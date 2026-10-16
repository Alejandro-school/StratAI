# StratAI

StratAI analiza partidas de Counter-Strike 2 a partir de demos de Steam y presenta estadísticas y repeticiones 2D. El AI Coach desarrolla el ciclo de análisis, prioridades y práctica; su alcance demostrado y siguiente paso se consultan en el [plan canónico](docs/AI_COACH_IMPLEMENTATION_PLAN.md).

Si te incorporas al proyecto, empieza por la [guía de incorporación](docs/getting-started.md).

## Mapa rápido

```text
Navegador
   │
   ▼
Frontend React :3000
   │
   ▼
API FastAPI :8000 ─────────────── Redis :6379
   │                                  ▲
   ▼                                  │
Servicio Steam (Node.js) :4000 ───────┤
   │                                  │
   ▼                                  │
Analizador de demos (Go) :8080 ───────┘
```

| Ruta | Responsabilidad |
| --- | --- |
| `frontend/` | Aplicación web React y experiencia de usuario. |
| `backend/app/` | API pública FastAPI, autenticación y agregación de datos. |
| `backend/node-service/` | Integración con Steam, descargas y orquestación del pipeline. |
| `backend/go-service/` | Análisis de demos y generación de datos de partida. |
| `ai_coach/` | Motor Python del coach: contratos, contextos, interpretación y validación. |
| `backend/tests/` | Pruebas de la API FastAPI. |
| `docs/` | Producto, plan, diseño, contratos, operación e histórico. |
| `outputs/` | Entregables y evidencia fechada; no es una aplicación. |
| `.github/` y `.agents/` | Automatización y configuración de asistentes; no contienen lógica del producto. |

Consulta [la guía de estructura](docs/repository-structure.md) para saber dónde realizar cada tipo de cambio.

## Flujo de una partida

1. FastAPI recibe la identidad de Steam y expone los datos al frontend.
2. Node.js descubre partidas, descarga sus demos y coordina los trabajos mediante Redis.
3. Go procesa cada demo y guarda los resultados estructurados.
4. FastAPI agrega esos resultados.
5. React presenta el historial, las métricas, el mapa táctico y el coaching.

## Requisitos

- Node.js 24 LTS (versión fijada en `.nvmrc`).
- Python 3.11 o 3.12.
- Go 1.26.9 o posterior compatible, conforme a `backend/go-service/go.mod`.
- Redis.

## Preparación local

Instala las dependencias JavaScript:

```powershell
npm ci
npm --prefix frontend ci
npm --prefix backend/node-service ci
```

Prepara Python:

```powershell
python -m venv backend/venv
backend\venv\Scripts\python.exe -m pip install --upgrade pip==26.2.1
backend\venv\Scripts\python.exe -m pip install --require-hashes -r backend/requirements-dev.txt
Copy-Item backend/.env.example backend/.env
```

Prepara Go y las configuraciones locales:

```powershell
Push-Location backend/go-service
go mod download
Pop-Location

Copy-Item frontend/.env.example frontend/.env
Copy-Item backend/node-service/.env.example backend/node-service/.env
```

Completa al menos `SESSION_SECRET_KEY`, `STEAM_API_KEY` y las credenciales del bot de Steam. `SESSION_SECRET_KEY` e `INTERNAL_SERVICE_SECRET` deben ser coherentes entre los servicios que los comparten.

## Desarrollo

Inicia Redis y ejecuta desde la raíz:

```powershell
npm run start:all
```

`start:all` comprueba la configuración compartida y la conexión con Redis antes
de iniciar FastAPI, Node, Go y el frontend. Si un servicio termina, cierra los
demás para evitar un arranque parcial. No hay un modo de interfaz que omita la
validación de Steam. Puedes ejecutar la comprobación por separado con
`npm run check:services`.

El usuario inicia sesión con Steam y conecta los códigos de su historial una
vez. Se importa la partida introducida, después de validar el acceso con Steam,
y se buscan las posteriores automáticamente. La cuenta técnica de Steam es
interna; no hay que agregarla como amigo. Consulta el [flujo de demos](backend/README.md#acceso-al-historial-y-descarga-de-demos).

| Servicio | Dirección | Arranque individual |
| --- | --- | --- |
| Frontend | `http://localhost:3000` | `npm run frontend` |
| FastAPI | `http://localhost:8000` | `npm run backend` |
| Node.js | `http://localhost:4000` | `npm run node-service` |
| Go | `http://localhost:8080` | `npm run go-service` |
| Redis | `redis://localhost:6379` | Depende de la instalación local. |

La documentación interactiva de FastAPI está disponible en `http://localhost:8000/docs` fuera de producción.

## Calidad

```powershell
npm run check:all
```

También puedes ejecutar comprobaciones concretas:

```powershell
npm run test:frontend
npm run test:node
npm run test:go
npm run test:python
npm run lint:python
npm run build:frontend
npm run check:docs
```

GitHub Actions ejecuta pruebas para React, Python, Node.js y Go, análisis de dependencias y detección de secretos. Consulta la [configuración de seguridad y despliegue](docs/operations/security-hardening-2026-10-15.md).

## Documentación por responsabilidad

| Para qué | Entrada |
|---|---|
| Entender el repositorio y ubicar un cambio | [Mapa del proyecto](docs/repository-structure.md) |
| Trabajar en la interfaz | [Frontend](frontend/README.md) |
| Trabajar en API, Steam o parsing | [Backend y servicios](backend/README.md) |
| Trabajar en contextos e interpretación | [Motor AI Coach](ai_coach/README.md) |
| Consultar producto, diseño y contratos | [Índice documental](docs/README.md) |
| Validar o publicar datos | [Operación](docs/operations/README.md) |
| Consultar revisiones y workbooks | [Entregables](outputs/README.md), referenciados desde el plan |

`check:all` reúne las comprobaciones de frontend, backend Python, Node y Go. Las pruebas específicas del motor se ejecutan además según [ai_coach/README.md](ai_coach/README.md).

## Problemas habituales

- Si FastAPI no arranca, revisa los secretos obligatorios de `backend/.env`.
- Si el frontend no conecta, revisa `VITE_API_URL`.
- Si trabajas con demos, comprueba Redis, las credenciales de Steam, `STEAM_API_KEY`, `GO_SERVICE_URL` y los permisos de `backend/data/`.
- Si Ubuntu se cierra al abrirse, consulta [Redis y WSL en Windows](backend/README.md#redis-y-wsl-en-windows). Un fallo de WSL impide arrancar Redis y el pipeline completo.

Conserva los datos locales, las salidas retenidas y los cambios previos del directorio de trabajo. Las reglas de colaboración y verificación están en [AGENTS.md](AGENTS.md).

Última revisión de estructura: 5 de octubre de 2026.
