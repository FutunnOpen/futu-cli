$ErrorActionPreference = "Stop"

$Product = "futu"
$Binary = "futu"
$ChecksumsAsset = "futu_checksums.txt"
$DefaultReleaseBase = "https://github.com/FutunnOpen/futu-cli"
$ReleaseBase = if ($env:FUTU_CLI_RELEASE_BASE) { $env:FUTU_CLI_RELEASE_BASE } else { $DefaultReleaseBase }
$Version = $env:FUTU_CLI_VERSION
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "FutuCLI\bin" }

$ReleaseBase = $ReleaseBase.TrimEnd("/")
$LatestUrl = "$ReleaseBase/releases/latest"

function Get-ResponseUrl($Response) {
  if ($Response.BaseResponse -and $Response.BaseResponse.ResponseUri) {
    return $Response.BaseResponse.ResponseUri.AbsoluteUri
  }
  if ($Response.BaseResponse -and $Response.BaseResponse.RequestMessage -and $Response.BaseResponse.RequestMessage.RequestUri) {
    return $Response.BaseResponse.RequestMessage.RequestUri.AbsoluteUri
  }
  return $LatestUrl
}

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

if ($Arch -ne "amd64") {
  throw "unsupported Windows architecture: $Arch"
}

if ([string]::IsNullOrWhiteSpace($Version)) {
  $LatestResponse = Invoke-WebRequest -UseBasicParsing -Uri $LatestUrl
  $Version = ((Get-ResponseUrl $LatestResponse).TrimEnd("/") -split "/")[-1]
}
if ($Version -notmatch "^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$") {
  throw "invalid version: $Version"
}

$Asset = "${Binary}_${Version}_windows_amd64.zip"
$TempDir = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
  $Archive = Join-Path $TempDir $Asset
  $Checksums = Join-Path $TempDir $ChecksumsAsset
  Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/releases/download/$Version/$Asset" -OutFile $Archive
  Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/releases/download/$Version/$ChecksumsAsset" -OutFile $Checksums

  $Expected = $null
  foreach ($Line in Get-Content $Checksums) {
    $Parts = $Line -split "\s+"
    if ($Parts.Length -ge 2 -and $Parts[1] -eq $Asset) {
      $Expected = $Parts[0]
    }
  }
  if ([string]::IsNullOrWhiteSpace($Expected)) {
    throw "checksum for $Asset not found"
  }
  $Actual = (Get-FileHash -Algorithm SHA256 -Path $Archive).Hash.ToLowerInvariant()
  if ($Expected.ToLowerInvariant() -ne $Actual) {
    throw "checksum mismatch for $Asset"
  }

  $UnpackDir = Join-Path $TempDir "unpack"
  Expand-Archive -Path $Archive -DestinationPath $UnpackDir -Force
  $Exe = Join-Path $UnpackDir "$Binary.exe"
  if (!(Test-Path $Exe)) {
    throw "$Binary.exe not found in archive"
  }
  $VersionOutput = & $Exe version
  if ($LASTEXITCODE -ne 0 -or ($VersionOutput -join "`n") -notmatch ([regex]::Escape($Version))) {
    throw "$Binary.exe version output does not contain $Version"
  }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Copy-Item -Path $Exe -Destination (Join-Path $InstallDir "$Binary.exe") -Force

  $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (($UserPath -split ";") -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to user PATH. Open a new terminal before running $Binary."
  }
  Write-Host "$Binary $Version installed to $InstallDir"
} finally {
  Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
