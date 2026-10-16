# Información disponible: propuesta para revisión experta

Fecha: 10/10/2026. Propuesta original del agente tras «Perfecto. Vamos con la siguiente
fase», con el encuadre posterior recibido y separado abajo.
El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la
siguiente acción. Esta ficha concreta el alcance para la revisión de AGENTS.md.

## Encuadre recibido el 10/10/2026

Literal del usuario: «Manten el modelo inicial. Eso si, ten en cuenta el que es
posible que tenga la información.» Se acepta el modelo inicial de comunicación
inmediata simulada y se exige distinguir quién puede disponer de cada dato.
Los criterios y ejemplos de la propuesta quedan encuadrados para implementación.
La propuesta original se conserva debajo como procedencia.

Concreción técnica dentro de ese alcance: el observador queda identificado;
los compañeros presentes al quedar disponible el contacto figuran como
posibles receptores. No se afirma escucha, atención, recuerdo ni voz real. Una
muerte posterior impide nuevos contactos visuales propios y permite recepción
compartida posible. La primera concreción del agente excluía a receptores muertos;
esa suposición se corrige durante la revisión técnica antes del cierre, sin
atribuirla al usuario. Se preservan por separado últimos informes directos y
compartidos: una recepción incierta no sustituye lo visto por el observador.
Las muestras colectivas de recursos aliados siguen
rotuladas como vista de equipo, sin convertirlas en conocimiento individual.

## Problema y base existente

La vía actual de replay por equipo publica contactos visuales recientes, con
fuente y fecha. Agrupa rivales por identidad, pero no entrega una memoria
explícita que separe todos sus observadores y la comunicación posterior. La
entrada Python permite consultar hechos anteriores; esto no equivale a presentar
una última observación consolidada por jugador y equipo en cada corte.

Fuentes contrastadas: `createTeamReplayProjector` en
`frontend/src/features/replay2d/domain/teamReplayProjection.js` y
`ReplayReadingSession` en `ai_coach/interpretation/replay_reading_context.py`.
El grafo identifica ambas rutas, pero informa cambios de metadatos respecto al
índice; se contrastó el código actual. No se afirma una auditoría exhaustiva.
El replay admite T/CT; la lectura contextual actual está especializada en T.

## Criterios ya recibidos

- Conservar quién pudo observar qué, cuándo y por qué fuente; una observación
  calculada sigue siendo estimada y no acredita percepción humana efectiva.
- Perder contacto no borra la observación ni confirma la posición actual.
- Separar observación individual y comunicación de equipo simulada. No atribuir
  conversaciones reales sin fuente; revelar contactos al bando ya está acordado.
- Conservar muertes comunicadas como historia, sin convertirlas en rivales vivos.
- Compartir entre mapa e IA la información permitida por perspectiva y corte.
  Los datos ocultos del parser no se convierten por ello en conocimiento.

## Alcance propuesto

1. Registrar contactos por observador, rival e instancia de ronda, con instante
   de observación, disponibilidad, fuente y carácter estimado/acreditado.
2. Derivar la información individual y la compartida del mismo registro. Mostrar
   quién originó cada contacto y por qué otro jugador puede recibirlo. Esto no
   acredita que un jugador conozca todos los recursos de sus aliados.
3. Separar contacto actual de última posición observada. Conservar esta última
   durante la ronda con antigüedad, sin caducidad táctica arbitraria, movimiento
   supuesto, radio creciente ni afirmación de que una zona está despejada.
4. Una nueva observación actualiza el último contacto; dos compañeros viendo al
   mismo rival no crean dos enemigos. La muerte conocida termina su estado de
   contacto vivo y conserva el suceso histórico. El reinicio de ronda separa las
   memorias; retroceder el replay reconstruye sólo el prefijo permitido.
5. Incorporar estos hechos a la validación y redacción factual cerradas. La ruta
   común se comprobará en T/CT; extender la interpretación táctica CT no forma
   parte de este bloque. Mantener los contratos e inputs históricos intactos.

## Única decisión nueva propuesta

