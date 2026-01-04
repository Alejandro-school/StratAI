
# backend/app/utils/maps.py
# -------------------------
# Utilidades y configuración para mapas de CS2 (radares, callouts, etc.)

# ============================================================================
# MAP RADAR CONFIG
# ============================================================================
# Map radar configuration (origin and scale from CS2 game files)
# These values convert game coordinates to radar % positions
# Source: https://github.com/SteamDatabase/GameTracking-CS2/tree/master/game/csgo/pak01_dir/resource/overviews
MAP_RADAR_CONFIG = {
    "de_dust2": {"origin_x": -2476, "origin_y": 3239, "scale": 4.4},
    "de_mirage": {"origin_x": -3230, "origin_y": 1713, "scale": 5.0},
    "de_inferno": {"origin_x": -2087, "origin_y": 3870, "scale": 4.9},
    "de_nuke": {"origin_x": -3453, "origin_y": 2887, "scale": 7.0},
    "de_overpass": {"origin_x": -4831, "origin_y": 1781, "scale": 5.2},
    "de_ancient": {"origin_x": -2953, "origin_y": 2164, "scale": 5.0},
    "de_anubis": {"origin_x": -2796, "origin_y": 3328, "scale": 5.22},
    "de_vertigo": {"origin_x": -3168, "origin_y": 1762, "scale": 4.0},
    "de_train": {"origin_x": -2308, "origin_y": 2078, "scale": 4.082077},
}

# ============================================================================
# CALLOUT FIXED POSITIONS - Predefined positions for each callout on each map
# ============================================================================
# These are fixed x,y percentages (0-100) for placing callout markers on the radar.
# This prevents overlapping caused by centroid calculations from tracking data.
# If a callout is not in this list, the system falls back to the calculated centroid.

