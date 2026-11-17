package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	width, height := 1280, 800
	if cfg := loadConfig(); cfg.WindowWidth >= 900 && cfg.WindowHeight >= 600 {
		width = cfg.WindowWidth
		height = cfg.WindowHeight
	}

	err := wails.Run(&options.App{
		Title:     "Clear Signal",
		Width:     width,
		Height:    height,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 9, G: 9, B: 11, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
			app.requestService,
			app.workspaceService,
			app.openapiService,
			app.importService,
			app.bundleService,
			app.settingsService,
			app.demoService,
			app.scenarioService,
			app.helpService,
			app.cliBuilder,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
