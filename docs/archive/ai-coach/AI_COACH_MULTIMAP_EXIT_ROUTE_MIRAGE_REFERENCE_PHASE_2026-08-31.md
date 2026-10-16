# Cierre 1B.9 — contrato multimapa de rutas de salida y referencia parcial Mirage

Fecha: 31 de agosto de 2026  
Estado: **cerrada en código**  
Gate 1B: **pendiente**  
Entrenamiento: **no autorizado**

## Resultado

La fase congela una infraestructura de `FLD-EXIT-ROUTE` independiente del mapa y una única referencia candidata parcial para `de_mirage`. El contrato distingue tres afirmaciones que no pueden confundirse:

1. conectividad NAV y coste físico entre un origen observado en T0 y un destino explícito;
2. escenarios temporales exactos sólo internos, como restricciones condicionales y no como predicción de llegada;
3. calidad, seguridad o recomendación táctica de la salida, que requiere revisión humana y continúa pendiente.

Se añadieron los schemas Draft 2020-12 `stratai.exit_route_map_package@1`, `stratai.exit_route_support_manifest@1` y `stratai.observed_exit_route_diagnostic@1`; loaders fail-closed equivalentes en Go y Python; un generador determinista; un productor causal full-demo; un modo CLI aislado; y un validador independiente. La resolución exige `map_id`, contrato, versión, hash del paquete y hashes de assets exactos. Mapa desconocido, paquete/asset ausente o alterado, versión incompatible, mismatch de mapa o referencia inválida produce `unavailable`, nunca un fallback a Mirage.

El código común no contiene nombres de lugares ni ramas de Mirage. El support manifest sólo registra un candidato parcial y no declara mapas soportados como producto.

## Semántica física y límites

Cada candidato dirigido conserva origen de muestra, destino explícito, asociación de endpoints a polígonos NAV, conectividad dirigida, distancia euclídea, camino Dijkstra y procedencia. Reutiliza `stratai.observed_navigation_path@2`, `nav_area_connections_with_polygons@2`, asociación por polígono contenedor antes del centro 3D y desempates estables. El destino no se infiere de un callout, de proximidad, cover, exposure ni clearance.

Los escenarios temporales conservan la política ya congelada: referencia optimista continua, máximo continuo del arma causal disponible y proyección del ritmo horizontal observado. Son soporte interno de decisión, no reloj conocido por el jugador, no ETA mostrable, no predicción de llegada y no recomendación. `exact_user_facing_time_allowed=false` y `arrival_prediction=false`.

NAV acredita polígonos, conexiones dirigidas y coste de camino. El mesh físico sólo fue auditado como colisión; `de_mirage_places.json` aportó coordenadas semilla, no etiquetas normativas. Los paquetes previos de exposure y cover permanecen geometría separada y no se consumen como verdad de salida. Ninguno de esos assets demuestra que una ruta sea segura, prudente o tácticamente buena.

## Referencia parcial de `de_mirage`

El paquete `1.0.0-candidate.1` contiene:

- 8 muestras parciales y 8 candidatos dirigidos;
- 8/8 candidatos estáticamente alcanzables en el NAV protegido;
- distancias de camino entre 1.047,440386 y 8.086,618578 unidades Source 2;
- 8 placeholders tácticos separados, todos `pending_review`, sin autor, revisor, evidencia ni clasificación;
- `full_map_coverage=false`, `product_supported=false` y `training_use_authorized=false`.

Dos regeneraciones independientes coincidieron byte a byte entre sí y con el paquete almacenado. El paquete ocupa 19.945 bytes y tiene SHA-256 `6007fb5680eb331b2433e3a0377e613a1e6a219f96c4ed3fbf37260ee8440200`; su revisión geométrica es `f02deca7d5d4a2f4f2aee448f976831aa12a1465e39410e8fff1046a455f8ca7`. El support manifest ocupa 604 bytes y tiene SHA-256 `dc180bf6e2840a300adda02a559f4a01c79a8e819ea182d8ef3d2ca664bf0650`.

