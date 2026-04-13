package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

var defaultCORSAllowHeaders = []string{
	"Accept",
	"Authorization",
	"Content-Type",
	"Origin",
	"X-Requested-With",
	"X-Qorvi-Plan",
	"X-Flowintel-Plan",
	"X-Whalegraph-Plan",
}

var defaultCORSAllowMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		normalized := normalizeCORSOrigin(origin)
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	allowMethods := strings.Join(defaultCORSAllowMethods, ", ")
	defaultAllowHeaders := strings.Join(defaultCORSAllowHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := normalizeCORSOrigin(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				headers := w.Header()
				headers.Set("Access-Control-Allow-Origin", origin)
				headers.Set("Access-Control-Allow-Credentials", "true")
				headers.Set("Access-Control-Allow-Methods", allowMethods)
				headers.Set("Access-Control-Max-Age", "600")
				headers.Add("Vary", "Origin")
				headers.Add("Vary", "Access-Control-Request-Method")
				headers.Add("Vary", "Access-Control-Request-Headers")

				requestHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
				if requestHeaders == "" {
					headers.Set("Access-Control-Allow-Headers", defaultAllowHeaders)
				} else {
					headers.Set("Access-Control-Allow-Headers", requestHeaders)
				}
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loadCORSAllowedOriginsFromEnv() []string {
	origins := []string{
		os.Getenv("APP_BASE_URL"),
		os.Getenv("NEXT_PUBLIC_APP_BASE_URL"),
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:3001",
		"http://127.0.0.1:3001",
	}

	extra := strings.Split(strings.TrimSpace(os.Getenv("QORVI_CORS_ALLOWED_ORIGINS")), ",")
	origins = append(origins, extra...)

	seen := map[string]struct{}{}
	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		normalized := normalizeCORSOrigin(origin)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	return out
}

func normalizeCORSOrigin(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}
