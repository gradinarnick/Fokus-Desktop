// Фокус для Windows и Mac: блок видео, ночной режим, правила для сайтов и программ.
// Ослабить что-либо можно только одноразовым кодом Google Authenticator владельца ключа.
package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

var version = "1.0.0" // подставляется при сборке

const listenAddr = "127.0.0.1:47890"
const uiURL = "http://127.0.0.1:47890/"

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "service":
		runService()
	case "install", "":
		install()
	case "version":
		fmt.Println(version)
	default:
		fmt.Println("fokus: install | service | version")
	}
}

func runService() {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return // уже запущен
	}
	loadState()
	go serveUI(ln)
	go updateLoop()
	tick := 0
	for {
		enforce(tick%200 == 0) // раз в ~10 минут — полная перезапись (на случай ручных правок)
		tick++
		time.Sleep(3 * time.Second)
	}
}

var lastKey string

// enforce применяет текущие блокировки. Пишет в систему только при изменениях.
func enforce(force bool) {
	stMu.Lock()
	a := computeActive(st, time.Now())
	stMu.Unlock()
	key := fmt.Sprint(a.Video, a.Night, a.Sites, a.Paths)
	if force || key != lastKey {
		applyBlocks(a)
		lastKey = key
	}
	killApps(a.Apps)
}
