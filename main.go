package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
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
