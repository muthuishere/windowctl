# Windows real-window smoke test for windowctl.
#
# Implements the scenario in docs/specs/10-ci-test-matrix.md:
#
#   1. Build the CLI binary.
#   2. Launch a real application (Notepad).
#   3. Detect it with `windowctl windows list`.
#   4. Execute a `move` and verify no errors.
#   5. Stop the application.
#
# Designed to run on the windows-latest GitHub Actions runner. Use as:
#
#   - if: matrix.os == 'windows-latest'
#     run: pwsh ./scripts/smoke-windows.ps1

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Write-Host '== windowctl Windows smoke test =='

# 1. Build the CLI.
go build -o windowctl.exe ./cmd/windowctl
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

# 2. Launch a real application (Notepad).
$np = Start-Process -FilePath notepad.exe -PassThru
$cleanup = {
  if ($np -and -not $np.HasExited) {
    try { $np | Stop-Process -Force } catch {}
  }
}

try {
  # Give the window time to register with the OS.
  Start-Sleep -Seconds 2

  # 3. Detect Notepad via `windowctl windows list --json`.
  $raw = & .\windowctl.exe windows list --json
  if ($LASTEXITCODE -ne 0) { throw 'windowctl windows list failed' }

  $windows = $raw | ConvertFrom-Json
  $notepad = $windows | Where-Object { $_.Title -match 'Notepad|Untitled' } | Select-Object -First 1
  if (-not $notepad) {
    Write-Host 'windowctl windows list output:'
    Write-Host $raw
    throw 'Notepad window was not detected'
  }
  Write-Host "Detected Notepad: ID=$($notepad.ID), Title='$($notepad.Title)'"

  # 4. Execute a move and verify no errors.
  & .\windowctl.exe move --title 'Notepad' --x 100 --y 100 --w 800 --h 600
  if ($LASTEXITCODE -ne 0) { throw 'windowctl move failed' }
  Write-Host 'Move command returned 0.'

  Write-Host '== Smoke test PASSED =='
}
finally {
  & $cleanup
}
