# Contrato atómico de combate v2

## Versiones

- Evento: `stratai.combat_event@2`.
- Export canónico: `3.4.0`.
- Parser: `v13`.
- Replay: `stratai.replay_index@5` y `stratai.replay_round@5`.
- Quality report: schema `9`.

El scope es exclusivamente el Bloque 4. La identidad de engagements,
iniciadores, ganadores y trades continúa perteneciendo al Bloque 5.

## Fuente de verdad

`pkg/combat.Tracker` es el único propietario de los callbacks de combate:

- `WeaponFire`.
- `BulletDamage`.
- `PlayerHurt`.
- `Kill`.
- `WeaponReload`.
- `ItemEquip`.

Los handlers solo convierten callbacks a inputs factuales. El tracker conserva
todos los inputs, incluidos world, self, friendly y actores no disponibles. La
correlación se ejecuta sobre un snapshot ordenado después de capturar los
callbacks; no depende del orden de registro de handlers ni de iterar maps.

Estadísticas, replay y export canónico se proyectan desde ese snapshot. El
scoreboard nativo se mantiene como resultado autoritativo post-match y sus
deltas frente a lo observado quedan explícitos.

## Tipos de evento

| `event_type` | Callback fuente | Semántica |
| --- | --- | --- |
| `weapon_fire` | `events.WeaponFire` | Acción factual; crea un `shot_id` estable |
| `bullet_damage` | `events.BulletDamage` | Evidencia de impacto dañino sin posición de impacto |
| `player_hurt` | `events.PlayerHurt` | Daño factual de vida/armadura, incluido fatal |
| `kill` | `events.Kill` | Resultado de muerte, assist y flash assist |
| `weapon_reload` | `events.WeaponReload` | Inicio de recarga observado; no implica fin |
| `weapon_equip` | `events.ItemEquip` | Equip/cambio observado cuando la demo lo emite |

No se fabrica `bullet_impact`: `demoinfocs v5.2.0` no lo expone en estas demos.
`bullet_damage` declara `impact_position_status=unavailable`. Si una fuente
futura aporta posición real, se añadirá sin convertir el endpoint visual del
replay en un hecho.

## Campos obligatorios de cada fila

### Identidad y tiempo

- `schema_id`, `match_id`, `event_id`.
- `round_id`, `round_number`, `tick`.
- `sequence_in_tick`, `sequence_in_round`.
- `event_type`, `source`.
- `source_event_ids`, siempre lista y solo con IDs anteriores o simultáneos ya
  ordenados causalmente.
- `tick_status=observed`.
- `subtick=null`, `subtick_status=unavailable` cuando la fuente no lo aporta.
- `time_seconds`, derivado de `tick/tick_rate`, con source explícito.

### Actores y relación

- `actor_player_id`, `actor_side`, `actor_status`, `actor_source`.
- `target_player_id`, `target_side`, `target_status`, `target_source`.
- `relation`: `enemy | friendly | self | world | unknown`.

Un actor ausente se representa con `null`, nunca con `steam:0`. World requiere
actor nulo y target observado. `unknown` se usa si faltan datos para clasificar.

### Arma y posiciones

- `weapon`, `weapon_status`, `weapon_source`.
- `actor_position`, `actor_position_status`, `actor_position_source`.
- `target_position`, `target_position_status`, `target_position_source`.

El arma factual procede del propio callback cuando existe. `ActiveWeapon()` no
sustituye el arma de un hurt/kill. Los valores ausentes son `null/unavailable`.

### Correlación de disparo e impacto

- `shot_id` en fire y en eventos correlacionados.
- `correlation_status`: `exact | inferred | unavailable`.
- `correlation_source`.
- `shot_result`: `hit | miss | unavailable` solo en `weapon_fire`.
- `shot_result_status`, `shot_result_source` y
  `shot_result_availability_tick` identifican el outcome posterior.
- `impact_position`, `impact_position_status`, `impact_position_source`.
- `bullet_distance_world_units`, `damage_direction`,
  `penetrated_objects`, `no_scope`, `attacker_in_air` cuando `BulletDamage`
  los aporta.

Una correlación nunca enlaza un fire con un evento de tick anterior. Empates se
resuelven con un orden total estable por ronda, tick, prioridad causal, IDs de
jugador, arma y payload factual. Los IDs se asignan después de ese orden.

### Daño y kill

