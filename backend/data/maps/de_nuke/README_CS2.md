# Map Geometry for CS2

To achieve **pixel-perfect visibility checks** (Raycasting) in CS2, we need the map geometry.
Since CS2 uses a closed format (`.vpk`), we cannot read it directly.

## The Solution: Export to OBJ
We use the "Physics Hull" (collision mesh) of the map exported to a standard `.obj` file.

### How to generate these files:
1. Download **Source 2 Viewer (VRF)**: https://github.com/ValveResourceFormat/ValveResourceFormat
2. Open the map file (e.g., `de_mirage.vpk`) from your CS2 installation.
3. Find the main world physics file (usually `world_physics.vmap` or similar inside the VPK).
4. Right-click -> **Export as OBJ**.
5. Rename the file to `de_mirage.obj`.
6. Place it in this folder (`backend/data/maps/`).

### Supported Formats
- **.obj**: For CS2 maps (Physics Mesh).
- **.bsp**: For legacy CS:GO maps.

If no file is found, the system automatically falls back to the **Hybrid Heuristic Mode** (Radar + FOV + Flash), which is ~90% accurate.
