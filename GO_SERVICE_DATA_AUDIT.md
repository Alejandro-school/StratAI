# StratAI - Go Service Data Extraction Audit & Enhancement Roadmap

**Date**: December 4, 2025  
**Role**: Service Manager - Data Architecture  
**Objective**: Audit current data extraction, identify library untapped potential, propose AI-relevant advanced metrics

---

## 📊 CURRENT STATE ANALYSIS

### What We're Extracting Now

#### ✅ Core Game Events (SOLID)
- **Kills**: Position, weapon, headshot, wallbang, distance, through smoke, no-scope, blind
- **Damage**: Health/armor damage, hit group, victim health
- **Grenades**: Flash (victims + duration), HE, Smoke, Molotov positions
- **Bomb Events**: Plant, defuse, explosion with site
- **Economy**: Start money, equipment value, loss bonus, buy classification
- **Round Data**: Winner, reason, scores, bomb plant status

#### ✅ Advanced Analytics (GOOD)
- **Mechanics**: Counter-strafe rating, recoil control, time-to-damage
- **Crosshair Placement**: Head-level placement, enemy proximity awareness
- **Spray Analysis**: Recoil control, burst accuracy, spray patterns
- **Reaction Times**: First-seen to first-shot timing with metadata (flash, smoke, distance)
- **Movement**: Position tracking, velocity, ducking state, view angles

#### ✅ Timeline System (EXCELLENT)
- Game state snapshots every second (128 ticks)
- Full player states: HP, armor, position, velocity, equipment
- Round lifecycle events
- Real-time tactical situation detection

#### ✅ Grenade Trajectories (NEW)
- Full projectile paths with bounces
- Position sampling every tick
- Useful for coaching/replay analysis

---

## 🔍 DEMOINFOCS LIBRARY UNTAPPED POTENTIAL

After deep analysis of `github.com/markus-wa/demoinfocs-golang`, here's what we're **NOT using** but should:

### 🎯 HIGH PRIORITY - Missing Gold

#### 1. **Weapon State & Ammo Management**
```go
// AVAILABLE BUT UNUSED
Equipment.AmmoInMagazine()    // Bullets left in mag
Equipment.AmmoReserve()       // Reserve ammo
Equipment.ZoomLevel()         // AWP/Scope zoom level
Equipment.ReloadState()       // Is reloading
```

**AI VALUE**: 
- Ammo management patterns (when players reload, panic reloads)
- Zoom level on AWP kills (quickscope vs hold angle)
- Reserve ammo tracking for economy decisions

**Implementation**:
```go
type WeaponStateEvent struct {
    Tick           int     `json:"tick"`
    PlayerSteamID  uint64  `json:"player_steam_id"`
    Weapon         string  `json:"weapon"`
    AmmoInMag      int     `json:"ammo_in_mag"`
    AmmoReserve    int     `json:"ammo_reserve"`
    IsReloading    bool    `json:"is_reloading"`
    ZoomLevel      int     `json:"zoom_level"`
}
```

#### 2. **Player Visibility & Spotted System**
```go
// AVAILABLE BUT UNUSED
Player.IsSpottedBy(other *Player)  // Is player spotted by enemy
Player.HasSpotted(other *Player)   // Has player spotted enemy
PlayerSpottersChanged event        // Spotters list updated
```

**AI VALUE**:
- **Information trading**: Who spots who first
- **Map awareness**: Players checking corners vs tunnel vision
- **Positioning mistakes**: Getting spotted unnecessarily

**Implementation**:
```go
type SpottingEvent struct {
    Tick           int     `json:"tick"`
    SpotterSteamID uint64  `json:"spotter_steam_id"`
    SpottedSteamID uint64  `json:"spotted_steam_id"`
    SpotDuration   int     `json:"spot_duration_ticks"`
}
```

#### 3. **Flash Effectiveness Metrics**
```go
// PARTIALLY USED, CAN BE ENHANCED
Player.FlashDurationTime()          // Total flash duration
Player.FlashDurationTimeRemaining() // Remaining blind time
Player.IsBlinded()                  // Currently blinded
```

