param(
  [string]$Version = "latest",
  [string]$InstallDir = "$env:LOCALAPPDATA\Programs\xelyon\bin",
  [switch]$DryRun,
  [switch]$Yes
)

$ErrorActionPreference = "Stop"
$Repo = "susugadx/xelyon-cli"

function Get-ReleaseApiUrl {
  if ($Version -eq "latest") {
    return "https://api.github.com/repos/$Repo/releases/latest"
  }
  return "https://api.github.com/repos/$Repo/releases/tags/$Version"
}

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
  throw "--InstallDir must not be empty"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$release = Invoke-RestMethod -Uri (Get-ReleaseApiUrl) -TimeoutSec 120 -Headers @{ "User-Agent" = "xelyon-installer" }
$asset = $release.assets | Where-Object { $_.name -match "_windows_$arch\.zip$" } | Select-Object -First 1
$checksums = $release.assets | Where-Object { $_.name -eq "checksums.txt" } | Select-Object -First 1
if (-not $asset -or -not $checksums) {
  throw "Could not find release asset for windows/$arch"
}

$target = Join-Path $InstallDir "xelyon.exe"
Write-Host "xelyon installer"
Write-Host "  release: $Version"
Write-Host "  asset:   $($asset.browser_download_url)"
Write-Host "  sums:    $($checksums.browser_download_url)"
Write-Host "  target:  $target"

if ($DryRun) {
  exit 0
}

if ((Test-Path $target) -and -not $Yes) {
  $answer = Read-Host "Overwrite $target? [y/N]"
  if ($answer -notin @("y", "Y", "yes", "YES")) {
    throw "Cancelled"
  }
}

$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("xelyon-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $temp | Out-Null
try {
  $archivePath = Join-Path $temp $asset.name
  $checksumsPath = Join-Path $temp "checksums.txt"
  Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archivePath -TimeoutSec 120 -Headers @{ "User-Agent" = "xelyon-installer" }
  Invoke-WebRequest -Uri $checksums.browser_download_url -OutFile $checksumsPath -TimeoutSec 120 -Headers @{ "User-Agent" = "xelyon-installer" }

  $line = Get-Content $checksumsPath | Where-Object { $_ -match "\s$([regex]::Escape($asset.name))$" } | Select-Object -First 1
  if (-not $line) {
    throw "Checksum entry not found for $($asset.name)"
  }
  $expected = ($line -split "\s+")[0].ToLowerInvariant()
  $actual = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLowerInvariant()
  if ($expected -ne $actual) {
    throw "Checksum mismatch for $($asset.name): expected $expected actual $actual"
  }

  $extractDir = Join-Path $temp "extract"
  Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
  $binary = Get-ChildItem -Path $extractDir -Recurse -File -Filter "xelyon.exe" | Select-Object -First 1
  if (-not $binary) {
    throw "xelyon.exe not found in archive"
  }

  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Copy-Item -Path $binary.FullName -Destination $target -Force

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $pathParts = $userPath -split ";" | Where-Object { $_ -ne "" }
  if ($pathParts -notcontains $InstallDir) {
    if ($Yes) {
      [Environment]::SetEnvironmentVariable("Path", ($pathParts + $InstallDir -join ";"), "User")
      Write-Host "Added $InstallDir to the User PATH. Open a new terminal to use it."
    } else {
      $answer = Read-Host "Add $InstallDir to the User PATH? [y/N]"
      if ($answer -in @("y", "Y", "yes", "YES")) {
        [Environment]::SetEnvironmentVariable("Path", ($pathParts + $InstallDir -join ";"), "User")
        Write-Host "Added $InstallDir to the User PATH. Open a new terminal to use it."
      }
    }
  }

  Write-Host "Installed $target"
  & $target --version
}
finally {
  Remove-Item -Recurse -Force $temp -ErrorAction SilentlyContinue
}
