//go:build windows

package main

import (
	"fmt"
	"github.com/wailsapp/go-webview2/webviewloader"
)

// Check before Wails starts so even its default download strategy cannot run.
func checkRuntime() error {
	version, e := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if e != nil {
		return fmt.Errorf("WebView2 check failed: %w", e)
	}
	if version == "" {
		return fmt.Errorf("Microsoft WebView2 Runtime is required. Install it separately before running Subtitle Doctor. No runtime will be downloaded")
	}
	cmp, e := webviewloader.CompareBrowserVersions(version, "94.0.992.31")
	if e != nil {
		return e
	}
	if cmp < 0 {
		return fmt.Errorf("WebView2 %s is too old; install a supported runtime separately", version)
	}
	return nil
}
