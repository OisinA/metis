package api

import (
	"encoding/json"
	"metis/pkg/provider"
	"net/http"

	"github.com/Strum355/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/spf13/viper"
)

const (
	VERSION = "0.1.0"
)

type API struct {
	serviceProvider provider.Provider
}

func NewAPI(provider provider.Provider) API {
	return API{serviceProvider: provider}
}

func (a *API) Register(r chi.Router) {
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Token")

			if token == viper.GetString("metis.secret") {
				next.ServeHTTP(w, r)
				return
			}

			w.WriteHeader(401)
			return
		})
	})
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			"METIS-AGENT", VERSION,
		})
		if err != nil {
			log.WithError(err).Error("Could not send API response")
			return
		}
	})

	r.Post("/service", a.CreateService)
	r.Post("/service/health", a.ServiceHealth)
	r.Post("/service/destroy", a.DestroyService)
}
