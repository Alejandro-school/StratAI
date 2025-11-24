# StratAI - CS:GO/CS2 Demo Analysis Platform

## 🏗️ Architecture Overview

StratAI is a **multi-service CS:GO/CS2 demo analysis platform** with automated match detection and comprehensive statistics generation.

### Core Services
- **Python FastAPI Backend** (`backend/app/`) - Authentication, Steam integration, main API
- **Node.js Service** (`backend/node-service/`) - Steam bot for demo downloads, cron jobs
- **Go Service** (`backend/go-service/`) - High-performance demo parsing and analysis
- **React Frontend** (`frontend/src/`) - Dashboard with stats visualization

### Key Data Flow
```
Steam API → Cron Job → Redis Queue → Steam Bot → Demo Download → Go Parser → Redis → Frontend
```

## 🚀 Development Workflow

### Starting All Services
Use the **PowerShell launcher** for Windows development:
```powershell
.\start_services.ps1
```
This opens separate terminals for each service and checks Redis connectivity.

### Service Ports
- FastAPI: `http://localhost:8000` (docs at `/docs`)
- Node.js: `http://localhost:4000`
- Go Service: `http://localhost:8080`
- React Frontend: `http://localhost:3000`

### Testing Automated Flow
```powershell
python backend/scripts/test_automatic_flow.py
```

## 📊 Redis as Central Data Store

**Critical**: Redis is the backbone for cross-service communication. Always verify Redis is running before debugging issues.

### Key Redis Patterns
- `all_steam_ids` - Set of registered users
- `{steam_id}:authCode` - Steam authentication codes
- `sharecodes:{steam_id}` - FIFO queue of match codes to process
- `sharecode_status:{steam_id}` - Hash tracking processing states (`pending`, `processed`)
- `processed_demos:{steam_id}` - List of analyzed match data
- `match_data:{match_id}` - Individual match statistics

### Monitoring Commands
```bash
redis-cli
> SMEMBERS all_steam_ids              # List users
> LLEN sharecodes:{steam_id}          # Queue length
> HGETALL sharecode_status:{steam_id} # Processing states
```

## 🤖 Automatic Match Detection System

### Cron Job Logic (`backend/node-service/services/cronJob.js`)
- Runs every 5 minutes by default (`CRON_INTERVAL` in `.env`)
- Calls `/steam/all-sharecodes` for each registered user
- Automatically queues new match codes in Redis

### Steam Bot Integration (`backend/node-service/services/steamDownloader.js`)
- **Redis Event Listener**: Monitors `rpush` events on `sharecodes:*` keys
- **Steam GC Integration**: Requests demo URLs from CS:GO Game Coordinator
- **Demo Processing Pipeline**: Downloads → Saves to `backend/data/demos/` → Triggers Go analysis

### Critical Files for Bot Debugging
- Check Steam credentials in `backend/node-service/.env`
- Monitor logs with `[CRON]` and `📡` prefixes
- Verify `BOT_USERNAME`, `BOT_PASSWORD`, `BOT_SHARED_SECRET` are set

## 🔧 Service-Specific Patterns

### Python Backend (`backend/app/`)
- **FastAPI with Redis Strategy**: Uses `fastapi_users` with Redis session storage
- **Steam OAuth**: Session-based authentication through Steam OpenID
- **CORS Configuration**: Hardcoded for `http://localhost:3000`
- **Route Structure**: `steam_auth.py` (OAuth), `steam_service.py` (API calls), `auth_status.py` (session checks)

### Go Service Architecture
- **Gorilla Mux Router**: Simple REST endpoints for demo processing
- **Concurrency Pattern**: Processes one demo at a time to avoid resource conflicts
- **Redis Integration**: Stores parsed results as JSON in `match_data:` keys
- **File Handling**: Reads from `../data/demos/` directory relative to go-service

