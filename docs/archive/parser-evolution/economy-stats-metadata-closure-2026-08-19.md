# Cierre del Bloque 6 — Economía, estadísticas y metadatos

Estado: **cerrado el 2026-08-19**. No se ha iniciado el Bloque 7.

## Prerrequisitos y alcance

La auditoría read-only inicial confirmó que los Bloques 1–5 estaban cerrados, que
sus contratos seguían pasando sobre Nuke y Cache y que sus exports coincidían con
los backups de cierre. No se encontró un P0/P1 heredado, un hash no determinista ni
una inconsistencia previa que impidiera trabajar en el Bloque 6.

El contrato se congeló antes de cambiar la implementación en
[`economy-stats-metadata-contract-v1.md`](economy-stats-metadata-contract-v1.md).
No se reconstruyeron agregados ni se modificaron contratos de los Bloques 1–5.

## Fallos demostrados y causas raíz

| Fallo real | Evidencia previa | Causa raíz | Resolución |
| --- | --- | --- | --- |
| Identidad económica ligada al lado | Algunas filas de ronda usaban entity IDs `2/3`; el cambio de lado podía cambiar la identidad | El exporter trataba la entidad CT/T del momento como equipo | `team_a/team_b` se resuelven desde participantes y `side_assignments`; `side` queda limitado a la ronda |
| Transición de dinero incompleta | `next_round_min_money` discrepaba en 141/180 transiciones de Nuke y 172/200 de Cache | Se calculaba sólo como dinero tras compra + loss bonus | Dinero observado, esperado y calculado están separados; la transición incompleta queda `not_evaluable` |
| Propietarios fabricados | `original_owner` contenía `purchased` en 1.629/1.987 observaciones | El tipo de adquisición se almacenaba en un campo de identidad | Compras y propietarios usan `steam:<steam64>`; ausencia queda `null` con estado explícito |
| Precio cero ambiguo y purchases sin lineage suficiente | C4 y precios ausentes terminaban mezclados con cero; varias compras carecían de entity ID | Tabla embebida sin versión y ausencia representada por el valor numérico | Precio `{amount,status,table_version}`, `known_zero` separado de `unknown`, tabla y checksum versionados |
| Pickup/drop potencialmente mal atribuido | Existía fallback por nombre de arma/tiempo y filtro de precio | La correlación no exigía identidad exacta del objeto | Sólo se correlaciona el mismo entity ID, arma y Steam ID; si falta evidencia no se fabrica exchange |
| Native sobrescrito por calculated | El equipo de freeze/fin de ronda mezclaba scoreboard con suma de inventario | Un único campo servía para dos fuentes | Los valores nativos y calculados tienen campos, estados y deltas distintos |
| Clutch perdido o sobrescrito | Un único puntero CT→T podía reemplazar el primer clutch; Cache omitía el 1v1 T ganado de la ronda 11 | Estado mutable compartido por lado y victoria inferida por supervivencia | Ledger atómico por jugador/equipo/ronda, dos attempts simultáneos permitidos y resultado confirmado por winner de ronda |
| Estadísticas contaminables | Algunos callbacks podían entrar fuera de ronda oficial | Los acumuladores no exigían frontera competitiva en todos los caminos | Los derivados se alimentan sólo de hechos atómicos dentro de rondas competitivas; warmup y callbacks descartados se contabilizan |
| Metadatos y rating sin lineage completo | Faltaban hashes de configuración y versión/fórmula explícita del rating | El manifest sólo describía el parser de forma global | `match_metadata@1` publica fuente, endpoint/demo, checksum, versiones, configuración y algoritmos; `played_at` no se infiere |
| Validador insuficiente | El validador estructural no reconstruía economía, clutch o metadata | No existían invariantes post-export del Bloque 6 | El validador Python y el auditor independiente reconstruyen identidades, referencias, transiciones y attempts |

## Contratos y versiones cerrados

- Export: `stratai.canonical_export@3.6.0`; manifest
  `stratai.canonical_manifest@2`; parser `v15`; quality schema `11`.
- Artefactos nuevos: `stratai.economy_round@1`,
  `stratai.economy_player@1`, `stratai.player_stats@1`,
  `stratai.clutch_event@1` y `stratai.match_metadata@1`.
