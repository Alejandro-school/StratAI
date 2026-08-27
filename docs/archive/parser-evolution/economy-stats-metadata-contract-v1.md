# Contrato canónico del Bloque 6 — Economía, estadísticas y metadatos

Estado: congelado el 2026-08-19 antes de modificar la implementación.

Este contrato añade cinco artefactos a `stratai.canonical_export@3.6.0`. No cambia
los contratos cerrados de los Bloques 1–5.

## Identidad y disponibilidad

- `team_id` es `team_a` o `team_b` durante toda la partida. `CT` y `T` solo son
  valores de `side` y se resuelven desde `rounds.side_assignments`.
- Toda persona se referencia mediante `player_id` (`steam:<steam64>`). Los nombres
  son presentación y nunca claves de unión.
- Los valores no observables son `null`; cada grupo medible declara `status` con
  uno de `observed`, `calculated`, `unavailable`, `not_observed` o
  `not_evaluable`. Un cero real lleva `status=observed|calculated`, nunca
  `unavailable`.
- Los arrays y mapas canónicos se ordenan antes de serializar. Los IDs se derivan
  de claves estables, no del orden de callbacks o de iteración de mapas.

## `stratai.economy_round@1`

Archivo: `canonical/derived/economy_rounds.json`.

Contiene una fila por ronda y equipo estable:

- `round_id`, `round_number`, `team_id`, `side`, `outcome`, `win_reason`.
- `loss_bonus`: nivel, cantidad calculada, versión de reglas y estado.
- `money_start`, `money_freeze_end` y `money_round_end`: sumas observadas por
  equipo. Los cálculos permanecen en campos separados en el contrato por jugador.
- `rewards`: victoria, derrota, plant, explosión, defuse y kills. Cada entrada
  conserva `observed_amount`, `calculated_amount`, `status` y `source_event_ids`.
- `diagnostics` contiene reconciliaciones sin sustituir la fuente nativa.

No se exporta un identificador de entidad `2/3` como `team_id`.

## `stratai.economy_player@1`

Archivo: `canonical/derived/economy_players.json`.

Contiene una fila por ronda y jugador:

- Identidad: `round_id`, `round_number`, `player_id`, `team_id`, `side`.
- Dinero separado: `round_start_observed`, `freeze_end_observed`,
  `after_buy_observed`, `after_buy_calculated`, `round_end_observed`,
  `next_round_observed` y `next_round_calculated`.
- Inventarios nativos y calculados separados en inicio, freeze y fin de ronda.
  Si el inventario no se observó, `status=not_observed`, ambos valores son `null`
  y `items` queda vacío; no se representa como inventario de valor cero.
- `transactions`: `purchase`, `pickup`, `drop`, `refund` y `exchange`, con actor,
  propietario original por `player_id`, entidad, estado de observación y eventos
  fuente.
- Cada precio es `{amount, status, table_version}`. `amount=null` representa precio
  desconocido; `amount=0,status=known_zero` un precio cero real; una compra no
  observada o un inventario no observable usan estados distintos.
- `purchased_item` y `observed_item` son campos distintos.

`next_round_calculated` solo se publica cuando la transición completa es evaluable;
nunca se deriva únicamente de dinero tras compra más loss bonus.

## `stratai.player_stats@1`

Archivo: `canonical/derived/player_stats.json`.

Contiene una fila por jugador con `player_id`, `team_id` y:

- `native_scoreboard`: estadísticas públicas nativas y `status`.
- `derived`: K/D/A, daño de combate, assists, utilidad, KAST, openings y trades
  derivados de hechos atómicos competitivos.
- `reconciliation`: deltas entre nativo y observado sin sobrescritura.
- `clutch`: attempts, wins, losses, not-evaluable y distribución 1vX.
- `rating`: valor aproximado, `approximate=true`, fórmula, algoritmo y versión.
- `provenance`: artefactos/eventos fuente, algoritmo y disponibilidad.

El daño de utilidad nativo y el observado permanecen separados. Warmup, callbacks
fuera de ronda y freeze inválido no alimentan estadísticas derivadas.

## `stratai.clutch_event@1`

Archivo: `canonical/derived/clutch_events.json`.

Cada attempt contiene `clutch_id`, jugador, equipo estable, lado, ronda,
`enemies_at_start`, `state` (`1v1`…`1v5`), `attempt=true`, `result`
(`won`, `lost`, `not_evaluable`), tick de inicio, evento disparador y
`source_event_ids`. Se permite un attempt por equipo en la misma ronda. Una victoria
requiere que `rounds.winner_team_id` confirme el equipo; sobrevivir no basta.

## `stratai.match_metadata@1`

Archivo: `canonical/core/match_metadata.json`.

Incluye:

- `played_at` y `origin_date` anulables con fuente y estado; no admite timestamps
  de procesamiento como fecha jugada.
- Fuente (`demo`/endpoint), fecha de consulta anulable, versión y checksum SHA-256.
- Versiones de parser, export, reglas económicas, tabla de precios y algoritmos.
- Hashes SHA-256 de configuración y transformaciones.
- Disponibilidad, warnings y limitaciones.

El manifiesto canónico replica checksum, versiones y hashes de configuración para
cerrar el linaje. Los campos operativos no deterministas quedan fuera del árbol
canónico.

## Versiones congeladas

- Export canónico: `3.6.0`.
- Parser schema: `v15`.
- Quality schema: `11`.
- Tabla de precios: `stratai.cs2_prices@1`, vigencia de la revisión
  `2026-08-19`; si `played_at` es nulo su aplicabilidad se marca
  `unverified_match_date`, y si es anterior a la vigencia se marca
  `unverified_outside_effective_range`. Su checksum se calcula sobre la versión,
  vigencia y todas las parejas item/precio ordenadas.
- Reglas económicas: `stratai.cs2_economy_rules@1`. El loss bonus usa la regla
  publicada por Valve en 2019 (1400, 1900, 2400, 2900 y 3400; nivel inicial de
  mitad 1). Otros premios quedan `unavailable` si la demo no expone una cantidad
  nativa o la regla aplicable no puede determinarse.
- Rating: `stratai.rating_hltv2_approx@1`; aproximación no oficial.
- Clutch: `stratai.clutch_ledger@1`.

## Quality gates obligatorios

El informe schema 11 contiene exactamente los gates de Bloque 6:

`economy_team_identity`, `economy_native_calculated_reconciliation`,
`economy_money_transition`, `economy_purchase_provenance`,
`economy_price_table_version`, `stats_scoreboard_reconciliation`,
`stats_utility_reconciliation`, `clutch_attempt_reconciliation`,
`warmup_contamination`, `metadata_provenance`, `metadata_checksum_lineage`,
`economy_determinism`, `stats_determinism` y
`economy_observation_coverage`.

Los trece primeros son críticos. Las diferencias reales entre native y calculated
se contabilizan en un diagnóstico separado y no hacen fallar el gate cuando ambos
valores y su delta se conservan. `economy_observation_coverage` es informativo y
puede ser `warning`; la ausencia se declara, no se rellena.
