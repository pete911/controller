package pkg

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Flags struct {
	LogLevel string
}

func (f Flags) SlogLevel() slog.Level {
	switch strings.ToUpper(f.LogLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	}
	return slog.LevelInfo
}

func ParseFlags() Flags {
	var flags Flags
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	f.StringVar(&flags.LogLevel, "log-level", getStringEnv("CTRL_LOG_LEVEL", "DEBUG"), "controller log level")

	if err := f.Parse(os.Args[1:]); err != nil {
		fmt.Printf("parse flags: %v", err)
		os.Exit(1)
	}
	if _, ok := map[string]struct{}{"DEBUG": {}, "INFO": {}, "WARN": {}, "ERROR": {}}[strings.ToUpper(flags.LogLevel)]; !ok {
		fmt.Printf("invalid log level %s", flags.LogLevel)
		os.Exit(1)
	}
	return flags
}

func getStringEnv(envName string, defaultValue string) string {
	if env, ok := os.LookupEnv(envName); ok {
		return env
	}
	return defaultValue
}
