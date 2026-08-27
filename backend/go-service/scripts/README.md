# Scripts de validación y reprocesamiento canónico

Este directorio contiene la interfaz operativa de Block 7 para validar y
republicar exports de demos. El contrato vigente es parser `v16`, export
`3.8.0`, quality schema `12` y manifest
`stratai.canonical_manifest@3`.

## Requisitos

- El entorno `backend/venv` con las dependencias de desarrollo instaladas.
- Redis disponible en `REDIS_ADDR`.
- El servicio Go iniciado desde `backend/go-service`.
- El mismo `INTERNAL_SERVICE_SECRET` de al menos 32 caracteres en el cliente y
  el servicio Go. La republicación no admite el modo local sin firma.
- Cada `backend/data/demos/match_<id>.dem` debe tener un sidecar
  `match_<id>.dem.sha256` válido.

Ejemplo de arranque local:

```powershell
Push-Location backend/go-service
$env:REDIS_ADDR = "127.0.0.1:6379"
$env:INTERNAL_SERVICE_SECRET = "replace-with-at-least-32-characters"
$env:STRATAI_BUILD_ID = "local-block7-build"
go run .
Pop-Location
```

`GET /health` debe responder `status=ok`, `redis=available`, parser `v16` y
export `3.8.0` antes de publicar.

## Reprocesado seguro

Ejecuta siempre desde la raíz del repositorio y selecciona los matches de
forma explícita. El flujo operativo de Block 7 usa exactamente un worker,
dos reintentos, 1.500 segundos de timeout por intento y no reconstruye
agregados:

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

Si el preflight pasa, repite el comando sin `--dry-run`:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\reprocess_parallel.py `
  --workers 1 `
  --retries 2 `
  --timeout 1500 `
  --skip-aggregate-rebuild `
  --match-id <match-id>
```

`--match-id` puede repetirse para un lote explícito. El proceso es secuencial
y se detiene en el primer fallo. Sólo reintenta fallos transitorios y genera
un nonce HMAC nuevo en cada intento; un error HTTP 4xx no se reintenta.

Para un ensayo de coherencia de diez demos, añade diez `--match-id` y el gate
`--expected-count 10`. El script aborta antes del backup si la selección final
no contiene exactamente diez demos:

```powershell
$matchIds = @(
  "<match-id-01>", "<match-id-02>", "<match-id-03>", "<match-id-04>",
  "<match-id-05>", "<match-id-06>", "<match-id-07>", "<match-id-08>",
  "<match-id-09>", "<match-id-10>"
)
$selection = foreach ($matchId in $matchIds) { "--match-id"; $matchId }

& backend\venv\Scripts\python.exe backend\go-service\scripts\reprocess_parallel.py `
  --workers 1 `
  --retries 2 `
  --timeout 1500 `
  --skip-aggregate-rebuild `
  --expected-count 10 `
  @selection `
  --dry-run
```

Tras un dry-run correcto, repite el comando sin `--dry-run`.

Antes de procesar, el script:

1. verifica el sidecar y recalcula el SHA-256 de cada demo;
2. exige staging limpio y ausencia de directorios `match_*.rollback.*`;
3. comprueba salud, Redis, versiones y firma interna;
4. copia los exports actuales fuera de `backend/data/exports/`;
5. compara SHA-256 origen/destino y escribe `backup_manifest.json`.

El backup predeterminado se conserva en
`backend/data/export_backups/block7_reprocess_<UTC>/`. Puede indicarse una ruta
externa a `exports` con `--backup-dir`, pero debe ser nueva. Un fallo conserva
el backup, no publica el bundle inválido, limpia su staging y detiene el lote.

`reprocess_all_demos.py` es sólo un alias de compatibilidad. Para el protocolo
de cierre no se usan `--limit`, varios workers ni la regeneración de agregados.

## Validadores

Validación fail-closed de un bundle y del corpus golden:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\publication_validator.py `
  backend\data\exports\match_<match-id> `
  --match-id <match-id> `
  --checksum <sha256-demo>
```

Validación independiente del corpus de contratos:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\golden_corpus_validator.py `
  backend\go-service\testdata\golden-corpus\v1
```

Validación del inventario secreto-free y de la selección determinista de 40
demos (las roots locales se pasan sólo en ejecución y nunca se guardan en el
manifest):

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\golden_demo_corpus_validator.py `
  backend\go-service\testdata\golden-demos\v2\manifest.json `
  --stratai-root <root-demos-stratai> `
  --faceit-root <root-demos-faceit>
```

`source_inventory_valid` prueba bytes, SHA-256, magic y mapa de las fuentes;
no prueba Gate 1. El manifest v2 mantiene los contadores actuales de evidencia
semántica y continúa parcial hasta reprocesar y validar todo el corpus.

Reprocesamiento semántico reanudable del corpus seleccionado:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\golden_demo_semantic_runner.py `
  backend\go-service\testdata\golden-demos\v2\manifest.json `
  --stratai-root <root-demos-stratai> `
  --faceit-root <root-demos-faceit> `
  --maps-root <root-geometria-mapas>
```

El runner guarda cada resultado terminado de forma atómica y puede reanudarse
sin repetir los casos vigentes. Las rutas locales solo se resuelven en memoria:
la salida y el manifest usan alias opacos. Los ficheros tácticos grandes se
validan línea a línea para limitar la memoria, y la lista de errores también
tiene un límite para que una fuente dañada no bloquee el equipo.

`passed` significa que export, identidad y contrato canónico coinciden.
`quarantined` aparta la demo con un motivo cerrado; no la borra. Aunque las 40
demos terminen, Gate 1 sigue pendiente hasta completar la revisión humana de
las dos familias de decisiones.

Suites específicas de Block 7:

```powershell
backend\venv\Scripts\python.exe -m pytest `
  backend\go-service\scripts\test_canonical_export_validator.py `
  backend\go-service\scripts\test_block7_publication.py `
  backend\go-service\scripts\test_reprocess_parallel.py
```

Los auditores semánticos de Blocks 4–6 siguen disponibles como diagnósticos
adicionales; no sustituyen al validador de publicación:

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\combat_export_audit.py `
  backend\data\exports\match_<match-id>

backend\venv\Scripts\python.exe backend\go-service\scripts\economy_stats_metadata_audit.py `
  backend\data\exports\match_<match-id> --json
```

Consulta también:

- [`docs/canonical-publication-rollback.md`](../../../docs/canonical-publication-rollback.md)
- [`docs/canonical-export-schema-v3.8.md`](../../../docs/canonical-export-schema-v3.8.md)
- [`docs/golden-corpus.md`](../../../docs/golden-corpus.md)

Última revisión: 21 de agosto de 2026.
