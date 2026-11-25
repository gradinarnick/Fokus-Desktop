package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Rule — правило для сайта или программы.
type Rule struct {
	Target string `json:"target"` // домен (youtube.com) или программа (Telegram.exe / Telegram)
	Kind   string `json:"kind"`   // "site" | "app"
	Mode   string `json:"mode"`   // "always" | "night" | "sched"
	From   int    `json:"from"`   // минуты от полуночи (для sched)
	To     int    `json:"to"`
}

// State — всё, что «Фокус» помнит. Файл доступен только системе/администратору.
type State struct {
	Secret     string           `json:"secret"`
	LastStep   int64            `json:"lastStep"`
	VideoBlock bool             `json:"videoBlock"`
	Night      bool             `json:"night"`
	NightStart int              `json:"nightStart"`
	NightEnd   int              `json:"nightEnd"`
	Rules      []Rule           `json:"rules"`
	Unlock     map[string]int64 `json:"unlock"` // "all" | "video" -> до какого времени (unix)
}

var (
	stMu sync.Mutex
	st   *State
)

func statePath() string { return filepath.Join(dataDir(), "state.json") }

func loadState() {
	stMu.Lock()
	defer stMu.Unlock()
	s := &State{VideoBlock: true, Night: false, NightStart: 22 * 60, NightEnd: 7 * 60, Unlock: map[string]int64{}}
	if b, err := os.ReadFile(statePath()); err == nil {
		_ = json.Unmarshal(b, s)
	}
	if s.Unlock == nil {
		s.Unlock = map[string]int64{}
	}
	st = s
}

// saveLocked — вызывать под stMu.
func saveLocked() {
	b, _ := json.MarshalIndent(st, "", "  ")
	tmp := statePath() + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, statePath())
	}
}

func unlocked(s *State, scope string) bool {
	return s.Unlock[scope] > time.Now().Unix()
}

func minuteOfDay(t time.Time) int { return t.Hour()*60 + t.Minute() }

func inRange(t time.Time, from, to int) bool {
	if from == to {
		return false
	}
	m := minuteOfDay(t)
	if from < to {
		return m >= from && m < to
	}
	return m >= from || m < to
}

// Что сейчас действует.
type Active struct {
	Video bool
	Night bool
	Sites []string // домены, закрытые полностью
	Paths []string // адреса с путём (только Chrome)
	Apps  []string // программы, которые закрываются
}

func computeActive(s *State, now time.Time) Active {
	a := Active{}
	if unlocked(s, "all") {
		return a
	}
	a.Video = s.VideoBlock && !unlocked(s, "video")
	a.Night = s.Night && inRange(now, s.NightStart, s.NightEnd)
	seen := map[string]bool{}
	add := func(d string) {
		if !seen[d] {
			seen[d] = true
			a.Sites = append(a.Sites, d)
		}
	}
	if a.Video {
		for _, d := range videoDomains {
			add(d)
		}
		for _, d := range dohDomains {
			add(d)
		}
		a.Paths = append(a.Paths, videoPaths...)
		a.Apps = append(a.Apps, videoApps...)
	}
	if a.Night {
		for _, d := range nightDomains {
			add(d)
		}
		a.Apps = append(a.Apps, nightApps...)
	}
	for _, r := range s.Rules {
		on := r.Mode == "always" || (r.Mode == "night" && a.Night) || (r.Mode == "sched" && inRange(now, r.From, r.To))
		if !on {
			continue
		}
		if r.Kind == "app" {
			a.Apps = append(a.Apps, r.Target)
		} else if strings.Contains(r.Target, "/") {
			a.Paths = append(a.Paths, r.Target)
		} else {
			add(r.Target)
		}
	}
	return a
}
