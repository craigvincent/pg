// The explorer: the tree of a database's objects, their definitions, and
// what autocomplete offers.

package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"pg/internal/catalog"
	"pg/internal/dockerpg"
)

func (s *Server) tree(w http.ResponseWriter, r *http.Request) {
	cat := s.catalog()
	if cat == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	nodes, err := cat.Children(ctx, r.FormValue("kind"), r.FormValue("id"))
	render(w, r, TreeNodes(nodes, err))
}

func (s *Server) definition(w http.ResponseWriter, r *http.Request) {
	cat := s.catalog()
	if cat == nil {
		http.Error(w, "connect to a service first", http.StatusConflict)
		return
	}
	s.mu.Lock()
	settings, image := s.settings, s.image
	s.mu.Unlock()
	// pg_dump may be Docker's, which takes a few seconds to start.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	def, err := cat.Definition(ctx, r.FormValue("def"), r.FormValue("id"), func(ctx context.Context, pattern string) (string, error) {
		return dockerpg.Dump(ctx, settings, image, pattern)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, def)
}

func (s *Server) table(w http.ResponseWriter, r *http.Request) {
	cat := s.catalog()
	if cat == nil {
		http.Error(w, "connect to a service first", http.StatusConflict)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	table, err := cat.Table(ctx, r.FormValue("oid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(table)
}

func (s *Server) completions(w http.ResponseWriter, r *http.Request) {
	var done catalog.Completions
	if cat := s.catalog(); cat != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		var err error
		if done, err = cat.Completions(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(done)
}
