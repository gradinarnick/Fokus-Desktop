//go:build darwin

package main

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func dataDir() string   { return "/Library/Fokus" }
func exePath() string   { return filepath.Join(dataDir(), "fokus") }
func assetName() string { return "fokus-mac-" + runtime.GOARCH }

func run(name string, args ...string) error { return exec.Command(name, args...).Run() }

const hostsPath = "/etc/hosts"
const hostsBegin = "# >>> Fokus >>>"
const hostsEnd = "# <<< Fokus <<<"
const chromePlist = "/Library/Managed Preferences/com.google.Chrome.plist"
const daemonPlist = "/Library/LaunchDaemons/com.kolya.fokus.plist"

func stripSection(txt, begin, end string) string {
	for {
		i := strings.Index(txt, begin)
		if i < 0 {
			return txt
		}
		j := strings.Index(txt[i:], end)
		if j < 0 {
			return txt[:i]
		}
		k := i + j + len(end)
		if k < len(txt) && txt[k] == '\n' {
			k++
		}
		txt = txt[:i] + txt[k:]
	}
}

// applyBlocks: /etc/hosts (для всех программ) и политики Chrome (адреса с путём, инкогнито, DNS).
func applyBlocks(a Active) {
	b, _ := os.ReadFile(hostsPath)
	old := string(b)
	txt := stripSection(old, hostsBegin, hostsEnd)
	if len(a.Sites) > 0 {
		var sb strings.Builder
		if !strings.HasSuffix(txt, "\n") && txt != "" {
			sb.WriteString("\n")
		}
		sb.WriteString(hostsBegin + "\n")
		for _, d := range a.Sites {
			hosts := []string{d}
			if !strings.HasPrefix(d, "www.") {
				hosts = append(hosts, "www."+d, "m."+d)
			}
			for _, h := range hosts {
				sb.WriteString("0.0.0.0 " + h + "\n:: " + h + "\n")
			}
		}
		sb.WriteString(hostsEnd + "\n")
		txt += sb.String()
	}
	if txt != old {
		_ = os.WriteFile(hostsPath, []byte(txt), 0644)
		_ = run("dscacheutil", "-flushcache")
		_ = run("killall", "-HUP", "mDNSResponder")
	}

	list := append(append([]string{}, a.Sites...), a.Paths...)
	if a.Video {
		list = append(list, "chrome://flags", "chrome://net-internals")
	}
	if len(list) == 0 && !a.Video && !a.Night {
		if _, err := os.Stat(chromePlist); err == nil {
			_ = os.Remove(chromePlist)
			_ = run("killall", "cfprefsd")
		}
		return
	}
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	sb.WriteString("<key>URLBlocklist</key>\n<array>\n")
	for _, d := range list {
		sb.WriteString("<string>" + html.EscapeString(d) + "</string>\n")
	}
	sb.WriteString("</array>\n")
	sb.WriteString("<key>IncognitoModeAvailability</key><integer>1</integer>\n")
	sb.WriteString("<key>DeveloperToolsAvailability</key><integer>2</integer>\n")
	sb.WriteString("<key>BrowserGuestModeEnabled</key><false/>\n")
	sb.WriteString("<key>BrowserAddPersonEnabled</key><false/>\n")
	if a.Video {
		sb.WriteString("<key>DnsOverHttpsMode</key><string>off</string>\n")
		sb.WriteString("<key>ProxyMode</key><string>direct</string>\n")
	}
	sb.WriteString("</dict>\n</plist>\n")
	newTxt := sb.String()
	cur, _ := os.ReadFile(chromePlist)
	if string(cur) != newTxt {
		_ = os.MkdirAll(filepath.Dir(chromePlist), 0755)
		_ = os.WriteFile(chromePlist, []byte(newTxt), 0644)
		_ = run("killall", "cfprefsd")
	}
}

type procEntry struct {
	pid  int
	path string
}

func processes() []procEntry {
	out, err := exec.Command("ps", "-axo", "pid=,comm=").Output()
	if err != nil {
		return nil
	}
	var res []procEntry
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		sp := strings.IndexByte(line, ' ')
		if sp < 0 {
			continue
		}
		pid, err := strconv.Atoi(line[:sp])
		if err != nil {
			continue
		}
		res = append(res, procEntry{pid, strings.TrimSpace(line[sp+1:])})
	}
	return res
}

// appName: "/Applications/Telegram.app/Contents/MacOS/Telegram" -> "Telegram".
func appName(path string) string {
	i := strings.Index(path, ".app/Contents/MacOS/")
	if i < 0 {
		return ""
	}
	return filepath.Base(path[:i])
}

