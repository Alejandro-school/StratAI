package demo

import (
	"github.com/golang/geo/r3"
)

// Zone representa una zona del mapa con su nombre y área
type Zone struct {
	Name   string
	MinX   float64
	MaxX   float64
	MinY   float64
	MaxY   float64
	MinZ   float64 // Opcional: altura mínima
	MaxZ   float64 // Opcional: altura máxima
	IsSafe bool    // Si es zona segura (spawn, etc.)
}

// MapZones contiene todas las zonas definidas por mapa
var MapZones = map[string][]Zone{
	"de_dust2": {
		// T Side
		{Name: "T Spawn", MinX: -1664, MaxX: -896, MinY: 2368, MaxY: 3136, IsSafe: true},
		{Name: "T Mid to B", MinX: -1024, MaxX: -256, MinY: 1408, MaxY: 2368},
		{Name: "Tunnels", MinX: 576, MaxX: 1344, MinY: 2048, MaxY: 2816},
		{Name: "Lower Tunnels", MinX: 832, MaxX: 1600, MinY: 1280, MaxY: 2048},

		// CT Side
		{Name: "CT Spawn", MinX: -448, MaxX: 512, MinY: -1024, MaxY: -128, IsSafe: true},
		{Name: "CT Mid", MinX: -1024, MaxX: 256, MinY: 512, MaxY: 1408},

		// Mid
		{Name: "Mid Doors", MinX: -1536, MaxX: -768, MinY: 512, MaxY: 1280},
		{Name: "Top Mid", MinX: -1536, MaxX: -256, MinY: -256, MaxY: 512},
		{Name: "Catwalk", MinX: -256, MaxX: 512, MinY: 640, MaxY: 1536},
		{Name: "Xbox", MinX: -768, MaxX: 0, MinY: 640, MaxY: 1152},

		// A Site
		{Name: "Long A", MinX: 512, MaxX: 1792, MinY: -384, MaxY: 640},
		{Name: "A Ramp", MinX: 768, MaxX: 1536, MinY: 384, MaxY: 1152},
		{Name: "A Site", MinX: 1024, MaxX: 2048, MinY: -640, MaxY: 384},
		{Name: "A Platform", MinX: 1536, MaxX: 2176, MinY: -896, MaxY: -256},

		// B Site
		{Name: "B Site", MinX: 1280, MaxX: 2304, MinY: 1280, MaxY: 2048},
		{Name: "B Doors", MinX: 1024, MaxX: 1600, MinY: 896, MaxY: 1280},
		{Name: "B Window", MinX: 832, MaxX: 1280, MinY: 512, MaxY: 1024},
	},

	"de_mirage": {
		// T Side
		{Name: "T Spawn", MinX: -1792, MaxX: -1088, MinY: -896, MaxY: -128, IsSafe: true},
		{Name: "T Ramp", MinX: -1344, MaxX: -512, MinY: -896, MaxY: -384},
		{Name: "T Stairs", MinX: -896, MaxX: -256, MinY: -512, MaxY: 128},

		// CT Side
		{Name: "CT Spawn", MinX: 128, MaxX: 1024, MinY: -1536, MaxY: -640, IsSafe: true},

		// Mid
		{Name: "Mid Top", MinX: -1024, MaxX: -256, MinY: -1152, MaxY: -384},
		{Name: "Connector", MinX: -512, MaxX: 256, MinY: -1408, MaxY: -896},
		{Name: "Window", MinX: -896, MaxX: -256, MinY: -384, MaxY: 128},
		{Name: "Underpass", MinX: -512, MaxX: 256, MinY: -640, MaxY: 0},

		// A Site
		{Name: "A Ramp", MinX: -1280, MaxX: -640, MinY: 128, MaxY: 896},
		{Name: "A Site", MinX: -768, MaxX: 256, MinY: 384, MaxY: 1152},
		{Name: "Stairs", MinX: -384, MaxX: 256, MinY: 896, MaxY: 1536},
		{Name: "Jungle", MinX: -1280, MaxX: -640, MinY: 768, MaxY: 1536},
		{Name: "Palace", MinX: -1792, MaxX: -1024, MinY: 512, MaxY: 1280},

		// B Site
		{Name: "B Apartments", MinX: 384, MaxX: 1280, MinY: 512, MaxY: 1536},
		{Name: "B Site", MinX: 384, MaxX: 1280, MinY: -256, MaxY: 512},
		{Name: "Market", MinX: 640, MaxX: 1408, MinY: -896, MaxY: -128},
		{Name: "Kitchen", MinX: 896, MaxX: 1664, MinY: 256, MaxY: 896},
	},

	"de_inferno": {
		// T Side
		{Name: "T Spawn", MinX: 1280, MaxX: 2304, MinY: 1536, MaxY: 2560, IsSafe: true},
		{Name: "T Apartments", MinX: 1024, MaxX: 1920, MinY: 896, MaxY: 1536},
		{Name: "Banana", MinX: 256, MaxX: 1024, MinY: 512, MaxY: 1280},

		// CT Side
		{Name: "CT Spawn", MinX: -256, MaxX: 896, MinY: -512, MaxY: 512, IsSafe: true},

		// Mid
		{Name: "Mid", MinX: 896, MaxX: 1664, MinY: 256, MaxY: 896},
		{Name: "Top Mid", MinX: 1152, MaxX: 1920, MinY: -128, MaxY: 512},

		// A Site
		{Name: "A Site", MinX: 1664, MaxX: 2560, MinY: -256, MaxY: 896},
		{Name: "Balcony", MinX: 1920, MaxX: 2560, MinY: 384, MaxY: 1024},
		{Name: "Apartments", MinX: 2048, MaxX: 2816, MinY: 768, MaxY: 1536},
		{Name: "Arch", MinX: 1536, MaxX: 2048, MinY: -640, MaxY: 128},

		// B Site
		{Name: "B Site", MinX: -384, MaxX: 640, MinY: 896, MaxY: 1792},
		{Name: "Coffins", MinX: 0, MaxX: 512, MinY: 1536, MaxY: 2048},
		{Name: "Sandbags", MinX: -128, MaxX: 512, MinY: 1024, MaxY: 1536},
	},

	"de_ancient": {
		// T Side
		{Name: "T Spawn", MinX: 512, MaxX: 1536, MinY: 1920, MaxY: 2944, IsSafe: true},

		// CT Side
		{Name: "CT Spawn", MinX: -1024, MaxX: 0, MinY: -1536, MaxY: -512, IsSafe: true},

		// Mid
		{Name: "Mid", MinX: -512, MaxX: 768, MinY: 256, MaxY: 1536},

		// A Site
		{Name: "A Site", MinX: -1280, MaxX: 256, MinY: 768, MaxY: 2048},
		{Name: "A Main", MinX: -768, MaxX: 512, MinY: 1792, MaxY: 2816},

		// B Site
		{Name: "B Site", MinX: 256, MaxX: 1536, MinY: -768, MaxY: 512},
		{Name: "B Main", MinX: 768, MaxX: 1792, MinY: 256, MaxY: 1280},
	},

	"de_nuke": {
		// T Side
		{Name: "T Spawn Outside", MinX: -1536, MaxX: -256, MinY: -1536, MaxY: -256, IsSafe: true},

		// CT Side
		{Name: "CT Spawn", MinX: 512, MaxX: 1792, MinY: -768, MaxY: 512, IsSafe: true},

		// Outside
		{Name: "Outside", MinX: -1280, MaxX: 256, MinY: -2048, MaxY: -512},
		{Name: "Yard", MinX: -896, MaxX: 512, MinY: -256, MaxY: 1024},

		// Upper Sites
		{Name: "A Site Upper", MinX: 256, MaxX: 1280, MinY: 1024, MaxY: 2304, MinZ: 0},
		{Name: "Ramp Upper", MinX: -384, MaxX: 512, MinY: 1280, MaxY: 2304},

		// Lower Sites
		{Name: "B Site Lower", MinX: 256, MaxX: 1536, MinY: 1024, MaxY: 2304, MinZ: -500, MaxZ: -100},
		{Name: "Ramp Lower", MinX: -512, MaxX: 512, MinY: 1536, MaxY: 2560, MinZ: -500},
	},

	"de_vertigo": {
		// T Side
		{Name: "T Spawn", MinX: -1536, MaxX: -256, MinY: 384, MaxY: 1408, IsSafe: true},

		// CT Side
		{Name: "CT Spawn", MinX: -1792, MaxX: -768, MinY: -1536, MaxY: -384, IsSafe: true},

		// Mid
		{Name: "Mid", MinX: -1280, MaxX: -256, MinY: -640, MaxY: 384},

		// A Site
		{Name: "A Site", MinX: -256, MaxX: 1024, MinY: -256, MaxY: 1024},
		{Name: "A Ramp", MinX: 256, MaxX: 1280, MinY: 384, MaxY: 1536},

		// B Site
		{Name: "B Site", MinX: -1536, MaxX: -256, MinY: -2048, MaxY: -768},
	},

	"de_anubis": {
		// T Side
		{Name: "T Spawn", MinX: -1536, MaxX: -256, MinY: 1792, MaxY: 3072, IsSafe: true},

		// CT Side
		{Name: "CT Spawn", MinX: 1280, MaxX: 2560, MinY: -768, MaxY: 512, IsSafe: true},

		// Mid
		{Name: "Mid", MinX: -256, MaxX: 1280, MinY: 512, MaxY: 1792},

		// A Site
		{Name: "A Site", MinX: 768, MaxX: 2048, MinY: 1280, MaxY: 2560},
		{Name: "A Main", MinX: -512, MaxX: 768, MinY: 1536, MaxY: 2816},

		// B Site
		{Name: "B Site", MinX: 256, MaxX: 1536, MinY: -1024, MaxY: 256},
		{Name: "B Main", MinX: -768, MaxX: 512, MinY: 256, MaxY: 1280},
	},
}