## Productor y semántica causal

`BuildObservedExitRouteFullDemoReport` muestrea jugadores vivos a 1 Hz. En cada T0 conserva alias opaco, posición, contexto causal disponible y la selección espacial única. Sólo dentro de una muestra declarada evalúa su candidato dirigido desde la posición real observada hasta el destino congelado; fuera de cobertura se abstiene.

La viabilidad objetiva conserva reachability, distancia física y escenarios temporales disponibles. La evaluación táctica vive en otra partición y se abstiene siempre mientras falte revisión humana. No se leen futuro, oracle, outcomes, identidades públicas ni estado enemigo; no se emiten etiquetas humanas, seguridad ni recomendaciones.

El sidecar candidato queda fuera del manifest canónico `3.8.0`; no altera replay, tactical, `enemy_los`, los 44 bundles ni la elegibilidad histórica.

## Smoke real obligatorio

Se procesó exactamente una demo aprobada. Los tres intentos usaron el mismo match, SHA-256 y una salida temporal nueva; nunca se seleccionó otra demo:

- alias: `demo-a11eb0d600ee702c9edb`;
- SHA-256: `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`;
- mapa: `de_mirage`;
- parser/export: `v16` / `3.8.0`;
- build válido: `working-tree-exit-route-mirage-1b9-20260831-r3`;
- duración del productor válido: 83,732 s.

R1 produjo el sidecar, pero el validador rechazó `reason_codes:null` en el resumen de paquete disponible. Tras convertirlo en una lista vacía explícita y añadir regresión, R2 reveló el mismo defecto en filas objetivas alcanzables. La corrección y regresión equivalentes dejaron R3 válido. Este historial demuestra que el gate independiente invalidó resultados estructuralmente ambiguos aunque el productor terminase con código cero.

Comando exacto válido, ejecutado desde `backend/go-service`:

```powershell
go run ./cmd/canonical-export --demo 'E:\Carpeta compartida\Proyecto IA\Faceit-Demos\demos\1-3f9e5224-791f-4a15-ac14-5acaacec9407.dem' --output 'C:\Users\kerch\AppData\Local\Temp\stratai-one-demo-exit-route-1b9-20260831-r3' --alias 'demo-a11eb0d600ee702c9edb' --checksum 'a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262' --maps-root 'E:\Carpeta compartida\Proyecto IA\StratAI\backend\data\maps' --exit-route-packages-root 'E:\Carpeta compartida\Proyecto IA\StratAI\ai_coach\spatial\exit_route' --build-id 'working-tree-exit-route-mirage-1b9-20260831-r3' --ai-coach-exit-route-validation
```

Resultado real de R3:

- 24 rondas, 29.520 frames fuente, 10 aliases y 12.802 filas a 1 Hz;
- 883 posiciones dentro de las ocho muestras y 11.919 ausencias `outside_declared_sample_coverage`;
- 883 evaluaciones de candidato: 878 alcanzables, 0 inalcanzables y 5 indisponibles por origen fuera de tolerancia NAV;
- 878 caminos físicos disponibles y 5 ausentes;
- 828 escenarios de arma activa disponibles y 55 ausentes;
- 750 escenarios de ritmo observado disponibles y 133 ausentes;
- las ocho muestras tuvieron datos: 55, 22, 63, 105, 193, 61, 208 y 176 filas;
- 0 clasificaciones/recomendaciones tácticas y 883 abstenciones `human_review_pending`.

El validador independiente tardó 6,745 s y recomputó schemas, package/assets, hashes, muestra y destino, distancia directa, álgebra temporal, conteos, privacidad y abstención. Obtuvo cero mismatches, campos futuros/oracle/outcome, identidades crudas o etiquetas humanas. El stream gzip ocupa 644.215 bytes y tiene SHA-256 `f2a9a03a8463d458e49834f80794d4915526b9beaa0524c82dc5d69d22ba4ba8`; su contenido lógico ocupa 22.387.194 bytes y tiene SHA-256 `bca59cb6e87492b8904411c4c6612ee7e556068ecc16b5210d8f5e8f202f274c`.

