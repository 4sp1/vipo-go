package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/4sp1/vipo-go/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vipo-server",
		Short: "Vipo Pomodoro API server",
	}

	root.PersistentFlags().String("db", "vipo.db", "Path to SQLite database file")
	root.PersistentFlags().String("config", "", "Path to config.json file")

	viper.BindPFlag("db", root.PersistentFlags().Lookup("db"))
	viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			debugVal, dbErr := cmd.Flags().GetBool("debug")
			if dbErr != nil {
				return fmt.Errorf("read --debug flag: %w", dbErr)
			}
			levelVal, lvErr := cmd.Flags().GetString("log-level")
			if lvErr != nil {
				return fmt.Errorf("read --log-level flag: %w", lvErr)
			}
			level, err := resolveLogLevel(
				cmd.Flags().Changed("debug"), debugVal,
				cmd.Flags().Changed("log-level"), levelVal,
			)
			if err != nil {
				return err
			}
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

			dbFile := viper.GetString("db")
			addr := viper.GetString("addr")

			srv, err := app.NewServer(dbFile)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			if err := srv.DB.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			fmt.Printf("listening on %s\n", addr)
			httpServer := &http.Server{
				Addr:              addr,
				Handler:           srv.Handler,
				ReadHeaderTimeout: 10 * time.Second,
			}
			return httpServer.ListenAndServe()
		},
	}

	serve.Flags().String("addr", "127.0.0.1:8080", "Listen address")
	viper.BindPFlag("addr", serve.Flags().Lookup("addr"))

	serve.Flags().Bool("debug", false, "Enable debug logging (sugar for --log-level=debug)")
	serve.Flags().String("log-level", "warn", "Log level (debug|info|warn|error)")

	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			debugVal, dbErr := cmd.Flags().GetBool("debug")
			if dbErr != nil {
				return fmt.Errorf("read --debug flag: %w", dbErr)
			}
			levelVal, lvErr := cmd.Flags().GetString("log-level")
			if lvErr != nil {
				return fmt.Errorf("read --log-level flag: %w", lvErr)
			}
			level, err := resolveLogLevel(
				cmd.Flags().Changed("debug"), debugVal,
				cmd.Flags().Changed("log-level"), levelVal,
			)
			if err != nil {
				return err
			}
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

			dbFile := viper.GetString("db")
			srv, err := app.NewServer(dbFile)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			if err := srv.DB.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			fmt.Println("migrations complete")
			return nil
		},
	}

	migrate.Flags().Bool("debug", false, "Enable debug logging (sugar for --log-level=debug)")
	migrate.Flags().String("log-level", "warn", "Log level (debug|info|warn|error)")

	root.AddCommand(serve, migrate)

	return root
}

func resolveLogLevel(debugChanged, debug bool, levelChanged bool, lvl string) (slog.Level, error) {
	if debugChanged && levelChanged {
		return 0, errors.New("--debug and --log-level are mutually exclusive; use only one")
	}
	if debugChanged {
		if debug {
			return slog.LevelDebug, nil
		}
		return slog.LevelWarn, nil
	}
	value := "warn"
	if levelChanged {
		value = lvl
	}
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid --log-level %q: must be one of debug|info|warn|error", value)
	}
}


func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("VIPD")
	viper.AutomaticEnv()
}