**AI VALUE**:
- **Flash effectiveness**: Did flash lead to kill?
- **Flash recovery**: How fast player peeks after flash
- **Self-flash detection**: Team flash damage

**Current**: We track flash victims + duration  
**Missing**: Post-flash actions, flash-to-kill conversion

#### 4. **Zone & Area Detection**
```go
// AVAILABLE BUT UNUSED
Player.IsInBombZone()  // In bomb plant area
Player.IsInBuyZone()   // In buy zone
Player.IsAirborne()    // Jumping/falling
```

**AI VALUE**:
- **Save detection**: Players staying in buy zone during loss
- **Jump peeking**: Airborne + shooting patterns
- **Post-plant positioning**: Time outside bombsite after plant

#### 5. **Weapon Pickup & Drop Events**
```go
// EVENTS AVAILABLE
ItemEquip event   // When player equips weapon
ItemPickup event  // When player picks up weapon
ItemDrop event    // When player drops weapon
```

**AI VALUE**:
- **Weapon sharing**: Team dropping rifles for buy-less teammates
- **Upgrade decisions**: When players switch weapons mid-round
- **Death location utility**: Tracking valuable weapon drops

#### 6. **Inferno (Molotov/Incendiary) Advanced Data**
```go
// AVAILABLE BUT UNDERUTILIZED
Inferno.Fires()         // Individual fire positions
Inferno.ConvexHull2D()  // Area coverage calculation
Inferno.Active()        // Currently burning fires
```

**AI VALUE**:
- **Molotov effectiveness**: Area denial coverage
- **Lineup precision**: Consistency of molly positions
- **Burn damage**: Players caught in fires

---

### 🎮 MEDIUM PRIORITY - Nice to Have

#### 7. **Footstep & Sound Events**
```go
PlayerSound event  // When player makes sound
PlayerFootstep event  // Footstep sounds
```

**AI VALUE**:
- Sound discipline tracking
- Walking vs running patterns
- Audio cues for positioning

#### 8. **Jump Events**
```go
PlayerJump event  // When player jumps
```

**AI VALUE**:
- Movement patterns
- Jump spot usage
- Bhop detection

#### 9. **Team Money Aggregation**
```go
TeamState.MoneySpentThisRound()
TeamState.MoneySpentTotal()
TeamState.CurrentEquipmentValue()
```

**AI VALUE**:
- Team economy health
- Coordinated buy patterns
- Equipment value at round end

#### 10. **Defuse Kit Tracking**
```go
Player.HasDefuseKit  // Defuse kit ownership
```

**AI VALUE**:
- Who bought kit
- Kit holder positioning
- Defuse attempts without kit

---

## 🧠 CUSTOM ADVANCED METRICS (Beyond Library)

Metrics that **require custom logic** beyond what demoinfocs provides:

### 🔥 TIER 1 - Critical for AI Training

#### 1. **Trade Efficiency**
```go
type TradeEvent struct {
    InitialKillTick    int     `json:"initial_kill_tick"`
    TradeKillTick      int     `json:"trade_kill_tick"`
    TradeTimeMs        int     `json:"trade_time_ms"`
    TradedPlayer       uint64  `json:"traded_player_steam_id"`
    TradingPlayer      uint64  `json:"trading_player_steam_id"`
    TradeDistance      float64 `json:"trade_distance"`
    WasSuccessful      bool    `json:"was_successful"`
}
```

**Why It Matters**: Crucial team coordination metric. AI can learn optimal trade timing.

#### 2. **Utility Usage Effectiveness**
```go
type UtilityEffectiveness struct {
    UtilityType        string  `json:"utility_type"` // "flash", "smoke", "molly"
    ThrowTick          int     `json:"throw_tick"`
    LandingTick        int     `json:"landing_tick"`
    
    // Effectiveness Metrics
    PlayersAffected    int     `json:"players_affected"`
    EnemiesAffected    int     `json:"enemies_affected"`
    TeammatesAffected  int     `json:"teammates_affected"` // Team damage
    
    // Impact
    LedToKill          bool    `json:"led_to_kill"`
    TicksUntilKill     int     `json:"ticks_until_kill"`
    DeniedPlant        bool    `json:"denied_plant"`
    DeniedDefuse       bool    `json:"denied_defuse"`
}
```

