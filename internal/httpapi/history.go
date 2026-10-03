package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/history"
)

func (s *server) listGenerations(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusBadRequest, "invalid_request", "limit must be between 1 and 100.")
			return
		}
		limit = n
	}
	var before int64
	if v := r.URL.Query().Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid_request", "before must be a generation id.")
			return
		}
		before = n
	}
	gens, err := s.History.List(r.Context(), limit+1, before)
	if err != nil {
		log.Printf("listing generations: %v", err)
		writeError(w, http.StatusInternalServerError, "internal", "Couldn't load the history.")
		return
	}
	// One extra row says whether another page exists.
	var next *int64
	if len(gens) > limit {
		gens = gens[:limit]
		id := gens[limit-1].ID
		next = &id
	}
	writeJSON(w, http.StatusOK, map[string]any{"generations": gens, "next_before": next})
}

func (s *server) deleteGeneration(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.History.Delete(r.Context(), id); err != nil {
		writeHistoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) favourites(w http.ResponseWriter, r *http.Request) {
	favs, err := s.History.Favourites(r.Context())
	if err != nil {
		writeHistoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"favourites": favs})
}

func (s *server) favourite(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		name, err := s.History.SetFavourite(r.Context(), id, on)
		if err != nil {
			writeHistoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, name)
	}
}

func writeHistoryError(w http.ResponseWriter, err error) {
	if errors.Is(err, history.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "That isn't in the history any more.")
		return
	}
	log.Printf("history: %v", err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong with the history.")
}
