package main

import (
	"context"
	"fmt"
	"log"
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

			log.Printf("listening on %s", addr)
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

	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFile := viper.GetString("db")
			srv, err := app.NewServer(dbFile)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			if err := srv.DB.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			log.Println("migrations complete")
			return nil
		},
	}

	root.AddCommand(serve, migrate)

	return root
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
