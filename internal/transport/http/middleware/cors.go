package middleware

import "net/http"

// CORS is a minimal CORS handler suitable for the MVP. allowedOrigins may
// contain "*" to allow any origin.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAny := false
	allow := map[string]struct{}{}
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAny = true
		}
		allow[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAny {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if _, ok := allow[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
