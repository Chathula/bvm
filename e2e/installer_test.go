//go:build e2e

package e2e

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestPowerShellInstallerSyntax validates install.ps1 parses cleanly so CI
// catches PowerShell regressions on any OS where pwsh is available.
func TestPowerShellInstallerSyntax(t *testing.T) {
	if runtime.GOOS != "windows" {
		pwsh, err := exec.LookPath("pwsh")
		if err != nil {
			t.Skip("pwsh not available on this platform")
		}
		_ = pwsh
	}

	script := `
$errs = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile(
    (Join-Path $env:BVM_REPO_ROOT 'install.ps1'), [ref]$null, [ref]$errs)
if ($errs -and $errs.Count -gt 0) {
    Write-Output ($errs | ForEach-Object { $_.Message })
    exit 1
}
`
	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(cmd.Environ(), "BVM_REPO_ROOT="+repoRoot(t))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.ps1 has parse errors:\n%s", out)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	return strings.TrimSpace(string(out))
}
