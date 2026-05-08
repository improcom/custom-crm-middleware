package config

import (
	"crm-middleware/build"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

var config atomic.Pointer[Config]

// Default configuration.
func defaults() (conf *Config) {
	conf = new(Config)

	conf.Server.Port = 9996

	conf.Host.Scheme = "http"
	conf.Host.Domain = "localhost"

	conf.Logger.Level = levelWarn
	conf.Logger.File = true
	conf.Logger.MaxSize = 10
	conf.Logger.MaxBackups = 3

	if build.Dev {
		conf.Logger.Level = levelDebug
		conf.Logger.MaxSize = 50
		conf.Logger.MaxBackups = 2
	}

	return
}

func Get() *Config {
	return config.Load()
}

// Reload with configuration file from state directory.
// If configuration file doesn't exist, default configuration file is created in state directory.
// Existing configuration is untouched on error.
func Reload() error {
	var conf *Config = defaults()

	if build.Dev {
		var err error
		if conf.State.Dir, err = os.Getwd(); err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		conf.State.Dir = filepath.Join(conf.State.Dir, "."+build.ServiceName())
	} else {
		if os.Getuid() != 0 {
			return fmt.Errorf("%s must be run as root", build.ServiceName())
		}
		conf.State.Dir = filepath.Join("/var/lib", build.ServiceName())
	}

	conf.State.Filepath.Config = filepath.Join(conf.State.Dir, "config.yaml")
	conf.State.Filepath.Database = filepath.Join(conf.State.Dir, build.ServiceName()+".db")
	conf.State.Filepath.Log = filepath.Join(conf.State.Dir, "app.log")
	conf.Logger.Filepath = conf.State.Filepath.Log

	if data, err := os.ReadFile(conf.State.Filepath.Config); err == nil {
		if err = yaml.Unmarshal(data, conf); err != nil {
			return fmt.Errorf("parsing config %s: %w", conf.State.Filepath.Config, err)
		}
	} else {
		// Missing configuration file in state directory, create default configuration.
		if err = os.MkdirAll(conf.State.Dir, 0750); err != nil {
			return fmt.Errorf("create state dir %s: %w", conf.State.Dir, err)
		}
		rawDefault, err := yaml.Marshal(conf)
		if err != nil {
			return fmt.Errorf("marshal default config %s: %w", conf.State.Filepath.Config, err)
		}
		rawDefault = append([]byte("# Reload "+build.ServiceName()+" service after modifying this file.\n\n"), rawDefault...)
		if err = os.WriteFile(conf.State.Filepath.Config, rawDefault, 0600); err != nil {
			return fmt.Errorf("writing default config %s: %w", conf.State.Filepath.Config, err)
		}
	}

	config.Store(conf)

	return nil
}
