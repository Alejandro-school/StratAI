# Contrato de capacidades y datos del AI Coach

Estado: contrato humano activo v1.0, 2026-08-27. Esta fase no autoriza entrenar, descargar demos, reprocesar, reemplazar datos canónicos ni desplegar cambios Go.

Jerarquía: la [definición de producto](AI_COACH_PRODUCT_DEFINITION.md) fija la promesa; el [plan de implementación](AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y orden; este documento explica el contrato humano; y el [catálogo máquina](../ai_coach/contracts/capability_catalog.json) fija IDs y referencias para software. El [esquema canónico 3.8](canonical-export-schema-v3.8.md) gobierna los bundles actuales. La [auditoría cuantitativa](AI_COACH_MIRAGE_44_DATA_AUDIT_2026-08-27.md) es evidencia fechada, no una fuente de progreso. Si este texto y el JSON divergen, se detiene el consumidor afectado y ambos se corrigen en la misma revisión.

## Resultado de la fase

El corpus local contiene 44 bundles Mirage aprobados y 6 exclusiones documentadas. Sirve para diseñar y verificar el contrato, pero `training_allowed=false`: todavía no constituye autorización para entrenar. La implementación existente aporta hechos de replay y algunas derivaciones causales; todavía no existe el pipeline futuro de datasets, especialistas, ranking, perfil histórico ni Evidence Pack.

El producto se divide sin ambigüedad en tres capas:

1. **Hechos observados:** lo que el jugador razonablemente podía conocer hasta T0, con `availability_tick <= t0_tick`.
2. **Evaluación:** outcomes futuros y estado oracle, usados sólo para etiquetas, métricas y análisis contrafactual prudente.
3. **Consejo:** candidatos físicamente válidos, ranking con soporte e incertidumbre y abstención. Un LLM sólo redacta evidencia estructurada; no inventa hechos ni decide por sí solo.

## Invariantes causales y de procedencia

- **T0** es el último estado legal anterior a la acción observada, no el instante posterior registrado por comodidad.
- `observed`, `oracle`, `outcomes` y `perfil_histórico` son particiones distintas. Oracle y outcomes nunca entran como features de T0.
- El historial permitido termina en T0. Las ventanas posteriores sólo crean etiquetas/outcomes.
- Todo campo consumido conserva productor, artefacto, ruta JSON, unidad, frecuencia y estado de disponibilidad.
- Los IDs de jugador, ronda, tick, episodio, decisión, acción y outcome se usan para joins; no son señales predictivas.
- Cada episodio solapado recibe deduplicación o peso explícito. Los splits se agrupan por match y, cuando exista identidad estable, por jugador.
- Una recomendación aprendida exige soporte mínimo, solapamiento entre acciones, calibración y prueba humana. Si falla cualquiera, se describe o se abstiene.
- `NO_OBSERVABLE` significa que una demo no puede probarlo: voz, intención, sonido subjetivamente oído y el futuro contrafactual real no se imputan como verdad.

## Los dos flujos

### Flujo 1: evaluación factual de una decisión observada

`bundle canónico -> reconciliación/eligibilidad -> detector de decisión -> snapshot legal en T0 -> acción observada -> outcomes posteriores -> especialista factual -> Evidence Pack`

Responde “qué ocurrió y qué señales legales había”. Puede puntuar ejecución y riesgo, pero no afirmar que otra acción era mejor sin pasar por el flujo 2.

### Flujo 2: candidatos alternativos y ranking prudente

`misma decisión/T0 -> generador de acciones parametrizadas -> filtro físico -> reglas -> modelos por especialista -> estimación de outcome/intervalo/soporte -> ranking común -> abstención o recomendación -> Evidence Pack`

El ground truth es compuesto: hechos del replay, resultados posteriores y revisión humana con adjudicación. Una acción alternativa no tiene “resultado verdadero” observado; se estima condicionadamente y se presenta con incertidumbre. Ray tracing sólo resuelve obstrucción geométrica puntual: no define por sí mismo zonas tácticas, rutas, coberturas, exposición, intención ni la decisión correcta.

## Registro común de decisiones, acciones y outcomes

Los 15 tipos `DEC-*`, 51 acciones `ACT-*` con parámetros físicos obligatorios y 18 outcomes `OUT-*` están enumerados en el [contrato máquina](../ai_coach/contracts/capability_catalog.json). El contrato distingue, entre otros, duelo/disparo, peek, posición, trade, entry, utilidad, economía, información, rotación, plant/defuse/retake, riesgo, clutch y perfil.

Toda acción alternativa tiene precondiciones concretas. Ejemplos: un peek requiere origen, destino, duración y exposición; una rotación requiere ruta alcanzable y tiempo de viaje; una utilidad requiere tipo, punto de salida, ángulos, fuerza y trayectoria; comprar requiere dinero y reglas de tienda. “Jugar mejor” no es una acción válida.

Los outcomes incluyen daño causado/recibido, kill, supervivencia, trade, tiempo a daño, cambio de exposición, control de mapa, utilidad efectiva, objetivo, ventaja numérica, economía a una/tres rondas, victoria de ronda/partida, espacio proxy y recurrencia/tendencia histórica. Todos declaran horizonte, unidad y si son sólo etiqueta. Ganar una ronda no se trata automáticamente como prueba de corrección de una decisión aislada.

## Fichas de capacidad

Cada fila hereda estas obligaciones completas: actor indicado y demás jugadores sólo como contexto factual; T0 legal; historial terminado en T0; futuro sólo como etiqueta/outcome; confusores de habilidad, spawn/plan, compañeros, utilidad, economía, política observada y voz/intención no observables; exclusión por warmup, actor sin reconciliar, T0 ausente, fuga futura, mapa sin assets, episodio duplicado sin peso o candidato no verificable. Las métricas comunes son fidelidad factual humana, validez física, métrica agrupada por match, calibración, cobertura/abstención y soporte efectivo por acción. El mínimo prescriptivo estándar es 200 decisiones y 30 matches por acción comparada, ESS >= 100, y una prueba ciega de 100 casos con dos revisores y adjudicación; una familia rara necesita tamaño prerregistrado y 50 casos de desafío. Alta confianza exige campos completos, slice soportado, intervalo calibrado y ausencia de OOD; media usa lenguaje condicional; baja sólo permite descripción. La evidencia al usuario siempre muestra ronda/T0, acción observada, candidato parametrizado, hechos de T0, consecuencias futuras marcadas, soporte/ESS/confianza, referencias al replay y límites.

| ID | Pregunta y comienzo/T0/final | Historia, acción observada y alternativas físicas | Observado / oracle / outcomes | Elegibilidad, método inicial y estado |
|---|---|---|---|---|
| `CAP-COMBAT-001` | ¿Fue razonable el duelo y cómo ejecutó disparo, ráfaga, spray, recarga o cambio? Empieza en `DEC-DUEL/SHOOT`; termina al romper contacto o cerrar ventana. | Estado físico/arma/equipo hasta T0; tomar/evitar duelo, parar/continuar/resetear spray, recargar/esperar. | Self 16 Hz, combate, shot-state, ammo, reloj / enemigo oracle sólo etiqueta / daño 2 s, daño recibido 2 s, kill 5 s, supervivencia 10 s, ronda. | Bloquea shot/ammo/T0 ausentes; reglas + frecuencias + regresión, CatBoost challenger; **PARCIAL**. |
| `CAP-CROSSHAIR-001` | ¿La mira estaba preparada y cuánto costó corregirla? Empieza antes de tiro/contacto; final en daño o ventana. | Ángulos y movimiento previos; mantener, corregir yaw/pitch, detenerse antes de disparar. | View angles, shot-state y proyección de error / enemigo oracle para etiqueta / tiempo a daño, daño, kill. | Bloquea proyección no verificable; reglas geométricas + regresión; **PARCIAL**. |
| `CAP-MOVEMENT-001` | ¿El movimiento ayudó o degradó la ejecución? Inicio en cambio de velocidad/stance; final al estabilizar o salir del episodio. | Velocidad, dirección, walk/duck y arma; parar, counter-strafe, caminar, agacharse o mantener desplazamiento con magnitud/duración. | Estado 2 Hz/16 Hz, stance, combate / enemigo oracle sólo etiqueta / daño, daño recibido, supervivencia. | Dirección es derivable, 16 Hz de velocidad no está materializada; reglas + regresión temporal; **PARCIAL**. |
| `CAP-PEEK-001` | ¿Debía asomar, mantener o abandonar y con qué geometría? Inicio `DEC-PEEK`; final al terminar exposición/contacto. | Trayectoria previa, posición, facing, compañeros y utilidad; hold/peek/disengage con origen, destino, duración, ángulo y ruta alcanzable. | Self/team, LOS parcial, facing, clocks / enemigo oracle / daño, kill, supervivencia y exposición. | Bloquea FOV/humo/ruta/exposición no verificados; reglas geométricas + CatBoost; **INSUFICIENTE**. |
| `CAP-POSITION-001` | ¿Era buena la posición y cuál era una recolocación válida? Inicio al ocupar/cambiar área; final al abandonar o cambiar fase. | Posiciones previas y objetivo; mantener/recolocar/usar cobertura/abrir salida con punto, ruta y tiempo. | Posición, área cruda, team y clocks / oracle enemigo / exposición, control proxy y supervivencia. | Zonas, cobertura, rutas y salidas requieren clasificación; reglas espaciales antes de aprender; **INSUFICIENTE**. |
| `CAP-TEAM-001` | ¿Mantuvo spacing, conexión y posibilidad de trade? Inicio `DEC-TRADE`; final en trade, desconexión o ventana. | Self/team, alive counts y conexión; conectar, espaciar, seguir, tradear o abortar con distancia/tiempo. | Team states, trade_possible, connection_time, causal trade / oracle sólo evaluación / trade 5 s, daño, supervivencia, ventaja. | Los trades existentes son verificables; alternativas de ruta aún no; reglas + regresión; **PARCIAL_AVANZADO**. |
| `CAP-ENTRY-001` | ¿El entry fue viable y habilitó al equipo? Inicio `DEC-ENTRY`; final en contacto, espacio/objetivo o muerte. | Estado de equipo, utilidad y objetivo; entrar, esperar apoyo/utilidad, cambiar choke o abortar con ruta y deadline. | Posición/team/utility/clocks / oracle / daño, trade, espacio proxy, objetivo, supervivencia. | Chokes/rutas/espacio necesitan clasificación o derivación; reglas + CatBoost secuencial; **PARCIAL**. |
| `CAP-UTILITY-001` | ¿La utilidad fue oportuna, físicamente válida y efectiva? Inicio `DEC-UTILITY`; final en detonation/effect window. | Inventario y movimiento previos; lanzar ahora/después, tipo/lineup/objetivo alternativo o guardar con posición, ángulos, fuerza y trayectoria. | Inventario, evento y trayectoria; in-flight parcial / oracle para afectados / efecto, daño, espacio y objetivo. | Evento/trajectory existen; inventario causal e intención táctica son parciales; reglas balísticas + modelos por tipo; **PARCIAL**. |
| `CAP-ECONOMY-001` | ¿Compra, ahorro o drop eran coherentes con dinero y horizonte? Inicio `DEC-ECONOMY` en freeze; final al cerrar tienda/ronda. | Dinero, equipo y resultados sólo de rondas previas; comprar/eco/force/drop/guardar con cesta legal. | Dinero y economía de equipo / sin oracle necesario / ronda, economía 1/3 rondas, partida. | Falta snapshot causal completo de freeze/inventario; reglas económicas + regresión jerárquica; **PARCIAL_FUERTE**. |
| `CAP-INFO-001` | ¿Actuó con información razonablemente disponible y suficientemente fresca? Inicio al recibir/perder contacto; final al actuar o caducar. | Contactos previos y edades; mantener, comprobar, comunicar-proxy, reposicionar o no sobreinterpretar. | LOS/contacto parcial, team / oracle sólo para medir error / daño recibido, control y objetivo. | Faltan FOV, humo, last-known e information-age; primero ledger observado; **INSUFICIENTE**. |
| `CAP-ROTATE-001` | ¿Debía rotar, mantener o fingir y podía llegar a tiempo? Inicio `DEC-ROTATION`; final en destino, reversión o deadline. | Posiciones, objetivo, reloj y contactos legales; hold/rotate/fake/reverse con origen, destino, ruta y ETA. | Posición/team/clocks/objetivo / oracle sólo evaluación / control, objetivo y ronda. | Faltan reachability, route y travel-time; reglas de pathfinding + modelo temporal; **INSUFICIENTE**. |
| `CAP-OBJECTIVE-001` | ¿Plant, defuse, tap, stick, retake o save eran legalmente viables? Inicio `DEC-PLANT/RETAKE`; final al resolver objetivo/ronda. | C4/kit, alive counts, clocks y trayectoria; plant/defuse/tap/stick/retake/save con sitio, duración, ruta y deadline. | Eventos/estado objetivo, kit, clocks y team / oracle sólo etiqueta / objetivo, supervivencia, ronda, economía. | Sitios/deadlines se deben materializar y verificar; reglas duras + survival model; **PARCIAL**. |
| `CAP-TIME-001` | ¿La acción respetó relojes y deadlines? Inicio al cambiar fase/deadline; final al vencer o completar. | Historia corta de relojes/posición; acelerar, esperar o abortar con deadline y ETA. | Round/bomb clock y objective state / oracle sólo evaluación / objetivo y ronda. | ETA/rutas faltan; reglas deterministas antes de modelos; **DERIVABLE_TRAS_RUTAS**. |
| `CAP-ADVANTAGE-001` | ¿La decisión preservó o desperdició ventaja numérica/material? Inicio al cambiar alive/economía; final en próximo cambio o ronda. | Alive counts derivables, equipo y objetivo; aislar, agrupar, tradear, evitar o convertir con spacing/route. | Team/self, combate, economía y clocks / oracle etiqueta / delta numérico, daño, supervivencia, ronda. | Alive counts derivables; causalidad multijugador limitada; reglas + regresión jerárquica; **DERIVABLE**. |
| `CAP-RISK-001` | ¿El riesgo estaba justificado por recompensa, tiempo y economía? Inicio ante decisión irreversible; final al resolver exposición/objetivo. | Estado T0 y contexto; asumir, reducir, transferir o abandonar riesgo con ruta/deadline. | Campos de especialistas y economía / oracle etiqueta / daño recibido, supervivencia, ronda y economía. | Sólo agrega especialistas elegibles; calibrador/ranker, no señal independiente; **INSUFICIENTE_ESPACIAL**. |
| `CAP-CLUTCH-001` | ¿La secuencia 1vN fue viable sin usar futuro como entrada? Inicio al quedar clutch; final ronda. | Historial observado desde inicio de clutch; aislar, reposicionar, usar utilidad, objetivo o save con tiempos/rutas. | Self, contactos legales, utilidad, objetivo, clocks / oracle sólo outcome / daño, kill, objetivo, supervivencia, ronda. | Requiere ledger de información y rutas; reglas + modelo temporal sólo con soporte raro; **PARCIAL_RARO**. |
| `CAP-PROFILE-001` | ¿Qué patrón estable aparece en partidas estrictamente anteriores? Inicio al cerrar una partida; T0 es inicio de la partida evaluada. | Hallazgos/contexto/ejecución de matches previos; practicar, mantener o vigilar un patrón con periodo y umbral. | Sólo registros históricos versionados / nunca oracle del match actual / recurrencia y tendencia. | No existe store histórico ni identidad estable autorizada; estadística jerárquica, sin prescripción hasta revisión; **FALTA_IMPLEMENTAR**. |

El detalle no abreviado de cada ficha —incluidos todos los confusores, exclusiones, umbrales, niveles de confianza, abstenciones y tests— está materializado por capacidad en el JSON y se comprueba con `python -m ai_coach.contracts.validate_contract`.

## Matriz de disponibilidad y procedencia

Estados válidos: `EXISTE_VERIFICADO`, `DERIVABLE`, `PARCIAL`, `FALTA_EXTRAER`, `REQUIERE_CLASIFICACION` y `NO_OBSERVABLE`. El inventario completo tiene 55 campos: 26 verificados, 3 derivables, 7 parciales, 10 por extraer, 5 por clasificar y 4 no observables.

| Grupo | Estado | Procedencia verificada o trabajo necesario | Uso permitido ahora |
|---|---|---|---|
| Match, participantes, rondas, score | EXISTE_VERIFICADO | `core/*.json`, Go `ExportCanonicalBundle`, Python `CanonicalMatch` | joins, contexto y splits |
| Posición, view, velocidad 2 Hz, stance, vida/armor, reloj | EXISTE_VERIFICADO | `states/player_states/round_*.jsonl` | hechos T0 con frecuencia declarada |
| Self/team tactical 16 Hz | EXISTE_VERIFICADO | `states/tactical/observed.jsonl.gz` | posición, facing, vida, arma, utilidad y dinero; respetar `availability_tick` |
| Enemigo completo | EXISTE_VERIFICADO/ORACLE | `states/tactical/oracle.jsonl.gz` | sólo etiquetas/evaluación |
| Combate, utilidad, objetivo | EXISTE_VERIFICADO | `events/*.jsonl`; trayectorias de utilidad incluidas | episodios y outcomes |
| Economía | EXISTE_VERIFICADO/PARCIAL | `derived/economy_*.json`; falta snapshot causal de freeze completo | descripción; no decisión prescriptiva completa |
| LOS/contacto | PARCIAL | raycast de StratAI; `enemy_los` no aplica de forma demostrada FOV ni humo | geometría auxiliar, nunca “lo vio” |
| FOV, humo, last-known, edad de información | FALTA_EXTRAER | demoinfocs ofrece base parcial; falta ledger causal propio | bloquea información/peek prescriptivos |
| Dirección de movimiento, alive counts, espacio proxy | DERIVABLE | diferencias temporales/estado de equipo/futuro aliado etiquetado | derivar con versión y tests |
| Zona, choke, sitio, cobertura, exposición, salida, ruta | REQUIERE_CLASIFICACION | nav/mesh y callouts no aportan semántica táctica suficiente | bloquea alternativas espaciales |
| Reachability, path distance, travel time | FALTA_EXTRAER | hace falta grafo navegable y pathfinding versionado | bloquea ETA/rotación |
| Perfil previo | FALTA_EXTRAER | futuro store con matches estrictamente anteriores | no usar aún |
| Voz, intención, sonido subjetivo, futuro contrafactual | NO_OBSERVABLE | una demo no lo prueba | declarar limitación; no imputar |

## Auditoría espacial y del parser

### 1. Qué entrega directamente demoinfocs v5.2.0

El parser expone ticks/frames y estado del juego; jugadores con posición/punto de ojos, yaw/pitch, vida, armor, dinero, armas, kit, walking/scoped/ducking/blinded/spotted; arma activa y munición; flags de reload/defuse/plant/flash; eventos de disparo, daño, kill, granadas/flash/smoke/fire, bomba, ronda e inventario; proyectiles e infernos. Son hechos del replay, no conceptos tácticos de StratAI.

### 2. Qué calcula StratAI hoy

`ParseDemoWithReplay` registra handlers y analizadores; `ExportCanonicalBundle` escribe core, eventos, estados, tactical, engagements, trades, economía, calidad y particiones causales. `MapManager` aporta mesh/nav/callouts y `IsVisible` hace raycast. `BuildTacticalExport` genera observed/oracle; `buildCanonicalCausalPartitions` produce decisiones actuales de peek/trade. El repositorio Python consume artefactos por manifest; Node sólo orquesta `/process-demo` y metadatos.

Hallazgo crítico: el `enemy_los` causal usa visibilidad geométrica; no hay prueba de que aplique FOV ni oclusión por humo. Existe una heurística con FOV/flash/humo simple, pero no está conectada al stream táctico. Por eso LOS y contacto son `PARCIAL`, no “visibilidad razonable”.

### 3. Qué es derivable sin nueva verdad externa

Dirección/aceleración desde posiciones/velocidad, alive counts desde estado de equipo, ventanas de episodio, deadlines simples desde relojes y un proxy de espacio desde posiciones aliadas futuras guardado exclusivamente en outcomes. Toda derivación necesita versión, unidad, test de borde y test de invariancia futura.

### 4. Qué exige clasificación táctica propia

Jerarquía de zonas, sites/chokes, rutas nominales, cobertura, ángulos de exposición, salidas y roles contextuales. Mesh, nav areas y callouts son insumos; no son esas etiquetas. Deben vivir en assets versionados por mapa, revisados por humanos y con casos positivos/negativos.

### 5. Qué no puede saberse con fiabilidad desde la demo

Lo comunicado por voz, intención, plan mental, qué sonido percibió realmente cada persona y qué habría ocurrido en un futuro alternativo. Sólo pueden aparecer como variables latentes/limitaciones, nunca como hechos.

## Tests y puertas obligatorias

Cada capacidad exige fixtures positivo, negativo e indisponible; invariancia al futuro; separación oracle; deduplicación; validez física del candidato; y abstención. Además:

- El contrato JSON debe parsear, tener IDs globalmente únicos y referencias válidas.
- Un test de mutación posterior a T0 no puede cambiar ninguna feature observada.
- Quitar FOV/humo/ruta debe volver inelegible la tarea afectada, no rellenarla silenciosamente.
- Todo candidato rechazado debe registrar la precondición física fallida.
- Métricas se reportan por match, acción, mapa/side/fase y slice de soporte; nunca sólo por filas.
- La prueba humana separa fidelidad factual, utilidad, accionabilidad, prudencia y calibración percibida.

## Plan versionado de migración del golden

Este contrato v1 fija el plan, pero no lo ejecuta:

1. conservar `3.8.0`, su release y `golden-demos-v2` sin cambios como referencia reproducible;
2. definir una versión canónica nueva sólo cuando estén aprobados los hechos P0, con schema/lineage/validador y release nuevos en el mismo cambio;
3. hacer dry-run por una demo hacia un directorio nuevo, comparar invariantes y validar rollback;
4. reprocesar 44/44 una a una únicamente tras autorización, sin sobrescribir el corpus actual;
5. publicar inventario, hashes, exclusiones, quality gates y compatibilidad de consumidores;
6. promover el candidato Mirage sólo con Gate 1B y firma humana; hasta entonces `training_allowed=false`.

No se intenta recuperar las fuentes borradas de `golden-demos-v2`. La migración cambia la referencia mediante una versión explícita; nunca reinterpretando silenciosamente `3.8.0`.

## Criterio de salida de este contrato

La siguiente fase puede construir datasets y validadores en una ubicación nueva y versionada, pero no entrenar. Para abrir entrenamiento deben estar verdes: procedencia/T0, aislamiento oracle, elegibilidad por tarea, soporte por acción, splits sin fuga, assets espaciales requeridos y protocolo humano. El corpus dorado antiguo conserva linaje técnico, no autorización ni fuente local que haya que recuperar.
