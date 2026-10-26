package main

import (
	"log/slog"
	"testing"
)

func TestServeFlagDefaults(t *testing.T) {
	cmd := newRootCmd()
	if cmd == nil {
		t.Fatal("newRootCmd returned nil")
	}

	serveCmd, _, err := cmd.Find([]string{"serve"})
	if err != nil {
		t.Fatal("serve subcommand not found")
	}
	if serveCmd == nil {
		t.Fatal("serve subcommand is nil")
	}

	addrFlag := serveCmd.Flag("addr")
	if addrFlag == nil {
		t.Error("addr flag not found")
	}
	if addrFlag.DefValue != "127.0.0.1:8080" {
		t.Errorf("addr default = %q, want '127.0.0.1:8080'", addrFlag.DefValue)
	}

	dbFlag := cmd.PersistentFlags().Lookup("db")
	if dbFlag == nil {
		t.Error("db flag not found")
	}
}

func TestResolveLogLevel(t *testing.T) {
	cases := []struct {
		name         string
		debugChanged bool
		debug        bool
		levelChanged bool
		lvl          string
		want         slog.Level
		wantErr      bool
	}{
		{"defaults", false, false, false, "", slog.LevelWarn, false},
		{"debug flag only", true, true, false, "", slog.LevelDebug, false},
		{"debug flag false", true, false, false, "", slog.LevelWarn, false},
		{"level debug", false, false, true, "debug", slog.LevelDebug, false},
		{"level info", false, false, true, "info", slog.LevelInfo, false},
		{"level warn", false, false, true, "warn", slog.LevelWarn, false},
		{"level error", false, false, true, "error", slog.LevelError, false},
		{"combo fails", true, true, true, "warn", 0, true},
		{"invalid value", false, false, true, "verbose", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveLogLevel(tc.debugChanged, tc.debug, tc.levelChanged, tc.lvl)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveLogLevel() err = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveLogLevel() err = %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveLogLevel() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMigrateSubcommand(t *testing.T) {
	cmd := newRootCmd()
	migrateCmd, _, err := cmd.Find([]string{"migrate"})
	if err != nil {
		t.Fatal("migrate subcommand not found")
	}
	if migrateCmd == nil {
		t.Fatal("migrate subcommand is nil")
	}
}
