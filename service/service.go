package service

import (
	"context"
	"crm-middleware/apps/salesforce"
	"crm-middleware/build"
	"crm-middleware/cmd"
	"crm-middleware/config"
	"crm-middleware/db"
	"crm-middleware/db/migrations"
	"crm-middleware/logger"
	"crm-middleware/model"
	"crm-middleware/repo"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 15 * time.Second

var stderr = logger.Stderr("service")

type Service struct {
	srv *http.Server
}

func (service *Service) Init() error {
	stderr("starting", build.ServiceName(), "(", build.Version(), ")")

	if err := config.Reload(); err != nil {
		stderr("failed to load configuration:", err)
		return err
	}

	var conf = config.Get()

	stderr("running at state directory:", conf.State.Dir)

	logger.Init()

	if err := db.Init(conf.State.Filepath.Database); err != nil {
		stderr("database initialization failed:", err)
		return err
	}

	if err := migrations.Run(); err != nil {
		stderr("migrations run failed:", err)
		return err
	}

	if build.Dev {
		// try to create default developer client if it doesn't exist
		_, _ = repo.Clients.Create("dev", model.ClientStatusNone)

		cmd.Run("version")

		// list available clients
		cmd.Run("client", "list")
	}

	salesforce.RefreshOAuthConfig()

	return nil
}

func (service *Service) StartHTTPServer(router *chi.Mux) {
	conf := config.Get()

	service.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Server.Port),
		Handler: router,
	}

	go func() {
		stderr("starting http server", conf.Host.Domain+":"+fmt.Sprint(conf.Server.Port))

		if err := service.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stderr("server failed:", err)
			os.Exit(1)
		}
	}()
}

func (service *Service) Reload() {
	stderr("reloading service -- refresh configuration")
	if err := config.Reload(); err != nil {
		stderr("config reload failed -- keeping existing config:", err)
		return
	}

	logger.Init()

	salesforce.RefreshOAuthConfig()
}

func (service *Service) Stop() {
	stderr("shutting down -- draining in-flight requests")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := service.srv.Shutdown(ctx); err != nil {
		stderr("graceful shutdown timed out:", err)
	}

	db.Close()
	stderr("shutdown complete")
}
