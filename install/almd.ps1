# PowerShell wrapper for launching the almd Go application
# Finds and executes almd-cli.exe with all arguments.

# Deferred self-update logic (Windows PowerShell)
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$InstallRoot = Join-Path $ScriptDir 'install' # Assuming install scripts are here relative to almd.ps1
$UpdatePending = Join-Path $ScriptDir 'update_pending' # Check in ScriptDir
$NextDir = Join-Path $ScriptDir 'next' # Check in ScriptDir
if ((Test-Path $UpdatePending -PathType Leaf) -and (Test-Path $NextDir -PathType Container)) {
  Write-Host "Applying deferred update..."
  # Remove old files
  if (Test-Path (Join-Path $ScriptDir 'src')) { Remove-Item -Recurse -Force (Join-Path $ScriptDir 'src') }
  if (Test-Path (Join-Path $ScriptDir 'almd')) { Remove-Item -Force (Join-Path $ScriptDir 'almd') }
  if (Test-Path (Join-Path $ScriptDir 'almd.ps1')) { Remove-Item -Force (Join-Path $ScriptDir 'almd.ps1') }
  if (Test-Path (Join-Path $ScriptDir 'almd-cli.exe')) { Remove-Item -Force (Join-Path $ScriptDir 'almd-cli.exe') }

  # Copy new files from 'next' structure (assuming structure mirrors final layout)
  $nextSrcPath = Join-Path $NextDir 'src' # Expecting source code here
  if (Test-Path $nextSrcPath -PathType Container) {
    Copy-Item -Recurse -Force $nextSrcPath $ScriptDir
  }
  $nextCliPath = Join-Path $NextDir 'almd-cli.exe' # Expecting executable here
  if (Test-Path $nextCliPath -PathType Leaf) {
    Copy-Item -Force $nextCliPath $ScriptDir
  }
  $nextAlmdPath = Join-Path $NextDir 'almd' # Expecting bash wrapper here
  if (Test-Path $nextAlmdPath -PathType Leaf) {
    Copy-Item -Force $nextAlmdPath $ScriptDir
  }
  $nextAlmdPs1Path = Join-Path $NextDir 'almd.ps1' # Expecting PS wrapper here
  if (Test-Path $nextAlmdPs1Path -PathType Leaf) {
    Copy-Item -Force $nextAlmdPs1Path $ScriptDir
  }

  # Cleanup
  Remove-Item -Recurse -Force $NextDir
  Remove-Item -Force $UpdatePending

  # Print update success and new version
  $CLI_BIN = Join-Path $ScriptDir 'almd-cli.exe'
  if (Test-Path $CLI_BIN -PathType Leaf) {
    try {
      $version = & $CLI_BIN --version 2>$null
      if ($version) {
        Write-Host "Almandine CLI updated successfully! New version: $($version -join '')"
      } else {
        Write-Host "Almandine CLI updated successfully! (version unknown)"
      }
    } catch {
      Write-Host "Almandine CLI updated successfully! (Error checking version: $_)"
    }
  } else {
    Write-Host "Almandine CLI updated successfully! (almd-cli.exe not found to check version)"
  }
}

# Determine the location of the main executable
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$CLI_BIN = Join-Path $ScriptDir 'almd-cli.exe'

if (-not (Test-Path $CLI_BIN -PathType Leaf)) {
  Write-Error "Main executable 'almd-cli.exe' not found in $ScriptDir"
  exit 1
}

# Execute the Go binary, passing all arguments
& $CLI_BIN @args
exit $LASTEXITCODE