**Why It Matters**: Separates good utility from wasted utility. Key for coaching.

#### 3. **Positioning Risk Score**
```go
type PositionRiskEvent struct {
    Tick              int      `json:"tick"`
    PlayerSteamID     uint64   `json:"player_steam_id"`
    Position          Vector3  `json:"position"`
    
    // Risk Factors
    ExposedAngles     int      `json:"exposed_angles"` // Number of angles player is exposed to
    NearbyEnemies     int      `json:"nearby_enemies"`
    TeamSupport       int      `json:"team_support"`   // Teammates within trade distance
    EscapeRoutes      int      `json:"escape_routes"`  // Available exits
    RiskScore         float64  `json:"risk_score"`     // 0-100
}
```

**Why It Matters**: Teaches positioning fundamentals. Detects over-aggression.

#### 4. **Rotation Timing**
```go
type RotationEvent struct {
    Tick              int     `json:"tick"`
    PlayerSteamID     uint64  `json:"player_steam_id"`
    FromSite          string  `json:"from_site"` // "A", "B", "Mid"
    ToSite            string  `json:"to_site"`
    
    // Timing Analysis
    InformationSource string  `json:"information_source"` // "spotted", "sound", "callout"
    RotationSpeed     float64 `json:"rotation_speed_units_per_sec"`
    WasLate           bool    `json:"was_late"`
    MissedAction      bool    `json:"missed_action"` // Arrived after round decided
}
```

**Why It Matters**: Critical tactical decision. AI can learn rotation triggers.

#### 5. **Engagement Distance Optimization**
```go
type EngagementProfile struct {
    PlayerSteamID     uint64            `json:"player_steam_id"`
    Weapon            string            `json:"weapon"`
    
    // Distance-based performance
    KillsByDistance   map[string]int    `json:"kills_by_distance"` // "0-5m", "5-15m", "15-30m", "30m+"
    DeathsByDistance  map[string]int    `json:"deaths_by_distance"`
    HeadshotByDistance map[string]float64 `json:"headshot_rate_by_distance"`
    OptimalRange      string            `json:"optimal_range"`
    WeakRange         string            `json:"weak_range"`
}
```

**Why It Matters**: Identifies weapon comfort zones. Teaches when to engage.

#### 6. **First Contact Win Rate**
```go
type FirstContactStats struct {
    PlayerSteamID      uint64   `json:"player_steam_id"`
    
    // Entry fragging
    FirstContactWins   int      `json:"first_contact_wins"`
    FirstContactLosses int      `json:"first_contact_losses"`
    FirstContactWinRate float64 `json:"first_contact_win_rate"`
    
    // Contextual
    AsAttacker        int      `json:"as_attacker"`
    AsDefender        int      `json:"as_defender"`
    WithFlashAssist   int      `json:"with_flash_assist"`
    WithoutUtility    int      `json:"without_utility"`
}
```

**Why It Matters**: Measures entry fragging effectiveness. Core T-side skill.

---

### 💎 TIER 2 - Coaching & Analysis

#### 7. **Grenade Lineup Consistency**
```go
type GrenadeLineupAnalysis struct {
    PlayerSteamID       uint64              `json:"player_steam_id"`
    GrenadeType         string              `json:"grenade_type"`
    TargetLocation      string              `json:"target_location"` // "A site smoke", "B flash"
    
    // Consistency Metrics
    Throws              []Vector3           `json:"throws"`
    LandingPoints       []Vector3           `json:"landing_points"`
    AverageDeviation    float64             `json:"avg_deviation_units"`
    ConsistencyScore    float64             `json:"consistency_score"` // 0-100
}
```

