@echo off
echo ===================================
echo Reiniciando servicio Go...
echo ===================================
echo.

REM Matar procesos Go existentes
taskkill /F /IM main.exe 2>nul
timeout /t 2 >nul

echo Compilando servicio...
go build -o main.exe

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ===================================
    echo Compilación exitosa!
    echo ===================================
    echo.
    echo Iniciando servicio en el puerto 8080...
    start "Go Service" cmd /k main.exe
    echo.
    echo Servicio iniciado correctamente!
) else (
    echo.
    echo ===================================
    echo ERROR en la compilación
    echo ===================================
    pause
)

