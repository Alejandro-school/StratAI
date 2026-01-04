# StratAI - CS:GO/CS2 Demo Analysis Platform

> **⚠️ CRITICAL REFERENCE**: For AI coaching architecture and future implementation plans, see [`AI_COACH_ARCHITECTURE.md`](../AI_COACH_ARCHITECTURE.md) at the project root.

## 🏗️ Architecture Overview

StratAI is a **multi-service CS:GO/CS2 demo analysis platform** with automated match detection, comprehensive statistics generation, and an **upcoming AI coaching system** using multi-agent architecture.

### Core Services
- **Python FastAPI Backend** (`backend/app/`) - Authentication, Steam integration, main API
- **Node.js Service** (`backend/node-service/`) - Steam bot for demo downloads, cron jobs
- **Go Service** (`backend/go-service/`) - High-performance demo parsing and analysis (using `demoinfocs-golang`)
- **React Frontend** (`frontend/src/`) - Dashboard with stats visualization
- **[PLANNED] Python ML Service** - Multi-agent AI coaching system (Economy, Positioning, Aim, Utility agents + LLM orchestrator)

### Current Data Flow (Phase 1: Statistics)
```
Steam API → Cron Job → Redis Queue → Steam Bot → Demo Download → Go Parser → Redis → Frontend
```

### Future Data Flow (Phase 2: AI Coaching)
```
Go Parser → PostgreSQL (JSONB) → Python ML Agents → Golden JSON → LLM Orchestrator → Coaching UI
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

### Common Tasks
- **Test Automated Flow**: `python backend/scripts/test_automatic_flow.py`
- **Reprocess All Demos**: `python backend/reprocess_all_simple.py` (Calls Go service for all local demos)
- **Debug Go Parser**: `go run process_demo.go` (in `backend/go-service/`) for standalone file testing

## 🤖 Automatic Match Detection System

### Cron Job Logic (`backend/node-service/services/cronJob.js`)
- Runs every 5 minutes by default (`CRON_INTERVAL` in `.env`)
- Calls `/steam/all-sharecodes` for each registered user
- Automatically queues new match codes in Redis

### Steam Bot Integration (`backend/node-service/services/steamDownloader.js`)
- **Redis Event Listener**: Monitors `rpush` events on `sharecodes:*` keys
- **Steam GC Integration**: Requests demo URLs from CS:GO Game Coordinator
- **Demo Processing Pipeline**: Downloads → Saves to `backend/data/demos/` → Triggers Go analysis

## 🔧 Service-Specific Patterns

### Python Backend (`backend/app/`)
- **FastAPI with Redis Strategy**: Uses `fastapi_users` with Redis session storage
- **Steam OAuth**: Session-based authentication through Steam OpenID
- **CORS Configuration**: Hardcoded for `http://localhost:3000`

### Go Service Architecture (`backend/go-service/`)

**Primary Library**: `demoinfocs-golang` (Markus Walther) - The industry-standard CS:GO/CS2 demo parser.

**Current Implementation Status**: ✅ Phase 1 Complete (Statistics & Timeline)

#### Core Components:
- **Router** (`main.go`): `gorilla/mux` handling:
  - `/process-demo` (POST) - Main demo processing endpoint
  - `/match-details/{matchID}` (GET) - Retrieve match statistics
  
- **Parser Package** (`parser/`):
  - `parser.go`: Core parsing orchestration, registers all handlers
  - `timeline_exporter.go`: Generates round-by-round timeline events + JSON exports
  - `output.go`: Builds final `MatchData` structure
  
- **Handlers** (`handlers/`): Event-driven data capture
  - ✅ `timeline.go` - GameState snapshots, round tracking
  - ✅ `combat.go` - Kills, damage, weapon fire, reloads
  - ✅ `player.go` - Movement, weapon state, spotting, zone tracking
  - ✅ `grenade.go` - All utility usage (smokes, flashes, HE, molotovs)
  - ✅ `round.go` - Round lifecycle, freezetime, end conditions
  - ✅ `economy.go` - Equipment purchases, money tracking
  - ✅ `bomb.go` - Plant/defuse events, defuse kit tracking
  - ✅ `chat.go` - In-game chat capture