// GetZoneForPosition retorna el nombre de la zona donde está el jugador
// Si no encuentra zona específica, retorna "Unknown"
func GetZoneForPosition(mapName string, pos r3.Vector) string {
	zones, exists := MapZones[mapName]
	if !exists {
		return "Unknown"
	}

	// Buscamos la zona más específica (la más pequeña que contenga el punto)
	var matchedZone *Zone
	smallestArea := float64(-1)

	for i := range zones {
		zone := &zones[i]

		// Check si el punto está dentro de esta zona
		if pos.X >= zone.MinX && pos.X <= zone.MaxX &&
			pos.Y >= zone.MinY && pos.Y <= zone.MaxY {

			// Si tiene restricción de altura, verificarla
			if zone.MinZ != 0 || zone.MaxZ != 0 {
				if pos.Z < zone.MinZ || pos.Z > zone.MaxZ {
					continue // No está en el rango de altura
				}
			}

			// Calcular área de esta zona
			area := (zone.MaxX - zone.MinX) * (zone.MaxY - zone.MinY)

			// Seleccionar la zona más pequeña (más específica)
			if smallestArea < 0 || area < smallestArea {
				matchedZone = zone
				smallestArea = area
			}
		}
	}

	if matchedZone != nil {
		return matchedZone.Name
	}

	return "Unknown"
}

// IsInSafeZone verifica si el jugador está en zona segura (spawn)
func IsInSafeZone(mapName string, pos r3.Vector) bool {
	zones, exists := MapZones[mapName]
	if !exists {
		return false
	}

	for _, zone := range zones {
		if zone.IsSafe &&
			pos.X >= zone.MinX && pos.X <= zone.MaxX &&
			pos.Y >= zone.MinY && pos.Y <= zone.MaxY {
			return true
		}
	}

	return false
}
