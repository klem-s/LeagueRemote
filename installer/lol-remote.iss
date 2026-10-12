; Inno Setup script for LoL Remote.
; AppVersion is passed in from CI: iscc /DAppVersion=0.8.2 lol-remote.iss
; (falls back to 0.0.0 for a local/manual compile so the script is still
; valid on its own).
#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{CD6EAE54-51AD-4923-A07D-8378D0D6E41C}
AppName=LoL Remote
AppVersion={#AppVersion}
AppPublisher=klem-s
AppPublisherURL=https://github.com/klem-s/LeagueRemote
; Same named mutex internal/singleinstance uses — if a running LoL Remote
; wasn't already closed by whatever triggered this install (the in-app
; "Update" button exits the process itself first, but a manually re-run
; installer wouldn't), Setup detects it and asks the user to close it
; rather than failing to overwrite the locked exe.
AppMutex=Global\LoLRemote-7F3C2E7B-7E2E-4C2E-9C7F-9B1A2D3E4F5A
; Per-user install under %LOCALAPPDATA% — no admin rights / UAC prompt,
; same no-friction install this project has always had as a portable exe.
DefaultDirName={localappdata}\Programs\LoLRemote
DefaultGroupName=LoL Remote
PrivilegesRequired=lowest
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
UninstallDisplayIcon={app}\lol-remote.exe
OutputBaseFilename=LoLRemote-Setup
OutputDir=.
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; Re-running this installer for a new version just overwrites the files
; in place (same AppId, same install dir) — that's the "update" story.
DisableProgramGroupPage=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: unchecked

[Files]
Source: "..\lol-remote.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\LoL Remote"; Filename: "{app}\lol-remote.exe"
Name: "{group}\Uninstall LoL Remote"; Filename: "{uninstallexe}"
Name: "{userdesktop}\LoL Remote"; Filename: "{app}\lol-remote.exe"; Tasks: desktopicon

[Run]
; No skipifsilent: the in-app Update button runs this installer with
; /VERYSILENT, and it still needs to relaunch the app afterward — that's
; the whole point of a one-click update.
Filename: "{app}\lol-remote.exe"; Description: "Launch LoL Remote"; Flags: nowait postinstall

[UninstallDelete]
; The app writes nothing else under {app} (no config/cache files there),
; so nothing extra needs cleaning up here.