### React Frontend Patterns
- **Route Protection**: Uses `RequireAuth` wrapper for authenticated routes
- **Context Providers**: `AuthProvider` + `UserProvider` for global state
- **Component Structure**: 
  - `components/Dashboard/` - Main dashboard
  - `components/Stats/` - Statistics visualization components
  - `auth/` - Authentication components and hooks

## 🔍 Common Debugging Scenarios

### "Cron not detecting new matches"
1. Verify user has `authCode` and `knownCode` in Redis
2. Check Steam Auth Code hasn't expired (refresh from Steam Support)
3. Confirm cron logs show `⏰ [CRON] Iniciando consulta`

### "Demo downloads failing"
1. Check bot Steam connection: Look for `✅ Bot conectado a Steam`
2. Verify Redis keyspace events: `CONFIG GET notify-keyspace-events` should include `KEA`
3. Ensure bot is friends with the user on Steam

### "Frontend shows no data"
1. Check Redis for processed demos: `LLEN processed_demos:{steam_id}`
2. Verify Go service processed the demo: Check `match_data:{match_id}` exists
3. Confirm FastAPI endpoints return data at `/docs`

## 🛠️ File Conventions

### Demo Files
- Stored in `backend/data/demos/` with pattern `match_{match_id}.dem`
- Match IDs extracted from Steam ShareCodes, not random

### Environment Files
- Main `.env` in project root for shared configs
- Service-specific `.env` files override as needed
- Redis connection defaults to `redis://localhost`

### PowerShell Scripts
- `start_services.ps1` - Development launcher
- `test_*.ps1` files - Individual service testing
- `check_current_state.ps1` - System health check

## 🎯 Key Integration Points

### Steam API Dependencies
- **Authentication**: Requires Steam OpenID for user login
- **Match Data**: Uses Steam Web API with user's auth codes
- **Demo Access**: Bot must be friends with users to access private matches

### Cross-Service Communication
- **Python → Node.js**: Via Redis queues (`rpush` to `sharecodes:*`)
- **Node.js → Go**: HTTP POST to `/process-downloaded-demo`
- **Go → Frontend**: Via Redis storage (`match_data:*` keys)

### Third-Party Dependencies
- **Steam Bot**: Uses `globaloffensive` npm package for CS:GO integration
- **Demo Parsing**: Go service uses custom demo parser (not demoinfocs)
- **Frontend Charts**: Uses `react-chartjs-2` for statistics visualization

## 🧠 Code Review & Feature Evaluation Philosophy

**Apply ruthless technical honesty when evaluating proposals, code changes, or new features.**

### When Reviewing Code or Ideas:
1. **Challenge weak reasoning** - If the technical approach is flawed, break it down and explain why
2. **Expose hidden complexity** - Point out underestimated implementation costs, technical debt, or architectural risks
3. **Question assumptions** - Don't accept "this should work" without evidence from the codebase
4. **Identify time-wasters** - Call out feature creep, over-engineering, or solutions looking for problems
5. **Show opportunity cost** - What's NOT being built by pursuing this approach?

### Feature Evaluation Criteria:
- **Problem-Solution Fit**: Does this solve a real user pain point or is it speculative?
- **Implementation Cost**: Realistic effort estimate including testing, edge cases, and integration
- **Architectural Impact**: Does it fit cleanly or require refactoring? Will it create maintenance burden?
- **User Value**: Is this user-validated or just "sounds cool"? What's the actual impact?
- **Alternatives**: Are there simpler approaches being overlooked? Can existing features be improved instead?

### Communication Style:
- **Be direct, not diplomatic** - Say "this won't scale" instead of "we might want to consider scalability"
- **Prioritize ruthlessly** - Not everything deserves implementation time
- **Demand specifics** - Vague ideas get challenged until they're concrete or abandoned
- **Focus on shipping** - Favor pragmatic working solutions over perfect architectures that never launch

**Goal**: Accelerate progress by cutting through noise, avoiding technical dead-ends, and focusing effort on high-impact work.