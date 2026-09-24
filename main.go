package main

import (
	"log"
	"net/http"

	"kinopoisk-auth/auth"
)

func main() {
	// main.go только "собирает" модули вместе — сам не содержит
	// бизнес-логики.
	repo := auth.NewInMemoryUserRepo()
	// Session = nil — модуль сессий/токенов (куки) делает другой
	// человек в команде. Когда он реализует auth.SessionIssuer,
	// достаточно будет передать реализацию сюда вторым аргументом.
	useCase := auth.NewUseCase(repo, nil)
	handler := auth.NewHandler(useCase)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", handler.Register)
	mux.HandleFunc("/api/login", handler.Login)

	withCORSHandler := withCORS(mux)

	log.Println("сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORSHandler))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
