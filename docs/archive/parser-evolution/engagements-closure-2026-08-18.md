# Cierre del Bloque 5 — Engagements, duelos, exchanges y trades

Fecha de cierre: 2026-08-18.

## Resultado

El Bloque 5 queda cerrado sobre las dos demos piloto. Los derivados se construyen exclusivamente desde `stratai.combat_event@2`, conservan cierres atómicos exactos, separan roles causales de outcomes y representan de forma explícita los casos `unavailable`, `inferred`, `not_tradeable` y `not_evaluable`.

Versiones finales:

- Export canónico `3.5.0`.
- Parser `v14`.
- Quality schema `10`.
- `stratai.engagements@2` con algoritmo `engagement_causal@2`.
- `stratai.trades@1` con algoritmo `trade_response@1`.
- `stratai.player_match_stats@2`.
- Ledger fuente sin cambios: `stratai.combat_event@2`.

Configuración temporal versionada:

- Continuación del mismo par: `1.500 ms`.
- Unión multi-target: `750 ms`.
- Duración máxima: `5.000 ms`.
- Prelude de agresión: `500 ms`.
- Respuesta de trade: `5.000 ms`.
- Todos los ticks se calculan con `ceil(ms * tick_rate_hz / 1000)`; no existen ventanas de 64/128 ticks hardcodeadas.

## Fallos encontrados y causa raíz

- El derivador anterior consumía eventos legacy, dependía de iteraciones de mapas, trataba `attacker` como winner implícito, reutilizaba sources, mezclaba contexto posterior con datos causales y aplicaba una ventana de trade fija de 320 ticks.
- El export anterior no distinguía de forma contractual `initiator`, `first_aggressor`, `first_damage_dealer`, participantes, winner/losers, exchanges exactos ni intentos de trade fallidos/no evaluables.
- La primera ejecución real de Nuke activó correctamente el nuevo gate pre-export: un `weapon_fire` inferido podía asignarse a dos engagements; algunos ancestros de disparo estaban ordenados después del primer daño; y un prelude inferido redundante podía coexistir con un ancestro causal observado.
- La corrección hace exclusiva la atribución del prelude, descarta inferencias ambiguas, solo admite ancestros ordenados causalmente y prioriza evidencia atómica observada. El intento fallido no modificó el export publicado.

## Contrato implementado

- IDs deterministas asignados después de un orden total.
- Exchanges como proyecciones exactas de un único `player_hurt` enemigo y su cierre `hurt → kill` cuando corresponde.
- Participantes reconciliados por `player_id`, nunca por nombre.
- `initiator`, `first_aggressor` y `first_damage_dealer` conservan evidencias, status, disponibilidad y confianza independientes del winner.
- `causal_context` usa únicamente estados con `availability_tick <= t0`; `outcome_context` contiene kills, winner/losers, disengagement y enlaces a trades.
- Una muerte enemiga produce exactamente un candidato de trade. Las respuestas se asignan uno-a-uno al candidato previo más reciente con desempate estable.
- `completed`, `failed`, `not_attempted`, `not_tradeable` y `not_evaluable` permanecen separados.
- Las estadísticas de trade se regeneran desde candidatos/completions, sin el heurístico legacy de KAST.

## Quality gates

Los nueve gates duros están en `pass` y su métrica es cero en ambas demos:

1. `engagement_event_contract`
2. `engagement_atomic_provenance`
3. `engagement_participant_reconciliation`
4. `engagement_role_consistency`
5. `engagement_temporal_consistency`
6. `engagement_causal_availability`
7. `engagement_trade_reconciliation`
8. `engagement_stats_reconciliation`
9. `engagement_determinism`

`engagement_observation_coverage` es un gate informativo en `warning`: 41 observaciones en Nuke y 50 en Cache. No bloquea entrenamiento porque las ausencias permanecen explícitas y no fabrican movimiento, arma, distancia, agresor ni tradeability.

## Métricas reales

| Métrica | Nuke | Cache |
| --- | ---: | ---: |
| Eventos atómicos de combate | 6.567 | 12.572 |
| Hurts enemigos | 533 | 728 |
| Kills enemigas | 135 | 160 |
| Engagements | 195 | 258 |
| Exchanges | 533 | 728 |
| Duelos | 182 | 243 |
| Multi-target | 11 | 14 |
| Collateral | 2 | 1 |
| Outcomes kill / disengaged | 132 / 63 | 157 / 101 |
| First aggressor unavailable | 35 | 41 |
| First aggressor distinto del first damage | 3 | 5 |
| Initiator distinto del winner | 19 | 32 |
| Candidatos de trade | 135 | 160 |
| Completed / failed / not attempted | 19 / 9 / 88 | 22 / 24 / 93 |
| Not tradeable / not evaluable | 19 / 0 | 20 / 1 |
| Trade attempts / failed attempts | 32 / 10 | 47 / 24 |
| Trade kills / traded deaths | 19 / 19 | 22 / 22 |

