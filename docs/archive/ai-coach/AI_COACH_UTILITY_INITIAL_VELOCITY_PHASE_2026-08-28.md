# Cierre: velocidad inicial causal del proyectil de utilidad

Fecha: 28 de agosto de 2026  
Fase: 1B.5  
Estado: cerrada en código y validada con exactamente una demo aprobada  
Contrato candidato: `stratai.observed_utility_initial_velocity@1` (`1.0.0`)  
Informe de smoke: `stratai.observed_utility_initial_velocity_full_demo@1`  
Evidencia: [AI_COACH_UTILITY_INITIAL_VELOCITY_SINGLE_DEMO_SMOKE_2026-08-28.json](../../evidence/AI_COACH_UTILITY_INITIAL_VELOCITY_SINGLE_DEMO_SMOKE_2026-08-28.json)

## Resultado

StratAI ya puede acreditar la velocidad inicial de un proyectil de utilidad como un hecho causal separado. Cada fila conserva el vector XYZ, la velocidad horizontal, unidades, sistema de coordenadas, ticks de observación y disponibilidad, ventana de medida, fuente, procedencia y aliases locales de join. El estado siempre es `observed` o `unavailable`.

La fuente preferida es `m_vecVelocity` durante `GrenadeProjectileThrow`. Sólo se acepta cuando el vector completo existe, es finito, ninguna componente cumple `abs(value) >= FLT_MAX` y no es el cero total ambiguo. Una componente cero dentro de un vector válido se conserva.

Cuando la propiedad nativa no acredita el hecho, el productor usa únicamente la posición de lanzamiento y la primera posición posterior válida recibida online:

`velocity = (end_position - start_position) * tick_rate_hz / tick_count`

La derivación sólo queda disponible en `end_tick`, que coincide con `observed_tick` y `availability_tick`; nunca se retropublica como conocida en el lanzamiento. En la demo real, las 398 observaciones usaron una ventana causal de 1 tick a 64 Hz.

El candidato permanece fuera del bundle y del manifest canónicos `3.8.0`. Los 44 bundles aprobados conservan su `projectile_initial_velocity` histórico como `unavailable`; ninguna capacidad cambia de elegibilidad. Gate 1B sigue pendiente y `training_allowed=false`.

## Semántica causal e identidad

- La identidad física es `(round, entity ID, serial)`, más actor y tipo de utilidad exactos. Un serial distinto y observado identifica otra entidad válida; reutilizar posteriormente el mismo scope físico exacto hace que sólo el nuevo candidato se abstenga.
- El candidato nativo se congela en el tick de lanzamiento. El fallback se congela en la segunda posición. Un callback posterior al tick de disponibilidad no puede cambiar un hecho ya acreditado, aunque traiga identidad discordante, no finitos o sentinels.
- Conflictos relevantes en el mismo tick siguen fallando cerrados. Bounce o lifecycle antes o en la segunda muestra, tickrate inválido, ticks no crecientes, delta nulo ambiguo o muestras insuficientes producen `unavailable`.
- `recordFinalTrajectory`, trayectoria completa, detonación, efectos, daños, outcomes y estado futuro no participan en la construcción.
- Los aliases de participante se asignan por primera aparición causal sobre un orden estable. Añadir un jugador o conflicto futuro no renumera una fila histórica.
- Los IDs físicos sólo se usan internamente para joins; no se publican nombres, Steam IDs, entity IDs ni seriales.

## Implementación y validadores

- `pkg/utility` mantiene `InitialVelocityCandidate` como estado hermano, separado de la semántica legacy.
- `handlers/grenade_throw.go` captura la propiedad nativa y el fallback online dentro del mismo parse.
- `BuildObservedUtilityInitialVelocityLedger` proyecta una fila por lanzamiento sin leer campos oracle ni trayectoria final.
- `BuildObservedUtilityInitialVelocityFullDemoReport` escribe un JSONL gzip determinista con hashes comprimido y lógico.
- `canonical-export --ai-coach-full-demo-validation` ejecuta una sola llamada a `ParseDemoResult` y publica el candidato fuera de `match_<alias>/canonical`.
- El JSON Schema Draft 2020-12 discrimina `observed/unavailable`, fuente y método, cierra joins, provenance, tipos y 25 razones de abstención, y rechaza sentinels por componente.
- El validador Python ejecuta el schema real y además recomputa causalidad, delta, hashes, tipos JSON estrictos, cobertura, orden, privacidad y resúmenes. Booleanos o floats no pueden pasar como ticks o conteos enteros.
- Los validadores Go, schema y Python exigen `initial_velocity_unavailable` en toda abstención. `reconciliation=ambiguous` exige exactamente una causa entre `projectile_observation_identity_conflict` y `projectile_entity_reused`; esas causas se prohíben con reconciliación no ambigua.
- Las regresiones demuestran que el candidato y el serial runtime no cambian los bytes de `projectCanonicalUtilityThrows`, y que la deduplicación legacy del mismo tick sigue produciendo un único throw.

## Intentos sobre la única demo

Se abrió un único match aprobado. Las correcciones reutilizaron siempre esa misma demo:

| Intento | Build | Resultado |
|---|---|---|
| r1 | `working-tree-utility-initial-velocity-full-demo-20260828-r1` | Productor exit 0, 403 filas, 398/5; invalidado por revisión independiente posterior: tickrate con default, identidad insuficiente, dependencia del orden, unidad incorrecta y sentinels parciales no cerrados. |
| r2 | `working-tree-utility-initial-velocity-full-demo-20260828-r2` | Productor exit 0, 386/17 y validadores del momento verdes; invalidado porque trataba 12 seriales físicos distintos como falsas reutilizaciones y no publicaba la causa específica. |
| r3 | `working-tree-utility-initial-velocity-full-demo-20260828-r3` | Cierre válido: productor exit 0, contrato final verde, 398/5 y cero errores canónicos. |

