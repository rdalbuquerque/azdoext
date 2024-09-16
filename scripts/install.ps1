# Determine OS and Architecture
$os = "windows" # Assuming Windows, as this is a PowerShell script
$arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Define download URL
$tar = "azdoext-$os-$arch.zip"
$url = "https://github.com/rdalbuquerque/azdoext/releases/latest/download"

# Get the latest version
$version = Invoke-RestMethod -Uri "https://api.github.com/repos/rdalbuquerque/azdoext/releases/latest" | Select-Object -ExpandProperty tag_name
$filename = "azdoext_${version}_$os-$arch.exe"

# Download and install
Write-Host "Downloading version $version of azdoext-$os-$arch..."
Invoke-WebRequest -Uri "$url/$tar" -OutFile "$env:TEMP\$tar"
if (!(Test-Path "$env:TEMP\$tar")) {
    Write-Host "Failed to download azdoext-$os-$arch"
    throw "Failed to download azdoext-$os-$arch"
}

# Extracting the ZIP file
Expand-Archive -Path "$env:TEMP\$tar" -DestinationPath $env:TEMP -Force

# Move the file to a specific location
$destination = "$env:LOCALAPPDATA\azdoext\celify.exe"
Write-Host "Moving $env:TEMP\$filename to $destination"
$null = New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\azdoext"
if (Test-Path $destination) {
    Write-Warning "Overwriting $destination"
    Move-Item -Path "$env:TEMP\$filename" -Destination $destination -Force
}

# Add the azdoext.exe to the PATH if it's not
$paths = $env:Path -split ";"
if ($paths -notcontains "$env:LOCALAPPDATA\azdoext") {
    Write-Host "Adding $env:LOCALAPPDATA\Programs\azdoext to the PATH"
    $env:Path += ";$env:LOCALAPPDATA\Programs\azdoext"
    Write-Host "Please alter your PATH variable to include '$env:LOCALAPPDATA\Programs\azdoext' permanently"
}
azdoext --version
if ($LASTEXITCODE -ne 0) {
    throw "Failed to install azdoext"
} else {
    Write-Host "azdoext installed successfully"
}

# Clean up
Remove-Item "$env:TEMP\$tar"