- `health_damage`, `health_damage_taken`.
- `armor_damage`, `armor_damage_taken`.
- `health_before`, `health_after`, `armor_before`, `armor_after`.
- `damage_status`, `damage_source`.
- `hitgroup`, `hitgroup_status`, `hitgroup_source`.
- `is_headshot`, `is_kill`.
- `assister_player_id`, `assister_side`, `assister_status`, `assister_source`.
- `assisted_flash`.

Solo `player_hurt` transporta daño. `kill` no vuelve a imputar daño ni inventa
hitgroup; se enlaza al hurt fatal cuando existe. `health_damage_taken` y
`armor_damage_taken` conservan el daño efectivo limitado por vida/armadura; los
campos sin semántica para el tipo de evento son `null` con status explícito.

### Lifecycle de arma

- `reload_phase=start` solo para `weapon_reload`.
- `reload_end_tick=null` y `reload_end_status=unavailable` si no hay callback
  fiable de fin.
- `previous_weapon` y `previous_weapon_status` en equip se basan únicamente en
  observaciones causales anteriores.
- `is_weapon_switch` es booleano solo si arma anterior y actual están
  observadas; en otro caso es `null`.
- `ammo_in_magazine` y `ammo_reserve` conservan disponibilidad/source.

## Proyecciones

Cada ronda de replay contiene `combat_shots`, una proyección única y ordenada de
todos los `weapon_fire` de la ronda. `frames[].shots` es solo la repetición
visual disponible cuando existe un frame causal posterior al disparo; nunca se
inventa un frame ni se copia estado de jugadores para alojar un disparo tardío.

### Estadísticas

- `shots_fired`, `shots_hit`, misses, body hits y weapon stats se calculan por
  `shot_id`/ledger, no desde callbacks paralelos.
- Friendly, self y enemy damage se calculan desde `player_hurt`.
- K/D/A permanecen reconciliados con el scoreboard nativo; se publican además
  los conteos observados y `native_minus_observed`.
- `total_damage` permanece nativo. `combat_damage_observed` y
  `combat_damage_unattributed_delta` hacen explícita la cobertura sin crear un
  evento falso.
- Flash assists observados proceden de `kill.assisted_flash`.

### Replay

- `player_hurt`, `kill` y shots se generan desde el ledger.
- El frontend selecciona desde `combat_shots` por tick causal; no depende de que
  exista un frame muestreado posterior.
- Cada evento de replay incluye `source_event_ids` canónicos.
- El endpoint de una línea de disparo se etiqueta
  `derived/view_direction_projection`; nunca se etiqueta como impacto.
- `hit` refleja cualquier hurt correlacionado con el shot, no solo una kill.

### Engagements

Se mantienen sus algoritmos actuales. Sus `source_event_ids` se remapean a
eventos `player_hurt`/`kill` válidos del nuevo ledger, sin cambiar quién se
considera iniciador, ganador o trade en este bloque.

## Invariantes y gates duros

1. Conservación: callbacks aceptados por ronda/tipo = filas del ledger por
   `source`; cualquier descarte lleva diagnóstico y motivo.
2. IDs únicos y estables, asignados después del orden total.
3. Orden estricto por ronda, tick, `sequence_in_tick`, `event_id`.
4. Toda referencia de jugador apunta a participants o es `null/unavailable`.
5. Toda `source_event_id` existe, no se duplica y no apunta al futuro causal.
6. Cada fire tiene exactamente un `shot_id`; ningún otro evento inventa uno.
7. `shot_result=miss` solo tras cerrar la ventana causal y con cobertura
   suficiente; de lo contrario es `unavailable`.
8. Hurt conserva daño de vida y armadura no negativo y estados before/after
   coherentes cuando están disponibles.
9. Kill no duplica daño; assist/flash assist conservan actor y disponibilidad.
10. Relation coincide con IDs/lados; world/self/friendly no se omiten.
11. Stats ledger-derived recomputadas coinciden con el export; los resultados
    nativos solo difieren mediante deltas explícitos.
12. Replay es una proyección exacta de fire/hurt/kill y su provenance es válida.
13. Callbacks permutados producen el mismo snapshot y los mismos hashes.
14. Ningún valor no finito llega al export.
15. La ausencia de impactos espaciales o fin de reload es un warning de
    cobertura explícito, no un hecho inventado ni un fallo silencioso.

Los incumplimientos 1–14 impiden el export en Go y fallan también en Python. El
gate 15 permite `warning` y mantiene `usable_for_training=true` solo si toda la
indisponibilidad está declarada.
