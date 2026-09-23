# MirageNet

<div align="center">
  <img src="miragenetlogo.png" alt="MirageNet Logo" width="200"/>
</div>

<div align="center">

```
███╗   ███╗██╗██████╗  █████╗  ██████╗ ███████╗███╗   ██╗███████╗████████╗
████╗ ████║██║██╔══██╗██╔══██╗██╔════╝ ██╔════╝████╗  ██║██╔════╝╚══██╔══╝
██╔████╔██║██║██████╔╝███████║██║  ███╗█████╗  ██╔██╗ ██║█████╗     ██║   
██║╚██╔╝██║██║██╔══██╗██╔══██║██║   ██║██╔══╝  ██║╚██╗██║██╔══╝     ██║   
██║ ╚═╝ ██║██║██║  ██║██║  ██║╚██████╔╝███████╗██║ ╚████║███████╗   ██║   
╚═╝     ╚═╝╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚══════╝   ╚═╝   
```

</div>

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react&logoColor=black)](https://react.dev)
[![License: AGPLv3](https://img.shields.io/badge/License-AGPL_v3-purple.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![MITRE ATT&CK](https://img.shields.io/badge/MITRE_ATT%26CK-27_Techniques-orange?style=flat)](https://attack.mitre.org/)

> **Adaptive deception network with attacker profiling**
>
> Multi-protocol honeypot network that simulates six real services, captures attacker behavior, maps to MITRE ATT&CK, computes risk scores, and visualizes threats through a command-deck dashboard.

## What It Does

- Simulates 6 services: SSH (fake shell with 25+ commands), HTTP (WordPress/phpMyAdmin fakes), FTP (PASV file capture), SMB (negotiate), MySQL (wire protocol), Redis (RESP)
- Captures every attacker interaction: credentials, commands, file uploads, scanning patterns, tool fingerprints
- Maps behavior to 27 MITRE ATT&CK techniques across 8 tactics with single-event and sliding-window detection
- **Computes attacker risk scores (0-100)** based on technique severity, session count, command sophistication, and escalation speed
- Extracts IOCs (IPs, URLs, domains, user-agents, credentials) with confidence scoring and deduplication
- Exports threat intelligence as STIX 2.1 bundles and firewall blocklists (iptables, nginx deny, plain text, CSV)
- Records SSH sessions in asciicast v2 format, replayable in the browser via xterm.js
- Streams events in real time via WebSocket to a React dashboard with attack maps, MITRE heatmaps, and session replay

## Quick Start

```bash
git clone <your-repo-url>
cd miragenet
cp .env.example .env
docker compose -f dev.compose.yml up -d
```

Dashboard loads at `http://localhost:56722`. Connect to the SSH honeypot to see your first captured session:

```bash
ssh root@localhost -p 1262
```

Use any password. Run commands like `ls`, `cat /etc/passwd`, `wget http://example.com/payload.sh`, and watch events stream into the dashboard.

> [!TIP]
> This project uses [`just`](https://github.com/casey/just) as a command runner. Type `just` to see all available commands.
>
> Install: `curl -sSf https://just.systems/install.sh | bash -s -- --to ~/.local/bin`

## Architecture

```
                    ┌─────────────────────────────────────────────┐
    Attackers       │            MirageNet Backend                │
                    │                                             │
  ┌──────┐         │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐   │
  │ SSH  │────2222──│──│ sshd │  │ httpd│  │ ftpd │  │ smbd │   │
  │Client│         │  └──┬───┘  └──┬───┘  └──┬───┘  └──┬───┘   │
  └──────┘         │     │         │         │         │        │
  ┌──────┐         │  ┌──┴───┐  ┌──┴───┐                        │
  │MySQL │────3307──│──│mysqld│  │redisd│                        │
  │Client│         │  └──┬───┘  └──┬───┘                        │
  └──────┘         │     │         │                             │
                   │     ▼         ▼                             │
                   │  ┌─────────────────┐                        │
                   │  │    Event Bus    │  (fan-out pub/sub)      │
                   │  └────────┬────────┘                        │
                   │           │                                 │
                   │     ┌─────┴─────┐                           │
                   │     │ Processor │  (4 worker goroutines)    │
                   │     │  GeoIP    │                           │
                   │     │  MITRE    │                           │
                   │     │  Risk     │  ← NEW: Risk Score Engine │
                   │     │  Store    │                           │
                   │     │  Stream   │                           │
                   │     └───────────┘                           │
                   │                                             │
                   │  ┌─────────────────┐                        │
                   │  │   REST API      │  Chi router :8000      │
                   │  │   WebSocket     │  /ws/events             │
                   │  └─────────────────┘                        │
                   └──────────────┬──────────────────────────────┘
                                  │
                   ┌──────────────┴──────────────────┐
                   │         Frontend                │
                   │   React 19 + TypeScript         │
                   │   Command Deck Layout           │
                   │   Risk Radar • Story Mode       │
                   └─────────────────────────────────┘
```

## Services

| Service | Port (dev) | Protocol | Interaction Depth |
|---------|------------|----------|-------------------|
| SSH | 1262 → 2222 | x/crypto/ssh | Full shell with filesystem, 25+ commands, session recording |
| HTTP | 8339 → 8080 | net/http | WordPress/phpMyAdmin fakes, scanner detection, vulnerability path traps |
| FTP | 2633 → 2121 | Raw TCP | AUTH + PASV data channel, file upload capture (1MB cap) |
| SMB | 4459 → 4450 | Raw TCP | NetBIOS framing + negotiate response, SMB1/SMB2 detection |
| MySQL | 3344 → 3307 | Raw TCP | Binary wire protocol greeting, auth capture, query handling |
| Redis | 6355 → 6380 | tidwall/redcon | RESP protocol, PING/AUTH/INFO/CONFIG/SET/GET/KEYS |

## Stack

**Backend:** Go 1.25, Chi v5, nhooyr.io/websocket, pgxpool (PostgreSQL), go-redis, zerolog, Cobra CLI

**Frontend:** React 19, TypeScript, Vite 6, SCSS (dark theme with OKLCH-inspired tokens), TanStack Query v5, Zustand, Recharts, react-leaflet, xterm.js

**Infrastructure:** Docker Compose, PostgreSQL 17, Redis 7.4, nginx reverse proxy, multi-stage builds

## API

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check with version and sensor ID |
| `GET /api/stats/overview` | Total events, events by service, active sessions |
| `GET /api/stats/countries` | Event counts by country |
| `GET /api/stats/credentials` | Top captured username/password pairs |
| `GET /api/events` | Paginated events with IP filtering |
| `GET /api/sessions` | Paginated session list |
| `GET /api/sessions/{id}` | Session detail with commands and techniques |
| `GET /api/sessions/{id}/replay` | Asciicast v2 recording for session replay |
| `GET /api/attackers` | Attacker list with geo and tool info |
| `GET /api/attackers/{id}/risk` | **NEW:** Risk score breakdown for an attacker |
| `GET /api/mitre/techniques` | Full technique catalog |
| `GET /api/mitre/heatmap` | Technique detection counts for heatmap |
| `GET /api/iocs` | Paginated IOC list |
| `GET /api/iocs/export/stix` | STIX 2.1 bundle export |
| `GET /api/iocs/export/blocklist` | Blocklist export (plain, iptables, nginx, csv) |
| `WS /ws/events` | Real-time event stream |

## MITRE ATT&CK Coverage

MirageNet detects 27 techniques across 8 tactics:

| Tactic | Techniques |
|--------|------------|
| Reconnaissance | T1595, T1595.002 |
| Initial Access | T1078, T1190 |
| Execution | T1059.004 |
| Persistence | T1053.003, T1543.002, T1098.004 |
| Credential Access | T1110, T1110.001, T1110.003, T1552.001 |
| Discovery | T1082, T1083, T1046, T1018, T1049, T1016 |
| Lateral Movement | T1021.004 |
| Command and Control | T1105, T1071.001 |
| Impact | T1496, T1485, T1489 |

Detection uses two strategies: single-event pattern matching (command → technique) and multi-event sliding windows (5+ auth attempts in 5 minutes → T1110 Brute Force, 3+ distinct services in 60 seconds → T1046 Network Service Discovery).

## CLI

```bash
mirage serve                       # Start all services
mirage serve --config mirage.yml   # Custom config file
mirage migrate up                  # Apply database migrations
mirage migrate down                # Rollback last migration
mirage migrate status              # Show migration status
mirage keygen                      # Generate SSH host key
mirage version                     # Show version and module info
```

## Configuration

All settings can be set via YAML config file or environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MIRAGE_SENSOR_ID` | `mirage-01` | Sensor identifier |
| `MIRAGE_SSH_ENABLED` | `true` | Enable SSH honeypot |
| `MIRAGE_SSH_PORT` | `2222` | SSH listen port |
| `MIRAGE_HTTP_ENABLED` | `true` | Enable HTTP honeypot |
| `MIRAGE_HTTP_PORT` | `8080` | HTTP listen port |
| `MIRAGE_FTP_ENABLED` | `true` | Enable FTP honeypot |
| `MIRAGE_FTP_PORT` | `2121` | FTP listen port |
| `MIRAGE_SMB_ENABLED` | `true` | Enable SMB honeypot |
| `MIRAGE_SMB_PORT` | `4450` | SMB listen port |
| `MIRAGE_MYSQL_ENABLED` | `true` | Enable MySQL honeypot |
| `MIRAGE_MYSQL_PORT` | `3307` | MySQL listen port |
| `MIRAGE_REDIS_ENABLED` | `true` | Enable Redis honeypot |
| `MIRAGE_REDIS_PORT` | `6380` | Redis listen port |
| `MIRAGE_API_PORT` | `8000` | Dashboard API listen port |
| `MIRAGE_DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `MIRAGE_REDIS_URL` | `redis://...` | Infrastructure Redis URL |
| `MIRAGE_GEOIP_DB_PATH` | `data/GeoLite2-City.mmdb` | MaxMind database path |
| `MIRAGE_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `MIRAGE_CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |

## Project Structure

```
miragenet/
├── cmd/mirage/            # CLI entry point
├── pkg/types/             # Shared domain types (Event, Session, IOC)
├── internal/
│   ├── sshd/              # SSH honeypot (shell, filesystem, commands)
│   ├── httpd/             # HTTP honeypot (WordPress, phpMyAdmin fakes)
│   ├── ftpd/              # FTP honeypot (auth capture, upload logging)
│   ├── smbd/              # SMB honeypot (negotiate-only)
│   ├── mysqld/            # MySQL honeypot (wire protocol, query logging)
│   ├── redisd/            # Redis honeypot (RESP commands)
│   ├── event/             # Event bus + processor pipeline
│   ├── store/             # PostgreSQL + Redis persistence
│   ├── mitre/             # ATT&CK technique detection engine
│   ├── risk/              # NEW: Risk score computation engine
│   ├── intel/             # IOC extraction, STIX export, blocklists
│   ├── api/               # REST + WebSocket dashboard API
│   └── ...                # config, geo, ratelimit, session, ui
├── frontend/              # React 19 + TypeScript dashboard
├── migrations/            # PostgreSQL schema (goose format)
├── infra/                 # Docker, nginx, Redis configs
├── meragenet/             # Project documentation and planning
└── compose.yml            # Production Docker Compose
```

## Common Issues

**SSH host key error on repeated starts**
```
ssh: handshake failed: ssh: no common algorithm for host key
```
Delete `data/hostkey_ed25519` and restart. A new key will be auto-generated.

**PostgreSQL connection refused**
Make sure the database is running. With Docker: `docker compose up -d postgres`. Check that PostgreSQL is listening on the correct port (53743 in dev mode).

**Frontend WebSocket not connecting**
The Vite dev server proxies `/ws/*` to the backend. Make sure the backend is running on port 8184 before starting the frontend.

## Legal Disclaimer

This tool is designed for authorized security research and educational purposes. Deploying honeypots on networks you do not own or control may violate local laws and regulations. Before deploying:

- Ensure you have authorization from network owners
- Check your cloud provider's acceptable use policy (some prohibit honeypots)
- Be aware that honeypots collect attacker data, which may include personal information subject to privacy regulations (GDPR, CCPA)
- Do not use captured data for offensive purposes
- If deploying on a public IP, understand that you are inviting connections from potentially hostile actors

The authors are not responsible for misuse of this software.

## Attribution

MirageNet is built upon open-source honeypot architecture patterns. The core event-driven design, protocol emulation approach, and MITRE ATT&CK detection engine draw from established honeypot systems. Original contributions in this implementation include:
- Attacker risk scoring algorithm
- Command-deck UI/UX design
- Adaptive features architecture (in development)

See `meragenet/` directory for full project planning and design documentation.

## License

AGPL 3.0
