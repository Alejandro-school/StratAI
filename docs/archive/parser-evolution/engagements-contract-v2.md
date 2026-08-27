# Contrato de engagements y trades v2

## Versiones congeladas

- Engagements: `stratai.engagements@2`.
- Trades: `stratai.trades@1`.
- Estadísticas: `stratai.player_match_stats@2`.
- Export canónico: `3.5.0`.
- Parser: `v14`.
- Quality report: schema `10`.
- Algoritmo de engagement: `engagement_causal@2`.
- Algoritmo de trade: `trade_response@1`.
- Ledger de entrada, sin cambios: `stratai.combat_event@2`.

El scope termina en el Bloque 5. No se corrige economía ni se avanza a los
Bloques 6 o 7.

## Fuente de verdad y fronteras

La identidad, exchanges, participantes y outcomes se derivan únicamente del
ledger `combat_event@2`. Ningún engagement modifica, completa o corrige una
fila atómica.

`player_state@3` solo puede enriquecer `causal_context` con un snapshot cuyo
`tick <= t0_tick`; cada valor conserva `state_id`, tick, status y source. Si no
existe observación causal, el valor es `null/unavailable`. Economía, visibilidad
o bomba no se reconstruyen desde outcomes.

Los nombres de jugador son presentación. Toda unión, rol, estadística y
referencia usa `player_id` estable.

## Configuración temporal

Toda configuración autoritativa se expresa en milisegundos:

| Campo | Valor |
| --- | ---: |
| `pair_continuation_window_ms` | 1500 |
| `multi_target_window_ms` | 750 |
| `max_engagement_duration_ms` | 5000 |
| `aggressor_prelude_window_ms` | 500 |
| `trade_window_ms` | 5000 |

La conversión inclusiva es `ceil(milliseconds * tick_rate_hz / 1000)`. Cada
artefacto publica milisegundos, ticks derivados y tick rate. No se permiten
constantes de ticks dependientes de 64/128 Hz.

## Orden total e IDs

Antes de asignar IDs, los eventos se ordenan por:

1. `round_number`;
2. `tick`;
3. `sequence_in_tick`;
4. prioridad causal `weapon_fire < bullet_damage < player_hurt < kill`;
5. actor ID nulo al final;
6. target ID nulo al final;
7. `shot_id` nulo al final;
8. `event_id`.

Engagements, candidatos, completions, participantes, exchanges y listas de
source IDs se ordenan explícitamente. Ningún map de Go define el resultado.

Los IDs se asignan después del orden total:

- `{match_id}:engagement:{sequence:06d}`.
- `{match_id}:trade_candidate:{sequence:06d}`.
- `{match_id}:trade_completion:{sequence:06d}`.

## Formación causal de engagements

Un exchange factual nace de un `player_hurt` enemy con actor y target
observados. Su closure incluye recursivamente los `source_event_ids` válidos,
el `weapon_fire`/`bullet_damage` correlacionado y la kill que referencia al hurt
fatal.

Dos exchanges se unen solo si están en la misma ronda, no exceden la duración
máxima y cumplen una regla documentada:

- mismo par no dirigido dentro de `pair_continuation_window_ms`;
- mismo `shot_id`, sin límite adicional dentro de la duración máxima;
- mismo actor y distinto target dentro de `multi_target_window_ms`.

Compartir únicamente una ventana temporal o una víctima no basta. La unión
transitiva no puede superar `max_engagement_duration_ms`. Si un exchange puede
unirse a varios grupos, gana: mismo shot, mismo par, multi-target, y después el
grupo con clave total menor.

Un `weapon_fire` miss puede actuar como preámbulo agresor solo en un engagement
de dos participantes, dentro de `aggressor_prelude_window_ms`, con el actor en
el engagement, lado opuesto observado y sin otro engagement dañino solapado
del actor. Su pertenencia es `inferred`, nunca `observed`, e incluye confidence
y availability tick. En caso ambiguo no se incorpora.

## Roles separados

- `initiator`: actor de la primera acción ofensiva causal incluida. Puede ser
  observed, inferred o unavailable. Nunca se deriva del ganador o daño total.
- `first_aggressor`: actor del primer `weapon_fire` incluido.
- `first_damage_dealer`: actor del primer `player_hurt` incluido.
- `participants`: unión ordenada de actor/target de exchanges.
- `winner_player_id`: actor de la kill terminal cuando existe un único ganador
  observable; en otro caso `null`.
- `loser_player_ids`: targets muertos observados, ordenados por su kill.

Cada rol publica `status`, `source`, `availability_tick`, `source_event_ids` y
`confidence`. Un rol desconocido usa ID nulo y status `unavailable`.

