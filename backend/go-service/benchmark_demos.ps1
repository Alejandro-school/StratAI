# Script de benchmarking para el demo parser optimizado

$demos = @(
    "match_2ycniSRrujsiUfXJf2WqQ4zAC.dem",
    "match_6DhZ28LiQKRMfj4tDG8CKUGnK.dem",
    "match_bCqFLZvWToJvc7pb3i26tzioE.dem"
)

$demoPath = "e:\Carpeta compartida\Proyecto IA\Proyecto IA\backend\data\demos"

Write-Host "=== BENCHMARK DE DEMOS OPTIMIZADAS ===" -ForegroundColor Cyan
Write-Host ""

$totalTime = 0
$results = @()

foreach ($demo in $demos) {
    $fullPath = Join-Path $demoPath $demo
    
    # Obtener tamaño del archivo
    $fileSize = (Get-Item $fullPath).Length / 1MB
    
    Write-Host "Procesando: $demo" -ForegroundColor Yellow
    Write-Host "  Tamaño: $([math]::Round($fileSize, 2)) MB"
    
    $body = @{
        demo_path = $fullPath
        match_id = "benchmark_$(Get-Random)"
        steam_id = ""
    } | ConvertTo-Json
    
    $elapsed = Measure-Command {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/process-demo" `
                -Method POST `
                -ContentType "application/json" `
                -Body $body `
                -ErrorAction Stop
        } catch {
            Write-Host "  ❌ Error: $_" -ForegroundColor Red
        }
    }
    
    $seconds = [math]::Round($elapsed.TotalSeconds, 2)
    $totalTime += $seconds
    
    Write-Host "  ⏱️  Tiempo: $seconds segundos" -ForegroundColor Green
    Write-Host ""
    
    $results += [PSCustomObject]@{
        Demo = $demo
        SizeMB = [math]::Round($fileSize, 2)
        TimeSeconds = $seconds
        MBPerSecond = [math]::Round($fileSize / $elapsed.TotalSeconds, 2)
    }
}

Write-Host "=== RESUMEN ===" -ForegroundColor Cyan
$results | Format-Table -AutoSize

$avgTime = [math]::Round($totalTime / $demos.Count, 2)
Write-Host "Tiempo promedio por demo: $avgTime segundos" -ForegroundColor Green
Write-Host "Tiempo total: $totalTime segundos" -ForegroundColor Green

# Comparación con tiempo anterior
Write-Host ""
Write-Host "=== COMPARACIÓN ===" -ForegroundColor Cyan
Write-Host "Tiempo promedio anterior: ~4.0 segundos" -ForegroundColor Yellow
Write-Host "Tiempo promedio actual: $avgTime segundos" -ForegroundColor Green
if ($avgTime -lt 4.0) {
    $improvement = [math]::Round((4.0 - $avgTime) / 4.0 * 100, 1)
    Write-Host "Mejora: $improvement porciento" -ForegroundColor Green
}
