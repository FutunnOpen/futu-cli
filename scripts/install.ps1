$ErrorActionPreference = "Stop"

$Product = "futu"
$Binary = "futu"
$ChecksumsAsset = "futu_checksums.txt"
$ChecksumsSignatureAsset = "futu_checksums.txt.sig"
$VersionFile = "VERSION"
$DefaultReleaseBase = "https://github.com/FutunnOpen/futu-cli"
$ReleasePublicKeyPem = @"
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEyk556CXX05/DwVDZl9lJIt74Vut5
+4Fj6BuwyUATzxXDI5O3GohEhC3X3m0ygsH/p3ClarvAUwfmRF+1DfEkWQ==
-----END PUBLIC KEY-----
"@
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
  $Signature = Join-Path $TempDir $ChecksumsSignatureAsset
  Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/releases/download/$Version/$Asset" -OutFile $Archive
  Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/releases/download/$Version/$ChecksumsAsset" -OutFile $Checksums
  Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/releases/download/$Version/$ChecksumsSignatureAsset" -OutFile $Signature
  $PublicKeyPath = Join-Path $TempDir "release.pub"
  Set-Content -Path $PublicKeyPath -Value $ReleasePublicKeyPem -Encoding ascii
  $openssl = Get-Command openssl -ErrorAction SilentlyContinue
  if (-not $openssl) {
    throw "openssl is required to verify the release signature"
  }
  & openssl dgst -sha256 -verify $PublicKeyPath -signature $Signature $Checksums | Out-Null
  if ($LASTEXITCODE -ne 0) {
    throw "checksum signature verification failed"
  }

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
  $PackageVersionPath = Join-Path $UnpackDir $VersionFile
  if (!(Test-Path $PackageVersionPath)) {
    throw "$VersionFile not found in archive"
  }
  $PackageVersion = (Get-Content -Raw $PackageVersionPath).Trim()
  if ($PackageVersion -ne $Version) {
    throw "package $VersionFile is $PackageVersion, want $Version"
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
