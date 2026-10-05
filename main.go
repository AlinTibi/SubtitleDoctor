package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"log"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := checkRuntime(); err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	app := NewApp()
	err := wails.Run(&options.App{Title: "Subtitle Doctor", Width: 1440, Height: 900, MinWidth: 1040, MinHeight: 640, AssetServer: &assetserver.Options{Assets: assets}, BackgroundColour: &options.RGBA{R: 18, G: 22, B: 29, A: 255}, OnStartup: app.startup, OnBeforeClose: app.beforeClose, Bind: []interface{}{app}, EnableDefaultContextMenu: true, DragAndDrop: &options.DragAndDrop{EnableFileDrop: true, DisableWebViewDrop: true}, Windows: &windows.Options{WebviewIsTransparent: false, WindowIsTranslucent: false, DisableWindowIcon: false}})
	if err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
}
