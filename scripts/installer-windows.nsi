; Per-user Windows installer. The profile under APPDATA is never removed.
Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "x64.nsh"

!ifndef PAYLOAD_DIR
  !error "PAYLOAD_DIR is required"
!endif
!ifndef OUTPUT_FILE
  !error "OUTPUT_FILE is required"
!endif
!ifndef APP_VERSION
  !error "APP_VERSION is required"
!endif

Name "Nekophage ${APP_VERSION}"
OutFile "${OUTPUT_FILE}"
InstallDir "$LOCALAPPDATA\Programs\Nekophage"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show

!insertmacro MUI_PAGE_LICENSE "${PAYLOAD_DIR}\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

Var SkipRegistration

Function .onInit
  SetShellVarContext current
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
  StrCpy $SkipRegistration "0"
  ${GetParameters} $0
  ClearErrors
  ${GetOptions} $0 "/NOREGISTER" $1
  ${IfNot} ${Errors}
    ; Isolated installation checks use the real payload and uninstaller without
    ; writing the user's Start menu or Add/Remove Programs registration.
    StrCpy $SkipRegistration "1"
  ${EndIf}
FunctionEnd

Section "Nekophage"
  SetOutPath "$INSTDIR"
  File "${PAYLOAD_DIR}\*"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteINIStr "$INSTDIR\install-state.ini" "install" "skipRegistration" "$SkipRegistration"
  ${If} $SkipRegistration != "1"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "DisplayName" "Nekophage"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "DisplayVersion" "${APP_VERSION}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "Publisher" "laincat"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "InstallLocation" "$INSTDIR"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "QuietUninstallString" '$\"$INSTDIR\Uninstall.exe$\" /S'
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "NoModify" 1
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "NoRepair" 1
    CreateDirectory "$SMPROGRAMS\Nekophage"
    CreateShortcut "$SMPROGRAMS\Nekophage\Nekophage.lnk" "$INSTDIR\Nekophage.exe"
  ${EndIf}
SectionEnd

Function un.onInit
  SetShellVarContext current
  ${If} ${RunningX64}
    SetRegView 64
  ${EndIf}
FunctionEnd

Section "Uninstall"
  ; Leave the install directory before deleting it: Windows refuses to remove a
  ; directory that is some process's working directory, which leaves an empty
  ; Nekophage folder behind after a silent uninstall.
  SetOutPath "$TEMP"
  ReadINIStr $SkipRegistration "$INSTDIR\install-state.ini" "install" "skipRegistration"
  ${If} $SkipRegistration != "1"
    ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage" "InstallLocation"
    ${If} $0 == $INSTDIR
      DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Nekophage"
      Delete "$SMPROGRAMS\Nekophage\Nekophage.lnk"
      RMDir "$SMPROGRAMS\Nekophage"
    ${EndIf}
  ${EndIf}
  Delete "$INSTDIR\Nekophage.exe"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\COPYING"
  Delete "$INSTDIR\BUILD-INFO.txt"
  Delete "$INSTDIR\README.zh.txt"
  Delete "$INSTDIR\README.en.txt"
  Delete "$INSTDIR\HLS-LICENSE.txt"
  Delete "$INSTDIR\HLS-APACHE-2.0.txt"
  Delete "$INSTDIR\HLS-NOTICE.zh.txt"
  Delete "$INSTDIR\HLS-NOTICE.en.txt"
  Delete "$INSTDIR\SHA256SUMS.txt"
  Delete "$INSTDIR\install-state.ini"
  Delete "$INSTDIR\Uninstall.exe"
  ; Non-recursive: leave any extra files the user put in this directory intact.
  RMDir "$INSTDIR"
SectionEnd
