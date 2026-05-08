package logger

import (
	"crm-middleware/config"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Init() {
	var (
		writer io.Writer = os.Stdout
		conf             = config.Get().Logger
	)
	if !conf.Stdout {
		writer = io.Discard
	}

	if conf.File {
		var rotateWriter = &lumberjack.Logger{
			Filename:   conf.Filepath,
			MaxSize:    conf.MaxSize,
			MaxBackups: conf.MaxBackups,
		}

		if conf.Stdout {
			writer = io.MultiWriter(os.Stdout, rotateWriter)
		} else {
			writer = rotateWriter
		}
	}

	level := conf.Level.SLog()
	slog.SetDefault(slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	})))
}

// Stderr returns a function that writes prefixed lines to stderr.
func Stderr(args ...string) func(...any) {
	if len(args) == 0 {
		return func(args ...any) { fmt.Fprintln(os.Stderr, args...) }
	}
	prefix := "[" + strings.Join(args, " ") + "]"
	return func(args ...any) { fmt.Fprintln(os.Stderr, append([]any{prefix}, args...)...) }
}
