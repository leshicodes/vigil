# Deploy Vigil to Raspberry Pi
$PiHost = "jjambrose1s@192.168.2.157"
$RemoteDir = "/home/jjambrose1s/vigil"

Write-Host "Building Vigil Frontend..." -ForegroundColor Yellow
Set-Location web
npm ci
npx vite build
if ($LASTEXITCODE -ne 0) { Write-Host "Frontend build failed" -ForegroundColor Red; exit 1 }
Set-Location ..

Write-Host "Building Vigil Go Binary (arm64)..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o vigil_bin .
if ($LASTEXITCODE -ne 0) { Write-Host "Go build failed" -ForegroundColor Red; exit 1 }

Write-Host "Creating remote directory structure..." -ForegroundColor Yellow
ssh $PiHost "mkdir -p $RemoteDir/data/captures $RemoteDir/web/dist $RemoteDir/hooks"

Write-Host "Stopping service and copying files to Raspberry Pi..." -ForegroundColor Yellow
ssh $PiHost "sudo systemctl stop vigil"
scp vigil_bin "${PiHost}:${RemoteDir}/vigil"
scp -r ./web/dist/* "${PiHost}:${RemoteDir}/web/dist/"
scp ./hooks/* "${PiHost}:${RemoteDir}/hooks/"
scp ./scripts/vigil.service "${PiHost}:/tmp/vigil.service"

Write-Host "Installing and starting systemd service..." -ForegroundColor Yellow
ssh $PiHost "chmod +x ${RemoteDir}/vigil && sudo mv /tmp/vigil.service /etc/systemd/system/vigil.service && sudo systemctl daemon-reload && sudo systemctl enable vigil && sudo systemctl restart vigil"

Write-Host "Cleaning up local build artifacts..." -ForegroundColor Yellow
Remove-Item vigil_bin -ErrorAction SilentlyContinue

Write-Host "`nDeployment completed successfully!" -ForegroundColor Green
Write-Host "You can view the logs on the Pi by running: journalctl -fu vigil" -ForegroundColor Cyan
