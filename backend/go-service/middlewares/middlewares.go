package middlewares

import (
	"net/http"
)

// WithCors aplica CORS permitiendo el origen http://localhost:3000 y las credenciales.
func WithCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Especifica el origen que deseas permitir:
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		// Permite el envío de cookies y otras credenciales:
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		// Permite los métodos necesarios:
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// Permite los headers que necesites (por ejemplo, Content-Type, Authorization)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Si es OPTIONS (preflight), devolvemos sin procesar la solicitud
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