- Tabla de precios `stratai.cs2_prices@1`, revisión efectiva
  `2026-08-19`, checksum
  `643ec4ff6e87351673d13aa8be784e4f371382ebae89dfab97fe0fe8e07fae0b`.
- Reglas `stratai.cs2_economy_rules@1`, checksum
  `cc9ea39fc3ede0430573bb71ffebaf1e3429e5dd42fa4a7d5ab549725ececc43`.
- Rating aproximado `stratai.rating_hltv2_approx@1`; clutch
  `stratai.clutch_ledger@1`; transformaciones de economía, estadísticas y
  metadata `@1`.

La regla de loss bonus documenta la fuente de Valve y los niveles
1.400/1.900/2.400/2.900/3.400. Los demás premios sólo exponen un importe cuando la
demo o una regla aplicable lo permite; el evento fuente se conserva aunque el
importe quede `unavailable`.

## Resultado económico y estadístico

### Nuke — `3GJOGmqKyZhYTtJrEmMvfNLsE`

- 40 filas equipo-ronda y 200 filas jugador-ronda; ambos equipos mantienen el
  mismo `team_id` al jugar CT y T.
- 686 compras, 26 pickups y 23 refunds observados. No se fabricaron drops o
  exchanges sin una pareja exacta de entity IDs.
- 2.457 precios `known` y 47 `known_zero`; ningún precio desconocido se convirtió
  silenciosamente en cero.
- 38 diferencias native/calculated de inventario/equipment conservadas con
  diagnóstico, sin sobrescritura.
- Premios enlazados: 135 kills, 7 plants, 20 victorias y 20 loss bonuses. Los
  importes de kill/plant/victoria quedan `unavailable`; los loss bonuses quedan
  `calculated` y versionados.
- 10 scoreboards nativos reconciliados y 10 ratings marcados como aproximados.

### Cache — `7D4W7HC7RsQ7ZsCj3UVKmS5KQ`

- 44 filas equipo-ronda y 220 filas jugador-ronda; ambos equipos mantienen el
  mismo `team_id` al jugar CT y T.
- 866 compras, 32 pickups y 20 refunds observados; no se infirieron drops o
  exchanges sin evidencia exacta.
- 2.794 precios `known` y 56 `known_zero`.
- 43 diferencias native/calculated preservadas.
- Premios enlazados: 160 kills, 13 plants, 2 explosions, 3 defuses, 22 victorias
  y 22 loss bonuses; los importes no expuestos permanecen `unavailable`.
- 10 scoreboards nativos reconciliados y 10 ratings aproximados.

En ambas demos, `next_round_calculated` queda `not_evaluable` para todas las filas
porque la transición completa no se puede reconstruir con certeza. El dinero
observado de la siguiente ronda se conserva para 190/200 y 210/220 filas; las diez
filas finales de cada demo quedan `not_observed`.

## Clutch

| Demo | Attempts | Won | Lost | 1v1 | 1v2 | 1v3 | 1v4 | 1v5 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Nuke | 21 | 1 | 20 | 1 | 3 | 8 | 4 | 5 |
| Cache | 24 | 2 | 22 | 2 | 9 | 5 | 6 | 2 |

Cada fila conserva jugador, ronda, enemigos al inicio, estado 1vX,
`attempt=true`, resultado y eventos fuente. El auditor reconstruye los attempts
desde kills atómicas y obtuvo cero discrepancias. Los tests cubren 1v1, 1v2, 1v3,
1v5, ganado, perdido, no evaluable y dos equipos con clutch simultáneo.

## Metadatos y disponibilidad

- `played_at=null` y `played_at_status=unavailable` en las dos demos: no existe
  una fecha de origen fiable y no se reutiliza el timestamp de procesamiento.
- Checksums de demo: Nuke
  `86058469da7bbf625e341a4391b743d341019c0e23e43ad0d60de0dfb1c29eaa`;
  Cache `e5710c75deebde58e791572741774ad09c119b25bd18949d215faaf5c1446744`.
- `queried_at` también queda nulo al no existir una consulta remota. Endpoint,
  parser, versiones de transformación y cinco hashes de configuración están
  presentes en metadata y manifest.