## Outcome

Enums de engagement:

- `kill`: existe una kill terminal coherente.
- `disengaged`: hubo daño, no hubo kill y la ventana cerró por timeout.
- `survived`: cierre de ronda observado con participantes vivos y sin kill.
- `unresolved`: faltan observaciones para clasificar.

El trade no reemplaza el outcome factual del engagement. Se enlaza mediante
IDs de candidato/completion para que una kill siga siendo una kill.

## Exchanges

Cada exchange contiene como mínimo:

- `exchange_id` igual al `event_id` del `player_hurt` factual;
- tick, secuencias y actor/target por player ID;
- arma, status y source;
- health/armor damage y before/after nulables;
- hitgroup, headshot y kill;
- `shot_id` y primer impacto `bullet_damage` cuando existe;
- posiciones de actor/target con status/source;
- distancia derivada en world units cuando ambas posiciones son factuales;
- reaction/TTD solo con source y availability trazables;
- closure completo y ordenado en `source_event_ids`.

Una fila atómica no puede respaldar dos exchanges distintos. Payloads iguales
en el mismo tick siguen siendo distinguibles por `exchange_id`.

## Separación causal

`causal_context` y `outcome_context` son objetos físicos distintos.

`causal_context` contiene únicamente observaciones con
`availability_tick <= t0_tick`: snapshots de participantes, posición,
velocidad, arma activa, health/armor, distancia inicial, objective phase,
relojes y availability. Economía y enemigos expuestos quedan
`null/unavailable` mientras no exista una fuente causal cerrada.

`outcome_context` contiene daños, kills, winner/losers, supervivencia,
disengagement, trades, tiempo hasta trade y cualquier dato disponible después
de `t0`. Ningún campo de este objeto puede copiarse a causal context.

## Trades

Cada kill enemy observada crea exactamente un `trade_candidate`. La respuesta
de trade exige:

1. misma ronda;
2. respuesta posterior a la muerte original;
3. elapsed `<= trade_window_ms` usando tick rate real;
4. target de la respuesta igual al killer original;
5. trader compañero de la víctima original;
6. kill de respuesta no consumida por otra completion.

Si una kill puede completar varios candidatos, se asigna al candidato previo
más reciente; el desempate usa el orden total del kill original. Los restantes
no reutilizan la kill.

Enums de evaluación:

- `completed`: existe completion 1:1.
- `failed`: un compañero dañó al killer dentro de ventana, pero no lo mató.
- `not_attempted`: había compañero vivo observado y no hubo daño de respuesta.
- `not_tradeable`: no había compañero vivo observado.
- `not_evaluable`: faltan estado, lado, actor, target o cobertura temporal.

Cada candidato incluye muerte, killer, víctima, teammates elegibles, intentos,
ventana configurada, resultado y sources. Cada completion incluye killer,
victim, trader, relación `teammate | enemy | unknown`, elapsed ms/ticks y
sources. Si una completion kill genera a su vez un candidato completado, el
segundo se marca como counter-trade mediante IDs, sin alterar los hechos.

## Estadísticas

`trade_kills`, `traded_deaths`, `trade_attempts`, `failed_trade_attempts`,
`untradeable_deaths` y `non_evaluable_trade_deaths` se proyectan únicamente
desde `trades@1`. Los contadores legacy no se publican.

Opening duels e indicadores de engagement que se toquen en este bloque se
reconcilian contra `engagements@2`. K/D/A, daño, armas, utilidad y scoreboard
del Bloque 4 no cambian.

## Quality gates Go y Python

Los diez gates obligatorios son:

- `engagement_event_contract`;
- `engagement_atomic_provenance`;
- `engagement_participant_reconciliation`;
- `engagement_role_consistency`;
- `engagement_temporal_consistency`;
- `engagement_causal_availability`;
- `engagement_trade_reconciliation`;
- `engagement_stats_reconciliation`;
- `engagement_determinism`;
- `engagement_observation_coverage`.

Los nueve primeros son duros y deben valer cero. Coverage puede ser warning
solo si cada missing/inferred/unavailable está explícito. Go valida antes de
escribir; Python repite la validación desde los artefactos publicados.

Los gates rechazan referencias inexistentes o incompletas, source IDs
reutilizados de forma incompatible, exchanges duplicados, jugadores fuera del
roster, roles derivados de outcomes, ticks fuera de ronda, final anterior al
inicio, trades temporales o relacionales inválidos, completions reutilizadas,
conversión temporal incorrecta, future leakage, clasificación peek/hold sin
velocidad, orden no determinista y stats no reconciliadas.