La diferencia entre candidatos fallidos e intentos fallidos de Nuke es intencional: un candidato puede contener intentos de más de un compañero.

## Validación y determinismo

- `gofmt` limpio en el alcance modificado.
- `go test ./... -count=1`: correcto.
- `go vet ./...`: correcto.
- Stress/permutación: `pkg/engagement` y `parser`, 25 repeticiones correctas.
- Validador/reprocesador Python: 106 pruebas correctas.
- Backend Python: 42 pruebas correctas.
- Frontend: 23 ficheros y 88 pruebas correctas; build Vite correcto.
- Node: 14 pruebas correctas y 3 skips declarados.
- Ruff: correcto en aplicación, tests y scripts canónicos.
- `canonical_export_validator.py`: 2/2 exports correctos después de cada commit final.
- `engagement_export_audit.py`: 2/2 exports y 4/4 informes pass1/pass2 sin errores.

Hashes SHA-256 del árbol canónico, idénticos en dos reprocesados consecutivos:

- Nuke: `e9bb84caff6696131f5bb38f6a396033bb85db59d23c702deb3cf0774fe52825`.
- Cache: `56e1a21904714dc1ea5356d12049b4da24535046eef610ab4dcd19a0e1bda5e9`.

Hashes de artefactos finales:

| Artefacto | Nuke | Cache |
| --- | --- | --- |
| `engagements.json` | `da1c0cea2fec0e4398b6920ffc1017f08f1fca6e2ed5dcba5e3a9c7a89304891` | `dae0f4f897f6f1ba16ffa736f3af6ad735cb622abd54f1fa6a4d622f85b5d0a0` |
| `trades.json` | `5a875016bd3438767245e74a6e8bb6ce4eca3f23bac2eb0a3164e70f23b7325f` | `dc20f02eb441d58bdc49e6dc008f98d33c489e01072fc546cdeb587ca590ea0d` |
| `player_match_stats.json` | `63a6a90fa8738ba96cfbe302b99fbaea5c8d2213df78a026f02909ae5781bd55` | `fb3d1f81184b922822ad5d2db8c02dc0e6ff52f42b582d95b04ac508df0e76b8` |
| `quality_report.json` | `00727dcca24cea58915775d7444a04736e804a067a0a969948a3ffdb2c5cef3e` | `dd097b8270fd75208d73bcbcfacc5b5fe47b9913afeb7faa5b7267bd003ea6db` |

El único cambio del árbol completo entre commits es el `manifest.json` operacional de raíz y su timestamp de publicación. El subárbol `canonical/`, sus manifests de contenido y todos sus SHA-256 son byte-idénticos, igual que en el criterio de cierre del Bloque 4.

## Backups y demos

- Demo Nuke: SHA-256 `86058469da7bbf625e341a4391b743d341019c0e23e43ad0d60de0dfb1c29eaa`.
- Demo Cache: SHA-256 `e5710c75deebde58e791572741774ad09c119b25bd18949d215faaf5c1446744`.
- Backups externos conservados en `backend/data/export_backups/block5_20260818/`:
  - `nuke_pre_v14`: 53 archivos, tree hash `b1a0b86b6bfc5009871a11b2c9c39e1f27d87de6af71a605e1e4d9481f37f17b`.
  - `nuke_pass1_v14`: 54 archivos, full tree pass1 `969b28027dde761fe43f3a63531daf5953d2d4898647a984fa12b98e70335e33`.
  - `cache_pre_v14`: 57 archivos, tree hash `b43fc9d2c974d1f2654ff3ddb7e682bce8dc33059df24186105f03a8d89b1e26`.
  - `cache_pass1_v14`: 58 archivos, full tree pass1 `76b9c45fbb06acff4e72665d44d4bfb0e616fb2aa7312b1594036b54fedd506e`.
- Informes independientes conservados en `backend/data/export_audits/block5_20260818/`.
- Los cuatro reprocesados usaron `--workers 1 --timeout 1500 --skip-aggregate-rebuild --match-id`; no se reconstruyeron agregados.

## Limitaciones explícitas

- Un engagement requiere daño enemigo atómico; no se fabrican engagements a partir de misses o proximidad temporal sin exchange factual.
- `first_aggressor` puede quedar unavailable. La inferencia por miss solo se conserva si su atribución a un engagement de dos jugadores es exclusiva y se publica con confianza y disponibilidad explícitas.
- Cache conserva una muerte `not_evaluable` porque faltan observaciones necesarias; no se fuerza a `not_tradeable`.
- El quality global de ambos exports permanece en `warning` por coberturas explícitas de varios bloques y discrepancias económicas ya inventariadas, pero `usable_for_training=true`; no queda ningún P0/P1 conocido del Bloque 5.

## Siguiente bloque recomendado

Bloque 6 — Economía, estadísticas y metadatos. No se implementó ninguna parte de ese bloque durante este cierre.