Warnings conservados:

- `economy_observation_coverage=warning` por las transiciones no evaluables y la
  última ronda sin observación posterior; no es un gate crítico.
- No puede verificarse la aplicabilidad temporal de la tabla de precios mientras
  `played_at` sea nulo.
- Los importes de victoria, plant, explode, defuse y kills no expuestos por estas
  demos quedan `unavailable`, con sus eventos fuente.
- Las dos demos no ofrecen evidencia exacta suficiente para publicar un drop o
  exchange; los casos positivos y negativos están cubiertos por tests.

## Quality gates

En las dos demos pasan y valen cero:

- `economy_team_identity`
- `economy_native_calculated_reconciliation`
- `economy_money_transition`
- `economy_purchase_provenance`
- `economy_price_table_version`
- `stats_scoreboard_reconciliation`
- `stats_utility_reconciliation`
- `clutch_attempt_reconciliation`
- `warmup_contamination`
- `metadata_provenance`
- `metadata_checksum_lineage`
- `economy_determinism`
- `stats_determinism`

`economy_observation_coverage` es el único gate en `warning`; representa ausencia
de observación y no una inconsistencia. Las auditorías independientes de combate y
engagements también siguen en `pass` después del reprocesado.

## Pruebas y validación

- `gofmt -l .`: limpio.
- `go test ./... -count=1`: pass en todos los paquetes.
- `go vet ./...`: pass.
- Python de validator/reprocessor/auditor: 119 pass.
- Backend completo: 43 pass; subset de proyecciones afectadas: 12 pass.
- Ruff crítico del backend y Ruff estricto sobre archivos editados: pass.
- Frontend: 23 archivos/88 tests pass; `vite build` pass.
- Node: 14 pass, 3 skips previstos, 0 fallos.
- Validador canónico oficial: `OK` para Nuke y Cache.
- Auditores independientes de combate, engagements y Bloque 6: `status=pass` en
  ambas demos.

Los tests del Bloque 6 incluyen cambio de lado, identidad estable, native frente a
calculated, precio desconocido y cero real, compra sin precio, pickup sin
propietario, drop/pickup, refund/exchange, loss bonus, win/loss, objetivos,
warmup/freeze, clutch 1vX, actor ledger-only, scoreboard ausente, fecha ausente,
rating aproximado, orden inverso de eventos/mapas, IDs y determinismo.

## Reprocesado y determinismo

Cada ejecución usó exactamente `--workers 1 --timeout 1500
--skip-aggregate-rebuild --match-id <id>` y procesó una única demo.

| Demo | Pass 1 | Pass 2 | JSON canónicos | Diferencias SHA-256 | SHA-256 árbol canónico |
| --- | ---: | ---: | ---: | ---: | --- |
| Nuke | 34,3 s | 34,2 s | 56/56 idénticos | 0 | `cdeae36cd9f02f13b647611647e234ebc99f14b77b3c960205653e4e4be7ed24` |
| Cache | 47,7 s | 47,1 s | 60/60 idénticos | 0 | `e4217e78a6dc2aa3993b42fc9ed761447687f7477cd646a49e36db864d616c5e` |

Los timestamps operativos de publicación quedan fuera del árbol canónico; todos
los JSON del contrato canónico se compararon por ruta y SHA-256.

## Backups y rollback

- Backup previo completo y verificado:
  `backend/data/export_backups/block6_pre_v15_20260819/` — 54 archivos Nuke y 58
  Cache, sin diferencias al copiar.
- Snapshots del primer pase determinista:
  `backend/data/export_backups/block6_determinism_20260819/pass1/` — 56 archivos
  Nuke y 60 Cache, verificados contra su origen.
- No se borró ningún backup ni se reconstruyeron agregados. Los exports canónicos
  vigentes son los segundos pases validados.

## Cierre

No quedan P0/P1 conocidos del Bloque 6. Native y calculated permanecen separados,
la identidad es estable, los campos no observables son explícitos, los attempts de
clutch están reconciliados, no existe contaminación de warmup y los dos exports
son deterministas.

El siguiente trabajo recomendado es el **Bloque 7 — Quality gates y cierre
causal**, sin iniciarlo como parte de este cierre.
