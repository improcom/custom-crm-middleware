package api

import (
	"crm-middleware/model/dataschema"
	"log/slog"

	"github.com/go-chi/chi/v5"
)

type AppV1 struct {
	Path     string
	Versions []string
	Objects  map[string]*dataschema.Model

	Config IntegrationConfig

	AuthHandlers   AuthHandlers
	ObjectHandlers ObjectHandlers
	SearchHandlers SearchHandlers
	TaskHandlers   TaskHandlers
}

func (app *AppV1) Mount(r chi.Router) {
	slog.Info("mounting app",
		slog.String("name", app.Config.Name),
		slog.String("path", app.Path),
		slog.Any("auth", app.Config.AuthMethod),
	)

	r.Route(app.Path+"/api/v1", func(r chi.Router) {
		r.Get("/token", tokenHandler)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			models := make([]*dataschema.Model, 0, len(app.Objects))
			for _, m := range app.Objects {
				models = append(models, m)
			}

			r.Route("/config", func(r chi.Router) {
				r.Get("/", configHandler(app.Config))
				r.Get("/versions", versionsHandlerFunc(app.Config.Name, app.Versions...))
				r.Get("/objects", objectTypesHandlerFunc(models...))
				r.Post("/custom-fields", updateCustomFieldsHandler)
			})

			r.Delete("/integration", clientDeletedHandler)

			r.Route("/auth", func(r chi.Router) {
				if app.AuthHandlers.Callback != nil {
					r.Get("/login_url", app.AuthHandlers.Login)
				} else {
					r.Post("/login", app.AuthHandlers.Login)
				}

				r.Get("/status", app.AuthHandlers.Status)
				r.Get("/logout", app.AuthHandlers.Logout)
			})

			r.Route("/object", func(r chi.Router) {
				r.Get("/describe/{object_type}", WithObject(describeObjectHandlerFunc(app.Objects)))
				r.Route("/{object_type}", func(r chi.Router) {
					r.Post("/", WithObject(app.ObjectHandlers.Create))
					r.Get("/{object_id}", WithObject(app.ObjectHandlers.Get))
					r.Put("/{object_id}", WithObject(app.ObjectHandlers.Update))
				})
			})

			r.Get("/search", app.SearchHandlers.All)
			r.Get("/associations", app.SearchHandlers.Associations)
			r.Post("/create-task", app.TaskHandlers.Create)
		})
	})

	if app.AuthHandlers.Callback != nil {
		r.Get(app.Path+CallbackEndpoint, app.AuthHandlers.Callback)
	}
}
