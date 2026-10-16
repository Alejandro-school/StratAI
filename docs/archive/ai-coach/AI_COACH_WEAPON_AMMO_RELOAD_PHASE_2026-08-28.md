# Cierre: arma activa, munición y reload causales a 16 Hz

Fecha: 28 de agosto de 2026  
Fase: 1B.4  
Estado: cerrada en código y validada causalmente en `r3` con una única demo aprobada  
Contrato candidato: `stratai.observed_weapon_state@1` (`1.0.0`)  
Informe de smoke: `stratai.observed_weapon_state_full_demo@1`  
Evidencia: [AI_COACH_WEAPON_AMMO_RELOAD_SINGLE_DEMO_SMOKE_2026-08-28.json](../../evidence/AI_COACH_WEAPON_AMMO_RELOAD_SINGLE_DEMO_SMOKE_2026-08-28.json)

## Resultado

StratAI dispone de un productor candidato que observa a 16 Hz, por jugador y sin *carry-forward* entre rondas:

1. arma activa;
2. munición en el cargador;
3. munición en reserva;
4. estado de recarga.

Cada hecho se publica por separado como `observed` o `unavailable`, con su valor, `availability_tick` y procedencia. Cero munición continúa siendo una observación válida. `is_reloading=false` sólo lo es cuando un callback causal ya inicializó el tracker para la entidad física activa exacta y el bool de demoinfocs fue revalidado en el tick del sample; el zero-value nativo nunca acredita un `false`. Cuchillo, C4 y utilidades se tratan como equipo sin munición aplicable y nunca reciben cargador, reserva o reload fabricados. Los sentinels negativos se normalizan a indisponibilidad.

El muestreo usa `round_anchored_right_closed_no_carry_forward`: cada instante toma sólo el snapshot causal disponible por la derecha dentro de la misma ronda. Reload mantiene identidad mediante `playerstate.EquipmentID`, que combina entity ID y serial; dos entidades del mismo tipo de arma no comparten estado. `WeaponReload` acredita `true`, mientras `ItemEquip`, `WeaponFire` o un cambio de clip válido pueden acreditar `false` sólo cuando la entidad del callback coincide exactamente con la activa. Una vez válido, cada sample vuelve a leer `Player.IsReloading` para esa misma entidad y publica `availability_tick == source_tick == tick`.

La reconciliación nominal posterior del ledger permanece separada: los eventos sólo pueden confirmar una snapshot cuando coinciden ronda, tick, actor y arma normalizada; nunca rellenan un valor ausente. Una reconciliación ambigua obliga a abstenerse.

Este resultado no modifica el bundle canónico. El contrato candidato y sus ficheros de validación viven fuera de `3.8.0`; el parser canónico continúa en `v16`. Los campos candidatos añadidos al replay en memoria tienen `json:"-"`, por lo que no alteran los JSON existentes. `FLD-WEAPON-AMMO` continúa `PARCIAL` para el corpus publicado, `CAP-COMBAT-001` continúa no materializada, Gate 1B sigue pendiente y `training_allowed=false`.

## Implementación

- `handlers` captura arma activa, cargador y reserva y mantiene el tracker causal de reload durante el mismo parse usado por el exportador.
- El tracker sólo acepta callbacks con `EquipmentID` activo exacto. Muerte o `Kill`, respawn sin nueva señal, arma o entidad nula, mismatch, cambio de entidad, límite de ronda y clip `nil`, negativo o de tipo inválido eliminan la validez y producen `unavailable`.
- `parser.BuildObservedWeaponStateLedger` produce el ledger causal y usa un índice O(1) por ronda, tick, actor y arma normalizada para reconciliar disparos y recargas después de construir el snapshot.
- `parser.BuildObservedWeaponStateFullDemoReport` genera el informe completo y el stream comprimido de filas.
- `canonical-export --ai-coach-full-demo-validation` escribe el candidato de forma atómica junto a las validaciones ya existentes, pero no lo incorpora al manifest canónico `3.8.0`.
- `ai_coach/contracts/observed_weapon_state.schema.json` define el contrato físico y `validate_weapon_state_full_demo.py` vuelve a validar schema, causalidad, privacidad, semántica, conteos y hashes.
- Los participantes se retienen únicamente como alias locales `participant-NNN`; los identificadores son claves de join y no se publican nombres ni Steam IDs.

La procedencia de reload usa `demoinfocs_causal_reload_tracker`; distingue el snapshot revalidado, la confirmación nominal exacta por evento y la ausencia. El `EquipmentID` permanece interno y no se exporta. No se produjeron etiquetas good/bad, recomendaciones, candidatos de acción, intención, outcomes ni campos oracle.

## Pruebas negativas y de regresión

Las pruebas Go y Python cubren, entre otros casos:

