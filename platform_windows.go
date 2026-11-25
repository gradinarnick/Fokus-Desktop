//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

func dataDir() string   { return `C:\ProgramData\Fokus` }
func exePath() string   { return filepath.Join(dataDir(), "fokus.exe") }
func assetName() string { return "fokus-windows.exe" }

func hidden(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd
}

func run(name string, args ...string) error { return hidden(exec.Command(name, args...)).Run() }

const chromeKey = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Google\Chrome`
const ifeoKey = `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options`

func regStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func writeUTF16(path, text string) error {
	u := utf16.Encode([]rune(text))
	b := make([]byte, 2+len(u)*2)
	b[0], b[1] = 0xFF, 0xFE
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[2+i*2:], v)
	}
	return os.WriteFile(path, b, 0600)
}

// applyBlocks: политики Chrome (адреса, инкогнито, DNS) и запрет запуска других браузеров (IFEO).
func applyBlocks(a Active) {
	var sb strings.Builder
	nl := "\r\n"
	sb.WriteString("Windows Registry Editor Version 5.00" + nl + nl)
	sb.WriteString("[-" + chromeKey + `\URLBlocklist]` + nl + nl)
	sb.WriteString("[" + chromeKey + "]" + nl)
	dwords := []struct {
		n string
		v uint32
	}{{"IncognitoModeAvailability", 1}, {"DeveloperToolsAvailability", 2}, {"BrowserGuestModeEnabled", 0}, {"BrowserAddPersonEnabled", 0}}
	for _, p := range dwords {
		if a.Video || a.Night {
			fmt.Fprintf(&sb, "%s=dword:%08x%s", regStr(p.n), p.v, nl)
		} else {
			fmt.Fprintf(&sb, "%s=-%s", regStr(p.n), nl)
		}
	}
	if a.Video {
		sb.WriteString(`"DnsOverHttpsMode"="off"` + nl + `"ProxyMode"="direct"` + nl)
	} else {
		sb.WriteString(`"DnsOverHttpsMode"=-` + nl + `"ProxyMode"=-` + nl)
	}
	sb.WriteString(nl)
	list := append(append([]string{}, a.Sites...), a.Paths...)
	if a.Video {
		list = append(list, "chrome://flags", "chrome://net-internals")
	}
	if len(list) > 0 {
		sb.WriteString("[" + chromeKey + `\URLBlocklist]` + nl)
		for i, d := range list {
			fmt.Fprintf(&sb, "\"%d\"=%s%s", i+1, regStr(d), nl)
		}
		sb.WriteString(nl)
	}
	for _, exe := range ifeoBrowsers {
		sb.WriteString("[" + ifeoKey + `\` + exe + "]" + nl)
		if a.Video {
			sb.WriteString(`"Debugger"="systray.exe"` + nl + nl)
		} else {
			sb.WriteString(`"Debugger"=-` + nl + nl)
		}
	}
	p := filepath.Join(dataDir(), "policy.reg")
	if writeUTF16(p, sb.String()) == nil {
		_ = run("reg", "import", p)
	}
}

type procEntry struct {
	pid  uint32
	name string
}

func processes() []procEntry {
	h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(h)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	var out []procEntry
	for err = syscall.Process32First(h, &e); err == nil; err = syscall.Process32Next(h, &e) {
		out = append(out, procEntry{e.ProcessID, syscall.UTF16ToString(e.ExeFile[:])})
	}
	return out
}

func killApps(names []string) {
	if len(names) == 0 {
		return
	}
	want := map[string]bool{}
	for _, n := range names {
		n = strings.ToLower(n)
		if !strings.HasSuffix(n, ".exe") {
			n += ".exe"
		}
		want[n] = true
	}
	for _, p := range processes() {
		if want[strings.ToLower(p.name)] {
			if h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, p.pid); err == nil {
				_ = syscall.TerminateProcess(h, 1)
				_ = syscall.CloseHandle(h)
			}
		}
	}
}

func runningApps() []string {
	seen := map[string]bool{}
	var out []string
	skip := map[string]bool{"svchost.exe": true, "system": true, "[system process]": true, "registry": true, "smss.exe": true,
		"csrss.exe": true, "wininit.exe": true, "services.exe": true, "lsass.exe": true, "winlogon.exe": true, "dwm.exe": true,
		"conhost.exe": true, "fontdrvhost.exe": true, "fokus.exe": true, "runtimebroker.exe": true, "dllhost.exe": true,
		"sihost.exe": true, "taskhostw.exe": true, "ctfmon.exe": true, "searchhost.exe": true, "memory compression": true}
	for _, p := range processes() {
		l := strings.ToLower(p.name)
		if skip[l] || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, p.name)
	}
	return out
}

// ---------------------------------------------------------------- установка

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	messageBox = user32.NewProc("MessageBoxW")
)

func message(text string) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString("Фокус")
	_, _, _ = messageBox.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x40)
}

func isAdmin() bool {
	f, err := os.Open(`\\.\PHYSICALDRIVE0`)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

const taskXML = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>Fokus</Description></RegistrationInfo>
  <Triggers>
    <BootTrigger><Enabled>true</Enabled></BootTrigger>
    <TimeTrigger>
      <Repetition><Interval>PT1M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition>
      <StartBoundary>2025-01-01T00:00:00</StartBoundary>
      <Enabled>true</Enabled>
    </TimeTrigger>
  </Triggers>
  <Principals><Principal id="Author"><UserId>S-1-5-18</UserId><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings><StopOnIdleEnd>false</StopOnIdleEnd><RestartOnIdle>false</RestartOnIdle></IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>true</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author"><Exec><Command>C:\ProgramData\Fokus\fokus.exe</Command><Arguments>service</Arguments></Exec></Actions>
</Task>
`