La validación semántica del bundle temporal tardó 280,493 s y devolvió cero errores. La auditoría read-only adicional tardó 7,314 s: 918 `DecisionExample`, 918 joins correctos y cero fallidos. El bundle conservó 73 ficheros, 72 artefactos declarados, contrato `3.8.0` y parser `v16`.

Antes de parsear se comprobaron los cierres negativos: salida no temporal, destino no vacío, intento de sobrescritura y combinación de modos fueron rechazados sin consumir la demo.

## Pruebas superadas

- `go test -count=1 ./pkg/exitroute ./parser ./cmd/canonical-export`: verde;
- `go vet ./pkg/exitroute ./parser ./cmd/canonical-export ./cmd/exit-route-package-generator`: verde;
- `python -m pytest ai_coach/tests -q`: 181 pruebas pasadas;
- `python -m ruff check ai_coach --select E9,F63,F7,F82`: sin errores;
- `python -m ai_coach.contracts.validate_contract`: 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 17 schemas runtime y 18 reason codes;
- `npm run check:all`: 88 pruebas frontend, Node 14 pasadas/3 omitidas, todos los paquetes Go, 43 pruebas Python backend y build frontend correcto;
- doble regeneración del paquete: ambos outputs y el almacenado con el mismo SHA-256;
- validador independiente de salida: aprobado en R3;
- bundle semántico y auditoría directa: 0 errores, 918/918 joins.

Las regresiones cubren paquete real, mapa desconocido, assets/hash/version/map mismatch, soporte ausente, cobertura parcial, bounds/solapamientos/referencias, regeneración determinista, conectividad dirigida, destinos, endpoint association, timing causal, política sin segundos user-facing, ausencia de arma/ritmo, no finitos, futuro/oracle, privacidad, abstención humana y guardas preparse.

## Invariantes y limpieza

El preflight de `2026-08-31T17:36:27.2966941Z` y el postflight de `2026-08-31T17:53:07.9273361Z` coincidieron en todos los estados protegidos: 100 demos, 3.013 ficheros bajo `analyzed_demos`, 727 en `failed`, staging vacío, 1.242 en `backend/data/exports`, los 44 manifests aprobados y sus hashes. `Faceit-Demos` siguió limpio en `f132c8de4bcc97650352ec5a647fdf3985f11353`. `CURRENT` siguió apuntando a `stratai-2c6db440231d463a`, con el mismo hash, 9 ficheros y digest de contenido. La demo y los assets auditados conservaron sus SHA-256.

La [evidencia máquina](../../evidence/AI_COACH_MULTIMAP_EXIT_ROUTE_MIRAGE_SINGLE_DEMO_SMOKE_2026-08-31.json) conserva comandos, versiones, hashes, conteos, abstenciones, intentos y pre/postflight. Después de escribirla se eliminaron las tres salidas del smoke y los cuatro directorios auxiliares: 228 ficheros y 2.207.839.838 bytes no recuperables. La verificación posterior confirmó que los siete roots y cualquier temporal coincidente de la fase están ausentes.

## Qué no cambia y qué queda pendiente

Gate 1B sigue pendiente y `training_allowed=false`. Ningún mapa obtiene soporte de producto; la referencia Mirage no cubre el mapa completo. Faltan revisión humana atribuible y ciega, adjudicación, clasificación táctica de exposure/cover/salidas, ampliación de cobertura, materialización canónica versionada y una futura migración 44/44 expresamente autorizada. `FLD-EXIT-ROUTE` conserva su estado histórico `REQUIERE_CLASIFICACION`; ninguna capacidad gana elegibilidad.

La siguiente fase es 1B.10: congelar el protocolo humano de revisión y adjudicación espacial y preparar un lote piloto Mirage exclusivamente para revisión de exposure, cover y rutas de salida. No puede fabricar etiquetas, modificar datos canónicos ni ejecutar todavía la revisión humana. Su prompt queda preparado, pero no se ejecuta en este cierre.