- **Analyzers** (`analyzers/`): Advanced metric calculation
  - ✅ `spray.go` - Spray pattern analysis
  - ✅ `mechanics.go` - **Counter-strafe detection**, movement mechanics
  - ✅ `crosshair.go` - Crosshair placement analysis
  - ⚠️ `reaction.go` - Reaction time (DISABLED: `IsSpottedBy()` unreliable in CS2)

- **Models** (`models/`): Data structures
  - `DemoContext` - Central state container for parsing
  - `MatchData` - Final output structure
  - `RoundTimeline` - Per-round event sequences

- **Output Structure**: Generates modular JSON exports in `backend/data/exports/match_{id}/`:
  - `match_summary.json` - High-level match info
  - `timeline/round_{n}.json` - Per-round detailed events
  - `analysis/players.json` - Player-specific statistics
  - `analysis/combat.json` - Damage, kills, duels breakdown
  - `analysis/grenades.json` - Utility usage patterns
  - Data also stored in Redis for quick access

#### What's Extracted (Current Capabilities):
- ✅ Full round-by-round timeline with tick precision
- ✅ Player positions (X, Y, Z) + map area callouts
- ✅ Economic decisions per player per round
- ✅ Damage events with weapon, distance, hit location
- ✅ Counter-strafe efficiency (velocity at shot moment)
- ✅ Spray patterns and recoil compensation
- ✅ Grenade trajectories and effectiveness
- ✅ Trade kill detection
- ✅ Bomb plant/defuse timings with player positions
- ✅ Chat messages (team/all)

#### AI Coaching Readiness (Per [`AI_COACH_ARCHITECTURE.md`](../AI_COACH_ARCHITECTURE.md)):

**Economy Agent Data**: ✅ READY (Updated: 7 Dec 2025)
- ✅ `round_number`, `team_equipment_value`, `player_spend`, `remaining_money`
- ✅ `loss_bonus` per team (CT/T) - IMPLEMENTED
- ✅ `spawn_area` per player ("CTSpawn"/"TSpawn") - IMPLEMENTED
- ✅ **`equipment_value_survived` + `end_round_items`** - IMPLEMENTED (7 Dec 2025)
  - ✅ Uses `player.EquipmentValueCurrent()` from demoinfocs-golang
  - ✅ Captures full inventory at round end
  - ✅ `survived` boolean to distinguish alive vs dead players
- ⚠️ Missing: `enemy_equipment_value_estimated` (needs inference logic)

**Positioning Agent Data**: ✅ READY (Updated: 7 Dec 2025)
- ✅ Position snapshots (`position_x, position_y, position_z`)
- ✅ **Map area names from callouts** - IMPLEMENTED
  - ✅ `killer_place` / `victim_place` in kills (e.g., "Mini", "Squeaky")
  - ✅ `attacker_place` / `victim_place` in damage
  - ✅ `spawn_area` in economy (e.g., "CTSpawn", "TSpawn")
- ✅ Critical events (Death/Kill with positions and callouts)
- ✅ `is_traded` flag for trade kills
- ⚠️ Missing: `view_angle_yaw` in critical events (captured in game_state only)
- ⚠️ Missing: Sampling frequency control (currently samples every frame during active rounds)

**Aim/Mechanics Agent Data**: ✅ MOSTLY READY
- ✅ Counter-strafe efficiency (velocity < 15 u/s detection)
- ✅ Shots fired vs shots hit
- ✅ Weapon used
- ✅ Recoil compensation (mouse movement during spray)
- ⚠️ Missing: `crosshair_distance_to_enemy_head` (needs geometric calculation)
- ⚠️ Missing: `time_to_damage_ms` (needs precise tick-window extraction around duels)
- ❌ Missing: Duel window extraction (2s before + 1s after engagement) - **HIGH PRIORITY FOR AI**

