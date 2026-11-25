package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/gradinarnick/Fokus-Desktop/releases/latest"
const downloadPrefix = "https://github.com/gradinarnick/Fokus-Desktop/releases/download/"

func updateLoop() {
	time.Sleep(90 * time.Second)
	for {
		checkUpdate()
		time.Sleep(3 * time.Hour)
	}
}

func verNum(v string) int {
	v = strings.TrimPrefix(v, "v")
	i := strings.LastIndex(v, ".")
	n, _ := strconv.Atoi(v[i+1:])
	return n
}

// checkUpdate: есть ли на GitHub версия новее — скачать и поставить.
func checkUpdate() {
	client := &http.Client{Timeout: 2 * time.Minute}
	req, _ := http.NewRequest("GET", releaseAPI, nil)
	req.Header.Set("User-Agent", "fokus/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var rel struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.NewDecoder(resp.Body).Decode(&rel) != nil || verNum(rel.Tag) <= verNum(version) {
		return
	}
	url := ""
	for _, a := range rel.Assets {
		if a.Name == assetName() {
			url = a.URL
		}
	}
	if !strings.HasPrefix(url, downloadPrefix) {
		return
	}
	r2, err := client.Get(url)
	if err != nil {
		return
	}
	defer r2.Body.Close()
	bin, err := io.ReadAll(io.LimitReader(r2.Body, 100<<20))
	if err != nil || r2.StatusCode != 200 || len(bin) < 1<<20 {
		return
	}
	// Проверка, что это исполняемый файл нужной системы.
	if runtime.GOOS == "windows" && !bytes.HasPrefix(bin, []byte("MZ")) {
		return
	}
	if runtime.GOOS == "darwin" && !bytes.HasPrefix(bin, []byte{0xCF, 0xFA, 0xED, 0xFE}) {
		return
	}
	if replaceSelf(bin) == nil {
		os.Exit(0) // служба перезапустится уже с новой версией
	}
}