**Why It Matters**: Measures lineup mastery. Identifies practice needs.

#### 8. **Multi-Kill Sequence Analysis**
```go
type MultiKillSequence struct {
    PlayerSteamID       uint64      `json:"player_steam_id"`
    KillCount           int         `json:"kill_count"`
    SequenceStartTick   int         `json:"start_tick"`
    SequenceDuration    int         `json:"duration_ticks"`
    
    // Sequence details
    KillPositions       []Vector3   `json:"kill_positions"`
    VictimDistances     []float64   `json:"victim_distances"`
    TimeBetweenKills    []int       `json:"time_between_kills_ticks"`
    
    // Difficulty
    DifficultyScore     float64     `json:"difficulty_score"` // Based on angles, flashes, HP
}
```

**Why It Matters**: Identifies clutch ability. Tracks composure under pressure.

#### 9. **Crossfire Setup Detection**
```go
type CrossfireSetup struct {
    Tick                int        `json:"tick"`
    Player1SteamID      uint64     `json:"player1_steam_id"`
    Player2SteamID      uint64     `json:"player2_steam_id"`
    
    // Geometry
    Player1Pos          Vector3    `json:"player1_pos"`
    Player2Pos          Vector3    `json:"player2_pos"`
    CoveredArea         []Vector3  `json:"covered_area"`
    AngleDifference     float64    `json:"angle_difference"` // Optimal: 90°
    
    // Effectiveness
    LedToKill           bool       `json:"led_to_kill"`
    EnemyTrapped        bool       `json:"enemy_trapped"`
}
```

**Why It Matters**: Teaches team positioning. Measures coordination.

#### 10. **Economy Decision Quality**
```go
type EconomyDecision struct {
    PlayerSteamID       uint64  `json:"player_steam_id"`
    Tick                int     `json:"tick"`
    
    // Context
    Money               int     `json:"money"`
    TeamMoney           int     `json:"team_money"`
    RoundType           string  `json:"round_type"` // "full_buy", "force", "eco", "semi_eco"
    
    // Decision
    BoughtWeapon        string  `json:"bought_weapon"`
    BoughtUtility       []string `json:"bought_utility"`
    TotalSpent          int     `json:"total_spent"`
    
    // Quality Scoring
    AlignedWithTeam     bool    `json:"aligned_with_team"`
    OptimalPurchase     bool    `json:"optimal_purchase"`
    QualityScore        float64 `json:"quality_score"` // 0-100
}
```

**Why It Matters**: Economy is half the game. Bad buys lose rounds.

---

### 🧠 TIER 3 - AI Contextual Awareness (Requires Map Geometry & Models)

#### 11. **Win Probability Added (WPA)**
```go
type WPAEvent struct {
    Tick                int     `json:"tick"`
    PlayerSteamID       uint64  `json:"player_steam_id"`
    Event               string  `json:"event"` // "kill", "death", "plant"
    
    // Probability Delta
    WinProbBefore       float64 `json:"win_prob_before"`
    WinProbAfter        float64 `json:"win_prob_after"`
    WPA                 float64 `json:"wpa"` // Delta
    
    // Context
    AliveCT             int     `json:"alive_ct"`
    AliveT              int     `json:"alive_t"`
    BombPlanted         bool    `json:"bomb_planted"`
}
```

**Why It Matters**: Measures true impact. A kill in a 5v1 is worth less than a kill in a 2v2.

#### 12. **Geometry-Based Exposure (Raycasting)**
```go
type ExposureAnalysis struct {
    Tick                int     `json:"tick"`
    PlayerSteamID       uint64  `json:"player_steam_id"`
    Position            Vector3 `json:"position"`
    
    // Geometry Metrics
    VisibleAngles       int     `json:"visible_angles"` // From common spots
    IsOffAngle          bool    `json:"is_off_angle"`
    CoverQuality        float64 `json:"cover_quality"` // 0-100
    CrosshairPlacement  float64 `json:"crosshair_placement_dist"` // Dist to corner
}
```

