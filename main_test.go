package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEloStateDirUsesXDGStateHome(t *testing.T) {
	got := eloStateDir("/tmp/custom-state", "/home/user")
	want := filepath.Join("/tmp/custom-state", "elo")
	if got != want {
		t.Fatalf("eloStateDir() = %q, want %q", got, want)
	}
}

func TestEloStateDirFallsBackToUserStateDirectory(t *testing.T) {
	got := eloStateDir("", "/home/user")
	want := filepath.Join("/home/user", ".local", "state", "elo")
	if got != want {
		t.Fatalf("eloStateDir() = %q, want %q", got, want)
	}
}

func TestConfigureLoggingCreatesWritableLog(t *testing.T) {
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	})

	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)

	logPath, closeLog := configureLogging()
	log.Print("teste de escrita")
	closeLog()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("não foi possível ler o log criado: %v", err)
	}
	if !strings.Contains(string(content), "teste de escrita") {
		t.Fatalf("o log não contém a mensagem esperada: %q", content)
	}
}
