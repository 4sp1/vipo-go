package main

import (
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