func killApps(names []string) {
	if len(names) == 0 {
		return
	}
	want := map[string]bool{}
	for _, n := range names {
		want[strings.ToLower(strings.TrimSuffix(n, ".app"))] = true
	}
	for _, p := range processes() {
		n := appName(p.path)
		if n != "" && want[strings.ToLower(n)] {
			_ = syscall.Kill(p.pid, syscall.SIGKILL)
		}
	}
}

func runningApps() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range processes() {
		n := appName(p.path)
		if n == "" || seen[n] || strings.HasPrefix(p.path, "/System/") {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// ---------------------------------------------------------------- установка

const daemonXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>com.kolya.fokus</string>
    <key>ProgramArguments</key>
    <array><string>/Library/Fokus/fokus</string><string>service</string></array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
</dict>
</plist>
`

func message(text string) { fmt.Println(text) }

func userHome(u string) string {
	out, err := exec.Command("dscl", ".", "-read", "/Users/"+u, "NFSHomeDirectory").Output()
	if err != nil {
		return ""
	}
	f := strings.Fields(string(out))
	if len(f) < 2 {
		return ""
	}
	return f[1]
}

func install() {
	if os.Geteuid() != 0 {
		fmt.Println("Установку нужно запускать с sudo (это делает команда установки).")
		os.Exit(1)
	}
	self, _ := os.Executable()
	_ = os.MkdirAll(dataDir(), 0700)
	_ = os.Chmod(dataDir(), 0700)
	removeVideoBlock()
	_ = run("launchctl", "bootout", "system/com.kolya.fokus")
	time.Sleep(time.Second)
	if filepath.Clean(self) != exePath() {
		b, err := os.ReadFile(self)
		if err != nil {
			fmt.Println("Не получилось прочитать программу:", err)
			os.Exit(1)
		}
		tmp := exePath() + ".new"
		_ = os.WriteFile(tmp, b, 0755)
		if err := os.Rename(tmp, exePath()); err != nil {
			fmt.Println("Не получилось установить:", err)
			os.Exit(1)
		}
	}
	_ = run("xattr", "-c", exePath())
	_ = os.WriteFile(daemonPlist, []byte(daemonXML), 0644)
	_ = os.Chown(daemonPlist, 0, 0)
	if run("launchctl", "bootstrap", "system", daemonPlist) != nil {
		_ = run("launchctl", "load", "-w", daemonPlist)
	}
	if u := os.Getenv("SUDO_USER"); !quiet && u != "" && u != "root" {
		if home := userHome(u); home != "" {
			p := filepath.Join(home, "Desktop", "Фокус.webloc")
			_ = os.WriteFile(p, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>URL</key><string>`+uiURL+`</string></dict></plist>
`), 0644)
			if out, err := exec.Command("id", "-u", u).Output(); err == nil {
				uid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
				_ = os.Chown(p, uid, 20)
			}
		}
	}
	fmt.Println("\n  «Фокус» установлен. Окно откроется в браузере.")
	fmt.Println("  Потом «Фокус» открывается значком «Фокус» на рабочем столе.")
}

// removeVideoBlock удаляет старую программу VideoBlock, если она стоит.
func removeVideoBlock() {
	_ = run("launchctl", "bootout", "system/com.kolya.videoblock")
	_ = os.Remove("/Library/LaunchDaemons/com.kolya.videoblock.plist")
	_ = os.RemoveAll("/Library/VideoBlock")
	_ = os.RemoveAll("/Users/Shared/VideoBlock")
	if b, err := os.ReadFile(hostsPath); err == nil {
		txt := stripSection(string(b), "# >>> VideoBlock >>>", "# <<< VideoBlock <<<")
		if txt != string(b) {
			_ = os.WriteFile(hostsPath, []byte(txt), 0644)
		}
	}
	if m, _ := filepath.Glob("/Users/*/Desktop/Разблокировать видео.command"); m != nil {
		for _, f := range m {
			_ = os.Remove(f)
		}
	}
}

// openUI открывает окно «Фокуса».
func openUI() { _ = exec.Command("open", uiURL).Run() }

func uninstallAll() {
	applyBlocks(Active{})
	_ = os.RemoveAll("/Applications/Фокус.app")
	_ = os.Remove(chromePlist)
	_ = run("killall", "cfprefsd")
	if m, _ := filepath.Glob("/Users/*/Desktop/Фокус.webloc"); m != nil {
		for _, f := range m {
			_ = os.Remove(f)
		}
	}
	_ = os.Remove(daemonPlist)
	_ = os.RemoveAll(dataDir())
	_ = run("launchctl", "bootout", "system/com.kolya.fokus")
	os.Exit(0)
}

// replaceSelf ставит новую версию; launchd сразу перезапустит её.
func replaceSelf(bin []byte) error {
	tmp := exePath() + ".new"
	if err := os.WriteFile(tmp, bin, 0755); err != nil {
		return err
	}
	return os.Rename(tmp, exePath())
}
