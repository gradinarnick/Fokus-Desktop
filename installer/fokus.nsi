; Установщик «Фокуса» для Windows (собирается в GitHub Actions: makensis -DVERSION=1.0.N)
Unicode true
!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "1.0.0"
!endif

Name "Фокус"
Caption "Установка «Фокуса»"
OutFile "../dist/Fokus-Setup.exe"
InstallDir "$PROGRAMFILES64\Fokus"
RequestExecutionLevel admin
BrandingText "Фокус ${VERSION}"
VIProductVersion "${VERSION}.0"
VIAddVersionKey /LANG=1049 "ProductName" "Фокус"
VIAddVersionKey /LANG=1049 "FileDescription" "Установка «Фокуса»"
VIAddVersionKey /LANG=1049 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=1049 "LegalCopyright" "Коля"

!define MUI_ICON "../assets/icon.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "Установка «Фокуса»"
!define MUI_WELCOMEPAGE_TEXT "«Фокус» блокирует видео во всех браузерах и программах, включает ночной режим и правила для сайтов и программ.$\r$\n$\r$\nОслабить блокировки можно только одноразовым кодом из Google Authenticator того, у кого ключ.$\r$\n$\r$\nСтарая программа VideoBlock, если она есть, удалится сама.$\r$\n$\r$\nНажмите «Далее»."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TITLE "«Фокус» установлен"
!define MUI_FINISHPAGE_TEXT "«Фокус» работает и сам обновляется.$\r$\n$\r$\nПервым делом владелец ключа открывает «Фокус» и на вкладке «Ключ» сканирует QR-код в Google Authenticator."
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Открыть «Фокус»"
!define MUI_FINISHPAGE_RUN_FUNCTION OpenFokus
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "Russian"

Function OpenFokus
  ; через explorer — браузер откроется обычным, без прав администратора
  Exec '"$WINDIR\explorer.exe" "http://127.0.0.1:47890/"'
FunctionEnd

Section "Фокус"
  SetShellVarContext all
  SetOutPath "$INSTDIR"
  File "/oname=Fokus.exe" "../dist/fokus-windows.exe"
  DetailPrint "Ставлю службу «Фокуса»…"
  nsExec::Exec '"$INSTDIR\Fokus.exe" install /quiet'
  Delete "$DESKTOP\Фокус.url"
  CreateShortCut "$SMPROGRAMS\Фокус.lnk" "$INSTDIR\Fokus.exe" "open" "$INSTDIR\Fokus.exe" 0
  CreateShortCut "$DESKTOP\Фокус.lnk" "$INSTDIR\Fokus.exe" "open" "$INSTDIR\Fokus.exe" 0
  !define UK "SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Fokus"
  WriteRegStr HKLM "${UK}" "DisplayName" "Фокус"
  WriteRegStr HKLM "${UK}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UK}" "Publisher" "Коля"
  WriteRegStr HKLM "${UK}" "DisplayIcon" "$INSTDIR\Fokus.exe"
  ; «Удалить» открывает «Фокус»: удаление — только по коду, кнопкой на вкладке «Доступ»
  WriteRegStr HKLM "${UK}" "UninstallString" '"$INSTDIR\Fokus.exe" uninstall'
  WriteRegDWORD HKLM "${UK}" "NoModify" 1
  WriteRegDWORD HKLM "${UK}" "NoRepair" 1
SectionEnd