CALLOUT_FIXED_POSITIONS = {
    # ==================== DE_NUKE (Upper Level) ====================
    "de_nuke": {
        # T Side
        "T Spawn": {"x": 8, "y": 45},
        "Lobby": {"x": 22, "y": 37},
        "Roof": {"x": 30, "y": 37},
        "Squeaky": {"x": 33, "y": 31},
        "Vending": {"x": 28, "y": 28},
        "Trophy": {"x": 32, "y": 22},
        "Control": {"x": 37, "y": 13},
        "Ramp": {"x": 45, "y": 10},
        "Admin": {"x": 54, "y": 14},
        
        # A Site Area
        "Hut": {"x": 41, "y": 32},
        "Hut Roof": {"x": 43, "y": 35},
        "A Site": {"x": 47, "y": 32},
        "Rafters": {"x": 46, "y": 26},
        "Heaven": {"x": 53, "y": 23},
        "Hell": {"x": 58, "y": 23},
        "Catwalk": {"x": 56, "y": 19},
        "Gateway": {"x": 62, "y": 23},
        "Lockers": {"x": 58, "y": 30},
        "Crane": {"x": 52, "y": 32},
        
        # Outside
        "Outside": {"x": 36, "y": 48},
        "Silo": {"x": 32, "y": 50},
        "Mini": {"x": 46, "y": 52},
        
        # CT Side
        "CT Spawn": {"x": 73, "y": 26},
        "Garage": {"x": 68, "y": 55},
        "Secret": {"x": 60, "y": 68},
        
        # Lower level (shown on upper radar too for some callouts)
        "Ramp Room": {"x": 41, "y": 15},
        "Vents": {"x": 55, "y": 40},
        "Toxic": {"x": 48, "y": 58},
        "Decon": {"x": 50, "y": 65},
        "B Site": {"x": 52, "y": 72},
        "Radio": {"x": 64, "y": 52},
        "Main": {"x": 32, "y": 60},
    },
    
    # ==================== DE_DUST2 ====================
    "de_dust2": {
        # T Spawn Area
        "T Spawn": {"x": 14, "y": 60},
        "T Ramp": {"x": 18, "y": 65},
        "Outside Long": {"x": 18, "y": 40},
        "Outside Tunnels": {"x": 18, "y": 72},
        
        # Long A
        "Long Doors": {"x": 26, "y": 38},
        "Blue Door": {"x": 30, "y": 35},
        "Long A": {"x": 38, "y": 30},
        "Long Corner": {"x": 45, "y": 27},
        "Pit": {"x": 55, "y": 22},
        
        # A Site
        "A Site": {"x": 68, "y": 27},
        "Goose": {"x": 72, "y": 22},
        "Car": {"x": 62, "y": 32},
        "A Cross": {"x": 58, "y": 36},
        "Barrels": {"x": 75, "y": 30},
        
        # Short/Catwalk
        "Short A": {"x": 58, "y": 42},
        "Short Stairs": {"x": 52, "y": 47},
        "Catwalk": {"x": 48, "y": 52},
        
        # Mid
        "Xbox": {"x": 42, "y": 55},
        "Top Mid": {"x": 38, "y": 50},
        "Mid Doors": {"x": 35, "y": 55},
        "Lower Mid": {"x": 45, "y": 62},
        "CT Mid": {"x": 62, "y": 60},
        
        # B Tunnels
        "B Tunnels": {"x": 28, "y": 78},
        "Upper Tunnels": {"x": 32, "y": 75},
        "Lower Tunnels": {"x": 28, "y": 82},
        "Tunnel Stairs": {"x": 35, "y": 80},
        
        # B Site
        "B Site": {"x": 48, "y": 82},
        "B Closet": {"x": 42, "y": 80},
        "B Doors": {"x": 55, "y": 78},
        "Big Box": {"x": 52, "y": 85},
        "B Window": {"x": 60, "y": 82},
        "B Back": {"x": 58, "y": 88},
        "B Platform": {"x": 45, "y": 88},
        
        # CT Spawn
        "CT Spawn": {"x": 75, "y": 52},
        "CT Mid": {"x": 68, "y": 58},
        "Back Alley": {"x": 78, "y": 42},
        
        # Other
        "Suicide": {"x": 32, "y": 60},
    },
    
    # ==================== DE_MIRAGE ====================
    "de_mirage": {
        # T Spawn
        "T Spawn": {"x": 15, "y": 55},
        "T Ramp": {"x": 20, "y": 50},
        "T Apps": {"x": 15, "y": 68},
        
        # A Site
        "A Site": {"x": 72, "y": 25},
        "A Ramp": {"x": 65, "y": 32},
        "Tetris": {"x": 68, "y": 28},
        "Firebox": {"x": 75, "y": 22},
        "Ninja": {"x": 78, "y": 18},
        "Ticket": {"x": 62, "y": 25},
        "Stairs": {"x": 60, "y": 22},
        "Jungle": {"x": 55, "y": 28},
        "Connector": {"x": 50, "y": 35},
        "Sandwich": {"x": 70, "y": 32},
        "Default": {"x": 74, "y": 26},
        "Triple": {"x": 78, "y": 28},
        "Bench": {"x": 68, "y": 18},
        
        # Palace
        "Palace": {"x": 48, "y": 18},
        "Palace Alley": {"x": 42, "y": 22},
        
        # Mid
        "Top Mid": {"x": 35, "y": 42},
        "Mid Boxes": {"x": 40, "y": 48},
        "Chair": {"x": 45, "y": 52},
        "Window": {"x": 55, "y": 50},
        "Short Stairs": {"x": 52, "y": 45},
        "Underpass": {"x": 42, "y": 58},
        "Ladder Room": {"x": 48, "y": 62},
        "Under A": {"x": 58, "y": 58},
        
        # B Site
        "B Site": {"x": 28, "y": 78},
        "Van": {"x": 32, "y": 82},
        "Arches": {"x": 38, "y": 75},
        "Kitchen": {"x": 42, "y": 80},
        "Kitchen Door": {"x": 45, "y": 78},
        "B Apartments": {"x": 20, "y": 72},
        "B Bench": {"x": 25, "y": 80},
        "Catwalk": {"x": 35, "y": 70},
        "Market": {"x": 50, "y": 70},
        "Market Door": {"x": 48, "y": 68},
        
        # CT
        "CT Spawn": {"x": 72, "y": 55},
    },
    
    # ==================== DE_INFERNO ====================
    "de_inferno": {
        # T Spawn
        "T Spawn": {"x": 8, "y": 28},
        "T Ramp": {"x": 15, "y": 32},
        
        # Mid/Second
        "First Mid": {"x": 22, "y": 38},
        "Second Mid": {"x": 28, "y": 45},
        "Alt Mid": {"x": 32, "y": 40},
        
        # Apps/Apartments
        "Apartments": {"x": 45, "y": 30},
        "Balcony": {"x": 52, "y": 28},
        "Boiler": {"x": 48, "y": 35},
        "Hay": {"x": 42, "y": 38},
        "Moto": {"x": 38, "y": 35},
        
        # A Site
        "A Site": {"x": 60, "y": 25},
        "Pit": {"x": 65, "y": 18},
        "Graveyard": {"x": 70, "y": 22},
        "Church": {"x": 72, "y": 30},
        "Library": {"x": 68, "y": 35},
        "Arch": {"x": 55, "y": 42},
        "Truck": {"x": 62, "y": 32},
        
        # Banana/B
        "Banana": {"x": 28, "y": 58},
        "Bottom Banana": {"x": 22, "y": 52},
        "Top Banana": {"x": 35, "y": 65},
        "Banana Car": {"x": 30, "y": 62},
        "Logs": {"x": 32, "y": 55},
        "Sandbags": {"x": 38, "y": 70},
        
        # B Site
        "B Site": {"x": 45, "y": 78},
        "New Box": {"x": 48, "y": 82},
        "Coffins": {"x": 42, "y": 85},
        "Oranges": {"x": 38, "y": 80},
        "Spools": {"x": 50, "y": 75},
        "Dark": {"x": 55, "y": 82},
        "Fountain": {"x": 52, "y": 88},
        
        # CT
        "CT Spawn": {"x": 75, "y": 58},
    },
    
    # ==================== DE_OVERPASS ====================
    "de_overpass": {
        # T Spawn
        "T Spawn": {"x": 12, "y": 45},
        
        # A Site
        "A Site": {"x": 55, "y": 22},
        "Bank": {"x": 48, "y": 25},
        "Truck": {"x": 60, "y": 18},
        "Long A": {"x": 42, "y": 18},
        "Short A": {"x": 50, "y": 30},
        "Divider": {"x": 58, "y": 28},
        "Toilets": {"x": 62, "y": 32},
        "Bathrooms": {"x": 68, "y": 28},
        "Graffiti": {"x": 72, "y": 22},
        "Pillar": {"x": 45, "y": 20},
        
        # Mid/Connector
        "Monster": {"x": 35, "y": 40},
        "Connector": {"x": 45, "y": 45},
        "Party": {"x": 52, "y": 48},
        "Playground": {"x": 38, "y": 52},
        "Fountain": {"x": 58, "y": 55},
        "Sandbags": {"x": 42, "y": 60},
        
        # B Site
        "B Site": {"x": 28, "y": 72},
        "Water": {"x": 22, "y": 68},
        "Construction": {"x": 35, "y": 78},
        "Heaven": {"x": 40, "y": 75},
        
        # CT
        "CT Spawn": {"x": 72, "y": 55},
    },
    
    # ==================== DE_ANCIENT ====================
    "de_ancient": {
        # T Spawn
        "T Spawn": {"x": 8, "y": 52},
        
        # Mid
        "Cave": {"x": 28, "y": 45},
        "Donut": {"x": 42, "y": 42},
        "Main Hall": {"x": 45, "y": 50},
        "Side Hall": {"x": 48, "y": 58},
        "Elbow": {"x": 55, "y": 55},
        
        # A Site
        "A Site": {"x": 68, "y": 28},
        "Temple": {"x": 62, "y": 22},
        "Red Room": {"x": 72, "y": 32},
        "Ramp": {"x": 58, "y": 35},
        
        # B Site
        "B Site": {"x": 32, "y": 75},
        "Water": {"x": 28, "y": 70},
        
        # CT
        "CT Spawn": {"x": 75, "y": 55},
    },
    
    # ==================== DE_ANUBIS ====================
    "de_anubis": {
        # T Spawn
        "T Spawn": {"x": 12, "y": 50},
        
        # Mid
        "Main": {"x": 32, "y": 45},
        "Canal": {"x": 45, "y": 42},
        "Bridge": {"x": 50, "y": 38},
        "Connector": {"x": 48, "y": 52},
        "Alley": {"x": 38, "y": 55},
        
        # A Site
        "A Site": {"x": 72, "y": 28},
        "Palace": {"x": 65, "y": 22},
        "Heaven": {"x": 78, "y": 32},
        "Street": {"x": 58, "y": 35},
        
        # B Site
        "B Site": {"x": 28, "y": 75},
        "Ruins": {"x": 22, "y": 70},
        "Tomb": {"x": 35, "y": 78},
        
        # CT
        "CT Spawn": {"x": 70, "y": 58},
    },
    
    # ==================== DE_VERTIGO ====================
    "de_vertigo": {
        # T Spawn (lower)
        "T Spawn": {"x": 15, "y": 65},
        
        # A Site
        "A Site": {"x": 72, "y": 25},
        "Ramp": {"x": 68, "y": 35},
        "Elevator": {"x": 58, "y": 30},
        "Generator": {"x": 75, "y": 20},
        
        # Mid
        "Mid": {"x": 45, "y": 45},
        "Stairs": {"x": 52, "y": 50},
        
        # B Site
        "B Site": {"x": 28, "y": 78},
        "Scaffolding": {"x": 22, "y": 72},
        "Heaven": {"x": 35, "y": 82},
        
        # CT
        "CT Spawn": {"x": 68, "y": 55},
    },
    
    # ==================== DE_TRAIN ====================
    "de_train": {
        # T Spawn
        "T Spawn": {"x": 12, "y": 50},
        
        # A Site
        "A Site": {"x": 72, "y": 25},
        "Pop Dog": {"x": 65, "y": 32},
        "Ivy": {"x": 55, "y": 22},
        "Ladder": {"x": 78, "y": 28},
        "Hell": {"x": 70, "y": 18},
        "Heaven": {"x": 75, "y": 22},
        
        # Mid/Connector  
        "Connector": {"x": 48, "y": 45},
        "T Main": {"x": 28, "y": 42},
        "Lower": {"x": 35, "y": 55},
        
        # B Site
        "B Site": {"x": 28, "y": 78},
        "Upper B": {"x": 32, "y": 72},
        "Spools": {"x": 22, "y": 82},
        
        # CT
        "CT Spawn": {"x": 68, "y": 60},
    },
}

