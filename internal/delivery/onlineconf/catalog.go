package onlineconf

import (
	_ "embed"
	"net/http"
)

//go:generate go run ../../../cmd/cataloggen
//go:embed catalog.json
var builtins []byte

func serveCatalog(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(builtins)
}