- cero frente a ausencia en cargador y reserva;
- `false` acreditado frente al zero-value nativo y la ausencia de callback;
- cuchillo, C4 y utilidad como `not_applicable`;
- sentinels negativos y valores no finitos de ammo, además de clip `nil`, negativo o de tipo inválido;
- disponibilidad posterior a `T0` rechazada;
- ausencia de *carry-forward* entre rondas;
- evento coincidente que sólo confirma y evento futuro que no rellena;
- conflicto o ambigüedad que produce abstención;
- dos entidades con el mismo nombre de arma, mismatch entre entidad activa y evento y refresh causal en el tick del sample;
- muerte, `Kill`, respawn sin señal, arma nula y cambio de entidad;
- determinismo, estado físico interno por entity+serial e identidades de participantes limitadas a joins;
- stream gzip, resumen, hashes y conteos recalculados de forma independiente;
- serialización obligatoria de una lista vacía como `abstention_reasons:[]` y no como `null`.

En la demo real, sentinel de munición inválido y reconciliación ambigua sumaron cero. Los negativos de identidad, muerte/respawn, clip inválido y evento futuro quedan demostrados por las pruebas sintéticas y no se atribuyen falsamente al smoke.

## Gate real con exactamente una demo

Se abrió un único match distinto en todos los intentos:

| Propiedad | Valor |
|---|---|
| Alias opaco | `demo-a11eb0d600ee702c9edb` |
| SHA-256 | `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262` |
| Tamaño | 330.945.844 bytes |
| Manifest aprobado | `b17deba4ef4f81c8f80e9541947f32ebf2baf22c8c7d8a459232e6f37bc33be5` |
| Estado previo | `passed`, parser `v16`, export `3.8.0` |

La diversidad se comprobó antes de abrir la fuente: el bundle aprobado contenía 11.222 eventos de combate, 3.083 `weapon_fire` en 24 rondas y 49 `weapon_reload_start` en 20 rondas. Sus valores históricos de pseudo-munición para cuchillo, C4 y utilidad sólo sirvieron para detectar diversidad; no se reutilizaron como verdad del candidato.

Hubo tres intentos sobre ese mismo archivo, no tres demos:

| Intento | Build | Productor | Validador independiente | Resultado |
|---:|---|---:|---:|---|
| 1 | `working-tree-weapon-ammo-reload-full-demo-20260828-r1` | 99.972 ms, exit 0 | 175 ms | fallo cerrado: `rows[8].abstention_reasons` era `null` |
| 2 | `working-tree-weapon-ammo-reload-full-demo-20260828-r2` | 99.744 ms, exit 0 | 13.599 ms | los validadores pasaron, pero la revisión independiente invalidó la semántica de reload |
| 3 | `working-tree-weapon-ammo-reload-full-demo-20260828-r3` | aproximadamente 97.838 ms, exit 0 | 13.119 ms | válido y único resultado autorizable |

El primer intento descubrió que un slice `nil` de Go serializaba una lista vacía como `null`. La validación interna del productor no había ejercitado esa forma física. Se corrigió `canonicalizeWeaponAbstentions` para asignar siempre una lista no nula y se añadió la prueba de serialización.

El segundo intento pasó los validadores físicos, pero una revisión independiente detectó que `Player.IsReloading=false` podía proceder del zero-value sin bit de validez. También exigió identidad entity+serial y que el refresh del snapshot publicara el tick fuente actual. `r2` quedó invalidado y sus conteos y hashes no se usan como resultado del cierre. Tras implementar el tracker causal y sus negativos se repitió exactamente la misma demo por tercera vez, desde `2026-08-28T16:56:44.6224422Z` hasta `2026-08-28T16:58:22.4600149Z`, en el directorio temporal nuevo `stratai-one-demo-weapon-c74d9ef46acc42a8a87531cd3e0656de`.

El comando exacto, los directorios temporales, tiempos UTC y build IDs se conservan en la evidencia JSON enlazada. Ninguna salida fue promovida o publicada. Tras transferir hashes, conteos y revisión independiente, las tres hojas temporales se eliminaron; `r3` se retiró a `2026-08-28T17:14:31.1867153Z` y no quedó ninguna salida grande retenida.

## Resultado observado en la demo

El tercer intento produjo 29.520 instantes de muestreo y 294.825 filas jugador-instante sobre 24 rondas y 10 alias locales.

| Hecho | `observed` | `unavailable` |
|---|---:|---:|
| Arma activa | 203.030 | 91.795 |
| Munición en cargador | 131.626 | 163.199 |
| Munición en reserva | 131.626 | 163.199 |
| Reload | 130.031 | 164.794 |

Los valores observados conservaron los negativos semánticos reales:

| Valor | Filas |
|---|---:|
| Cargador igual a cero | 368 |
| Cargador positivo | 131.258 |
| Reserva igual a cero | 607 |
| Reserva positiva | 131.019 |
| Reload `true` | 940 |
| Reload `false` | 129.091 |