# ============================================================================
# CALLOUT NAME NORMALIZATION - NavMesh names → Readable callout names
# ============================================================================
# The Go parser extracts raw NavMesh area names which are often cryptic.
# This mapping translates them to standard CS2 callout names.

CALLOUT_NORMALIZATION = {
    # ==================== DE_MIRAGE ====================
    # B Site area
    "position.alleyac": "B Apartments",
    "position.alley": "B Apartments", 
    "BAlley": "B Apartments",
    "Apartments": "B Apartments",
    "BApartments": "B Apartments",
    
    # Window/Market area (mid)
    "position.shop0": "Window",
    "position.shop1": "Market",
    "position.sniper0": "Window",
    "position.sniper1": "Window",
    "position.sniper": "Window",
    "SnipersNest": "Window",
    "Snipers": "Window",
    "Shop": "Market",
    "MarketDoor": "Market Door",
    
    # T Spawn / Apps
    "position.actspawn": "T Apps",
    "position.tspawn": "T Spawn",
    "TRamp": "T Ramp",
    
    # CT Spawn
    "position.ctspawn": "CT Spawn",
    "position.ctspawn0": "CT Spawn",
    
    # Ladder / Underpass
    "position.ladder0": "Ladder Room",
    "position.ladder": "Ladder Room",
    
    # Tunnels (could be B apps)
    "position.tunnel0": "Underpass",
    "position.tunnel": "Underpass",
    
    # A Site
    "position.asite": "A Site",
    "ASite": "A Site",
    "ARamp": "A Ramp",
    "Tetris": "Tetris",
    "Firebox": "Firebox",
    "Ninja": "Ninja",
    "Ticket": "Ticket",
    "TicketBooth": "Ticket",
    "Stairs": "Stairs",
    "Jungle": "Jungle",
    "JungleBox": "Jungle",
    "Connector": "Connector",
    "CTSpawn": "CT Spawn",
    "Bench": "Bench",
    "Default": "Default",
    "Triple": "Triple",
    "Sandwich": "Sandwich",
    
    # Mid
    "TopMid": "Top Mid",
    "MidBox": "Mid Boxes",
    "MidBoxes": "Mid Boxes",
    "Ladder": "Ladder Room",
    "LadderRoom": "Ladder Room",
    "ShortStairs": "Short Stairs",
    "Chair": "Chair",
    "Underpass": "Underpass",
    "UnderA": "Under A",
    
    # Palace
    "Palace": "Palace",
    "PalaceAlley": "Palace",
    "PalaceInterior": "Palace",
    "position.palace0": "Palace",
    "position.palacealley0": "Palace Alley",
    "position.palace": "Palace",
    
    # B Site
    "BSite": "B Site",
    "BenchB": "B Bench",
    "Van": "Van",
    "Arches": "Arches",
    "Kitchen": "Kitchen",
    "KitchenDoor": "Kitchen Door",
    "Cat": "Catwalk",
    "Catwalk": "Catwalk",
    
    # ==================== DE_DUST2 ====================
    "LongA": "Long A",
    "LongDoors": "Long Doors",
    "LongCorner": "Long Corner",
    "BlueDoor": "Blue Door",
    "Pit": "Pit",
    "PitPlat": "Pit",
    "Goose": "Goose",
    "Car": "Car",
    "Barrels": "Barrels",
    "CrossA": "A Cross",
    "ShortA": "Short A",
    "Xbox": "Xbox",
    "MidDoors": "Mid Doors",
    "TopMid": "Top Mid",
    "LowerMid": "Lower Mid",
    "CTMid": "CT Mid",
    "BTunnel": "B Tunnels",
    "UpperTunnel": "Upper Tunnels",
    "LowerTunnel": "Lower Tunnels",
    "TunnelStairs": "Tunnel Stairs",
    "BCloset": "B Closet",
    "BDoors": "B Doors",
    "BigBox": "Big Box",
    "BWindow": "B Window",
    "BBacksite": "B Back",
    "BPlat": "B Platform",
    "BombsiteA": "A Site",
    "BombsiteB": "B Site",
    "TSpawn": "T Spawn",
    "TRamp": "T Ramp",
    "OutsideLong": "Outside Long",
    "OutsideTunnel": "Outside Tunnels",
    "Suicide": "Suicide",
    "BackAlley": "Back Alley",
    
    # ==================== DE_INFERNO ====================
    "BananaCar": "Banana Car",
    "Banana": "Banana",
    "BananaBottom": "Bottom Banana",
    "BananaTop": "Top Banana",
    "Logs": "Logs",
    "Sandbags": "Sandbags",
    "NewBox": "New Box",
    "Coffins": "Coffins",
    "Oranges": "Oranges",
    "Spools": "Spools",
    "Dark": "Dark",
    "Fountain": "Fountain",
    "Graveyard": "Graveyard",
    "Church": "Church",
    "Pit": "Pit",
    "Library": "Library",
    "Apts": "Apartments",
    "Boiler": "Boiler",
    "Balcony": "Balcony",
    "Hay": "Hay",
    "Moto": "Moto",
    "Second": "Second Mid",
    "FirstMid": "First Mid",
    "SecondMid": "Second Mid",
    "Arch": "Arch",
    "Truck": "Truck",
    "Alt": "Alt Mid",
    "AltMid": "Alt Mid",
    "TopBanana": "Top Banana",
    
    # ==================== DE_NUKE ====================
    "Ramp": "Ramp",
    "Lobby": "Lobby",
    "Main": "Main",
    "Squeaky": "Squeaky",
    "Hut": "Hut",
    "Vents": "Vents",
    "Heaven": "Heaven",
    "Hell": "Hell",
    "Control": "Control",
    "Secret": "Secret",
    "Silo": "Silo",
    "OutsideA": "Outside",
    "Garage": "Garage",
    "Radio": "Radio",
    "Trophy": "Trophy",
    "Mini": "Mini",
    "Toxic": "Toxic",
    "Decon": "Decon",
    
    # ==================== DE_OVERPASS ====================
    "BankA": "Bank",
    "Bathrooms": "Bathrooms",
    "Toilets": "Toilets",
    "Water": "Water",
    "Monster": "Monster",
    "Party": "Party",
    "Sandbags": "Sandbags",
    "Playground": "Playground",
    "Fountain": "Fountain",
    "Connector": "Connector",
    "LongA": "Long A",
    "ShortA": "Short A",
    "Truck": "Truck",
    "Divider": "Divider",
    "Graffiti": "Graffiti",
    "Pillar": "Pillar",
    "Construction": "Construction",
    "Heaven": "Heaven",
    
    # ==================== DE_ANCIENT ====================
    "Cave": "Cave",
    "Temple": "Temple",
    "MainHall": "Main Hall",
    "SideHall": "Side Hall",
    "Donut": "Donut",
    "Elbow": "Elbow",
    "RedRoom": "Red Room",
    "Water": "Water",
    "Ramp": "Ramp",
    
    # ==================== DE_ANUBIS ====================
    "Canal": "Canal",
    "Bridge": "Bridge",
    "Alley": "Alley",
    "Main": "Main",
    "Connector": "Connector",
    "Heaven": "Heaven",
    "Street": "Street",
    "Ruins": "Ruins",
    "Tomb": "Tomb",
    "Palace": "Palace",
    
    # ==================== GENERIC / COMMON ====================
    "position.spawn": "Spawn",
    "position.ctspawn": "CT Spawn",
    "Unknown": None,  # Skip unknown areas
    "": None,  # Skip empty
}

