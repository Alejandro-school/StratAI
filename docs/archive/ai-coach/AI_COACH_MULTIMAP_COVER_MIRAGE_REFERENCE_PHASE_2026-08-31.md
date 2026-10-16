# Cierre 1B.8 — contrato multimapa de cobertura y referencia parcial Mirage

Fecha: 31 de agosto de 2026  
Estado: **cerrada en código**  
Gate 1B: **pendiente**  
Entrenamiento: **no autorizado**

## Resultado

La fase congela una infraestructura de `FLD-COVER` independiente del mapa y una única referencia candidata parcial para `de_mirage`. El contrato distingue tres afirmaciones que ya no pueden confundirse:

1. la oclusión objetiva de un volumen de jugador, recomputable desde geometría;
2. la clasificación táctica humana de esa geometría como cobertura útil, todavía pendiente;
3. una ruta de salida viable, expresamente fuera de alcance hasta la fase siguiente.

Se añadieron los schemas Draft 2020-12 `stratai.cover_map_package@1`, `stratai.cover_support_manifest@1` y `stratai.observed_cover_diagnostic@1`; loaders equivalentes en Go y Python; un generador determinista sobre el BVH físico real; un evaluador causal full-demo; un modo aislado de CLI; y un validador independiente streaming. La resolución es por `map_id`, contrato, versión, hashes y assets fuente exactos. Mapa desconocido, paquete o asset ausente/alterado, versión incompatible, `map_id` incorrecto, solapamiento o dato de posición/postura/vista inválido produce `unavailable`, nunca un fallback implícito a Mirage.

El código común no contiene nombres de lugares ni ramas de Mirage. El soporte declarado vive en un manifest versionado y actualmente contiene sólo un paquete candidato, sin mapas soportados como producto.

## Contrato objetivo de volumen

El volumen explícito es `cylinder_cross_section_probes@1`, radio 16 unidades Source 2. Cada dirección usa una cuadrícula de nueve rayos: tres offsets laterales `[-16, 0, 16]` por tres alturas dependientes de postura. Para `crouching`, el volumen mide 54 unidades y las alturas son `[18, 34, 46]`; para `standing`, mide 72 y usa `[18, 45, 64]`.

Las direcciones parten de `+X`, giran en sentido antihorario hacia `+Y`, se normalizan en `[0,360)` y representan una posible línea entrante hacia el volumen. El paquete muestrea cada 15°, limita el raycast a 128 unidades y sólo declara oclusión completa dentro de 96 si los nueve probes quedan bloqueados. Un único rayo bloqueado no acredita protección de volumen. Las distancias y los nueve resultados permanecen en el asset para recomputación independiente.

La geometría objetiva y la anotación humana tienen IDs, versiones, procedencia, revisión y referencias separadas. Un perfil objetivo puede indicar oclusión sin convertirse por ello en «buen cover», consejo, riesgo contextual o ruta de escape.

## Referencia parcial de `de_mirage`

El paquete `1.0.0-candidate.1` contiene:

- 8 volúmenes de muestra parciales, cada uno con radio de selección 192 y tolerancia vertical 96;
- 192 direcciones —24 por muestra— y 384 perfiles postura/dirección;
- 9 probes por perfil;
- 384 placeholders tácticos separados, todos `pending_review`, sin autor, revisor, evidencia ni clasificación;
- 11 perfiles completamente ocluidos para `crouching` y 11 para `standing` en los orígenes de muestra;
- `full_map_coverage=false`, `product_supported=false` y `training_use_authorized=false`.

Dos regeneraciones independientes desde los inputs protegidos coincidieron byte a byte entre sí y con el paquete almacenado. El SHA-256 es `94313136aebd680a5b5c4a0b582fda6132499e82845e3e4bc0e98767c58f187c`; la revisión geométrica es `46924b6e46278ec51ea3dcf69016f15b23ed859be37671c2124e10b074effef1`.

El GLTF/bin físico aporta la geometría y `de_mirage_places.json` únicamente semillas de coordenadas. Los nombres de origen no son etiquetas tácticas. NAV, callouts y el paquete previo de exposición se auditaron, pero no se usaron como verdad de cover: conectividad, nombre de zona o clearance angular no prueban oclusión del volumen ni calidad táctica.

## Productor y semántica causal

`BuildObservedCoverFullDemoReport` muestrea jugadores vivos a 1 Hz con estrategia `round_anchored_right_closed_alive_players`. En cada T0 publica sólo alias opaco, posición y yaw disponibles en ese instante, postura real `crouching` o `standing`, selección de una muestra única y los 24 perfiles objetivos absolutos/relativos a la vista. No consulta futuro, oracle, outcomes, identidades públicas ni estado enemigo.

La posición no finita se conserva como `null` con ausencia explícita. Fuera de los ocho volúmenes parciales se abstiene. Dentro de cobertura, la geometría puede publicarse, pero la conclusión táctica siempre se abstiene mientras falte revisión humana. La salida de ruta se mantiene separada y devuelve `exit_route_out_of_scope` en todas las filas.

El sidecar candidato queda fuera del manifest canónico `3.8.0`; no altera `enemy_los`, replay, tactical, los 44 bundles ni la elegibilidad histórica.

## Smoke real obligatorio

Se procesó exactamente una demo aprobada, en un único intento del productor y una salida nueva bajo `%TEMP%`:

- alias: `demo-a11eb0d600ee702c9edb`;
- SHA-256: `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`;
- mapa: `de_mirage`;
- parser/export: `v16` / `3.8.0`;
- build: `working-tree-cover-mirage-1b8-20260831-r1`;
- duración del productor: 95,236 s.

