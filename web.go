package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

//go:embed ui
var uiFiles embed.FS

var pendingSecret string // новый ключ, ещё не подтверждённый кодом

func serveUI(ln net.Listener) {
	sub, _ := fs.Sub(uiFiles, "ui")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", apiState)
	mux.HandleFunc("/api/do", apiDo)
	mux.HandleFunc("/api/apps", apiApps)
	srv := &http.Server{Handler: guard(mux), ReadTimeout: 20 * time.Second}
	_ = srv.Serve(ln)
}

// guard: только с этого компьютера и только со страницы «Фокуса» (защита от чужих сайтов).
func guard(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != listenAddr && r.Host != "localhost:47890" {
			http.Error(w, "forbidden", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get("X-Fokus") != "1" {
			http.Error(w, "forbidden", 403)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func stateView() map[string]any {
	now := time.Now()
	a := computeActive(st, now)
	unl := map[string]int64{}
	for k, v := range st.Unlock {
		if v > now.Unix() {
			unl[k] = v
		}
	}
	v := map[string]any{
		"version":      version,
		"os":           runtime.GOOS,
		"keySet":       st.Secret != "",
		"video":        st.VideoBlock,
		"night":        st.Night,
		"nightStart":   st.NightStart,
		"nightEnd":     st.NightEnd,
		"rules":        st.Rules,
		"unlocks":      unl,
		"activeVideo":  a.Video,
		"activeNight":  a.Night,
		"blockedSites": len(a.Sites),
	}
	if pendingSecret != "" {
		v["pendingKey"] = prettyKey(pendingSecret)
		v["pendingURI"] = "otpauth://totp/Fokus:" + runtime.GOOS + "?secret=" + pendingSecret + "&issuer=Fokus"
	}
	return v
}

func apiState(w http.ResponseWriter, r *http.Request) {
	stMu.Lock()
	defer stMu.Unlock()
	writeJSON(w, stateView())
}

func apiApps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, runningApps())
}

type doReq struct {
	Action  string `json:"action"`
	Code    string `json:"code"`
	Scope   string `json:"scope"`
	Minutes int    `json:"minutes"`
	From    int    `json:"from"`
	To      int    `json:"to"`
	Index   int    `json:"index"`
	Rule    Rule   `json:"rule"`
}

func apiDo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "post", 405)
		return
	}
	var q doReq
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&q) != nil {
		http.Error(w, "bad", 400)
		return
	}
	stMu.Lock()
	res, uninstall := doAction(q)
	if res == "" {
		saveLocked()
	}
	view := stateView()
	stMu.Unlock()
	if res != "" {
		view["error"] = res
	}
	writeJSON(w, view)
	if uninstall {
		go func() {
			time.Sleep(time.Second)
			uninstallAll()
		}()
		return
	}
	go enforce(true)
}

// code проверяет код владельца ключа. Пока ключ не настроен — всё разрешено.
func codeOK(q doReq) bool {
	if st.Secret == "" {
		return true
	}
	return verifyLocked(st.Secret, q.Code)
}

const badCode = "Код неверный или уже использован. Попроси новый."

func doAction(q doReq) (errText string, uninstall bool) {
	switch q.Action {
	case "key_new":
		if st.Secret != "" && !codeOK(q) {
			return badCode, false
		}
		pendingSecret = newSecret()
	case "key_cancel":
		pendingSecret = ""
	case "key_confirm":
		if pendingSecret == "" || !verifyLocked(pendingSecret, q.Code) {
			return "Код не подошёл. Проверь, что ключ введён верно и время на телефоне автоматическое.", false
		}
		st.Secret = pendingSecret
		pendingSecret = ""
	case "video_on":
		st.VideoBlock = true
	case "video_off":
		if !codeOK(q) {
			return badCode, false
		}
		st.VideoBlock = false
	case "night_on":
		st.Night = true
	case "night_off":
		if !codeOK(q) {
			return badCode, false
		}
		st.Night = false
	case "night_hours":
		if !codeOK(q) {
			return badCode, false
		}
		if q.From < 0 || q.From >= 1440 || q.To < 0 || q.To >= 1440 {
			return "Неверное время", false
		}
		st.NightStart, st.NightEnd = q.From, q.To
	case "rule_add":
		t := strings.ToLower(strings.TrimSpace(q.Rule.Target))
		if q.Rule.Kind == "site" {
			t = strings.TrimPrefix(strings.TrimPrefix(t, "https://"), "http://")
			t = strings.TrimPrefix(t, "www.")
			if i := strings.IndexAny(t, "?#"); i >= 0 {
				t = t[:i]
			}
			t = strings.TrimSuffix(t, "/")
		} else {
			t = strings.TrimSpace(q.Rule.Target)
		}
		if t == "" || (q.Rule.Kind != "site" && q.Rule.Kind != "app") {
			return "Пустое правило", false
		}
		if q.Rule.Mode != "always" && q.Rule.Mode != "night" && q.Rule.Mode != "sched" {
			q.Rule.Mode = "always"
		}
		q.Rule.Target = t
		st.Rules = append(st.Rules, q.Rule)
	case "rule_del":
		if q.Index < 0 || q.Index >= len(st.Rules) {
			return "Нет такого правила", false
		}
		if !codeOK(q) {
			return badCode, false
		}
		st.Rules = append(st.Rules[:q.Index], st.Rules[q.Index+1:]...)
	case "unlock":
		if q.Scope != "all" && q.Scope != "video" {
			return "Неверный выбор", false
		}
		if q.Minutes < 5 || q.Minutes > 240 {
			return "Неверное время", false
		}
		if !codeOK(q) {
			return badCode, false
		}
		st.Unlock[q.Scope] = time.Now().Add(time.Duration(q.Minutes) * time.Minute).Unix()
	case "lock":
		delete(st.Unlock, q.Scope)
	case "uninstall":
		if !codeOK(q) {
			return badCode, false
		}
		return "", true
	default:
		return "Неизвестное действие", false
	}
	return "", false
}