const shortcutPath = `C:\Users\Public\Desktop\Фокус.url`

func install() {
	self, _ := os.Executable()
	if !isAdmin() {
		_ = run("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
			"Start-Process -FilePath '"+strings.ReplaceAll(self, "'", "''")+"' -ArgumentList 'install' -Verb RunAs")
		return
	}
	_ = os.MkdirAll(dataDir(), 0700)
	removeVideoBlock()
	_ = run("schtasks", "/end", "/tn", "Fokus")
	time.Sleep(1500 * time.Millisecond)
	if !strings.EqualFold(filepath.Clean(self), exePath()) {
		if b, err := os.ReadFile(self); err == nil {
			_ = os.Remove(exePath() + ".old")
			_ = os.Rename(exePath(), exePath()+".old")
			if err := os.WriteFile(exePath(), b, 0755); err != nil {
				message("Не получилось скопировать программу: " + err.Error())
				return
			}
		}
	}
	// Папка — только для системы и администраторов (ключ кодов никто не прочитает).
	_ = run("icacls", dataDir(), "/inheritance:r", "/grant:r", "*S-1-5-18:(OI)(CI)F", "*S-1-5-32-544:(OI)(CI)F")
	xmlPath := filepath.Join(dataDir(), "task.xml")
	if err := writeUTF16(xmlPath, taskXML); err == nil {
		_ = run("schtasks", "/create", "/tn", "Fokus", "/xml", xmlPath, "/f")
	}
	_ = run("schtasks", "/run", "/tn", "Fokus")
	if !quiet {
		_ = os.WriteFile(shortcutPath, []byte("[InternetShortcut]\r\nURL="+uiURL+"\r\nIconFile="+exePath()+"\r\nIconIndex=0\r\n"), 0644)
	}
	for i := 0; i < 40; i++ {
		if c, err := net.Dial("tcp", listenAddr); err == nil {
			c.Close()
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if quiet {
		return
	}
	_ = run("rundll32", "url.dll,FileProtocolHandler", uiURL)
	message("«Фокус» установлен.\n\nОткроется окно «Фокуса» в браузере. Первым делом владелец ключа настраивает Google Authenticator.\n\nПотом «Фокус» открывается значком «Фокус» на рабочем столе.")
}

// removeVideoBlock удаляет старую программу VideoBlock, если она стоит.
func removeVideoBlock() {
	_ = run("schtasks", "/delete", "/tn", "VideoBlockGuard", "/f")
	_ = run("schtasks", "/delete", "/tn", "VideoBlockGuardBoot", "/f")
	for _, exe := range append(append([]string{}, videoApps...), ifeoBrowsers...) {
		_ = run("reg", "delete", ifeoKey+`\`+exe, "/v", "Debugger", "/f")
	}
	_ = run("reg", "delete", chromeKey+`\ExtensionInstallBlocklist`, "/f")
	_ = run("cmd", "/c", "rmdir", "/s", "/q", `C:\ProgramData\VideoBlock`)
	_ = os.Remove(`C:\Users\Public\Desktop\Разблокировать видео.lnk`)
}

// openUI открывает окно «Фокуса» (ярлыки в «Пуске» и на рабочем столе).
func openUI() {
	for i := 0; i < 20; i++ {
		if c, err := net.Dial("tcp", listenAddr); err == nil {
			c.Close()
			_ = run("rundll32", "url.dll,FileProtocolHandler", uiURL)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	message("«Фокус» ещё запускается. Попробуйте открыть его через минуту.")
}

const appDir = `C:\Program Files\Fokus`
const uninstKey = `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Fokus`

func uninstallAll() {
	applyBlocks(Active{})
	_ = os.Remove(shortcutPath)
	_ = os.Remove(`C:\Users\Public\Desktop\Фокус.lnk`)
	_ = os.Remove(`C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Фокус.lnk`)
	_ = run("reg", "delete", uninstKey, "/f")
	_ = os.RemoveAll(appDir)
	_ = run("schtasks", "/delete", "/tn", "Fokus", "/f")
	cmd := hidden(exec.Command("cmd", "/c", "ping -n 4 127.0.0.1 >nul & rmdir /s /q \""+dataDir()+"\""))
	_ = cmd.Start()
	os.Exit(0)
}

// replaceSelf ставит новую версию; планировщик запустит её в течение минуты.
func replaceSelf(bin []byte) error {
	exe := exePath()
	_ = os.Remove(exe + ".old")
	if err := os.Rename(exe, exe+".old"); err != nil {
		return err
	}
	if err := os.WriteFile(exe, bin, 0755); err != nil {
		_ = os.Rename(exe+".old", exe)
		return err
	}
	return nil
}
