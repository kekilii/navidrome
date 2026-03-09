package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/openlist"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type openListConfigPayload struct {
	ID            string `json:"id"`
	Enabled       bool   `json:"enabled"`
	OpenListBase  string `json:"openlistBase"`
	OpenListUser  string `json:"openlistUser"`
	OpenListPass  string `json:"openlistPass"`
	CoverEnabled  bool   `json:"coverEnabled"`
	StreamEnabled bool   `json:"streamEnabled"`
}

type openListStreamPayload struct {
	RawURL string `json:"rawUrl"`
}

func (api *Router) addOpenListRoute(r chi.Router) {
	r.Route("/openlist", func(r chi.Router) {
		r.With(adminOnlyMiddleware).Get("/{id}", getOpenListConfig())
		r.With(adminOnlyMiddleware).Put("/{id}", updateOpenListConfig(api.ds))
		r.Get("/stream/{id}", resolveOpenListStream(api.ds))
	})
}

func getOpenListConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id != openlist.RecordID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		cfg := openlist.Current()
		resp := openListConfigPayload{
			ID:            openlist.RecordID,
			Enabled:       cfg.Enabled,
			OpenListBase:  cfg.OpenListBase,
			OpenListUser:  cfg.OpenListUser,
			OpenListPass:  "",
			CoverEnabled:  cfg.CoverEnabled,
			StreamEnabled: cfg.StreamEnabled,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func updateOpenListConfig(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id != openlist.RecordID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var payload openListConfigPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cfg, err := openlist.Update(ds, openlist.Config{
			Enabled:       payload.Enabled,
			OpenListBase:  payload.OpenListBase,
			OpenListUser:  payload.OpenListUser,
			OpenListPass:  payload.OpenListPass,
			CoverEnabled:  payload.CoverEnabled,
			StreamEnabled: payload.StreamEnabled,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := openListConfigPayload{
			ID:            openlist.RecordID,
			Enabled:       cfg.Enabled,
			OpenListBase:  cfg.OpenListBase,
			OpenListUser:  cfg.OpenListUser,
			OpenListPass:  "",
			CoverEnabled:  cfg.CoverEnabled,
			StreamEnabled: cfg.StreamEnabled,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func resolveOpenListStream(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		target, err := resolveOpenListStreamURL(r.Context(), ds, id)
		if err != nil {
			log.Debug(r.Context(), "OpenList stream resolve failed", "id", id, err)
			target = ""
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openListStreamPayload{RawURL: target})
	}
}

func resolveOpenListStreamURL(ctx context.Context, ds model.DataStore, id string) (string, error) {
	return openlist.ResolveStreamRawURLBySongID(ctx, ds, id)
}
