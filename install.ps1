# bvm installer for Windows.
# Usage: powershell -c "irm https://raw.githubusercontent.com/chathula/bvm/main/install.ps1 | iex"

$ErrorActionPreference = "Stop"

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
	"ARM64" { "arm64" }
	default { "x86_64" }
}
$url = "https://github.com/chathula/bvm/releases/latest/download/bvm_windows_${arch}.zip"

$bvmDir = if ($env:BVM_DIR) { $env:BVM_DIR } else { Join-Path $env:USERPROFILE ".bvm" }
$binDir = Join-Path $bvmDir "bin"
$exe = Join-Path $binDir "bvm.exe"

New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$tmpZip = Join-Path $env:TEMP "bvm_install.zip"
Invoke-WebRequest -Uri $url -OutFile $tmpZip -UseBasicParsing
Expand-Archive -Path $tmpZip -DestinationPath $binDir -Force
Remove-Item $tmpZip

Write-Host "bvm was installed successfully to $exe"

$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$binDir*") {
	[Environment]::SetEnvironmentVariable("PATH", "$userPath;$binDir", "User")
	Write-Host "Added $binDir to your user PATH — reopen your terminal to get started."
} else {
	Write-Host "Run 'bvm --help' to get started."
}