def normalize_callout(raw_callout: str, map_name: str = None) -> str | None:
    """
    Normalize a raw NavMesh callout name to a readable name.
    Returns None if the callout should be skipped.
    """
    if not raw_callout:
        return None
    
    # Direct match
    if raw_callout in CALLOUT_NORMALIZATION:
        return CALLOUT_NORMALIZATION[raw_callout]
    
    # Try lowercase match
    lower = raw_callout.lower()
    for key, value in CALLOUT_NORMALIZATION.items():
        if key.lower() == lower:
            return value
    
    # Try stripping "position." prefix and matching
    if raw_callout.startswith("position."):
        stripped = raw_callout[9:]  # Remove "position."
        # Try direct match of stripped version
        for key, value in CALLOUT_NORMALIZATION.items():
            if key.lower() == stripped.lower() or key.lower().endswith(stripped.lower()):
                return value
        # Return cleaned up version (capitalize)
        return stripped.replace("_", " ").title()
    
    # Return original if no mapping found (already clean enough)
    return raw_callout


def game_to_radar_percent(game_x: float, game_y: float, map_name: str) -> dict:
    """Convert game coordinates to radar percentage (0-100)."""
    config = MAP_RADAR_CONFIG.get(map_name, MAP_RADAR_CONFIG["de_dust2"])
    # Radar images are 1024x1024, scale converts units to pixels
    x_percent = ((game_x - config["origin_x"]) / config["scale"]) / 1024 * 100
    y_percent = ((config["origin_y"] - game_y) / config["scale"]) / 1024 * 100  # Y is inverted
    return {"x": round(max(0, min(100, x_percent)), 1), "y": round(max(0, min(100, y_percent)), 1)}
