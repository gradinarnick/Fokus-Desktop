#!/bin/bash
# Установка «Фокуса» на Mac:
#   curl -fsSL https://github.com/gradinarnick/Fokus-Desktop/releases/latest/download/install-mac.sh | bash
set -e
if [ "$(uname -m)" = "arm64" ]; then A=arm64; else A=amd64; fi
URL="https://github.com/gradinarnick/Fokus-Desktop/releases/latest/download/fokus-mac-$A"
TMP=$(mktemp -d)
echo "Скачиваю «Фокус»…"
curl -fsSL "$URL" -o "$TMP/fokus"
chmod +x "$TMP/fokus"
echo "Введи пароль администратора Mac (символы при вводе не видны):"
sudo "$TMP/fokus" install < /dev/tty
rm -rf "$TMP"
sleep 3
open "http://127.0.0.1:47890/"
