package main

import (
	"crm-middleware/api"
	"crm-middleware/apps/salesforce"
	"crm-middleware/cmd"
	"crm-middleware/httputil"
	srv "crm-middleware/service"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
)

var service = new(srv.Service)

func init() {
	if len(os.Args) > 1 {
		os.Exit(cmd.Run(os.Args...))
	}

	if err := service.Init(); err != nil {
		os.Exit(1)
	}
}

func main() {
	router := chi.NewRouter()
	router.Use(httputil.RequestLoggerMiddleware)
	router.NotFound(httputil.UnknownRouteHandler)

	salesforce.App(api.AuthMethodOAuth2).Mount(router)
	salesforce.App(api.AuthMethodBasicAuth).Mount(router)
	salesforce.App(api.AuthMethodNone).Mount(router)

	salesforce.RouterCustomSource(router)

	service.StartHTTPServer(router)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	for sig := range sigs {
		switch sig {
		case syscall.SIGHUP:
			service.Reload()

		case syscall.SIGTERM, syscall.SIGINT:
			service.Stop()
			return
		}
	}
}
