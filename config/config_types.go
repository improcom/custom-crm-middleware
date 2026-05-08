package config

import (
	"fmt"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Port int `yaml:"port"`
}

type Host struct {
	Scheme string `yaml:"scheme"`
	Domain string `yaml:"domain"`
}

type Logger struct {
	Level      logLevel `yaml:"level"`
	File       bool     `yaml:"file"`
	Filepath   string   `yaml:"-"`
	Stdout     bool     `yaml:"stdout"`
	MaxSize    int      `yaml:"max_size"` // MB
	MaxBackups int      `yaml:"max_backups"`
}

type Apps struct {
	Salesforce struct {
		Domain   string `yaml:"domain"`
		Consumer struct {
			Key    string `yaml:"key"`
			Secret string `yaml:"secret"`
		} `yaml:"consumer"`
	} `yaml:"salesforce"`
}

type State struct {
	Dir      string `yaml:"-"`
	Filepath struct {
		Config   string
		Database string
		Log      string
	} `yaml:"-"`
}

type Config struct {
	Server Server `yaml:"server"`
	Host   Host   `yaml:"host"`
	Logger Logger `yaml:"logger"`
	Apps   Apps   `yaml:"apps"`
	State  State  `yaml:"-"`
}

// Accepts a number (0=error, 1=warn, 2=info, >=3=debug)
// or a case-insensitive keyword (abbreviation) in config.yaml.
type logLevel int

const (
	levelError logLevel = 0
	levelWarn  logLevel = 1
	levelInfo  logLevel = 2
	levelDebug logLevel = 3
)

func (l logLevel) String() string {
	switch l {
	case levelError:
		return "error"
	case levelWarn:
		return "warn"
	case levelInfo:
		return "info"
	case levelDebug:
		return "debug"
	}
	return levelWarn.String()
}

func (l logLevel) SLog() slog.Level {
	switch l {
	case levelError:
		return slog.LevelError
	case levelWarn:
		return slog.LevelWarn
	case levelInfo:
		return slog.LevelInfo
	case levelDebug:
		return slog.LevelDebug
	}
	return slog.LevelWarn
}

func (l logLevel) MarshalYAML() (any, error) {
	return l.String(), nil
}

func (l *logLevel) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("loglevel: expected a scalar value")
	}
	switch strings.ToLower(strings.TrimSpace(value.Value)) {
	case "0", "e", "err", "error":
		*l = levelError
	case "1", "w", "warn", "warning":
		*l = levelWarn
	case "2", "i", "info":
		*l = levelInfo
	case "3", "d", "dbg", "debug":
		*l = levelDebug
	default:
		*l = levelWarn
	}
	return nil
}
