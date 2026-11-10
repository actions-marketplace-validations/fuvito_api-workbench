; Clear Signal — Windows Installer
; Build with: iscc /DMyAppVersion=v0.2.0 installer.iss
; Run from the apps/desktop/wails directory (where wails build outputs build\bin\clrsig.exe).

#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif

#define MyAppName      "Clear Signal"
#define MyAppExeName   "clrsig.exe"
#define MyAppPublisher "fuvito"
#define MyAppURL       "https://github.com/fuvito/api-workbench"

[Setup]
AppId={{B4E9A237-6F1C-4D8E-A523-07C5B1F8D962}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases

; Use the user's local Programs folder when run without admin, or Program Files
; when elevated — the dialog lets the user choose.
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

AllowNoIcons=yes
OutputDir=build\bin
OutputBaseFilename=clrsig-windows-amd64-setup
SetupIconFile=
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
WizardSmallImageFile=
DisableProgramGroupPage=yes
UninstallDisplayName={#MyAppName}
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "build\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}";  Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; \
  Description: "{cm:LaunchProgram,{#MyAppName}}"; \
  Flags: nowait postinstall skipifsilent
