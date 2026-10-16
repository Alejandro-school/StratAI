# Utilidad visual estimada y deducción de lanzadores — v1

Encuadre: [alcance experto v0.2](../../ai-coach/AI_COACH_GRENADE_INFORMATION_SCOPE_v0.2.md)
y Excel v46/IC56. El estado y la siguiente acción pertenecen al plan canónico.
Esta extensión runtime opt-in no migra el corpus ni modifica export 3.8.

## Fuente y certeza

`stratai.calculated_utility_vision@1`, productor
`go.replay.utility_point_occlusion_v1`, registra observador vivo, bando,
objeto opaco, clase de señal, tipo como proxy, XYZ, tick/availability iguales,
callback/estado nativo, origen de ojos y base de posición. No contiene autor
oculto, reconocimiento humano ni bando inferido del lanzador.

`ParseDemoWithReplayUtilityInformation` habilita esta fuente y los contactos
calculados anteriores. Las entradas existentes conservan el opt-in apagado.
El lanzamiento se observa en su callback nativo; muestras de proyectil sólo
usan objetos nativos vigentes, cada cuatro ticks. Humo usa un punto representativo
desde inicio hasta expiración. Fuego vuelve a consultar celdas nativas activas;
no reobserva una celda histórica apagada. HE/flash aportan un punto en su
explosión, sin reconstruir fecha de lanzamiento.

Se requieren geometría cargada, posición finita, orientación y scope nativos.
Se aplican el FOV de referencia, paredes, humo y cegado existentes. Ojos pueden
proceder de un proxy de postura identificado. Falta de fuente abstiene, y no
prueba ausencia de señales. Build del mapa, forma/radio del humo, clasificación
visual del tipo, atención y percepción humana siguen sin validación independiente.
Ver un centro no acredita toda la geometría o eficacia del efecto.

El humo se enlaza por entidad nativa cuando falta la identidad del equipo.
Si faltan ambas identidades, se abstienen nuevas observaciones de utilidad
hasta el reinicio de ronda: una oclusión sin identificar no equivale a visión
despejada.

## Proyección y memoria

La proyección @10 conserva información @1 y añade
`stratai.replay_utility_information@1`; el input T nuevo es @8.
Opciones: `{information:true, utilityInformation:true}`. T/CT usan la misma
proyección; la lectura experimental T conserva su alcance anterior.

Las observaciones incluyen IDs, observador, objeto, tipo/clase, punto y fecha,
fuente, afiliación propia sólo por enlace positivo a un lanzamiento aliado,
base de receptores y posibles recepciones inmediatas. Los muertos presentes
pueden recibir información posible y no originan nuevas observaciones.
Se conservan por separado las últimas señales directas/compartidas por objeto
y las señales de lanzamiento necesarias para auditar la deducción. Un avance
o retroceso reconstruye el prefijo causal sin borrar informes originales.

`objects` fecha el último punto de cada objeto; sólo un tick idéntico admite
`current_estimate`. Un punto posterior no se extrapola. `segments` conserva
dos muestras positivas de proyectil del mismo observador/objeto separadas
cuatro ticks; el espacio entre puntos se etiqueta interpolado. Nunca se unen
huecos de oclusión ni se recupera una trayectoria final omnisciente. Mapa e
input consumen los mismos marcadores/tramos. El mapa usa rombos y trazos
discontinuos para distinguir puntos históricos de jugadores y efectos activos.
Los segmentos se ordenan por observador y objeto, independientemente del orden
de inserción antes de compactar sus muestras.

Callbacks repetidos del mismo objeto/tick con XYZ distintos se conservan como
una sola señal: `position_status=ambiguous_same_tick`, `position=null` y `points`
retiene coordenadas/fuentes originales. No se promedian ni se inventa otra
granada. El mapa abstiene su marcador único y la lectura explica la ambigüedad.
Una muerte en el mismo tick abstiene la observación por falta de orden más fino;
las muertes anteriores al inicio de esa instancia no contaminan su vida.

## Regla experta acordada

Tres **objetos distintos**, con **lanzamiento observable**, del **mismo tipo**,
en un máximo de **cinco segundos inclusivos entre primero y tercero** producen
un indicio estimado de al menos tres lanzadores. Tipos: HE, humo, molotov e
incendiaria, conservando su clasificación; no se mezclan. **Flashes excluidas**.
Una granada vista por dos observadores cuenta como un objeto. Explosiones,
avistamientos tardíos o ticks de nacimiento omniscientes no aportan ese reloj.
Sin tick rate finito positivo no se calcula una ventana en segundos.

Se publican objetos/evidencias, intervalo, receptores posibles comunes y certeza
`estimated_expert_deduction`. Recogidas o suministro adicional no están
descartados; el tipo es estimado. Grupos solapados no se suman como personas.
Bando, origen exacto y posición actual permanecen desconocidos. Un lanzamiento
aliado positivamente enlazado se excluye de esta deducción de señales de bando
desconocido. Una caída en B no demuestra tres CT en B; callouts de puntos
observados son referencias locales del mapa, no ubicaciones actuales del rival.

## Validación y límites

El consumidor Python valida campos, fuentes, fechas, vida, receptores, memorias,
segmentos y marcadores. Recalcula grupos, cinco segundos y excepción de flashes
independientemente del productor JS. La redacción factual protegida conserva
esas condiciones; el modelo sólo selecciona referencias/principios existentes.
Un texto «hay tres CT en B» no puede sustituir el texto fechado de la deducción.

Pruebas sintéticas cubren límites, duplicados, mezcla, muerte, fuente ausente,
futuro/oculto, reverse seek y huecos. El cierre requiere además una demo aprobada
con fuentes positivas y auditoría nativa independiente; no basta exit code cero.
Sonido, audibilidad, intención, eficacia y calidad táctica permanecen pendientes.
La comparación de información + granadas usa N01/N04/N10 y el mismo Qwen/configuración.
