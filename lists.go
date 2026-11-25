package main

import "runtime"

// Программы, которые закрываются при блоке видео: другие браузеры и видеоплееры.
var videoApps = func() []string {
	if runtime.GOOS == "windows" {
		return []string{
			"msedge.exe", "firefox.exe", "browser.exe", "opera.exe", "opera_gx.exe", "brave.exe", "vivaldi.exe",
			"iexplore.exe", "waterfox.exe", "librewolf.exe", "tor.exe", "chromium.exe", "yandex.exe", "arc.exe",
			"vlc.exe", "wmplayer.exe", "mpc-hc.exe", "mpc-hc64.exe", "mpc-be.exe", "mpc-be64.exe", "PotPlayer.exe",
			"PotPlayerMini.exe", "PotPlayerMini64.exe", "KMPlayer.exe", "KMPlayer64.exe", "mpv.exe", "GOM.exe",
			"smplayer.exe", "5KPlayer.exe", "Video.UI.exe", "Microsoft.Media.Player.exe", "Kodi.exe", "Plex.exe",
		}
	}
	return []string{
		"Safari", "Firefox", "Microsoft Edge", "Opera", "Opera GX", "Brave Browser", "Yandex", "Vivaldi", "Arc",
		"Orion", "Chromium", "Tor Browser", "VLC", "QuickTime Player", "IINA", "Infuse", "Infuse 7", "Movist",
		"Movist Pro", "Elmedia Video Player", "TV", "5KPlayer", "Plex", "Kodi",
	}
}()

// Программы, закрытые ночью (соцсети, игры).
var nightApps = func() []string {
	if runtime.GOOS == "windows" {
		return []string{"Telegram.exe", "Discord.exe", "steam.exe", "EpicGamesLauncher.exe", "Battle.net.exe", "Instagram.exe", "TikTok.exe"}
	}
	return []string{"Telegram", "Discord", "Steam", "Steam Helper", "Battle.net"}
}()

// Exe-файлы, которые Windows не даёт запустить (IFEO), пока действует блок видео: только браузеры.
var ifeoBrowsers = []string{
	"msedge.exe", "firefox.exe", "browser.exe", "opera.exe", "opera_gx.exe", "brave.exe", "vivaldi.exe",
	"iexplore.exe", "waterfox.exe", "librewolf.exe", "chromium.exe", "yandex.exe",
}
