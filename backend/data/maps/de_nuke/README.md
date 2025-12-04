# Map Files

To enable high-precision visibility checks (Raycasting) for CS:GO demos, place the `.bsp` map files in this directory.

## Supported Maps
- de_mirage.bsp
- de_inferno.bsp
- de_dust2.bsp
- etc.

## CS2 Support
Currently, CS2 maps (`.vpk`, `.vmap`) are not supported for geometry analysis due to the lack of a public Source 2 map parser for Go.
The system will automatically fallback to a Heuristic method (FOV + Flashbang + Smoke proximity) for CS2 demos or if the map file is missing.

## Where to find maps?
You can extract them from your CS:GO installation (`csgo/maps`) or download them.
