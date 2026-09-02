/*
 * Created on Wed Mar 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"yoyo-judge/server"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type BuildRouterRequest struct {
	Store *server.Store
}

// basePath returns the URL prefix everything is served under: "" for a
// same-origin deploy (the embedded single-binary mode, e.g. behind a
// reverse proxy that preserves the whole path, or accessed directly), or a
// path like "/yoyojudge" when the frontend is instead hosted separately
// (e.g. pasted into an existing nginx docroot this app doesn't control the
// config of) and calls this backend directly, cross-origin, at a matching
// prefix — mirrors ../orkestrator-v2's BASE_PATH env var.
func basePath() string {
	bp := os.Getenv("BASE_PATH")
	if bp == "" {
		bp = "/yoyojudge"
	}
	return strings.TrimSuffix(bp, "/")
}

// allowedOrigins is the CORS allowlist: the built-in defaults plus any
// extra origins from the comma-separated CORS_ALLOWED_ORIGINS env var —
// needed for the cross-origin split-deploy shape, where the frontend
// (served by e.g. nginx from a docroot) and this backend (running
// standalone on its own port) are different origins.
func allowedOrigins() []string {
	origins := []string{
		"http://localhost:3000", "https://app.mimo.id", "https://app-staging.mimo.id",
		"http://localhost:5173", // Vite dev server default port
		"https://rizkiyoist.duckdns.org",
	}
	if extra := os.Getenv("CORS_ALLOWED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	}
	return origins
}

// firstExisting returns envValue if set, else fallbackPath if that file
// exists on disk, else "".
func firstExisting(envValue, fallbackPath string) string {
	if envValue != "" {
		return envValue
	}
	if _, err := os.Stat(fallbackPath); err == nil {
		return fallbackPath
	}
	return ""
}

// port is the TCP port to listen on, overridable via the PORT env var
// (mirrors ../orkestrator-v2's own PORT variable, which defaults to 8080
// there — 8081 here so the two can run on the same box without colliding).
func port() string {
	p := os.Getenv("PORT")
	if p == "" {
		p = "8081"
	}
	return p
}

func BuildRouter(br *BuildRouterRequest) {
	bp := basePath()

	router := mux.NewRouter()
	router.HandleFunc(bp+"/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server.Mount(router, br.Store, bp)

	// Serve the embedded frontend (frontend/dist, embedded via static.go)
	// under the same prefix, with manual SPA fallback: a request for a path
	// with a file extension is served as a static asset; anything else
	// (a client-side route) gets index.html so Vue Router can take over.
	// This must be registered last so it doesn't shadow the routes above.
	// Unused when the frontend is instead served directly by nginx from its
	// own docroot (the cross-origin deployment shape) — harmless either way.
	distFS, err := fs.Sub(staticFiles, "bin/static")
	if err != nil {
		log.Fatal("failed to load embedded frontend: ", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, bp)
		if path == "" {
			path = "/"
		}
		r2 := r.Clone(r.Context())
		if strings.Contains(path, ".") {
			r2.URL.Path = path
			fileServer.ServeHTTP(w, r2)
			return
		}
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
	if bp != "" {
		// PathPrefix(bp+"/") alone wouldn't match the bare prefix with no
		// trailing slash (e.g. a request for exactly "/yoyojudge").
		router.Handle(bp, spaHandler)
	}
	router.PathPrefix(bp + "/").Handler(spaHandler)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	// Serving TLS directly here (instead of only via a reverse proxy) lets
	// this backend be called from an HTTPS page without hitting the
	// browser's mixed-content block, even when it's running standalone on
	// its own port rather than behind a proxy that terminates TLS for it.
	// TLS_CERT_FILE/TLS_KEY_FILE override the path; otherwise cert.pem/
	// key.pem next to the binary are used automatically if present, so
	// running it is just "drop the cert files next to it and start it" -
	// no env vars to remember to set on every launch.
	certFile := firstExisting(os.Getenv("TLS_CERT_FILE"), "cert.pem")
	keyFile := firstExisting(os.Getenv("TLS_KEY_FILE"), "key.pem")
	addr := ":" + port()
	if certFile != "" && keyFile != "" {
		fmt.Println("serving HTTPS on", addr, "using", certFile, "/", keyFile)
		err = http.ListenAndServeTLS(addr, certFile, keyFile, corsHandler)
	} else {
		fmt.Println("serving HTTP on", addr)
		err = http.ListenAndServe(addr, corsHandler)
	}
	if err != nil {
		fmt.Println("failed to start server:", err)
	}
}
