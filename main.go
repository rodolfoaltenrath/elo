package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logPath, closeLog := configureLogging()
	defer closeLog()
	log.Printf("iniciando Elo")

	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("falha fatal durante a inicialização: %v\n%s", recovered, debug.Stack())
			fmt.Fprintf(os.Stderr, "Elo não pôde iniciar. Consulte o log em %s\n", logPath)
			os.Exit(1)
		}
	}()

	if err := run(); err != nil {
		log.Printf("falha ao executar o Elo: %v", err)
		fmt.Fprintf(os.Stderr, "Elo não pôde iniciar. Consulte o log em %s\n", logPath)
		os.Exit(1)
	}
}

func run() error {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	return wails.Run(&options.App{
		Title:  "Elo",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
}

func configureLogging() (string, func()) {
	logPath := filepath.Join(eloStateDir(os.Getenv("XDG_STATE_HOME"), userHomeDir()), "elo.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		log.Printf("não foi possível criar o diretório de log %s: %v", filepath.Dir(logPath), err)
		return logPath, func() {}
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("não foi possível abrir o arquivo de log %s: %v", logPath, err)
		return logPath, func() {}
	}

	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return logPath, func() {
		_ = logFile.Close()
	}
}

func eloStateDir(xdgStateHome, home string) string {
	if xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "elo")
	}
	if home != "" {
		return filepath.Join(home, ".local", "state", "elo")
	}
	return filepath.Join(os.TempDir(), "elo")
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
