# Candidato económico: frontera inicial y rondas, versión 2

Contrato `1.0.0-candidate.2`. Extiende el [candidato v1](canonical-economy-candidate-v1.md) en una salida aislada. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado de entrega. No modifica los bundles `3.8.0`, la elegibilidad ni las ventanas del catálogo.

## Dos momentos de observación

`rounds` conserva el significado de v1: primera captura en `FrameDone`, ejecución durante freeze y reconciliación posterior. `initial_frontiers`, con una entrada por instancia, añade:

| Campo | Significado |
|---|---|
| `round_start_probe` | Copia de propiedades nativas dentro del callback `RoundStart`, con tick y secuencia reales. Incluye ausencias; no rellena valores desde la captura posterior. |
| `native_rules` | Presencia, valor y tiempo de `m_pGameRules.m_bHasMatchStarted`, `m_bWarmupPeriod` y `m_bFreezePeriod`. Un `false` observado es distinto de una propiedad ausente. |
| `pre_probe_callbacks` | Callbacks de objetos del mismo tick anteriores a la sonda, incluso anteriores a la apertura de la instancia. Conservan actor opaco, tipo, tick y secuencia. |
| `official_end_tick` | Callback `RoundEndOfficial`; puede estar ausente. Es una frontera de correspondencia, no información inicial del jugador. |
| `round_link` | Resultado de contrastar identidad, límites y participantes con los artefactos canónicos acreditados por el manifest. |

La versión instalada de demoinfocs, `v5.2.0`, procesa `RoundStart` y callbacks de objetos aplazados al terminar el fotograma nativo. Una sonda anterior en el orden de callbacks puede ver propiedades que ya han cambiado. Por eso sus actores llevan `prepurchase_status=unavailable` y `not_initial_snapshot` —o `participant_unavailable`—: ese registro aislado no acredita precompra.

La primera captura en `FrameDone` sigue disponible como control de v1. Su contador histórico no es el contador competitivo de v2. El validador comprueba ambos compartimentos; nunca aumenta cobertura reinterpretando una recogida como compra o como aparición inocua.

## Acceso al estado inicial

El consumidor de v2 es `economy_initial_frontiers.initial_team_state`, sobre un candidato previamente cargado y validado mediante `validate_economy_candidate.validate_candidate`.

El acceso común `validate_economy_candidate.initial_team_state` también selecciona este lector al recibir v2. Un consumidor anterior no puede omitir el enlace competitivo utilizando el acceso v1 sobre datos nuevos.

1. Exige instancia existente, T0 exactamente igual a su inicio y correspondencia competitiva verificada.
2. Identifica cinco aliados en la sonda inicial. Para cada actor elige su observación completa más temprana: sonda, o primera captura de frame si la sonda carece de propiedades. No mezcla campos de dos observaciones. La selección no depende del dinero ni de las propiedades del rival.
3. Exige conexión, las ocho propiedades nativas presentes, captura exactamente en T0, gasto nativo cero y reglas nativas observadas compatibles con partida iniciada, sin warmup y en freeze. Una captura posterior permanece indisponible; no se mueve T0.
4. Cualquier callback de recogida, drop o refund en el tick inicial bloquea ese actor, aunque llegue después de la sonda o del primer frame. Los callbacks anteriores al reinicio tampoco se pierden. Los eventos de ticks posteriores no cambian los hechos ni la calificación inicial.
5. Devuelve una copia del equipo propio, procedencia de cada observación y contexto observado en la sonda. No devuelve rival, ejecución, reconciliación, desenlace ni límites finales. Los valores de contexto ausentes siguen ausentes.

El equipo puede contener observaciones completas de dos callbacks distintos del mismo T0; esto no acredita orden dentro del tick, simultaneidad de subtick, atención, voz o coordinación. Sigue siendo un estado factual para desarrollo, sin decisión económica ni recomendación. La falta de una regla económica como `mp_maxmoney` no se resuelve con un valor por defecto: su compatibilidad prescriptiva continúa sin verificar.

## Correspondencia competitiva

El productor contrasta cada instancia con `core/rounds.json` del bundle recién exportado. Se exige un único registro con el mismo número, inicio exacto y final oficial exacto, ID `source_alias:round:NNN`, y diez identidades reconciliadas mediante el puente opaco con `derived/economy_players.json`, cinco por lado. No basta el número visible. Dos instancias no pueden apropiarse de la misma ronda.

El validador independiente verifica hashes y tamaños desde el manifest base y recalcula la correspondencia. Un inicio preliminar, reinicio sin correspondencia, final oficial ausente, identidad o lado incoherente, o enlace duplicado queda excluido con motivo explícito. Un final truncado no se sustituye por el último frame del replay.

Esta comprobación es una condición de procedencia realizada pospartida. Lee límites e identidades del registro canónico, sin incorporar ganador, resultados o dinero futuro a las entradas del coach. La pertenencia a una ronda competitiva y la disponibilidad de los hechos iniciales son comprobaciones distintas.

## Schemas, compatibilidad y ejecución

- [Datos v2](../../../ai_coach/contracts/observed_economy_candidate_v2.schema.json): `stratai.observed_economy_candidate@2`.
- [Manifest v2](../../../ai_coach/contracts/economy_candidate_manifest_v2.schema.json): `stratai.economy_candidate_manifest@2`.
- [Validador común v1/v2](../../../ai_coach/contracts/validate_economy_candidate.py) y [lector inicial v2](../../../ai_coach/contracts/economy_initial_frontiers.py).

Los dos schemas v1 y su lector permanecen disponibles. El selector `--economy-candidate-version 2` requiere `--ai-coach-economy-candidate`; sin selector se conserva v1. Ambas versiones mantienen la validación de salida TEMP vacía antes de parsear y son incompatibles con los modos espaciales.

```powershell
# Mismas precondiciones y variables de identidad que en el contrato v1.
# $economyRunRoot es una carpeta nueva, vacía y dedicada bajo TEMP.
Push-Location backend/go-service
try {
  go run ./cmd/canonical-export --demo $approvedDemo --output $economyRunRoot `
    --alias $sourceAlias --checksum $sourceSHA256 --maps-root ../data/maps `
    --build-id <identificador-de-build> --ai-coach-economy-candidate `
    --economy-candidate-version 2
} finally { Pop-Location }
backend/venv/Scripts/python.exe -B -m ai_coach.contracts.validate_economy_candidate `
  "$economyRunRoot/smoke_run_$sourceAlias/economy_candidate_manifest.json" `
  --expected-source-sha256 $sourceSHA256 --require-observed
```

El informe diferencia capturas de control, enlaces competitivos, actores iniciales acreditados, equipos completos y abstenciones. Ninguno de estos conteos equivale a errores detectados, consejos correctos o capacidad elegible. `training_allowed` y `product_supported` permanecen en `false`.
