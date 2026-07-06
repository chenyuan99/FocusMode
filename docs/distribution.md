# Distribution

FocusMode release builds are available from GitHub Releases. Windows users can download the latest release with `wget` or PowerShell's `Invoke-WebRequest`.

## Download With wget

```powershell
wget https://github.com/chenyuan99/FocusMode/releases/latest/download/focusmode-windows-amd64.zip -OutFile focusmode-windows-amd64.zip
Expand-Archive .\focusmode-windows-amd64.zip -DestinationPath .\FocusMode -Force
.\FocusMode\focusmode-windows-amd64.exe -tray
```

In Windows PowerShell, `wget` is usually an alias for `Invoke-WebRequest`.

## Install With Script

Download the installer script:

```powershell
wget https://raw.githubusercontent.com/chenyuan99/FocusMode/master/scripts/install.ps1 -OutFile install-focusmode.ps1
```

Run it:

```powershell
.\install-focusmode.ps1
```

By default, it installs to:

```text
%LOCALAPPDATA%\FocusMode
```

Choose a custom install directory:

```powershell
.\install-focusmode.ps1 -InstallDir "$env:USERPROFILE\Apps\FocusMode"
```

Install a specific release tag:

```powershell
.\install-focusmode.ps1 -Version v1.2.3
```

## Stable Release Asset

The release workflow publishes a stable asset named:

```text
focusmode-windows-amd64.zip
```

This stable name makes the latest release downloadable from:

```text
https://github.com/chenyuan99/FocusMode/releases/latest/download/focusmode-windows-amd64.zip
```

The workflow also publishes the versioned archive for people who prefer immutable filenames.