**Why It Matters**: Distinguishes between bad positioning and bad luck.

#### 13. **Anti-Strat & Game Sense**
```go
type GameSenseMetric struct {
    PlayerSteamID       uint64  `json:"player_steam_id"`
    MetricType          string  `json:"metric_type"` // "anti_eco", "predictability"
    
    // Analysis
    Score               float64 `json:"score"`
    Context             string  `json:"context"` // "Played long range vs glocks"
    MistakeCount        int     `json:"mistake_count"`
}
```

**Why It Matters**: Detects high-level decision making errors.

---

## 📋 IMPLEMENTATION PRIORITY ROADMAP

### **Phase 1: Quick Wins** (1-2 days)
1. ✅ Weapon ammo tracking (already in library)
2. ✅ Spotted/Spotting events
3. ✅ Zone detection (bomb zone, buy zone, airborne)
4. ✅ Defuse kit tracking

### **Phase 2: Trade & Utility** (3-4 days)
5. 🔄 Trade efficiency calculation
6. 🔄 Utility effectiveness metrics
7. 🔄 Flash-to-kill conversion
8. 🔄 Molotov area coverage

### **Phase 3: Positioning & Tactics** (5-7 days)
9. 🔄 Positioning risk scoring
10. 🔄 Rotation timing analysis
11. 🔄 Crossfire detection
12. 🔄 First contact stats

### **Phase 4: Advanced Profiles** (7-10 days)
13. 🔄 Engagement distance optimization
14. 🔄 Grenade lineup consistency
15. 🔄 Multi-kill sequence analysis
16. 🔄 Economy decision quality

### **Phase 5: AI Context & Geometry** (Future / Research)
17. 🧪 Win Probability Added (WPA) Model
18. 🧪 Map Geometry / Raycasting Integration
19. 🧪 Anti-Strat / Game Sense Logic

---

## 🎯 IMMEDIATE ACTION ITEMS

### DO NOW (This Sprint):
1. **Add Weapon State Tracking** - Low-hanging fruit from library
2. **Implement Spotted/Spotting Events** - Pure library call
3. **Add Zone Detection** - Bomb zone, buy zone checks
4. **Trade Kill Detection** - Custom logic but high value

### DO NEXT (Next Sprint):
5. **Utility Effectiveness** - Requires correlation analysis
6. **Positioning Risk** - Needs map awareness logic
7. **Flash Effectiveness** - Post-flash action tracking

### BACKLOG (Future):
8. Advanced profile metrics
9. Grenade lineup analysis
10. Multi-kill difficulty scoring

---

## 💡 KEY INSIGHTS

### What Makes StratAI Different:
- **Not just stats, but context**: Every metric tied to tactical situation
- **AI-first design**: Metrics chosen for ML training value
- **Coaching focus**: Data that translates to actionable advice

### Technical Debt to Avoid:
- **Over-extraction**: Don't track everything "just in case"
- **Storage bloat**: Timeline system already pushes limits
- **Computation cost**: Real-time analysis vs post-processing

### Data Quality Priorities:
1. **Accuracy**: Wrong data worse than no data
2. **Consistency**: Same metric calculated same way always
3. **Context**: Never store metric without situation context
4. **Actionability**: Every metric must answer "what can I improve?"

---

## 🚀 CONCLUSION

**Current State**: Solid foundation with excellent core tracking  
**Untapped Potential**: ~10 high-value library features unused  
**Custom Metrics**: ~15 advanced AI-relevant metrics to build  
**AI Frontier**: 3 critical context-aware systems (WPA, Geometry, Game Sense) to truly differentiate the product.

**Recommendation**: Focus Phase 1 & 2 (6-7 days work) before tackling custom metrics. This gives us 80% of the value with 20% of the effort.

**Next Steps**:
1. Review this doc with team
2. Prioritize Phase 1 features
3. Create implementation tickets
4. Update data models

---

**Document Version**: 1.1 (Updated with AI Context)  
**Last Updated**: December 5, 2025  
**Next Review**: After Phase 1 completion
