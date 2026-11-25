package metrics

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Listen serves this process's registry on a dedicated HTTP listener.
func (m *Metrics) Listen(addr string) (func(), error) {
	if addr == "" {
		return func() {}, nil
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for metrics on %s: %w", addr, err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", m.Handler())
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		defer listener.Close()
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("serve metrics on %s: %v", addr, err)
		}
	}()
	return func() { _ = server.Close() }, nil
}
