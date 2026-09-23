# Honeypot Network - Stop Script
# This script stops all services

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "  STOPPING HONEYPOT NETWORK" -ForegroundColor Yellow
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Stopping Docker services..." -ForegroundColor Yellow
docker compose -f dev.compose.yml down

Write-Host ""
Write-Host "All services stopped" -ForegroundColor Green
Write-Host ""
Write-Host "Note: Frontend window needs to be closed manually (Ctrl+C)" -ForegroundColor Yellow
Write-Host ""
