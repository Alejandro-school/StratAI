@echo off
title Re-procesamiento de Demos CS2
color 0A

echo.
echo ========================================
echo    Re-procesamiento de Demos CS2
echo ========================================
echo.

REM Verificar si el ejecutable existe
if not exist "reprocess_demos.exe" (
    echo [INFO] Compilando el script por primera vez...
    echo.
    go build -o reprocess_demos.exe scripts/reprocess_all.go
    if errorlevel 1 (
        echo.
        echo [ERROR] No se pudo compilar el script
        echo Verifica que Go este instalado correctamente
        pause
        exit /b 1
    )
    echo [OK] Script compilado exitosamente
    echo.
)

REM Ejecutar el script
echo [INFO] Iniciando re-procesamiento...
echo.
reprocess_demos.exe

echo.
echo ========================================
echo.
pause