**Modelo de comunicación entre jugadores:** mantener la visión conjunta del
bando y concretar que un contacto se comparte de inmediato como *comunicación
ideal simulada*. Para quien lo origina sería observación; para sus compañeros,
información compartida simulada. No añadir retrasos ni omisiones inventados.

Ejemplo: T1 obtiene un contacto calculado de CT2 en corta. T1 registra la
observación estimada; T2 puede recibirla en el mismo instante sólo bajo ese
supuesto explícito de comunicación. Esto no afirma que T1 haya hablado ni que
T2 lo haya visto. La alternativa es limitar la perspectiva individual a sus
propias observaciones y reservar la agregación para una vista colectiva.

La distinción y las etiquetas ya están acordadas; el alcance de esa transferencia
inmediata a la perspectiva individual es la propuesta que necesita encuadre.

## Ejemplos y excepciones que se comprobarán

| Situación | Resultado esperado de la propuesta |
|---|---|
| T1 pierde un contacto en corta hace 8 s | «Última observación en corta hace 8 s»; posición actual desconocida. |
| T1 y T2 observan al mismo CT | Un rival, dos procedencias conservadas. |
| CT2 muere después de ser visto | Muerte histórica; no sigue figurando como contacto vivo. |
| Muere T1, que había visto a CT2 | Su observación pasada se conserva; T1 no genera nuevos contactos. |
| Falta una muestra de visión | No renueva la última observación ni demuestra ausencia del rival. |
| El parser conoce a un CT nunca observado | No se filtran identidad, posición, recursos ni movimiento ocultos. |

## Dependencias y verificación prevista

Reutilizar contactos calculados, identidad de ronda, bajas y relojes ya reparados.
La precisión de la visión sigue estimada; no hay fuente de voz que demuestre
comunicación real. Granadas y sonido conservan sus fases posteriores y no se
añaden inferencias de origen, identidad o audibilidad en esta fase.

Tras recibir el encuadre: registrar el literal y los criterios en las filas
pertinentes de una versión nueva del Excel, conservando v43, IDs y anotaciones.
Implementar por tramos: registro causal, proyecciones, integración mapa/entrada
y guardas de salida. Probar pérdida/reobservación, muerte, reinicio, búsqueda
hacia atrás, contactos simultáneos y diferencias entre información individual y
compartida. Mutar datos futuros u ocultos no debe alterar la salida permitida.

Ejecutar las suites públicas correspondientes y el gate de exactamente una demo
aprobada en TEMP nuevo, vacío, aislado y retenido, después de las pruebas. Validar
registros afectados no vacíos, fechas, procedencia y conteos con recomputación
independiente cuando sea práctica. No seleccionar otra demo si falla. La revisión
de interpretación se mantiene por pares: información y granadas, con encuadre
y cierre técnico separados. La preparación documental inicial no ejecutó nuevas
pruebas, demos ni modelos; la ejecución posterior se registra por separado.

## Ejecución posterior y cierre técnico — 10/10/2026

El alcance aceptado está implementado con proyección @9, input @7 y memoria @1,
últimos informes directos/compartidos separados y receptores presentes posibles,
incluidos muertos. Excel v44/IC54 registra el literal preservando v43, catorce
hojas, estilos y comentarios. La fase pasa 2.522 pruebas Python, 284 frontend,
suites Go/Node y el smoke de una única partida aprobada con tres capturas
aisladas y retenidas. El proyector se contrasta en 46.796 vistas T/CT y los diez
inputs suman 6.840 vistas. La inspección de Chromium y la exportación común
pasan; cero llamadas nuevas al modelo.

Comparar capturas detectó una dependencia del orden de callbacks en el productor
previo de acontecimientos. Creación y captura se unificaron para no perder
lanzamientos/enlaces; 347 lanzamientos contrastados con el observador nativo
separado, sin ampliar a deducciones desde granadas enemigas. El alcance original
y los intentos anteriores se conservan. [Resultado y ejemplos](../../outputs/stratai-information-20261010/RESULTADO.md).
La siguiente acción y los límites de CS3 siguen en el plan canónico.
