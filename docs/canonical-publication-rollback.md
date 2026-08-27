# Publicación canónica, rollback y recuperación

Estado: runbook vigente para parser `v16` y export `3.8.0`.

La publicación de una demo es una transacción compensable entre filesystem y
Redis. Ningún bundle se hace visible antes de superar el validador fail-closed.

## Flujo de commit

```text
request HMAC
  → lock por match/checksum/schema
  → recuperar commit interrumpido y limpiar staging obsoleto
  → parsear en exports/.tmp/<match-id>-<token>/match_<match-id>
  → construir catálogo y manifest
  → validar bundle + corpus golden
  → snapshot de Redis con TTL
  → mover export previo a match_<id>.rollback.<token>
  → rename atómico de staging a match_<id>
  → persistir Redis
  → eliminar rollback y responder receipt
```

El lock e idempotency key incorporan `match_id`, checksum de demo y versión de
schema. Un éxito idempotente sólo se acepta si el manifest comprometido es un
archivo regular, único, tiene el match/checksum/version esperados y su catálogo
coincide con tamaño y SHA-256 reales en disco.

## Precondiciones

- Redis responde y `GET /health` expone `status=ok`, `redis=available`, parser
  `v16`, export `3.8.0` y validator
  `stratai.canonical_validator@2`.
- La solicitud interna lleva firma HMAC v1, timestamp vigente y nonce no
  reutilizado. El secreto debe tener al menos 32 caracteres.
- La demo tiene un sidecar `.dem.sha256` y el checksum recalculado coincide.
- `backend/data/exports/.tmp/` no contiene staging activo de otro proceso para
  el match y no existe más de un rollback ambiguo.

## Fallos y compensación

| Punto de fallo | Resultado |
| --- | --- |
| Parseo, catálogo o validador | Se elimina staging; el export previo y Redis no cambian. |
| Snapshot de Redis | No comienza el commit del filesystem. |
| Rename del export previo | Se aborta sin exponer staging. |
| Rename de staging | Se restaura inmediatamente el export previo. |
| Escritura Redis posterior al rename | Se restaura el snapshot Redis con TTL y se revierte el filesystem. |
| Limpieza final del rollback | El commit queda válido y se registra warning operacional para limpieza segura. |
| Rollback fallido o ambiguo | Fail-closed, HTTP 500 y recuperación manual; nunca se declara éxito. |

Las operaciones Redis de snapshot, save y restore usan reintentos acotados y
contextos con timeout. La clasificación pública distingue al menos
`parse_failure`, `contract_validation`, `validator_timeout`, `commit_failure`,
`redis_failure` y `rollback_failure` para métricas y logs estructurados.

## Recuperación tras crash

Al recibir un nuevo trabajo del mismo match, el servicio ejecuta la recuperación
bajo el lock:

1. si falta el directorio final y hay un solo `match_<id>.rollback.*`, lo
   restaura;
2. si el directorio final es válido, elimina sólo sus backups de rollback
   hijos con prefijo exacto;
3. si hay varios candidatos ambiguos, se detiene sin borrar nada;
4. limpia únicamente staging hijo de `.tmp` cuyo nombre empiece por el
   `match_id` saneado.

Todas las rutas se resuelven y se comprueba que sean hijos exactos del
directorio esperado antes de mover o eliminar. No se aceptan symlinks, globs
amplios ni rutas proporcionadas por el cliente.

## Reprocesado de un match

Primero ejecuta el preflight:

```powershell
$env:INTERNAL_SERVICE_SECRET = "replace-with-at-least-32-characters"

backend\venv\Scripts\python.exe backend\go-service\scripts\reprocess_parallel.py `
  --workers 1 `
  --retries 2 `
  --timeout 1500 `
  --skip-aggregate-rebuild `
  --match-id <match-id> `
  --dry-run
```

Publica sólo si el dry-run termina con código cero:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\reprocess_parallel.py `
  --workers 1 `
  --retries 2 `
  --timeout 1500 `
  --skip-aggregate-rebuild `
  --match-id <match-id>
```

No cambies esos cuatro parámetros en el cierre de Block 7. Para un lote,
repite `--match-id`; se procesa en orden, con un solo worker, y se detiene en
el primer fallo.

## Backup verificable

Antes de enviar la primera demo, el script crea un directorio nuevo fuera de
`backend/data/exports/`, copia cada export previo, calcula SHA-256 de todos los
archivos antes y después y escribe `backup_manifest.json` con:

- `schema_id=stratai.reprocess_backup@1`;
- fecha UTC de creación;
- checksum de cada demo;
- hash de cada archivo por match.

El backup se conserva tanto en éxito como en fallo. No se reconstruyen
`backend/data/users/` ni otros agregados durante este protocolo.

## Verificación posterior

Después del lote:

1. ejecuta `publication_validator.py` para cada match y detente en el primer
   error;
2. confirma que `.tmp` está vacío y no existen
   `match_*.rollback.*`;
3. verifica de nuevo los hashes de `backup_manifest.json`;
4. para una prueba de determinismo, republica los pilotos con los mismos flags
   y compara SHA-256 de todos los archivos bajo `canonical/`.

El acta de referencia de este procedimiento está en
[`block7-closure-2026-08-21.md`](archive/parser-evolution/block7-closure-2026-08-21.md).
