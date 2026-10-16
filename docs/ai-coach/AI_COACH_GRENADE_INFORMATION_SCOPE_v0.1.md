# Información deducible de granadas — propuesta v0.1

Fecha: 11/10/2026. Documento de revisión experta, sin criterios nuevos aceptados
ni implementación. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md)
conserva el estado, los gates y la siguiente acción.

## Qué problema resuelve

Incorporar las señales que una granada puede aportar al jugador: utilidad que
aparece, recorrido observado, efecto y posibles orígenes o lanzadores. Mantener
separados el hecho, la información posible y la hipótesis táctica. Una granada
que cae en B no sitúa automáticamente a su lanzador en B.

Esta propuesta desarrolla el ejemplo ya recibido de **tres HE al comienzo de un
rush B**. Su pertinencia está recogida en el plan; el encuadre nuevo debe concretar
qué lectura justificarían sus condiciones y excepciones. No se atribuyen al
usuario las reglas propuestas aquí.

## Acuerdos que se conservan

- Comunicación inmediata simulada y receptores **posibles** presentes, incluidos
  compañeros muertos. Un muerto no origina nuevas observaciones visuales.
- Informes directos y compartidos por separado, con fuente, disponibilidad y
  antigüedad. No se demuestra voz, escucha, atención ni recuerdo efectivos.
- Contactos calculados como estimaciones; IDs opacos como enlaces de la demo,
  sin reconocimiento de personas ni movimientos percibidos por el mero ID.
- Posiciones históricas distintas de actuales; información nueva sin borrar
  informes anteriores ni crear rivales adicionales desde una granada anónima.
- Interpretación revisada tras el par **información + granadas**, con los mismos
  casos, modelo y configuración y todos los intentos conservados.

## Fuentes actuales contrastadas

La investigación parte del grafo del checkout exacto. El índice del 09/10
informa `metadata_changed` o `not_tracked` en las rutas relevantes; se contrasta
el código actual. El alcance es estas rutas y la captura final retenida de una
partida aprobada; no se declara una auditoría de las 44 partidas.

| Fuente | Qué aporta ahora | Límite para esta fase |
|---|---|---|
| `backend/go-service/handlers/replay_event_context.go` | Lanzamiento atómico, tipo, autor nativo, tick, disponibilidad, enlaces positivos de proyectil/efecto y abstenciones. | La identidad y posición nativas no prueban conocimiento del rival. |
| `backend/go-service/handlers/grenade_throw.go` | Observador nativo separado: lanzamiento, tipo, actor, posición y estado de fuente. | `observed` significa observado por el parser; no por un jugador. |
| `backend/go-service/models/replay.go` y `handlers/replay.go` | Posición XYZ de proyectiles en frames, historial XY, humo/fuego y geometría. | Historial XY sin fechas por punto ni observadores; no publicar el recorrido oculto previo a la primera observación. Comprobar vigencia nativa y fin de efecto, no asumir que una entrada cacheada sigue en vuelo. |
| `backend/go-service/analyzers/reaction.go`, `visibility_fov.go` y `replay_calculated_vision.go` | Estimación de contactos entre jugadores: vida, ojos, orientación/FOV de referencia, flash, humo y oclusión. | Los sujetos publicados son jugadores; no existe en estas fuentes una observación positiva de proyectiles/efectos por observador. Reutilizar geometría requiere producir y verificar esa evidencia nueva. |
| `frontend/src/features/replay2d/domain/replayEvents.js` | Utilidad propia/aliada con seguimiento colectivo simulado; lanzamiento rival si su actor tenía contacto calculado en el instante del evento. | Excluye geometría de utilidad enemiga y efectos enemigos. Ver al actor no autoriza todo el recorrido ni el efecto posterior. |
| `frontend/src/features/replay2d/domain/replayInformation.js` | Memoria y posibles receptores de contactos de jugadores. | No añade todavía observaciones de granadas ni hipótesis desde ellas. |
| `ai_coach/interpretation/replay_reading_context.py` | Validación causal y texto factual protegido, actualmente especializado en T. | La extensión de granadas debe compartir la proyección con el mapa y mantener la separación hecho/hipótesis. |