Comando exacto, ejecutado desde `backend/go-service`:

```powershell
go run ./cmd/canonical-export --demo 'E:\Carpeta compartida\Proyecto IA\Faceit-Demos\demos\1-3f9e5224-791f-4a15-ac14-5acaacec9407.dem' --output 'C:\Users\kerch\AppData\Local\Temp\stratai-one-demo-cover-1b8-20260831-r1' --alias 'demo-a11eb0d600ee702c9edb' --checksum 'a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262' --maps-root 'E:\Carpeta compartida\Proyecto IA\StratAI\backend\data\maps' --cover-packages-root 'E:\Carpeta compartida\Proyecto IA\StratAI\ai_coach\spatial\cover' --build-id 'working-tree-cover-mirage-1b8-20260831-r1' --ai-coach-cover-validation
```

Resultado real:

- 24 rondas, 29.520 frames fuente, 10 aliases y 12.802 filas a 1 Hz;
- 883 posiciones dentro de los ocho volúmenes y 11.919 ausencias `outside_declared_sample_coverage`;
- 21.192 perfiles direccionales observados —883 × 24—;
- 1.119 filas con postura `crouching` y 11.683 con `standing`;
- 26 direcciones completamente ocluidas en las filas crouching y 865 en las standing;
- las ocho muestras tuvieron datos: 55, 22, 63, 105, 193, 61, 208 y 176 filas;
- 0 clasificaciones tácticas, 12.802 abstenciones tácticas;
- 0 rutas de salida, 12.802 abstenciones explícitas por estar fuera de alcance.

El validador independiente tardó 9,784 s y recomputó schema, hashes físico y lógico, procedencia, paquete/assets, selección espacial, postura, transformación de yaw, los nueve probes de cada perfil, agregados, conteos y privacidad. No encontró mismatches, ticks futuros, campos oracle/outcome ni identidades crudas. El gzip tiene 571.456 bytes y SHA-256 `8d4cab6d6cde0ba3790f5486014285f43dcb7a54d4d9ca3cfbcc1a7c32d3d959`; su contenido lógico tiene 20.767.931 bytes y SHA-256 `f8d27767dee0b00d965673bf4fce6954b38fc5ba69475e3bc7c1aff610c5bbfc`.

La validación semántica del bundle temporal tardó 345,950 s y devolvió cero errores. La auditoría read-only adicional tardó 8,694 s: 918 `DecisionExample`, 918 joins correctos y cero fallidos. El bundle canónico conservó 73 ficheros, 72 artefactos declarados, contrato `3.8.0` y parser `v16`.

## Pruebas superadas

- `go test -count=1 ./pkg/cover ./parser ./cmd/canonical-export ./cmd/cover-package-generator`: verde;
- `go vet` sobre esos cuatro paquetes: verde;
- `python -m pytest -q ai_coach/tests`: 175 pruebas pasadas;
- `python -m ruff check ai_coach`: sin errores;
- `python -m ai_coach.contracts.validate_contract`: 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 14 schemas runtime y 18 reason codes;
- `npm run check:all`: 88 pruebas frontend, Node 14 pasadas/3 omitidas, todos los paquetes Go, 43 pruebas Python backend y build frontend correcto;
- doble regeneración del paquete: ambos outputs y el almacenado con el mismo SHA-256;
- validador independiente de cover: aprobado;
- bundle semántico y auditoría directa: 0 errores, 918/918 joins.

Las regresiones cubren paquete real, mapa desconocido, asset/hash/version/map mismatch, soporte ausente, solapamientos, bounds, direcciones y wrap, las dos posturas, los nueve probes, ausencia de vista/posición, no finitos, determinismo, invariancia ante futuro y mutación de campos ajenos, privacidad, abstención táctica y separación de salida.

## Invariantes y limpieza

El preflight de `2026-08-31T15:41:16.1250364Z` y el postflight de `2026-08-31T15:59:56.6891964Z` coincidieron en todos los estados protegidos: 100 demos, 3.013 ficheros bajo `analyzed_demos`, 727 en `failed`, staging vacío, 1.242 en `backend/data/exports`, 44 manifests aprobados y sus hashes. `Faceit-Demos` siguió limpio en `f132c8de4bcc97650352ec5a647fdf3985f11353`. `CURRENT` siguió apuntando a `stratai-2c6db440231d463a`, con el mismo hash, 9 ficheros y digest de contenido. La demo y los cuatro assets auditados conservaron sus SHA-256.

La [evidencia máquina](../../evidence/AI_COACH_MULTIMAP_COVER_MIRAGE_SINGLE_DEMO_SMOKE_2026-08-31.json) registra comandos, versiones, hashes, conteos, abstenciones, pruebas y pre/postflight. La salida grande del smoke y los directorios auxiliares de regeneración se eliminaron únicamente después de escribir esa evidencia.

## Qué no cambia y qué queda pendiente

Gate 1B sigue pendiente y `training_allowed=false`. Ningún mapa obtiene soporte de producto; la muestra Mirage no se extrapola al mapa completo. Faltan revisión humana atribuible, clasificación táctica, ampliación/revisión de cobertura, materialización canónica versionada y la futura migración 44/44 expresamente autorizada. `FLD-COVER` conserva su estado histórico `REQUIERE_CLASIFICACION`; ninguna capacidad gana elegibilidad.

La siguiente fase es 1B.9: congelar un contrato agnóstico al mapa de `FLD-EXIT-ROUTE` y construir sólo una referencia candidata parcial de Mirage, manteniendo separadas viabilidad física y valoración táctica humana. Su prompt se deja preparado, pero no se ejecuta en este cierre.
