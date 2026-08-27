# Acta de cierre de Block 7 — 21 de agosto de 2026

Veredicto: **cerrado**. No quedan P0/P1 heredados ni violaciones hard de Block
7 en el corpus local. No se inició ML, FACEIT ni Phase 2.

## Baseline auditado

Antes de editar se revisaron en modo read-only los contratos y las dos demos de
referencia de Blocks 1–6. No se detectó ningún P0/P1 heredado; los warnings
existentes eran gaps de observación ya explícitos y no justificaban detener el
bloque.

Los exports de referencia coincidían con sus backups de Block 6:

| Demo | SHA-256 demo | SHA-256 árbol canónico Block 6 |
| --- | --- | --- |
| Nuke `3GJOGmqKyZhYTtJrEmMvfNLsE` | `86058469da7bbf625e341a4391b743d341019c0e23e43ad0d60de0dfb1c29eaa` | `cdeae36cd9f02f13b647611647e234ebc99f14b77b3c960205653e4e4be7ed24` |
| Cache `7D4W7HC7RsQ7ZsCj3UVKmS5KQ` | `e5710c75deebde58e791572741774ad09c119b25bd18949d215faaf5c1446744` | `e4217e78a6dc2aa3993b42fc9ed761447687f7477cd646a49e36db864d616c5e` |

## Evidencia del corpus real

| Métrica | Resultado |
| --- | ---: |
| Demos con sidecar verificadas | 20/20 |
| Bundles validados individualmente | 20/20 |
| Bundles utilizables con warnings explícitos | 20/20 |
| Rondas | 411 |
| Archivos bajo `canonical/` | 1.222 |
| `decision_features` | 4.358 |
| `oracle_context` | 4.358 |
| `outcomes` | 4.358 |
| `quality_masks` | 4.358 |
| Violaciones hard Block 7 | 0 |
| Staging residual | 0 |
| Directorios rollback residuales | 0 |

Distribución de mapas: 5 Nuke, 1 Cache, 4 Dust2, 5 Inferno, 2 Ancient y
3 Anubis.

Todos los bundles terminan en `warning`, no en `fail`, porque preservan gaps de
cobertura reales. Resumen de dominios no perfectos:

| Dominio | Bundles warning | Cobertura min / media / max | Unavailable | Inferred |
| --- | ---: | ---: | ---: | ---: |
| `causal_availability` | 20 | 77,38% / 84,34% / 92,06% | 831 | 0 |
| `engagement_consistency` | 20 | 77,38% / 84,34% / 92,06% | 831 | 0 |
| `combat_consistency` | 20 | 82,05% / 89,53% / 92,31% | 20.321 | 0 |
| `economy_consistency` | 20 | 2,64% / 3,15% / 3,77% | 19.044 | 0 |
| `utility_consistency` | 20 | 35,82% / 39,95% / 44,66% | 12.533 | 1.209 |
| `lineage_completeness` | 20 | 88,24% / 92,10% / 93,75% | 26 | 0 |
| `objective_consistency` | 3 | 97,03% / 99,66% / 100% | 2.148 | 0 |

Las 2.148 observaciones objective no disponibles son discrepancias de flags
nativos sticky (`IsPlanting`/roles) reconciliadas contra el ledger objetivo.
No son eventos fabricados ni fallos de lifecycle. Los trece dominios restantes
tienen cobertura 100% y todos los dominios hard están en `pass`.

## Pilotos y determinismo

El protocolo final se ejecutó dos veces sobre Nuke y Cache con:

```text
--workers 1 --retries 2 --timeout 1500 --skip-aggregate-rebuild --match-id
```

Primera pasada: Nuke 41,0 s; Cache 56,9 s. Segunda pasada: Nuke 44,3 s;
Cache 72,0 s. Ambas procesaron 2/2 sin skip ni fallo.

Se compararon todos los SHA-256 bajo `canonical/` entre pasadas:

- Nuke: 60/60 archivos idénticos;
- Cache: 64/64 archivos idénticos;
- total: 124/124, 0 diferencias.

Los backups finales
`block7_reprocess_20260821T143534Z` y
`block7_reprocess_20260821T143742Z` se conservaron fuera de `exports`. Sus dos
manifests y 252 archivos copiados se revalidaron contra SHA-256: 0 errores.

## Lineage real de los pilotos

