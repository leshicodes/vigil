# Vigil Release Automation (PowerShell)

$Image = "leshicodes/vigil"
$Platforms = "linux/amd64,linux/arm64"

# Get Git context
$Branch = (git rev-parse --abbrev-ref HEAD).Trim()
$Sha = (git rev-parse --short HEAD).Trim()

# Determine tag
if ($Branch -eq "main") {
    $Tag = "latest"
} elseif ($Branch -eq "develop") {
    $Tag = "edge"
} else {
    $Tag = $Branch -replace "[^a-zA-Z0-9.-]", "-"
}

Write-Host "Vigil Release System" -ForegroundColor Cyan
Write-Host "--------------------"
Write-Host "Branch: $Branch"
Write-Host "SHA:    $Sha"
Write-Host "Tag:    $Tag"
Write-Host ""

$Confirm = Read-Host "Build and push to $Image? (y/n)"
if ($Confirm -ne "y") {
    Write-Host "Cancelled."
    exit
}

Write-Host "Starting multi-platform build and push (Standard)..." -ForegroundColor Yellow

docker buildx build --platform $Platforms `
    -t "${Image}:${Tag}" `
    -t "${Image}:${Sha}" `
    --push .

if ($LASTEXITCODE -ne 0) {
    Write-Host "`nStandard release failed." -ForegroundColor Red
    exit 1
}

Write-Host "Starting ARM64 build and push (Raspberry Pi)..." -ForegroundColor Yellow

docker buildx build --platform linux/arm64 `
    -t "${Image}:${Tag}-rpi" `
    -t "${Image}:${Sha}-rpi" `
    -f Dockerfile.pi `
    --push .

if ($LASTEXITCODE -eq 0) {
    Write-Host "`nRelease successful!" -ForegroundColor Green
} else {
    Write-Host "`nRelease failed." -ForegroundColor Red
}
