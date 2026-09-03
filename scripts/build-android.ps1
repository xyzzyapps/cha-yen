# Builds the debug-signed Android APK for Cha-Yen.
# Requires: Android SDK (ANDROID_HOME), NDK, Temurin JDK 17, fyne CLI, gomobile.
# Output: cmd/cha-yen/cha-yen.apk
#
# Usage:
#   pwsh scripts/build-android.ps1            # arm64 (physical devices)
#   pwsh scripts/build-android.ps1 -Arch amd64  # x86_64 (emulator)
param(
    [ValidateSet("arm64", "amd64")]
    [string]$Arch = "arm64"
)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# --- toolchain environment -------------------------------------------------
$jdk = Get-ChildItem "C:\Program Files\Eclipse Adoptium" -Directory -Filter "jdk-17*" |
    Select-Object -First 1
if (-not $jdk) { throw "JDK 17 (Temurin) not found" }
$env:JAVA_HOME = $jdk.FullName

if (-not $env:ANDROID_HOME) {
    $env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
}
$ndk = Get-ChildItem "$env:ANDROID_HOME\ndk" -Directory |
    Sort-Object Name -Descending | Select-Object -First 1
$env:ANDROID_NDK_HOME = $ndk.FullName
$env:ANDROID_NDK_ROOT = $ndk.FullName

$bt = Get-ChildItem "$env:ANDROID_HOME\build-tools" -Directory |
    Sort-Object Name -Descending | Select-Object -First 1
$env:Path = "$($jdk.FullName)\bin;$env:USERPROFILE\go\bin;$env:ANDROID_HOME\platform-tools;$bt.FullName;$env:Path"

Write-Host "JAVA_HOME=$env:JAVA_HOME"
Write-Host "ANDROID_HOME=$env:ANDROID_HOME"
Write-Host "ANDROID_NDK_HOME=$env:ANDROID_NDK_HOME"

# --- gomobile sanity -------------------------------------------------------
gomobile init

# --- package ----------------------------------------------------------------
# NOTE: the fyne CLI resolves --icon relative to --source-dir and writes the
# APK next to the source dir. FyneApp.toml at the repo root is metadata only.
Copy-Item assets\icon.png cmd\cha-yen\icon.png -Force
& fyne package --target "android/$Arch" `
    --source-dir ./cmd/cha-yen `
    --icon icon.png `
    --app-id dev.chayen.app `
    --name Cha-Yen `
    --app-version 0.1.0 --app-build 1
if ($LASTEXITCODE -ne 0) { throw "fyne package failed ($LASTEXITCODE)" }

$apk = Join-Path $root "cmd\cha-yen\Cha-Yen.apk"
Write-Host "APK: $apk"
if (Get-Command adb -ErrorAction SilentlyContinue) {
    Write-Host "Install with: adb install -r $apk"
}
