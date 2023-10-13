package pkg

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Flags struct {
	Kubeconfig string
	Namespace  string
	LogLevel   string
	LogJson    bool
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
	f.StringVar(&flags.Kubeconfig, "kubeconfig", getStringEnv("KUBECONFIG", ""), "path to kubeconfig file, or empty for in-cluster kubeconfig")
	f.StringVar(&flags.Namespace, "namespace", getStringEnv("CTRL_NAMESPACE", "kube-system"), "controller namespace")
	f.StringVar(&flags.LogLevel, "log-level", getStringEnv("CTRL_LOG_LEVEL", "INFO"), "controller log level")
	f.BoolVar(&flags.LogJson, "log-json", getBoolEnv("CTRL_LOG_JSON", true), "json log format")

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

func getBoolEnv(envName string, defaultValue bool) bool {
	if env, ok := os.LookupEnv(envName); ok {
		out, err := strconv.ParseBool(env)
		if err != nil {
			fmt.Printf("parse bool %s env var: %v", envName, err)
			os.Exit(1)
		}
		return out
	}
	return defaultValue
}