La [relectura de la captura retenida](../../outputs/stratai-grenades-framing-20261011/retained-capture-survey.json)
confirma **347 lanzamientos** dentro de los intervalos del replay: 92 humos,
106 flashes, 72 HE, 36 molotov, 40 incendiarias y un decoy; 184 T y 163 CT.
La fuente nativa completa tiene 348 registros: uno de R9 con tick **0** queda
fuera de su intervalo **58052–63877**, como ya exige el validador del cierre.
Se conserva esa exclusión, sin convertirla en un lanzamiento causal válido.
Los conteos de apariciones de proyectiles/efectos por frame repiten objetos;
no son granadas distintas ni lanzadores distintos.

Ejemplos reales del registro nativo, **sin acreditar quién los percibió**:

- R1/P01 lanza dos flashes en los ticks 8013 y 8096: separación **1,296875 s**.
  Dos flashes cercanas en el tiempo no equivalen automáticamente a dos jugadores.
- R11/P08 lanza dos HE en los ticks 72467 y 73853: separación **21,65625 s**.
  Contar HE a lo largo de una ronda no permite contar personas por igualdad.
  Estos campos no explican por sí solos cómo obtuvo la segunda HE.

## Alcance y criterios propuestos, pendientes del experto

1. **Registrar la señal antes de inferir.** Observador, instante, disponibilidad,
   objeto local, muestra de posición/recorrido/efecto realmente habilitada,
   fuente, carácter estimado y posibles receptores. Si falta la fuente, marcar
   desconocido; no rellenar con datos omniscientes ni declarar que nadie lo vio.
2. **Separar objeto, tipo y bando.** Un objeto visible puede tener tipo u origen
   desconocidos. El tipo del parser no acredita que el observador lo identificara.
   Proponer tipo estimado cuando una acción/efecto distinguible lo respalde.
   El autor/bando oculto no basta para llamarlo enemigo: conservar procedencia
   desconocida cuando no hay atribución respaldada. El seguimiento aliado ya
   simulado mantiene su etiqueta colectiva.
3. **Mostrar sólo el recorrido permitido.** Si la granada aparece tras una
   esquina, conservar desde la primera muestra observable y sus discontinuidades.
   No reconstruir puntos ocultos anteriores, unir saltos como recorrido visto,
   extrapolar automáticamente hacia atrás ni tratar un rebote como línea al origen.
4. **Distinguir lugar de efecto y origen.** El efecto permite describir utilidad
   en una zona estimada. Un tramo observado puede sugerir una dirección de llegada
   o región compatible; no entrega las coordenadas del lanzador. El origen sólo
   gana precisión con evidencia observable adicional, separada del impacto.
5. **Contar objetos sin contar personas por defecto.** Deduplicar el mismo
   proyectil observado por varios compañeros y separar lanzamiento de explosión.
   Un mínimo de lanzadores exige evidencia accesible de personas distintas —por
   ejemplo, varios actores estimados simultáneos realizando lanzamientos— o una
   restricción de juego causal verificada para ese caso. Los `thrower_id` ocultos,
   la ausencia de recogidas registradas y varias trayectorias no bastan por sí solos.
6. **Permitir hipótesis útiles con condiciones explícitas.** Proponer que varias
   HE identificadas como enemigas, casi simultáneas, al inicio del avance a B y
   con recorridos compatibles con su defensa apoyen «posible/probable defensa de
   varios jugadores en esa región». La etiqueta y las condiciones quedan para
   el experto; no introducir porcentajes, ventana de segundos o «tres CT en B»
   como regla universal. Si sólo se conoce la caída, limitarse a presión de
   utilidad en B, con número de jugadores y origen sin determinar.
7. **Conservar la fecha de la inferencia.** Una región compatible con un
   lanzamiento pasado no es una posición actual. Una nueva señal se añade a la
   memoria de utilidad y a sus informes posibles; no actualiza contactos visuales
   de personas ni se suma automáticamente al recuento público de vivos.

## Casos para la revisión y futuras regresiones

