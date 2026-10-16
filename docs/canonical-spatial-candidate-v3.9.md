# Contrato espacial canónico candidato 3.9

`3.9.0-candidate.2` corrige la frontera NAV de la materialización local iniciada en 1B.12, conforme a AUD-C01.
No sustituye el export `3.8.0`, el parser `v16`, sus manifests ni su lineage.
`training_allowed=false` y `product_supported=false` son obligatorios.

## Versiones y composición

| Componente | Contrato |
|---|---|
| Manifest candidato | `stratai.canonical_spatial_candidate_manifest@2` |
| Descriptor de artefacto | `stratai.spatial_candidate_artifact@1` |
| Loader | `stratai.spatial_candidate_loader@2` |
| Validator | `stratai.spatial_candidate_validator@2` |
| Puente de participantes | `stratai.spatial_participant_bridge@1` |
| Join espacial | `stratai.spatial_fact_join@1` |
| Ejemplo derivado | `stratai.ai_coach_decision_example@3`, contrato `3.0.0` |
| Elegibilidad | `stratai.ai_coach_task_eligibility@1`; conserva su semántica agregada por capacidad |

El manifest `spatial_candidate_manifest.json` contiene descriptores con schema,
versión, formato, compresión, conteos, bytes y SHA-256 de:

- el manifest base `match_<alias>/canonical/manifest.json`, íntegramente `3.8.0`;
- el puente opaco de participantes;
- los tres reportes factuales `@2` y sus JSONL gzip;
- `spatial_fact_joins.jsonl`;
- `spatial_decision_examples.jsonl`.

El consumidor verifica recursivamente los artefactos del manifest base. Cada
familia espacial declara paquete/hash/revisión/versión, muestreo y cobertura.
Los loaders independientes verifican los paquetes contra GLTF/bin/NAV locales.
Las rutas relativas no admiten escapes, symlinks ni reparse points. No se usa
descubrimiento implícito de candidatos ni se añade un descriptor a `3.8.0`.

## Join causal

La política `exact_source_round_participant_tick_no_carry_forward@1` exige igualdad
de partida, ronda, alias y `sample.t0_tick == decision.t0_tick`. También exige
`sample.availability_tick == sample.t0_tick`; ninguna disponibilidad anidada puede
exceder T0. Las filas fuente se ordenan estrictamente por ronda, tick y alias;
duplicados y desorden se rechazan. Los joins se ordenan por `decision_key`, campo,
y se recomputan al cargar: no se aceptan referencias ambiguas o modificadas.

El muestreo factual existente de 1 Hz sigue siendo de 1 Hz. Una muestra anterior
no acredita el estado actual; una futura tampoco. La ausencia de igualdad se
registra como `missing:no_exact_t0_sample`, nunca como una ausencia geométrica.
Una muestra exacta fuera de cobertura o con NAV no disponible conserva su causa
original. Para NAV, los joins describen presencia del diagnóstico: `false` indica
ausencia de camino en el subgrafo retenido, y `true` conexión en ese mismo grafo.
Ninguno acredita conexión física ni ausencia física. No se cambian los valores,
IDs, muestreo o asociaciones heredadas para aparentar una acreditación adicional.

El puente usa los mismos alias `participant-NNN` del productor, ordenados por
identidad reconciliada en memoria. La clave de actor es
`SHA256(source_sha256 + LF + canonical_actor_id)`. Su correspondencia se verifica
contra los participantes del bundle base. Las identidades originales sólo se
leen localmente para el join y nunca se retienen en el candidato. Los IDs de
decisión y estado del ejemplo también se sustituyen por hashes locales al match.

Además, el validador reconcilia posiciones, yaw float32 y postura con el frame
original del replay, limitado a los campos propios de ese participante en T0.
No toma eventos, ganador, outcomes ni posiciones enemigas como features. Los
campos de movimiento que el contrato de replay antiguo no serializa conservan
la procedencia y validación del sidecar; no se inventan desde el replay.

## Features, evidencia y elegibilidad

`DecisionExample@3.observed.spatial_facts` conserva los tres IDs estables. Los dos
campos geométricos mantienen sus valores y reglas. `FLD-NAV-EXIT-PATH` tiene siempre
`status=unavailable` y `values={}`; su evidencia física es `null` con muestra exacta
o `missing` sin ella, con la razón `nav_physical_connection_not_accredited`.

