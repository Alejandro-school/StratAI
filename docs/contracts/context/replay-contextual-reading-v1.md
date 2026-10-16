# Lectura contextual — contrato local v1

Instrumento optativo `stratai.replay_contextual_reading@1`, adaptador `1.8.0-local.1`; el adaptador por defecto y los formatos históricos se conservan. Encuadre recibido el 16/10, [alcance](../../ai-coach/AI_COACH_CONTEXTUAL_INTERPRETATION_SCOPE_v0.1.md) y estado en el [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md).

## Entrada

Parte de `ReplayReadingSession` y sus guardas actuales. Mantiene situación, acontecimientos, límites y referencias completos. Añade explícitamente los hechos propios de posición, zona, objetivo y estado de ronda del inicio disponible, con sus fechas; pueden citarse sin consulta adicional. Todas las demás muestras permanecen íntegras en el archivo y accesibles por `sample(index)`, con catálogo completo de tiempos y límites originales. No se citan sin entregarlas ni se inventan puntos intermedios. La cobertura declara esa entrega inicial y muestras discretas; no acredita trayectoria continua, visión ni continuidad de una intención.

La primera implementación que empujaba todo el historial a la entrada produjo 234.956 tokens en N01 (241 muestras), por encima del presupuesto original. Se conserva esa medición negativa y se repara la entrega mediante consultas causales, sin descartar datos, elevar contexto o comprimirlos semánticamente. Las muestras existen y se pueden solicitar; no se presentan todas como ya leídas por el modelo.

La región B Apps se calcula sobre cada posición propia actual con zona correspondiente y conserva fuente, aproximación y build no verificada. Es una medición contextual: no hay catálogo de candidatos, combinaciones admisibles ni filtro geométrico de hipótesis. Un estado cerrado se describe como tal; no se convierte en una decisión activa por disponer de una posición.

## Decisión y representación

El sobre mantiene `action`, `indices` y `reading`. Una consulta tiene índices no vacíos y `reading:null`; terminar tiene índices vacíos y una lectura. Las secciones factuales seleccionan referencias y conservan todos los hechos obligatorios. `interpretation` es una única decisión:

- `status=hypotheses`: una a seis propuestas, cada una con `hypothesis`, `plausibility` (`possible`, `preferred` o `undetermined`), referencias `evidence`, referencias `counterevidence`, `limits` y `would_change`. No hay catálogo de textos tácticos. Hipótesis compatibles pueden coexistir. El texto no se completa con una etiqueta de la geometría.
- `status=insufficient`: ninguna propuesta y `missing_information` no vacío. La insuficiencia se muestra explícitamente. No hay otra selección que pueda declarar apoyo y ocultarlo en una lista vacía.

Las cadenas están limitadas; las referencias son únicas y deben haber sido entregadas. La selección contiene sólo referencias en las secciones factuales. Se rechazan cantidades escritas en las cadenas contextuales mediante dígitos o palabras numéricas comunes; `C4` y alias propios suministrados son identificadores, no cantidades. Esta guarda detecta la regresión «dos C4», pero **no demuestra la verdad factual de lenguaje arbitrario**.

La salida contextual se representa como **borrador del modelo pendiente de revisión**, con `semantic_status=unverified_model_inference`. Se muestran todos sus componentes y hechos citados con fecha. Los hechos protegidos siguen reconstruidos desde las referencias, separados del borrador. La verificación declara explícitamente `factual_integrity_scope=situation_evolution_uncertainty_only` y `contextual_factual_integrity=not_mechanically_established`; nunca presenta la nueva prosa como factual o tácticamente validada. Una inferencia sin números aún puede equivocarse o describir mal un hecho: la revisión humana y la evaluación independiente siguen siendo necesarias. No es un contrato de consejos al jugador.

## Ejecución y verificación

Modelo/digest/presupuestos originales; transporte reversible @3 y llamadas/consultas limitadas, sin reintentos ocultos. Las peticiones, respuestas y reconstrucción causal quedan reproducibles. Pruebas distinguen mismo estado final con antecedentes distintos, cobertura escasa, hipótesis compatibles, insuficiencia, referencias no consultadas, futuro, cifras, contradicciones del sobre y alteraciones del resultado. Los controles de esquema no exigen ninguna interpretación táctica concreta.

El smoke posterior ejecuta exactamente una demo aprobada y el productor contextual sobre su captura; recomputa historial/mediciones y comprueba no vacío, procedencia y causalidad. La comparación real es desarrollo asistido, no calibración ni evaluación táctica independiente. La aceptación mecánica no acredita la verdad del borrador, entrenamiento o promoción.

## Vista factual legible

`stratai.contextual_factual_overview@1` añade un tercer mensaje con textos reconstruidos por el presentador factual para ronda, objetivo, jugadores propios y zonas actuales. Un catálogo completo enlaza tipos y estados de los hechos actuales con sus IDs. Un extracto espacial copia las pertenencias calculadas y conserva sus límites; no añade una hipótesis. El segundo mensaje sigue conteniendo el contexto completo reversible, sin sustitución o recorte. Las pruebas verifican identidad de textos, referencias y cobertura del catálogo.

Esta entrega responde a un fallo observable: el modelo podía declarar inexistentes datos presentes y, en W01, negar la zona de P02 aunque el contexto la suministraba. No se afirma conocer su mecanismo interno. La vista legible permite revisar los hechos sin depender sólo de interpretar el transporte comprimido, manteniendo la procedencia y sin una lectura experta insertada como respuesta.

## Progreso y presupuesto de consultas

Una tanda posterior a reparar la gramática mostró repetición de consultas ya entregadas: N01 pide el corte actual repetidamente; N04 repite un lote incluso después de agotar consultas. Se conservan los cuatro resultados y sus fuentes anteriores a la reparación. No son abstenciones tácticas ni prueba de falta de comprensión del mapa: el protocolo no garantizaba progreso o finalización.

El esquema vigente sólo ofrece índices que puedan aportar hechos aún no entregados para el alcance solicitado. `consult_history` devuelve posición/zona propias, objetivo y ronda, con fechas y referencias íntegras; `consult` conserva la consulta completa. Un índice puede ampliar su alcance después, pero no repetir un alcance suministrado o intentado. Se mantienen los límites originales de operaciones, bytes y ocho llamadas; la última sólo permite finalizar con hipótesis o insuficiencia explícita.

Antes de entregar una respuesta de consulta se aplica una estimación conservadora de crecimiento basada en el conteo real del último prompt y bytes UTF-8 del nuevo intercambio, con margen de formato y reserva de salida. Si no cabe, se rechaza la entrega **entera**, se restaura la disponibilidad de citas y se envía `not_delivered_context_budget`; el intento queda registrado y no se repite. No se recortan hechos ni se aumenta el contexto. La estimación puede rechazar una respuesta que un tokenizador más preciso sí permitiría; se declara como límite técnico local, no como inexistencia de información. La reconstrucción offline reproduce ámbitos, intentos, rechazos y llamada final.