**Utility Agent Data**: ✅ READY (Updated: 7 Dec 2025)
- ✅ Grenade type, throw/landing positions
- ✅ **Thrower area names** - IMPLEMENTED (e.g., "Ramp", "Heaven", "Garage")
- ✅ **Team flash detection** - IMPLEMENTED
  - ✅ `blinded_players[]` array with name, duration, and team
  - ✅ `enemies_blinded` and `allies_blinded` counters
- ✅ Damage dealt (HE/Molotov)
- ⚠️ `land_area_name` empty (requires spatial lookup - low priority)
- 🔴 Missing: Comparison with "perfect lineups" database (future ML component)

#### Next Implementation Steps for AI Coaching:
1. 🔴 **Add Duel Window Extraction**: Detect `PlayerHurt`/`WeaponFire` events and export tick snapshots ±2 seconds
2. 🔴 **Extract View Angles**: Add `view_angle_yaw`/`pitch` to position snapshots
3. 🔴 **Enemy Equipment Estimation**: Calculate estimated enemy economy based on last round's survivors + loss bonus
4. 🔴 **Modular JSON Outputs**: Restructure exports to match AI agent requirements:
   - `match_{id}_economy.json`
   - `match_{id}_positioning.json`
   - `match_{id}_aim.json`
   - `match_{id}_utility.json`
5. 🔴 **PostgreSQL Migration**: Replace Redis with PostgreSQL JSONB storage for historical analysis

#### Code Quality Notes:
- **Concurrency**: Not yet leveraging goroutines for parallel processing (future optimization)
- **Map Manager**: Integrated for callout name resolution (`pkg/maps/`)
- **Error Handling**: Robust with graceful degradation when optional data unavailable
- **Performance**: ~5-10 seconds per full competitive match (30 rounds)

### React Frontend Patterns (`frontend/src/`)
- **Stats Components** (`components/Stats/`):
  - Modular components: `PersonalPerformance`, `MapPerformance`, `Replays2D`
  - Centralized exports in `components/Stats/index.js`
- **Design System**:
  - **Glass Morphism**: `backdrop-filter: blur(20px)`, dark backgrounds (`#0f172a`)
  - **Styles**: CSS modules in `styles/Stats/` with specific prefixes (e.g., `pp-*` for PersonalPerformance)
- **State**: `AuthProvider` + `UserProvider` for global state

## 📊 Redis as Central Data Store

**Critical**: Redis is the backbone for cross-service communication. Always verify Redis is running before debugging issues.

### Key Redis Patterns
- `all_steam_ids` - Set of registered users
- `{steam_id}:authCode` - Steam authentication codes
- `sharecodes:{steam_id}` - FIFO queue of match codes to process
- `sharecode_status:{steam_id}` - Hash tracking processing states (`pending`, `processed`)
- `processed_demos:{steam_id}` - List of analyzed match data
- `match_data:{match_id}` - Individual match statistics

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

### "Go parser fails or returns incomplete data"
1. Check demo file exists in `backend/data/demos/`
2. Verify map files present in `backend/data/maps/` (for callout resolution)
3. Review Go service logs for parsing errors
4. Test standalone: `cd backend/go-service && go run process_demo.go path/to/demo.dem`
5. Check exported JSON structure in `backend/data/exports/match_{id}/`

## 🛠️ File Conventions

### Demo Files
- Stored in `backend/data/demos/` with pattern `match_{match_id}.dem`
- Match IDs extracted from Steam ShareCodes, not random

### Environment Files
- Main `.env` in project root for shared configs
- Service-specific `.env` files override as needed
- Redis connection defaults to `redis://localhost`

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