# Script de Verificación Rápida - Landing 3D

Write-Host "🔍 VERIFICANDO CONFIGURACIÓN DE LANDING 3D..." -ForegroundColor Cyan
Write-Host ""

# 1. Verificar archivos GLB
Write-Host "📁 1. Verificando archivos GLB..." -ForegroundColor Yellow
$glbPath = ".\frontend\public\images\Landing"

if (Test-Path $glbPath) {
    $files = Get-ChildItem -Path $glbPath -Filter "*.glb"
    
    if ($files.Count -eq 0) {
        Write-Host "❌ No se encontraron archivos .glb en $glbPath" -ForegroundColor Red
        Write-Host "   Descarga los modelos y colócalos en esa carpeta" -ForegroundColor Red
    } else {
        foreach ($file in $files) {
            $sizeMB = [math]::Round($file.Length / 1MB, 2)
            $icon = if ($sizeMB -gt 20) { "⚠️" } else { "✅" }
            Write-Host "   $icon $($file.Name) - $sizeMB MB" -ForegroundColor Green
        }
    }
} else {
    Write-Host "❌ Directorio no encontrado: $glbPath" -ForegroundColor Red
    Write-Host "   Crea la carpeta: mkdir '$glbPath'" -ForegroundColor Yellow
}

Write-Host ""

# 2. Verificar dependencias
Write-Host "📦 2. Verificando dependencias npm..." -ForegroundColor Yellow
$packagePath = ".\frontend\package.json"

if (Test-Path $packagePath) {
    $package = Get-Content $packagePath -Raw | ConvertFrom-Json
    
    $requiredDeps = @{
        "three" = $package.dependencies.three
        "@react-three/fiber" = $package.dependencies."@react-three/fiber"
        "@react-three/drei" = $package.dependencies."@react-three/drei"
    }
    
    foreach ($dep in $requiredDeps.GetEnumerator()) {
        if ($dep.Value) {
            Write-Host "   ✅ $($dep.Key): $($dep.Value)" -ForegroundColor Green
        } else {
            Write-Host "   ❌ $($dep.Key): NO INSTALADO" -ForegroundColor Red
            Write-Host "      npm install $($dep.Key)" -ForegroundColor Yellow
        }
    }
} else {
    Write-Host "❌ No se encontró package.json en frontend/" -ForegroundColor Red
}

Write-Host ""

# 3. Verificar componentes creados
Write-Host "🎨 3. Verificando componentes..." -ForegroundColor Yellow
$components = @(
    ".\frontend\src\components\Landing3D\Landing3DPage.jsx",
    ".\frontend\src\components\Landing3D\DebugHelper.jsx",
    ".\frontend\src\components\Landing3D\TestR3F.jsx",
    ".\frontend\src\components\Landing3D\Landing3DPage.FALLBACK.jsx"
)

foreach ($comp in $components) {
    if (Test-Path $comp) {
        $lines = (Get-Content $comp).Count
        Write-Host "   ✅ $(Split-Path $comp -Leaf) - $lines líneas" -ForegroundColor Green
    } else {
        Write-Host "   ❌ $(Split-Path $comp -Leaf) - NO ENCONTRADO" -ForegroundColor Red
    }
}

Write-Host ""

# 4. Verificar estructura de archivos críticos
Write-Host "🔧 4. Verificando configuración en Landing3DPage.jsx..." -ForegroundColor Yellow
$mainComponent = ".\frontend\src\components\Landing3D\Landing3DPage.jsx"

if (Test-Path $mainComponent) {
    $content = Get-Content $mainComponent -Raw
    
    # Verificar valores críticos
    $checks = @{
        "updateMatrixWorld(true)" = $content -match "updateMatrixWorld\(true\)"
        "size={0.02}" = $content -match "size=\{0\.0[2-9]"
        "scale={[2" = $content -match "scale=\{\[2"
        "useFrame" = $content -match "useFrame"
        "preserveDrawingBuffer" = $content -match "preserveDrawingBuffer"
    }
    
    foreach ($check in $checks.GetEnumerator()) {
        if ($check.Value) {
            Write-Host "   ✅ $($check.Key)" -ForegroundColor Green
        } else {
            Write-Host "   ⚠️ $($check.Key) - Verifica manualmente" -ForegroundColor Yellow
        }
    }
}

Write-Host ""

# 5. Resumen
Write-Host "📋 RESUMEN:" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
Write-Host "✅ Pasos completados:" -ForegroundColor Green
Write-Host "   1. Componente principal actualizado" -ForegroundColor White
Write-Host "   2. Herramientas de debugging creadas" -ForegroundColor White
Write-Host "   3. Documentación completa generada" -ForegroundColor White
Write-Host ""
Write-Host "🚀 Siguiente paso:" -ForegroundColor Yellow
Write-Host "   cd frontend" -ForegroundColor White
Write-Host "   npm start" -ForegroundColor White
Write-Host ""
Write-Host "🔍 Para debugging:" -ForegroundColor Yellow
Write-Host "   1. Abre DevTools (F12)" -ForegroundColor White
Write-Host "   2. Busca en consola: '✅ Cs2_agent.glb'" -ForegroundColor White
Write-Host "   3. Verifica BoundingBox values" -ForegroundColor White
Write-Host ""
Write-Host "📚 Lee la documentación:" -ForegroundColor Yellow
Write-Host "   frontend\src\components\Landing3D\TROUBLESHOOTING.md" -ForegroundColor White
Write-Host "   frontend\src\components\Landing3D\IMPLEMENTACION.md" -ForegroundColor White
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