| Situación | Lectura propuesta y excepción |
|---|---|
| Tres HE identificadas como enemigas llegan casi juntas al comienzo de un rush B; se observan tramos compatibles con una región defensiva, sin ver CT | Señal pertinente de posible/probable defensa de varios jugadores en el origen compatible. La cifra mínima y la precisión del origen requieren respaldo adicional y encuadre. |
| Tres explosiones en B, separadas, sin observar los recorridos | «Se detecta utilidad en B; origen y número de lanzadores sin determinar». Si se conocieron sólo por sonido, esperar a la fase sonora. |
| Tres actores rivales estimados distintos lanzan en el mismo episodio observable | Lanzadores estimados distintos, con fuentes y fechas; usar los contactos existentes, sin crear tres contactos nuevos desde los proyectiles. Su ubicación posterior queda desconocida al perder contacto. |
| Dos flashes cercanas, o varias utilidades a lo largo de la ronda | No aplicar una correspondencia universal objeto/persona; considerar tipo, secuencia, límites verificados y posibles recogidas. |
| T1 y T2 observan el mismo objeto | Un objeto, dos procedencias; receptores posibles y memoria directa/compartida conservados. |
| Una HE aparece tras un rebote o sólo al final del vuelo | Recorrido parcial y dirección local; región inicial desconocida o hipótesis expresamente limitada. |
| Humo visible en B, lanzador oculto | Humo estimado allí; no localización del enemigo ni garantía de bloqueo total de visión. |
| Fuego afecta al acceso | Efecto/área estimados si la fuente permite observarlos; intención, autor oculto, daño final y eficacia táctica siguen sin inferirse. |
| Observador muerto, cegado o con fuente geométrica ausente | No inventar observación positiva. La recepción posible desde otro observador conserva el modelo acordado; ausencia de registro no prueba ausencia de información. |
| Granada propia atraviesa una zona sin enemigos observados | Seguimiento colectivo simulado ya existente; no demuestra que la zona esté despejada. |

## Dependencias y ejecución después del encuadre

**Primera entrega propuesta:** información visual **estimada** de utilidad;
sonido mantiene su siguiente fase. Reutilizar ojos/FOV, oclusión y humo actuales
con sus límites de build y proxy, sin afirmar percepción humana ni rehacer
callouts. El productor nuevo debe acreditar observaciones causales positivas de
proyectiles y, donde sea verificable, efectos; una extensión definida en una
dependencia o un test sintético no satisface el gate real. La observabilidad del
centro de un efecto no demuestra que toda su geometría sea observable.

Si una subfuente —tipo percibible, dirección u origen, geometría del efecto—
no queda respaldada, abstener ese campo y explicar la carencia. Una reducción
material respecto al encuadre debe volver al experto. No convertir los datos
canónicos de trayectoria/eficacia final en información disponible al instante.

Orden de tramos después de recibir respuesta:

1. Registrar literal/criterios y excepciones en este alcance y en una nueva
   versión del Excel; preservar v44, IDs, catorce hojas y comentarios.
2. Producir y validar observaciones causales de utilidad estimada, con identidad
   local vigente y abstenciones; mantener contratos y consumidores anteriores.
3. Integrar observaciones, receptores posibles y memorias de utilidad en la
   proyección común T/CT, mapa e input T con redacción factual protegida.
4. Añadir sólo las hipótesis y conteos respaldados por los criterios acordados;
   mantener condiciones y nivel de certeza, sin inventar contactos personales.
5. Verificar, cerrar granadas y después comparar información + granadas en
   N01/N04/N10 con Qwen/configuración fijados. Conservar éxitos y rechazos.

## Verificación prevista

- Regresiones con visión/oclusión/humo/flash, fuente ausente, objeto caducado,
  efectos con identidad ambigua, observación parcial y discontinuidades.
- Causalidad de observación y recepción, datos futuros/ocultos mutados, muerte,
  reinicio, cambio de lado, avance y retroceso; nada de observaciones retroactivas.
- Conteos recalculados de objetos y lanzadores respaldados, duplicados entre
  observadores, tipos desconocidos, recogidas no acreditadas y contraste entre
  caída/origen. Ninguna cifra táctica a partir de identidad oculta.
- Igualdad de mapa/input y validación del texto factual. QA de navegador sobre
  los nuevos estados visibles; preservar contratos, inputs y análisis históricos.
- Suites públicas pertinentes y una única demo aprobada por cierre funcional
  en TEMP nuevo, vacío, aislado y retenido, con productor cambiado. Repetir esa
  misma partida si falla; no seleccionar otra. Verificar registros afectados
  no vacíos, procedencia, esquema, fechas, semántica, abstenciones y hashes,
  con recomputación independiente cuando sea práctica.

Esta preparación documental no cambia productores, contratos runtime ni UI y no
ejecuta demos, suites funcionales o modelos nuevos. Revisa únicamente el enfoque
de granadas; no abre evaluación de eficacia, sonido, CT táctico general,
entrenamiento o promoción. El motivo para esperar el encuadre figura en
[AGENTS.md](../../AGENTS.md), «AI Coach phase collaboration».
