# StratAI

StratAI es una plataforma de análisis y coaching para partidas de Counter-Strike 2. Convierte demos de Steam en estadísticas, repeticiones 2D y recomendaciones tácticas.

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
| `backend/tests/` | Pruebas de la API FastAPI. |
| `docs/` | Guías operativas y documentación transversal. |
| `.github/` y `.agents/` | Automatización y configuración de asistentes; no contienen lógica del producto. |

Consulta [la guía de estructura](docs/repository-structure.md) para saber dónde realizar cada tipo de cambio.

## Flujo de una partida

1. FastAPI recibe la identidad de Steam y expone los datos al frontend.
2. Node.js descubre partidas, descarga sus demos y coordina los trabajos mediante Redis.
3. Go procesa cada demo y guarda los resultados estructurados.
4. FastAPI agrega esos resultados.
5. React presenta el historial, las métricas, el mapa táctico y el coaching.

## Requisitos

- Node.js 20.19 o superior.
- Python 3.11 o 3.12.
- Go 1.23.4.
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
backend\venv\Scripts\python.exe -m pip install -r backend/requirements-dev.txt
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
```

GitHub Actions ejecuta pruebas para React, Python, Node.js y Go.

### Export canónico y republicación

El contrato de datos vigente es parser `v16`, export `3.8.0`, quality schema
`12` y manifest `stratai.canonical_manifest@3`. La publicación se valida en
staging y se confirma de forma compensable entre filesystem y Redis.

Para republicar una demo, inicia Redis y el servicio Go con el mismo
`INTERNAL_SERVICE_SECRET` de al menos 32 caracteres. Ejecuta primero el dry-run
desde la raíz:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\reprocess_parallel.py `
  --workers 1 `
  --retries 2 `
  --timeout 1500 `
  --skip-aggregate-rebuild `
  --match-id <match-id> `
  --dry-run
```

Si el preflight pasa, repite el mismo comando sin `--dry-run`. El script
verifica sidecars SHA-256, crea un backup fuera de `exports`, procesa de forma
secuencial y se detiene en el primer fallo. Consulta el
[runbook completo](docs/canonical-publication-rollback.md) antes de operar un
lote.

## Documentación

- [Estructura y convenciones del repositorio](docs/repository-structure.md)
- [Backend y responsabilidades de sus servicios](backend/README.md)
- [Frontend](frontend/README.md)

### AI Coach

Estas son las fuentes activas. El plan de implementación es el único documento que indica el estado y la siguiente acción:

Estado actual: Gate 1A cerrado, `training_allowed=false`. La fase vigente construye contratos y ejemplos estructurales de solo lectura; todavía no entrena modelos.

- [Definición del producto AI Coach](docs/AI_COACH_PRODUCT_DEFINITION.md)
- [Plan de implementación del AI Coach](docs/AI_COACH_IMPLEMENTATION_PLAN.md)
- [Contrato humano de capacidades y datos](docs/AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md)
- [Catálogo máquina de capacidades](ai_coach/contracts/capability_catalog.json)

Referencias necesarias para la implementación actual:

- [Arquitectura de modelos](docs/AI_COACH_MODEL_ARCHITECTURE.md)
- [Prompt vigente: contrato y constructor read-only de ejemplos](docs/prompts/AI_COACH_DECISION_DATASET_CONTRACT_PHASE_PROMPT.md)

Las auditorías fechadas y las evidencias conservan la historia de cada gate. `docs/prompts/` contiene únicamente el prompt vigente; cada prompt ejecutado se elimina después de trasladar su resultado al plan y al informe de cierre.

### Operación y contratos canónicos

- [Despliegue progresivo del pipeline v2](docs/pipeline-v2-rollout.md)
- [Contrato de export canónico 3.8](docs/canonical-export-schema-v3.8.md)
- [Publicación y rollback](docs/canonical-publication-rollback.md)
- [Corpus golden](docs/golden-corpus.md)

## Problemas habituales

- Si FastAPI no arranca, revisa los secretos obligatorios de `backend/.env`.
- Si no se detectan partidas, comprueba Redis, las credenciales de Steam y `STEAM_API_KEY`.
- Si el frontend no conecta, revisa `VITE_API_URL`.
- Si Go no puede procesar una demo, comprueba `GO_SERVICE_URL`, Redis y los permisos de `backend/data/`.

Última revisión: 27 de agosto de 2026.