`spatial_evidence.FLD-NAV-EXIT-PATH.diagnostic` conserva `scope=partial_nav_graph`,
el estado y las razones originales del join, y los valores originales de sample,
candidates y posture_context, incluidos sus IDs locales. Sólo se excluye
`tactical_assessment`; la referencia a la fila original, tick, paquete, unidades y
procedencia se mantienen. Los nombres heredados `objective_viability` y
`physical_cost` dentro del diagnóstico son nombres de serialización, no garantías
de viabilidad o coste físico. Sin join, los valores diagnósticos quedan vacíos;
con fuente no disponible se conserva la ausencia original, sin sustituirla por
false/0. Se validan también los límites temporales dentro del diagnóstico.

Los outcomes y oracle continúan en sus particiones originales. Los ocho schemas
humanos v1/v2 siguen rechazados por la entrada de ground truth.

La auditoría devuelve por familia y campo `present`, `missing`, `null`,
`unavailable`, razones físicas, razones de join, joins fallidos y violaciones
temporales. Un candidato corrupto falla antes de acreditar evidencia. Los joins
sin muestra son abstenciones contabilizadas, no errores ocultos. La cobertura que
consume `TaskEligibility` procede de los ejemplos unidos; los sidecars sin join
no acreditan campos. NAV no aporta cobertura física ni siquiera con diagnóstico
`present`: `spatial_by_family` conserva sus conteos con alcance explícito de grafo,
mientras `field_coverage` permanece null/missing. No se habilita ninguna capacidad.
La elegibilidad sigue siendo agregada, no una etiqueta por
fila ni una autorización de entrenamiento. Los tiempos NAV son escenarios
internos; se mantienen sus límites de presentación cualitativa y condicional.

## Ejecución temporal y validación

Después de las pruebas automatizadas se ejecuta una sola demo aprobada. El
exportador rechaza un destino no temporal, no vacío o existente antes de parsear.
El modo combinado genera los tres productores desde el mismo replay:

```powershell
$runRoot = Join-Path ([IO.Path]::GetTempPath()) "stratai-spatial-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $runRoot -ErrorAction Stop | Out-Null
Push-Location backend/go-service
try {
  go run ./cmd/canonical-export --demo <demo-aprobada> --output $runRoot `
    --alias <alias-opaco> --checksum <sha256> --maps-root ../data/maps `
    --exposure-packages-root ../../ai_coach/spatial/exposure `
    --cover-packages-root ../../ai_coach/spatial/cover `
    --exit-route-packages-root ../../ai_coach/spatial/exit_route `
    --build-id <working-tree-id> --ai-coach-spatial-candidate
} finally { Pop-Location }

$candidateRoot = Join-Path $runRoot 'smoke_run_<alias-opaco>'
backend/venv/Scripts/python.exe -m ai_coach.datasets.spatial_candidate materialize `
  $candidateRoot --alias <alias-opaco> --checksum <sha256> --map-id de_mirage `
  --maps-root backend/data/maps
backend/venv/Scripts/python.exe -m ai_coach.datasets.spatial_candidate validate `
  $candidateRoot --alias <alias-opaco> --checksum <sha256> --map-id de_mirage `
  --maps-root backend/data/maps
```

La materialización rechaza sobrescrituras, valida los sidecars con recomputación,
escribe los joins y ejemplos y sólo entonces escribe el manifest. La validación
posterior recompone los ejemplos y comprueba igualdad con el artefacto persistido.
También se exige el validador semántico canónico del bundle base.

## Compatibilidad y rollback ejecutables

`load_direct_bundle_manifest` mantiene su aceptación exclusiva de manifest `@3`,
export `3.8.0` y parser `v16`. `load_spatial_candidate` acepta exclusivamente el
manifest candidato y sus versiones. El consumidor debe elegir explícitamente
la entrada: nunca hay detección por la mera presencia de archivos adyacentes.

Los schemas anteriores de manifest `@1` y `DecisionExample@2` se conservan
íntegros para identificar evidencia histórica. El lector actual exige manifest
`@2`, contrato `2.0.0`, candidato `.2` y ejemplos `@3`; rechaza un manifiesto antiguo
en vez de reinterpretarlo silenciosamente. No se migran ni sobrescriben salidas.

Para rollback, el consumidor vuelve a la entrada `load_direct_bundle_manifest`
del mismo bundle base. `process_manifest(base, ...)` vuelve a informar los tres
campos como `missing` aunque los sidecars continúen en el directorio vecino. Las
pruebas verifican esta lectura y la identidad de todos los bytes del bundle base.
No hay que mutar `CURRENT`, borrar datos ni revertir una release.

Este candidato no cierra Gate 1B, no migra las 44 partidas y no es una release.
Su estado de ejecución y la siguiente acción sólo se deciden en el plan canónico.
