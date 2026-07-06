param(
    [string]$InstallDir = "$env:LOCALAPPDATA\FocusMode",
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$repo = "chenyuan99/FocusMode"
$assetName = "focusmode-windows-amd64.zip"

if ($Version -eq "latest") {
    $downloadUrl = "https://github.com/$repo/releases/latest/download/$assetName"
} else {
    $downloadUrl = "https://github.com/$repo/releases/download/$Version/$assetName"
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "focusmode-install-$([System.Guid]::NewGuid())"
$zipPath = Join-Path $tempDir $assetName

New-Item -ItemType Directory -Path $tempDir | Out-Null
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

try {
    Write-Host "Downloading FocusMode from $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath

    Write-Host "Installing to $InstallDir"
    Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force

    $exePath = Join-Path $InstallDir "focusmode-windows-amd64.exe"
    if (-not (Test-Path $exePath)) {
        throw "Expected executable was not found: $exePath"
    }

    Write-Host ""
    Write-Host "FocusMode installed."
    Write-Host "Run:"
    Write-Host "  & `"$exePath`" -tray"
    Write-Host ""
    Write-Host "Or add this folder to PATH:"
    Write-Host "  $InstallDir"
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

