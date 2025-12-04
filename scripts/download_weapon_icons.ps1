# Script para descargar iconos de armas de CS:GO desde GitHub

$baseURL = "https://raw.githubusercontent.com/ByMykel/counter-strike-image-tracker/main/static/panorama/images/econ/weapons/base_weapons"
$outputDir = "..\frontend\public\images\weapons"

# Lista de armas a descargar
$weapons = @(
    "weapon_ak47",
    "weapon_m4a1",
    "weapon_m4a1_silencer",
    "weapon_awp",
    "weapon_deagle",
    "weapon_glock",
    "weapon_usp_silencer",
    "weapon_hkp2000",
    "weapon_p250",
    "weapon_fiveseven",
    "weapon_tec9",
    "weapon_cz75a",
    "weapon_elite",
    "weapon_revolver",
    "weapon_mp9",
    "weapon_mp7",
    "weapon_mp5sd",
    "weapon_ump45",
    "weapon_p90",
    "weapon_bizon",
    "weapon_mac10",
    "weapon_famas",
    "weapon_galilar",
    "weapon_aug",
    "weapon_sg556",
    "weapon_ssg08",
    "weapon_scar20",
    "weapon_g3sg1",
    "weapon_nova",
    "weapon_xm1014",
    "weapon_mag7",
    "weapon_sawedoff",
    "weapon_m249",
    "weapon_negev",
    "weapon_knife",
    "weapon_c4",
    "weapon_hegrenade",
    "weapon_flashbang",
    "weapon_smokegrenade",
    "weapon_incgrenade",
    "weapon_molotov",
    "weapon_decoy",
    "weapon_taser"
)

Write-Host "Descargando iconos de armas..." -ForegroundColor Cyan
$count = 0
$total = $weapons.Count

foreach ($weapon in $weapons) {
    $count++
    $url = "$baseURL/${weapon}_png.png"
    $outputFile = Join-Path $outputDir "${weapon}.png"
    
    try {
        Write-Host "[$count/$total] Descargando $weapon..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $url -OutFile $outputFile -ErrorAction Stop
        Write-Host "  OK $weapon descargado" -ForegroundColor Green
    }
    catch {
        Write-Host "  ERROR descargando $weapon : $_" -ForegroundColor Red
    }
}

Write-Host "`nDescarga completada! Iconos guardados en: $outputDir" -ForegroundColor Green
Write-Host "Total descargado: $count de $total armas" -ForegroundColor Cyan