Se observaron 22 armas activas distintas. Hubo 71.404 filas de equipo sin munición aplicable, incluidas 54.106 de cuchillo, 1.249 de C4 y presencia real de utilidad. En esas filas se fabricaron cero valores de cargador, reserva o reload.

### Abstenciones

| Motivo | Filas |
|---|---:|
| Arma activa no disponible | 91.795 |
| Equipo sin munición aplicable | 71.404 |
| Tracker/callback de reload no disponible (`reload_callback_unavailable`) | 1.595 |
| Snapshot causal de reload no disponible (`reload_snapshot_unavailable`) | 1.595 |
| Reconciliación de evento ambigua | 0 |
| Sentinel de munición inválido | 0 |
| Otros | 0 |

Los dos motivos de reload describen las mismas 1.595 filas en capas distintas y pueden coexistir: el productor declara que no hubo tracker acreditado y el ledger declara que no pudo materializar el snapshot.

### Confirmaciones exactas por evento

| Hecho | `weapon_fire` | `weapon_reload_start` |
|---|---:|---:|
| Arma activa | 702 | 8 |
| Cargador | 581 | 8 |
| Reserva | 581 | 8 |
| Reload | — | 8 |

Las ocho confirmaciones de reload quedaron clasificadas como `confirmed_reload`. No hubo filas ambiguas. El validador independiente encontró cero disponibilidades posteriores a `T0`, cero municiones observadas negativas o no finitas, cero Steam IDs, cero nombres, cero campos oracle y cero rellenos desde eventos futuros. También encontró cero reloads observados sin tracker causal, con source distinto de `demoinfocs_causal_reload_tracker`, con identidad inválida o con `availability_tick`, `source_tick` y tick de sample desalineados.

## Artefactos del intento válido `r3`

| Artefacto temporal | Bytes | SHA-256 |
|---|---:|---|
| `observed_weapon_state_full_demo.json` | 2.578 | `a4a5758aaad448118c3cc55605c9296ae061d064f9228d143d4778688de6bde9` |
| `observed_weapon_state_rows.jsonl.gz` | 5.251.825 | `724a78955b17c1dc9ba8493e4aa9fd1c156a716d953dbcc3d02acd1a44be3931` |
| Stream lógico descomprimido | 345.322.718 | `d64b9b218c5efec1881e75b1185a5e0396e1aaa0e82dfacaf9701fc0a82c89f0` |
| `observed_navigation_full_demo.json` | 34.079 | `d7e8ce820fc90361842240dfcfc73a6b554e07b73984fa3e9834ebf86931c138` |
| `observed_navigation_full_demo_rows.jsonl.gz` | 3.346.479 | `fa0b3ffba4debf9398c3317f866e493cf94801346adc0c25f52ef854cb148a3c` |
| `observed_visual_contact_full_demo.json` | 41.762 | `174ac5d17aed2188b6319a437def22eb7a6037eb1b80f49d8cac0f36bb4d8c91` |
| `observed_visual_contact_full_demo_audit.jsonl.gz` | 8.788.325 | `6d923f254e6d3cb29cd9a10a3615dbb20d360b4c418bb7a8a4b803a9e50f7da7` |
| Manifest del bundle canónico temporal | 33.489 | `99c6ca71aeaffd74ace7e5d56ff68ef2db11420fd82e4e8343ed498739b33cab` |

El bundle canónico temporal continuó declarando 72 artefactos, export `3.8.0` y parser `v16`; su validación semántica directa devolvió cero errores. El candidato de arma permaneció como salida separada.

Las regresiones completas también pasaron: 28.718 filas de navegación, 28.277 alcanzables, 777.284 evaluaciones visuales, 2.992 episodios visuales, 99.690 intersecciones del smoke visual y 4.717 bloqueos finales por humo. La auditoría de decisión produjo dos veces el mismo SHA-256, `5792eb3ec3248461db8166224ac6f0a1445cc743f976645f77095f26a7b56491`, en 7.380 y 7.400 ms, con 918/918 joins, cero ejemplos persistidos y elegibilidad sin cambios: 0 `eligible`, 3 `ineligible` y 14 `not_materialized`.

## Verificación automatizada

