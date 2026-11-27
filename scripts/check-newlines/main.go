package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	log.SetFlags(0)

	if err := checkNewlines(); err != nil {
		log.Fatal(err)
	}
}

func checkNewlines() error {
	files, err := exec.Command("git", "ls-files", "--eol", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		return fmt.Errorf("list repository files: %w", err)
	}

	var failures []error

	for entry := range strings.SplitSeq(string(files), "\x00") {
		metadata, path, ok := strings.Cut(entry, "\t")
		if !ok || strings.Contains(metadata, "w/-text") {
			continue
		}

		if err := checkFile(path); err != nil {
			failures = append(failures, err)
		}
	}

	return errors.Join(failures...)
}

func checkFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("inspect %q: %w", path, err)
	}

	if !info.Mode().IsRegular() || info.Size() == 0 {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}

	if len(data) > 0 && data[len(data)-1] != '\n' {
		return fmt.Errorf("missing final newline: %s", path)
	}

	return nil
}
