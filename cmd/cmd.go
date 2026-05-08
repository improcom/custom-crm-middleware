package cmd

import (
	"crm-middleware/build"
	"crm-middleware/cmd/cli"
	"crm-middleware/config"
	"crm-middleware/db"
	"crm-middleware/logger"
	"os"
	"path/filepath"
)

var stderr = logger.Stderr()

func Run(args ...string) int {
	cli.Register(
		cli.Command("version", "", "show service version", func(...string) int {
			stderr(build.ServiceName(), "(", build.Version(), ")")
			return 0
		}),
	)

	cli.Register(
		cli.Group("client", "manage middleware clients",
			cli.Command("list", "", "show list of all middleware clients", withDatabase(clientList)),
			cli.Command("create", "<name> [--routing]", "create new client with <name> - (use 'routing' flag to create routing client)", withDatabase(clientCreate)),
			cli.Command("reset", "<id>", "reset client secret for <id> (invalidate active sessions)", withDatabase(clientReset)),
		),
	)

	if systemdAvailable() {
		cli.Register(
			cli.Group("systemd", "manage systemd service",
				cli.Command("install", "", "install/update and start executable as systemd service", systemdInstall),
				cli.Command("remove", "", "stop, disable and remove systemd service and executable", systemdRemove),
			),
		)
	}

	// allows commands to be run within the service excluding service name (command root)
	if len(args) > 0 && filepath.Base(args[0]) != build.ServiceName() {
		args = append([]string{build.ServiceName()}, args...)
	}

	return cli.Exec(args...)
}

func systemdAvailable() bool {
	info, err := os.Stat("/run/systemd/system")
	return err == nil && info.IsDir() && !build.Dev
}

func withDatabase(run func(...string) int) func(...string) int {
	return func(args ...string) int {
		if err := config.Reload(); err != nil {
			stderr("load config:", err)
			return 1
		}
		if err := db.Init(config.Get().State.Filepath.Database); err != nil {
			stderr("init database:", err)
			return 1
		}
		return run(args...)
	}
}
