# Cierre de contacto visual causal observed — 28 de agosto de 2026

## Resultado

La subfase 1B.2 está completa en código. StratAI ya tiene un ledger factual y causal que sólo afirma contacto visual cuando coinciden tres pruebas disponibles en T0:

1. el raycast del mapa está libre;
2. el objetivo está dentro del FOV del observador;
3. ningún humo activo tapa el segmento visual.

Si falta geometría, posición de ojos, orientación, zoom o snapshot de humo, el contacto queda en `null/unavailable` con un motivo de abstención. Un LOS geométrico aislado nunca se convierte en «lo vio».

Esto no modifica el corpus actual. El contrato nuevo `stratai.observed_visual_contact@1` es candidato y está separado de los bundles canónicos `3.8.0`, donde `enemy_los` conserva su significado histórico de proxy geométrico. `training_allowed=false` y Gate 1B sigue pendiente.

## Qué se implementó

- Contrato Go y schema JSON versionados para evaluaciones y episodios de contacto.
- Frustum de FOV determinista compartido con el analizador de reacción existente.
- Raycast mediante el contrato mínimo `TacticalGeometry`/`MapManager.IsVisible`.
- Oclusión por humo mediante cilindro 2D versionado, porque el replay actual publica centro X/Y y radio, no Z.
- Captura en memoria de posición de ojos, procedencia, orientación y zoom sin añadir campos a los JSON de replay ni reinterpretar `3.8.0`.
- Evaluaciones positivas, negativas e indisponibles a 16 Hz, con `availability_tick <= tick <= t0_tick`.
- Boundaries `started`, `continued`, `ended`, `inactive` y `unavailable`; episodios abiertos en T0 o cerrados con inicio/fin causales.
- IDs limitados a `join_keys`; no se exportan nombres ni estado completo enemigo.
- Validador fail-closed que vuelve a calcular la fórmula `raycast && FOV && !smoke` y rechaza disponibilidad posterior a T0.

No se añadieron etiquetas good/bad, recomendaciones, rutas, zonas, cobertura, exposición, intención, voz, sonido subjetivo, modelos ni candidatos de acción.

## Pruebas causales

Las 9 pruebas Go nuevas, con 7 subcasos, cubren:

- dentro y fuera del FOV;
- raycast libre y bloqueado;
- humo entre observador y objetivo;
- geometría, orientación o humo ausentes;
- disponibilidad posterior a T0 rechazada;
- mutación futura sin cambio del ledger;
- mutación de campos oracle sin cambio del ledger;
- IDs sólo para joins;
- determinismo ante orden, nombres y orden de humos distintos;
- inicio y fin de contacto;
- regresión explícita: LOS geométrico sin FOV no prueba contacto visual.

No se usaron las 44 partidas como sustituto de estas pruebas.

## Contratos y disponibilidad

El catálogo máquina sube a `1.1.0`:

- `FLD-FOV-CONTACT`, `FLD-SMOKE-OCCLUSION` y `FLD-VISIBLE-CONTACT` pasan a `DERIVABLE` por tener productor y pruebas;
- `FLD-LINE-OF-SIGHT` sigue `PARCIAL` para los bundles `3.8.0`;
- ninguna ruta de artefacto nueva se atribuye a los bundles antiguos;
- ninguna capacidad pasa a `eligible`.

El inventario de 55 campos queda en 26 `EXISTE_VERIFICADO`, 6 `DERIVABLE`, 6 `PARCIAL`, 8 `FALTA_EXTRAER`, 5 `REQUIERE_CLASIFICACION` y 4 `NO_OBSERVABLE`.

## Verificación

| Comando | Resultado | Duración |
|---|---|---:|
| `go test -count=1 -v ./parser` | 112 pruebas pasadas; 1 gate de demo real omitido por no solicitar corpus | 3,601 s de pared; 1,357 s Go |
| `npm run test:go` | Todos los paquetes Go pasaron | 5,444 s |
| `python -m ai_coach.contracts.validate_contract` con el venv del backend | 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 55 campos, 4 schemas y 18 reason codes válidos | 0,113 s |
| `python -m pytest ai_coach/tests -q` con el venv del backend | 23 pasadas | 0,979 s de pared; 0,55 s pytest |
| `npm run test:python` | 43 pasadas | 4,708 s de pared; 3,47 s pytest |
| `npm run lint:python` | Sin errores | 3,070 s |
| `python -m ruff check ai_coach` con el venv | 4 avisos preexistentes en `ai_coach/audits/bundle_audit.py`, archivo no modificado por esta fase | 0,213 s |
| `git diff --check` | Sin errores de whitespace; sólo avisos LF→CRLF | 0,060 s |

El lint amplio de `ai_coach` es informativo y no forma parte del script público actual. No se corrigieron esos cuatro avisos porque pertenecen a un archivo ajeno a esta fase y había que preservar cambios existentes.

## Estado preservado

Comprobación de solo lectura antes de implementar:

| Área | Archivos | Bytes |
|---|---:|---:|
| `Faceit-Demos/demos` | 100 | 13.589.310.951 |
| `Faceit-Demos/analyzed_demos` | 3.013 | 27.264.064.316 |
| `Faceit-Demos/failed` | 727 | 5.245.908.800 |
| `Faceit-Demos/staging` | 0 | 0 |

La comprobación final conserva exactamente esos valores. Faceit-Demos continúa sin cambios de Git. No se descargó ni reprocesó ninguna demo, no se entrenó ningún modelo, no se creó una release y no se modificaron `CURRENT`, staging, Docker, manifests, bundles ni cuarentena. La release activa documentada sigue siendo `stratai-2c6db440231d463a`.

## Lo que sigue bloqueado

Para usar este hecho con las 44 partidas hará falta una migración canónica y un reprocesado aislado expresamente autorizados; esta fase no los adelantó. Peek sigue además sin exposición, cobertura y salida. Trade y tiempo conservan el siguiente blocker factual medido: reachability, path distance y travel time.

La fase siguiente fue [reachability, path distance y travel time](AI_COACH_NAV_REACHABILITY_PATH_ETA_PHASE_2026-08-28.md); se ejecutó después y conserva este cierre como antecedente.