Datos del intento válido:

| Propiedad | Valor |
|---|---|
| Alias opaco | `demo-a11eb0d600ee702c9edb` |
| SHA-256 | `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262` |
| Tamaño | 330.945.844 bytes |
| Manifest aprobado | `b17deba4ef4f81c8f80e9541947f32ebf2baf22c8c7d8a459232e6f37bc33be5` |
| Estado previo | `passed`, parser `v16`, export `3.8.0` |
| Inicio UTC | `2026-08-28T19:55:34.9247958Z` |
| Fin UTC | `2026-08-28T19:57:50.9607996Z` |
| Duración | 136.031 ms |
| Cobertura | 24 rondas; una sola lectura de la demo |

Comando exacto del intento válido, ejecutado desde `backend/go-service`:

```powershell
go run ./cmd/canonical-export --demo 'E:\Carpeta compartida\Proyecto IA\Faceit-Demos\demos\1-3f9e5224-791f-4a15-ac14-5acaacec9407.dem' --output 'C:\Users\kerch\AppData\Local\Temp\stratai-one-demo-utility-velocity-2952c64387d14def992188dec1ed16b3' --alias 'demo-a11eb0d600ee702c9edb' --checksum 'a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262' --maps-root 'E:\Carpeta compartida\Proyecto IA\StratAI\backend\data\maps' --build-id 'working-tree-utility-initial-velocity-full-demo-20260828-r3' --ai-coach-full-demo-validation --navigation-smoke-queries 64 --visual-contact-smoke-frames 256
```

## Resultado real

| Métrica | Valor |
|---|---:|
| Lanzamientos / filas | 403 / 403 |
| Rondas | 24 |
| Participantes locales | 10 |
| `observed` | 398 |
| `unavailable` | 5 |
| `projectile_position_delta` | 398 |
| Ventana de 1 tick / 64 Hz | 398 / 398 |
| `observed_tick == availability_tick` | 398 |
| Error máximo de recomputación | 0 |

Tipos: 120 flashbang, 108 smoke, 69 HE, 49 molotov, 47 incendiary y 5 decoy. Otras 5 filas no tenían tipo causal acreditable.

Los valores cubrieron los tres ejes. Se conservó una componente `y=0`; no hubo ceros observados en `x` o `z`. Rangos: `x=[-966, 954]`, `y=[-943.99609375, 982]`, `z=[-1010, 820.000244140625]` y velocidad horizontal `[2, 982.2036448720804]`, en unidades Source 2 por segundo.

Las cinco filas `unavailable` —dos en ronda 16 y tres en ronda 23— conservaron estos seis motivos cada una:

| Motivo | Filas |
|---|---:|
| `initial_velocity_unavailable` | 5 |
| `launch_tick_unavailable` | 5 |
| `launcher_unavailable_at_launch` | 5 |
| `native_velocity_property_unavailable` | 5 |
| `projectile_entity_unavailable` | 5 |
| `utility_type_unavailable_at_launch` | 5 |

No hubo mismatch del delta, inferencia nominal, identidades crudas, Steam IDs, nombres ni rutas absolutas en el stream. Los negativos no presentes en esta demo se cubren con fixtures sintéticos.

## Artefactos acreditados

| Artefacto | Bytes | SHA-256 |
|---|---:|---|
| `observed_utility_initial_velocity_full_demo.json` | 1.544 | `09eb00ff4e135c56a1cee7227c21aab5fd2d7edff636aade8a4ee5dabf9043ce` |
| `observed_utility_initial_velocity_rows.jsonl.gz` | 33.517 | `3827c0bc6974443f8f0f6dde86ea54f0f6ad03d7dd6f36c6103e484062c4a7ec` |
| JSONL lógico | 412.380 | `ceed420d6bb8be68ac302f02fc85708fb1c00c8d4f5a8d611d5f339496261218` |
| manifest canónico temporal | 33.495 | `033258a0c023c8ebdf46662c2679cda3fb13351ae5665a810e2c1ea8e08804c9` |

El bundle temporal tuvo 73 ficheros, 735.286.379 bytes y 72 artefactos declarados. `golden_demo_semantic_runner.validate_direct_bundle` devolvió cero errores con `3.8.0` y parser `v16`.

## Estado protegido y límites

El postflight reprodujo exactamente los digests del preflight: 100 demos, 3.013 ficheros bajo `analyzed_demos`, 727 bajo `failed`, `staging` vacío, 1.242 ficheros en `backend/data/exports` y 44 manifests aprobados. `Faceit-Demos` siguió limpio en `f132c8de4bcc97650352ec5a647fdf3985f11353`. `CURRENT` siguió apuntando a `stratai-2c6db440231d463a` y la release activa conservó su digest.

No se entrenó, descargó, reprocesó el corpus, publicó, promovió, creó una release ni alteró un bundle aprobado. La evidencia máquina conserva comandos, intentos, tiempos, pruebas, hashes, inventarios y limpieza.

## Siguiente blocker demostrado

El siguiente paso es `FLD-LAST-KNOWN-POSITION` junto con `FLD-INFORMATION-AGE`, derivados sólo de episodios causales de contacto visual. Debe conservar tick de última observación, edad y una política explícita de caducidad; ante ausencia o ambigüedad debe abstenerse. No se permiten posiciones oracle, contactos futuros ni reconstrucción retrospectiva. Esta fase deja el prompt sucesor, pero no lo ejecuta.
