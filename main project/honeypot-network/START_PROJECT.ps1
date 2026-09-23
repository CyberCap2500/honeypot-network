# Honeypot Network - Quick Start Script
# This script starts all services for local development

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "  HONEYPOT NETWORK - STARTING" -ForegroundColor Yellow
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "[1/4] Checking Docker..." -ForegroundColor Yellow
try {
    docker ps | Out-Null
    Write-Host "✓ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Docker is not running. Please start Docker Desktop first." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[2/4] Starting backend services (Docker)..." -ForegroundColor Yellow
docker compose -f dev.compose.yml up -d postgres redis backend

# Wait for services to be healthy
Write-Host "Waiting for services to be ready..." -ForegroundColor Cyan
Start-Sleep -Seconds 10

# Check if backend is responding
$backendReady = $false
for ($i = 1; $i -le 5; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8184/api/health" -UseBasicParsing -TimeoutSec 2
        if ($response.StatusCode -eq 200) {
            $backendReady = $true
            break
        }
    } catch {
        Write-Host "  Attempt $i/5: Backend not ready yet..." -ForegroundColor Gray
        Start-Sleep -Seconds 3
    }
}

if ($backendReady) {
    Write-Host "✓ Backend is ready" -ForegroundColor Green
} else {
    Write-Host "⚠ Backend may not be fully ready. Check logs with: docker logs honeypot-network-backend-1" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "[3/4] Starting frontend..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PSScriptRoot\frontend'; pnpm dev" -WindowStyle Normal

Write-Host "✓ Frontend starting..." -ForegroundColor Green

Write-Host ""
Write-Host "[4/4] Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "  HONEYPOT NETWORK - READY!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Dashboard:     http://localhost:5173" -ForegroundColor Cyan
Write-Host "Backend API:   http://localhost:8184" -ForegroundColor Cyan
Write-Host ""
Write-Host "Honeypot Ports:" -ForegroundColor Yellow
Write-Host "   SSH:    localhost:1262" -ForegroundColor White
Write-Host "   HTTP:   localhost:8339" -ForegroundColor White
Write-Host "   FTP:    localhost:2633" -ForegroundColor White
Write-Host "   SMB:    localhost:4459" -ForegroundColor White
Write-Host "   MySQL:  localhost:3344" -ForegroundColor White
Write-Host "   Redis:  localhost:6355" -ForegroundColor White
Write-Host ""
Write-Host "Test SSH Honeypot:" -ForegroundColor Yellow
Write-Host "   ssh root@localhost -p 1262" -ForegroundColor White
Write-Host ""
Write-Host "View Logs:" -ForegroundColor Yellow
Write-Host "   docker logs honeypot-network-backend-1 -f" -ForegroundColor White
Write-Host ""
Write-Host "Stop Services:" -ForegroundColor Yellow
Write-Host "   docker compose -f dev.compose.yml down" -ForegroundColor White
Write-Host ""
