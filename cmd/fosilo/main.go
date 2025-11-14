package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/siohaza/fosilo/internal/server"
	"github.com/siohaza/fosilo/pkg/config"

	"github.com/spf13/cobra"
)

var (
	configPath string
	logLevel   string
	version    = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:   "foslio",
	Short: "Fosilo - Ace of Spades v0.75 Dedicated Server",
	Long: `Fosilo is a production-grade Golang dedicated server for Ace of Spades v0.75
with full protocol support, Lua scripting, and multiple game modes.`,
	Version: version,
	Run:     runServer,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Fosilo server",
	Long:  "Start the Fosilo dedicated server with the specified configuration",
	Run:   runServer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Fosilo v%s\n", version)
		fmt.Println("Ace of Spades v0.75 Dedicated Server")
		fmt.Println("Built with Go")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "configs/config.toml", "path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "log level (debug, info, warn, error)")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

func runServer(cmd *cobra.Command, args []string) {
	level := slog.LevelInfo
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	var logWriter io.Writer = os.Stdout
	var logFile *os.File

	if cfg.Server.LogToFile {
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
			os.Exit(1)
		}

		timestamp := time.Now().Unix()
		logPath := filepath.Join(logDir, fmt.Sprintf("fosilo_%d.log", timestamp))

		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			os.Exit(1)
		}
		defer logFile.Close()

		logWriter = io.MultiWriter(os.Stdout, logFile)
	}

	logger := slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	logger.Info("starting foslio server", "version", version)

	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("server running",
		"name", cfg.Server.Name,
		"address", fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port),
		"gamemode", cfg.Server.Gamemode,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("shutting down server")

	srv.Stop()
	logger.Info("server stopped successfully")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
