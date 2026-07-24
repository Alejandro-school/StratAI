# StratAI

StratAI es una plataforma de análisis y coaching para partidas de Counter-Strike 2. Combina una interfaz React, una API FastAPI, un servicio Node.js para Steam y descargas, y un analizador de demos escrito en Go.

## Arquitectura

```text
React :3000
    |
FastAPI :8000
    |--------- Redis :6379
    |
Node.js :4000 ------ Steam / descargas
    |
Go :8080 ----------- análisis de demos
```

La descripción detallada del sistema de coaching está en [AI_COACH_ARCHITECTURE.md](AI_COACH_ARCHITECTURE.md).

## Requisitos

- Node.js 20.19 o superior
- Python 3.11 o 3.12
- Go 1.23.4
- Redis

## Preparación

Instala las dependencias JavaScript:

```powershell
npm ci
npm --prefix frontend ci
npm --prefix backend/node-service ci
```

Prepara Python:

```powershell
python -m venv backend/venv
backend\venv\Scripts\python.exe -m pip install -r backend/requirements-dev.txt
Copy-Item backend/.env.example backend/.env
backend\venv\Scripts\Activate.ps1
```

Prepara el servicio Go:

```powershell
Push-Location backend/go-service
go mod download
Pop-Location
```

Copia también la configuración del frontend y del servicio Steam:

```powershell
Copy-Item frontend/.env.example frontend/.env
Copy-Item backend/node-service/.env.example backend/node-service/.env
```

Completa al menos `SESSION_SECRET_KEY`, `STEAM_API_KEY` y las credenciales del bot Steam. El valor de `SESSION_SECRET_KEY` debe ser el mismo en FastAPI y Node.js.

## Desarrollo

Inicia Redis y después ejecuta todos los servicios desde la raíz:

```powershell
npm run start:all
```

Servicios locales:

| Servicio | Dirección |
| --- | --- |
| Frontend | `http://localhost:3000` |
| FastAPI | `http://localhost:8000` |
| Node.js | `http://localhost:4000` |
| Go | `http://localhost:8080` |
| Redis | `redis://localhost:6379` |

También puedes iniciar cada proceso por separado con `npm run frontend`, `npm run backend`, `npm run node-service` y `npm run go-service`.

## Calidad

```powershell
npm run test:all
npm run build:frontend
```

Las mismas comprobaciones se ejecutan en GitHub Actions para frontend, Python, Node.js y Go.

## Problemas habituales

- Si FastAPI no arranca, revisa que `backend/.env` contenga los dos secretos obligatorios.
- Si no se detectan partidas, comprueba Redis, las credenciales Steam y `STEAM_API_KEY`.
- Si el frontend no conecta, revisa `VITE_API_URL` y `VITE_NODE_URL`.
- Los modelos requeridos por la portada están en `frontend/public/images/Landing`.

Última revisión: 2026-07-24.
