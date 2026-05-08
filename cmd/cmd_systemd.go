package cmd

import (
	"crm-middleware/build"
	"crm-middleware/config"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func systemdInstall(...string) int {
	if os.Getuid() != 0 {
		stderr("must be run as root")
		return 1
	}

	stopIfRunning()

	exe, err := resolveExecutable()
	if err != nil {
		stderr("failed to resolve executable:", err)
		return 1
	}

	if err = copyExecutable(exe, filepath.Join("/usr/local/bin", build.ServiceName())); err != nil {
		stderr("failed to copy executable:", err)
		return 1
	}

	if err = config.Reload(); err != nil {
		stderr("load config:", err)
		return 1
	}

	if err = installUnitFile(config.Get().State.Dir); err != nil {
		stderr("installing systemd unit:", err)
		return 1
	}

	if err = exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		stderr("failed to reload systemd:", err)
		return 1
	}

	if err = exec.Command("systemctl", "enable", build.ServiceName()).Run(); err != nil {
		stderr("failed to enable service:", err)
		return 1
	}

	if err = exec.Command("systemctl", "start", build.ServiceName()).Run(); err != nil {
		stderr("failed to start service:", err)
		return 1
	}

	stderr(build.ServiceName(), build.Version(), "installed and started successfully")
	return 0
}

func systemdRemove(...string) int {
	if os.Getuid() != 0 {
		stderr("must be run as root")
		return 1
	}

	stopIfRunning()

	exec.Command("systemctl", "disable", build.ServiceName()).Run()

	unitLink := filepath.Join("/etc/systemd/system", build.ServiceName()+".service")
	os.Remove(unitLink)

	if err := config.Reload(); err == nil {
		os.Remove(filepath.Join(config.Get().State.Dir, build.ServiceName()+".service"))
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		stderr("failed to reload systemd:", err)
		return 1
	}

	exec.Command("systemctl", "reset-failed").Run()

	if err := os.Remove(filepath.Join("/usr/local/bin", build.ServiceName())); err != nil && !os.IsNotExist(err) {
		stderr("failed to remove executable:", err)
		return 1
	}

	stderr(build.ServiceName(), "removed successfully")
	return 0
}

func stopIfRunning() {
	if err := exec.Command("systemctl", "is-active", "--quiet", build.ServiceName()).Run(); err != nil {
		return
	}
	if err := exec.Command("systemctl", "stop", build.ServiceName()).Run(); err != nil {
		stderr("failed to stop running", build.ServiceName(), "service")
		return
	}
	stderr(build.ServiceName(), "service stopped successfully")
}

func resolveExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func installUnitFile(stateDir string) error {
	unitFile := filepath.Join(stateDir, build.ServiceName()+".service")
	unitLink := filepath.Join("/etc/systemd/system", build.ServiceName()+".service")

	if _, err := os.Stat(unitFile); os.IsNotExist(err) {
		if err = os.WriteFile(unitFile, systemdUnit(), 0644); err != nil {
			return fmt.Errorf("failed to write unit file: %w", err)
		}
	}

	os.Remove(unitLink)
	if err := os.Symlink(unitFile, unitLink); err != nil {
		return fmt.Errorf("failed to create unit symlink: %w", err)
	}
	return nil
}

func systemdUnit() []byte {
	return fmt.Appendf(nil, `[Unit]
Description=PBXware Integrations Middleware
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/var/lib/%s
ExecStart=/usr/local/bin/%s
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
`, build.ServiceName(), build.ServiceName())
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
