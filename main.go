package main

import (
	"embed"
	"log"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

const buildMarker = "2026-09-22-calc-sign-diagnostic"

func main() {
	log.Printf("[BUILD] marker=%s built=%s", buildMarker, time.Now().Format(time.RFC3339Nano))
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