Nuke registra hashes observados de physics map, nav y callouts. Cache registra
physics map y nav; callouts permanece explícitamente no disponible. El nav de
Nuke figura como inspeccionado pero no usado por el fallback del parser. Build,
price table, algoritmos, schemas, metadata, warnings, abstenciones y corpus
quedan versionados en cada manifest.

## Golden corpus

`golden-corpus-v1` contiene 21 casos y su validador devolvió:

```json
{"errors": [], "status": "passed"}
```

El corpus declara `training_allowed=false`. Overtime está omitido de forma
explícita porque no hay una demo overtime confirmada entre las 20 disponibles;
no se inventó cobertura.

## Verificación pública

| Comando | Resultado |
| --- | --- |
| `npm run lint:python` | pass |
| `npm run test:python` | 43 passed |
| `npm run test:node` | 14 passed, 3 integration skips, 0 failed |
| `npm run test:go` | pass en todos los paquetes |
| `npm run test:frontend` | 23 files, 88 tests passed |
| `npm run build:frontend` | pass |
| `go vet ./...` | pass |
| Suite específica Block 7 | 133 passed |
| `npm run check:all` | pass |

El warning de Node sobre `--localstorage-file` y los tres skips de integración
son preexistentes/no bloqueantes; no hubo fallo de test.

## Revalidación ampliada de 10 demos

Como comprobación posterior se añadió al script el gate
`--expected-count 10` y se reprocesó una selección explícita de diez demos con
los mismos parámetros de seguridad. La muestra cubre Nuke, Cache, Dust2,
Inferno, Ancient y Anubis.

- 10 procesadas, 0 saltadas y 0 fallidas en 513,9 s;
- 206 rondas;
- 2.197 filas en cada una de las cuatro particiones causales;
- 0 violaciones hard y 10/10 receipts independientes en `passed`;
- 612/612 archivos canónicos con el mismo hash antes y después;
- backup `block7_reprocess_20260821T160745Z`: 622 archivos verificados, 0
  diferencias;
- staging vacío y 0 directorios rollback residuales.

Los warnings conservan la misma taxonomía de cobertura ya documentada y no
aparecieron nuevos tipos de inconsistencia.

## Incidencias cerradas durante el reprocesado

1. demoinfocs mantenía flags nativos de plantado de forma sticky. Los roles se
   reconcilian ahora desde el objective ledger SSOT y la discrepancia se mide.
2. El validador consideraba inválida una ronda con plant abortado en un site y
   plant final en otro. La consistencia se valida ahora por `attempt_id` y por
   terminal final.
3. Se endurecieron casos adversariales del validador, health JSON, sidecars,
   backup, retries HMAC y detención en primer fallo.

## Decisión

### Trazabilidad de las auditorías anteriores

Las auditorías iniciales de parser v3 y v6 se retiran como instrucciones
operativas porque quedaron superadas. Sus hallazgos conservan esta trazabilidad:

| Hallazgo inicial | Corrección consolidada | Contrato final |
| --- | --- | --- |
| Replay sin cierre, límites erróneos y flashes duplicadas | Finalización explícita, cronología y deduplicación verificadas | Manifest, quality report y validadores 3.8 |
| Participación, desconexiones, sustituciones e IDs no seguros | Roster reconciliado, IDs opacos como strings y joins validados | Player/combat/engagement ledgers canónicos |
| Daño aliado/propio y campos de combate ambiguos | Separación enemy/friendly/self, schemas y quality masks | `combat_event@2` y dominios de calidad |
| Compras, inventario y bonus por derrota no reconciliables | Valores nativos como SSOT, ledger explicativo y deltas explícitos | Contratos de economía y metadata |
| CT/T usados como identidad lógica | Team ID, side y cambios de mitad separados | Lineage y estados de ronda/equipo |
| Reglas hardcoded y manifest insuficiente | Reglas/versiones/procedencia y hashes incluidos | Manifest v3 y quality schema 12 |
| Raytracing interpretado como visibilidad completa | Geometría, disponibilidad y límites perceptivos explícitos | Observed/oracle/gaps separados; FOV/humo siguen siendo blockers |
| Rating y métricas sin lineage suficiente | Definiciones versionadas, inputs y abstención ante gaps | Métricas y quality report auditables |

Los criterios de salida están satisfechos: canonical-only, 20/20 válidos,
publicación atómica y compensable, causalidad separada, lineage verificable,
determinismo real y corpus golden disponible. Block 7 se cierra sin promover
ningún modelo ni comenzar Phase 2.