| Comando | Resultado | Duración |
|---|---|---:|
| `go test -count=1 -run '^TestReplayReload' -v ./handlers` | pasaron callback ausente, true/false acreditados, entity+serial, refresh, muerte/respawn, nil, mismatch, futuro, reset y clip inválido | 635 ms reportados por Go |
| `go test -count=1 ./handlers ./models ./parser ./cmd/canonical-export` | todos los paquetes focales pasaron después del tracker causal | handlers 971 ms; models 966 ms; parser 1.642 ms; CLI 950 ms |
| Productor `r3 --ai-coach-full-demo-validation` | exit 0 sobre el mismo alias y SHA | aproximadamente 97.838 ms |
| `validate_weapon_state_full_demo` | stream completo, causalidad, identidad, source, schema, hashes, privacidad, abstenciones y conteos válidos | 13.119 ms |
| Smoke de navegación | pasó | 140 ms |
| Smoke visual | pasó | 528 ms |
| Validación full NAV/visual | pasó | 35.294 ms |
| Semántica canónica directa | 0 errores | 231.883 ms |
| Auditoría directa, dos ejecuciones | 918/918 joins y mismo digest; elegibilidad 0/3/14 | 7.380 / 7.400 ms |
| `npm run check:all` | frontend 88; Node 14 pasadas/3 omitidas; todos los paquetes Go; backend Python 43; build frontend correcto | 65.874 ms |
| `python -m pytest ai_coach/tests -q` | 100 pruebas pasadas | 2.295 ms |
| `python -m ruff check ai_coach` / `validate_contract` | sin errores / 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 6 schemas y 18 reason codes | 259 / 161 ms |

Exit code cero no fue el criterio de cierre: `r2` también había terminado con cero. Sólo `r3`, tras el recálculo independiente del stream físico y las regresiones canónica, NAV, visual y de decisión, se acepta como evidencia.

## Estado protegido antes y después

El preflight de `r3` se tomó a `2026-08-28T16:56:20.890847Z` y el postflight a `2026-08-28T17:06:49.9969115Z`. La comparación de metadatos y contenido dio `all_equal=true`.

| Árbol protegido | Archivos | Bytes | SHA-256 ordinal de metadatos |
|---|---:|---:|---|
| `Faceit-Demos/demos` | 100 | 13.589.310.951 | `b2e68292260f20c0edfa663acb4dd8d3fdcd78cc7acf5c02c1abd63b6a6081df` |
| `Faceit-Demos/analyzed_demos` | 3.013 | 27.264.064.316 | `57452310039c829854b1cd3a39b98b8b0f29268511a52e50df1a2c100ebf97e5` |
| `Faceit-Demos/failed` | 727 | 5.245.908.800 | `7003e3a0c8353be4642cd5f1556a3dfd2fff84ec7e4f72a3882b6c9c08c95f59` |
| `Faceit-Demos/staging` | 0 | 0 | `01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b` |
| `backend/data/exports` | 1.242 | 1.353.261.344 | `ec459e860895e8b16ab6cae4fbb0a0ef5d366fe3d0c6f52eface0d293cf5e466` |

Los 44 manifests raíz aprobados continuaron `passed`, `v16` y `3.8.0`; su árbol de contenido mantuvo 44 ficheros, 1.048.029 bytes y SHA-256 `a236d872c0272ade790c3af5fc688f71897147492596a6fceea7951281418571`. `Faceit-Demos` permaneció limpio en `main` y HEAD `f132c8de4bcc97650352ec5a647fdf3985f11353`.

La release activa siguió siendo `stratai-2c6db440231d463a`. `CURRENT` mantuvo SHA-256 `600d3f69631c12112fa2af11fdf2a23813529bfba536806ba19627db3a8ee799`; la release conservó 9 ficheros, 24.047.407 bytes, digest ordinal `c04d3a7be1bddb567efa38c43781ec4975e593b50d293fe0ae649753b0852310` y `RELEASE.json` con SHA-256 `fe62c434595d7eb29c6796cf85661039fe3fc8a3c7d1551d77e27a5ddcac5572`. Sus seis artefactos declarados siguieron válidos. El inventario estricto sólo permaneció en `false` por dos `.pyc` preexistentes, registrados con hash en la evidencia; esta fase no los tocó.

No se descargó ninguna demo, no se abrió un segundo match, no se reprocesaron las 44 partidas, no se escribió en corpus, exports, staging, failed, bundles aprobados, releases ni `CURRENT`, y no se promovió o publicó ningún artefacto temporal. El cleanup quedó completo y no conserva outputs grandes.

## Límites y siguiente blocker

El candidato aún no está materializado en ninguna de las 44 partidas aprobadas. Por tanto, `r3` demuestra productor, contrato, causalidad y comportamiento real sobre una demo, pero no autoriza migración canónica, reprocesado 44/44, entrenamiento ni inferencias sobre calidad de decisión.

Gate 1B permanece pendiente y `training_allowed=false`.

El siguiente blocker factual demostrado es la velocidad inicial causal del proyectil en eventos de utilidad. La fase sucesora queda limitada a ese hecho y sólo se deja preparada como prompt: no se ejecutó aquí. No incluye clasificación espacial, modelos, recomendaciones, corpus, release, promoción ni cambio silencioso de `3.8.0`.
