//go:build !windows && !darwin

package main

import (
	"fmt"
	"os"
)

// Заглушки для проверки на Linux (настоящая работа — только Windows и Mac).
func dataDir() string {
	d := os.Getenv("FOKUS_DIR")
	if d == "" {
		d = "/tmp/fokus"
	}
	_ = os.MkdirAll(d, 0700)
	return d
}
func exePath() string   { return dataDir() + "/fokus" }
func assetName() string { return "fokus-linux" }
func applyBlocks(a Active) {
	fmt.Println("apply: video", a.Video, "night", a.Night, "sites", len(a.Sites), "paths", len(a.Paths))
}
func killApps(names []string)    {}
func runningApps() []string      { return []string{"Telegram", "Steam"} }
func install()                   { fmt.Println("install: only windows/mac") }
func uninstallAll()              { fmt.Println("uninstall"); os.Exit(0) }
func replaceSelf(b []byte) error { return nil }
