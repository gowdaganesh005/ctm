package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	api        *API
	hub        *SSEHub
	httpServer *http.Server
	port       int
	staticDir  string
}

func NewServer(api *API, hub *SSEHub, port int, staticDir string) *Server {
	return &Server{
		api:       api,
		hub:       hub,
		port:      port,
		staticDir: staticDir,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/status", s.cors(s.api.HandleStatus))
	mux.HandleFunc("/api/revisions", s.cors(s.api.HandleListRevisions))
	mux.HandleFunc("/api/revisions/", s.cors(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/diff") {
			s.api.HandleGetRevisionDiff(w, r)
		} else {
			s.api.HandleGetRevision(w, r)
		}
	}))
	mux.HandleFunc("/api/manual-changes", s.cors(s.api.HandleManualChanges))
	mux.HandleFunc("/api/timetravel", s.cors(s.api.HandleTimeTravel))
	mux.HandleFunc("/api/namespaces", s.cors(s.api.HandleNamespaces))
	mux.HandleFunc("/api/resources/", s.cors(s.api.HandleResourceHistory))
	mux.HandleFunc("/api/incidents", s.cors(s.api.HandleListIncidents))
	mux.HandleFunc("/api/incidents/", s.cors(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/rca") {
			s.api.HandleIncidentRCA(w, r)
		} else {
			s.api.HandleGetIncident(w, r)
		}
	}))
	mux.HandleFunc("/api/scenarios/run", s.cors(s.api.HandleRunScenario))
	mux.HandleFunc("/api/stream", s.hub.HandleSSE)

	// SPA Static File Serving
	mux.HandleFunc("/", s.spaHandler())

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("[Server] Cluster Time Machine listening on http://%s", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (s *Server) spaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.staticDir == "" {
			writeJSON(w, http.StatusOK, map[string]string{
				"app":     "Cluster Time Machine API",
				"version": "1.0.0",
				"docs":    "/api/status",
			})
			return
		}

		path := filepath.Join(s.staticDir, filepath.Clean(r.URL.Path))
		info, err := os.Stat(path)
		if os.IsNotExist(err) || (err == nil && info.IsDir()) {
			// Fallback to index.html for SPA client-side routing
			http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
			return
		}

		http.ServeFile(w, r, path)
	}
}
