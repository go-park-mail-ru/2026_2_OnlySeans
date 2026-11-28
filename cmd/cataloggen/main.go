// Generate the admin inventory from versioned defaults, never from process environments.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type entry struct {
	Scope  string `json:"scope"`
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Source string `json:"source"`
	Group  string `json:"group"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	root, err := os.Getwd()
	must(err)
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			panic("go.mod missing")
		}
		root = parent
	}
	must(os.Chdir(root))
	var entries []entry
	paths, err := filepath.Glob("configs/*.yaml")
	must(err)
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		must(err)
		var fields map[string]any
		must(yaml.Unmarshal(raw, &fields))
		scope := strings.TrimSuffix(filepath.Base(p), ".yaml")
		if scope == "config" {
			scope = "backend"
		}
		var walk func(string, map[string]any)
		walk = func(prefix string, fields map[string]any) {
			for key, v := range fields {
				if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "access_key") {
					continue
				}
				name := prefix + key
				if nested, ok := v.(map[string]any); ok {
					walk(name+".", nested)
					continue
				}
				entries = append(entries, entry{scope, name, v, filepath.ToSlash(p), "YAML"})
			}
		}
		walk("", fields)
	}
	dirs := []string{"internal/domain", "internal/domain/onlineconf", "internal/config", "pkg/utils"}
	for _, pattern := range []string{"internal/services/*", "internal/usecase/*", "internal/delivery/*"} {
		matches, err := filepath.Glob(pattern)
		must(err)
		dirs = append(dirs, matches...)
	}
	for _, dir := range dirs {
		productDefaults := strings.HasPrefix(filepath.ToSlash(dir), "internal/domain") || dir == "internal/config" || dir == "pkg/utils"
		files, err := filepath.Glob(dir + "/*.go")
		must(err)
		for _, p := range files {
			if strings.HasSuffix(p, "_test.go") {
				continue
			}
			set := token.NewFileSet()
			f, err := parser.ParseFile(set, p, nil, 0)
			must(err)
			for _, decl := range f.Decls {
				d, ok := decl.(*ast.GenDecl)
				seedFile := strings.HasSuffix(p, "seed.go") && d != nil && d.Tok == token.VAR
				if !ok || (d.Tok != token.CONST && !seedFile) {
					continue
				}
				for _, spec := range d.Specs {
					v := spec.(*ast.ValueSpec)
					for i, n := range v.Names {
						if i >= len(v.Values) || strings.HasPrefix(n.Name, "Env") {
							continue
						}
						var buf bytes.Buffer
						must(format.Node(&buf, set, v.Values[i]))
						raw := buf.String()
						if !productDefaults && !seedFile {
							// Collect service limits and timing defaults, without embedding SQL or transport internals.
							if !strings.ContainsAny(raw, "0123456789") && !strings.HasPrefix(raw, "time.") {
								continue
							}
							if strings.ContainsAny(raw, "\"`") {
								continue
							}
						}
						var val any = raw
						if s, err := strconv.Unquote(raw); err == nil {
							val = s
						} else if num, err := strconv.ParseInt(raw, 0, 64); err == nil {
							val = num
						}
						scope := strings.TrimSuffix(filepath.Base(p), ".go")
						if !productDefaults {
							scope = filepath.Base(dir)
						}
						entries = append(entries, entry{scope, n.Name, val, fmt.Sprintf("%s:%d", filepath.ToSlash(p), set.Position(n.Pos()).Line), "Go"})
					}
				}
			}
		}
	}
	seed, err := os.ReadFile("cmd/cataloggen/seed.json")
	must(err)
	var tables map[string]any
	must(json.Unmarshal(seed, &tables))
	for key, rows := range tables {
		entries = append(entries, entry{"catalog", key, rows, "db/migrations · " + key, "SQL"})
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		return a.Scope+"."+a.Key+a.Source < b.Scope+"."+b.Key+b.Source
	})
	data, err := json.MarshalIndent(entries, "", "  ")
	must(err)
	must(os.WriteFile("internal/delivery/onlineconf/catalog.json", append(data, '\n'), 0644))
}
