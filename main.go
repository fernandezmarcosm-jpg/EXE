package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var buildVersion = "dev"
var logFile *os.File

func logFilePath() string {
	if x, err := os.Executable(); err == nil && x != "" { return filepath.Join(filepath.Dir(x), "GestionSO_log.txt") }
	if d, err := os.Getwd(); err == nil && d != "" { return filepath.Join(d, "GestionSO_log.txt") }
	return filepath.Join(os.TempDir(), "GestionSO_log.txt")
}

func initFileLogger() {
	path := logFilePath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil { return }
	logFile = f
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("=== GestionSO V57 sesión iniciada === version=%s log=%s timestamp=%s", buildVersion, path, time.Now().Format(time.RFC3339))
}

func main() {
	initFileLogger()
	app := NewApp()
	err := wails.Run(&options.App{
		Title: "GestionSO V57",
		Width: 1400,
		Height: 850,
		MinWidth: 900,
		MinHeight: 500,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup: app.startup,
		Bind: []interface{}{app},
	})
	if err != nil { log.Fatal(err) }
}
