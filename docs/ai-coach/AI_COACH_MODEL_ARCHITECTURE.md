# Arquitectura del motor del AI Coach

Referencia técnica propuesta v2.1, 2026-09-17. Describe el diseño, no su grado de implementación ni un backlog. La [definición de producto](../AI_COACH_PRODUCT_DEFINITION.md) fija la promesa y el [plan de implementación](../AI_COACH_IMPLEMENTATION_PLAN.md) es la única fuente de estado, orden, autorizaciones y siguiente acción. El [contrato de capacidades y datos](../AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md) y el [catálogo máquina](../../ai_coach/contracts/capability_catalog.json) conservan los IDs y contratos consumidos por software; este documento no los modifica ni acredita capacidades implementadas.

## Objetivo

Construir un coach personalizado que conecte decisiones de una partida con hábitos, práctica y progreso. Debe explicar qué alternativa era razonable con la información disponible, por qué encajaba con el jugador y qué puede entrenar para ejecutarla mejor.

Se conserva la promesa: no hay error sin una acción mejor respaldada. Una descripción de riesgo o una sugerencia de práctica no se presenta como prueba de que otra decisión habría sido superior.

La cadena completa debe poder probarse con reglas, casos revisados y ayuda humana antes de entrenar modelos. El prototipo incluye memoria y misiones acotadas desde el principio, con incertidumbre visible y sin presentarse como el coach integral. Cada familia se amplía según su evidencia; una cobertura parcial no acredita el conjunto de Mirage ni otros mapas.

## Seis partes del motor

| Parte | Responsabilidad | Límite |
|---|---|---|
| **1. Observador** | Reconstruir hechos, episodios, información disponible en T0, geometría e inventario; acreditar procedencia y elegibilidad. | No reconstruye intención, atención ni conversaciones ausentes de la demo. |
| **2. Especialistas** | Proponer acciones concretas y estimar consecuencias de economía, táctica, mecánica y utilidad, compartiendo hechos de objetivo, tiempo y equipo. | Una regla o un cálculo pueden ser la solución final; no se exige un modelo por capacidad del catálogo. |
| **3. Comparador** | Filtrar acciones imposibles, comparar ventajas y costes y elegir entre recomendar, equivalencia o abstención. Después prioriza hallazgos sin duplicar una misma causa. | No convierte ganar, matar o imitar a un experto en prueba automática de buena decisión. |
| **4. Memoria previa** | Resumir capacidad de ejecución, patrones y práctica de partidas anteriores. Alimentar la comparación de acciones y su prioridad. | No usa la partida evaluada como información previa ni afirma hábitos con muestras insuficientes. |
| **5. Planificador de práctica** | Traducir prioridades respaldadas en ejercicios o misiones medibles y revisar su evolución en oportunidades posteriores comparables. | Completar una tarea no demuestra por sí solo haber mejorado ni que la misión causó el cambio. |
| **6. LLM con herramientas** | Explicar evidencia, conversar, pedir contexto y adaptar ejercicios aprobados al usuario. | No determina por su cuenta qué jugada era mejor ni inventa hechos ni da por completada una misión. |

Son responsabilidades dentro de un sistema mantenible, no seis servicios independientes. La implementación puede compartir un proceso batch y el servicio web existente.

## Interpretación del plan colectivo

El [diseño ICT-D1](AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md) concreta una responsabilidad compartida de los especialistas: inferir planes y funciones probables desde distribución, secuencia, recursos e historial causal de la partida. La IA realiza esa interpretación sin exigir una declaración previa del jugador. El Observador conserva hechos y disponibilidad; la interpretación añade hipótesis, señales, alternativas, desconocidos y actualizaciones por corte.

El plan inferido y sus posibles razones se mantienen separados de intención mental observada, conocimiento individual y valoración de conveniencia. La información rival oculta del replay no entra en la perspectiva usada para juzgar al jugador. Un plan identificado no demuestra que sea bueno ni que sus funciones tengan apoyo físico viable. El Comparador mantiene sus requisitos antes de recomendar. No se crea un séptimo servicio ni se exige un nuevo modelo entrenado. Las fichas actuales son material ficticio de diseño, sin implementación acreditada.

## Dos circuitos conectados

### Análisis de una partida

~~~text
Demo y bundle validados
  -> observador: episodios + información legal en T0 + elegibilidad
  -> interpretación colectiva: contexto + hipótesis de plan/funciones + incertidumbre
  -> especialistas: acción observada + alternativas parametrizadas, con el plan cuando afecte al juicio
  -> filtro de física, reglas y soporte
  -> comparador, usando memoria estrictamente anterior
  -> recomendación, opciones equivalentes o abstención
  -> Evidence Pack + replay + explicación
  -> actualización de memoria después de cerrar el análisis
~~~

La acción observada debe poder reconstruirse y superar las mismas comprobaciones físicas que sus alternativas. Si no ocurre, se bloquea el episodio y se investiga la extracción o el contrato. Los parámetros de una alternativa se construyen desde T0; no se copian de una trayectoria futura realizada.

La comparación dentro de una decisión y la prioridad entre hallazgos son operaciones distintas. Una alternativa puede ser razonable y tener poca prioridad de entrenamiento; varios hallazgos pueden describir el mismo problema y deben agruparse.

### Mejora del jugador y del sistema

~~~text
Hallazgos respaldados + historial + preferencias del usuario
  -> habilidad prioritaria y misión concreta
  -> práctica y nuevas partidas
  -> observación de oportunidades, ejecución y progreso
  -> revisión de la misión y actualización de memoria

Casos con dudas + muestra aleatoria de control
  -> revisión humana de decisiones completas
  -> datos versionados, separados del test final
  -> evaluación de reglas o entrenamiento global offline, cuando se autorice
  -> comparación y posible promoción de una nueva versión
~~~

El primer circuito de mejora personal no necesita reentrenar el modelo global. El historial del usuario se actualiza bajo su contrato de privacidad; reutilizar sus partidas o respuestas para aprendizaje global requiere consentimiento y controles independientes. Un clic o una respuesta humana no actualizan un modelo en producción.

## Qué calcular y qué aprender

| Dominio | Cálculo y búsqueda de alternativas | Aprendizaje que puede aportar valor |
|---|---|---|
| **Economía** | Dinero, inventario conservado, compras legales, drops y unas pocas distribuciones viables del equipo. Comparar escenarios de la ronda actual y las siguientes. | Estimar consecuencias de esas configuraciones en contextos comparables, teniendo en cuenta capacidad personal e incertidumbre. |
| **Game sense, posiciones y rotaciones** | Distinguir contactos confirmados, información antigua y desconocidos; calcular rutas, obstáculos, exposición, apoyo y márgenes temporales. Generar mantener, esperar, reposicionar, reagruparse o rotar. | Estimar riesgo, apoyo, control y valor de alternativas o planes condicionados. Los compañeros y rivales pueden reaccionar; no son piezas inmóviles. |
| **Mecánica** | Hechos verificables de disparo, movimiento, arma, mira y contacto, sólo con la resolución y disponibilidad que soporte el contrato. Separar decisión y ejecución. | Comparar ejecución en situaciones equivalentes, estimar dificultad personal y detectar patrones estables. No inferir teclas, intención ni precisión que la demo no registra. |
| **Utilidad** | Inventario, lanzamiento posible, trayectoria, colisiones y efectos observados; comprobar las alternativas en el estado concreto. | Estimar oportunidad, coordinación y consecuencias tácticas. Una trayectoria legal no demuestra que lanzar fuera mejor que guardar. |

En economía, el planificador explora configuraciones razonables y escenarios de ganar, perder o conservar equipamiento. Las reglas calculan legalidad y recursos; no simulan exactamente el resultado de una ronda. Una propuesta que necesita cooperación se expresa como condición o acuerdo de equipo, no como acción que el jugador podía garantizar solo.

En táctica, las alternativas proceden de reglas revisadas, casos comparables y búsqueda espacial. Una acción recuperada se adapta y vuelve a validar: una jugada profesional puede exigir apoyo, utilidad o capacidad ausentes aquí. Los tiempos físicos son referencias internas. Que una referencia optimista no quepa en el margen puede descartar una acción; que quepa no demuestra que el jugador pudiera completarla entre combates y paradas.

## Cuatro clases de evidencia

| Evidencia | Qué acredita | Qué no acredita |
|---|---|---|
| **Hechos disponibles** | Qué consta en la demo y estaba disponible en T0, con fuente, versión y disponibilidad. | Conocimiento subjetivo o información comunicada fuera del registro. |
| **Resultados observados** | Lo ocurrido después de la acción realizada, con horizonte y censura. | Lo que habría sucedido al elegir otra acción. |
| **Preferencias tácticas humanas** | Qué opción prefieren revisores ante una decisión completa, sus razones, confianza y desacuerdo. | Un futuro alternativo observado ni una verdad indiscutible. |
| **Evidencia de práctica y progreso** | Qué se practicó, oportunidades posteriores y evolución medida. | Efecto causal de la misión sin una evaluación que permita atribuirlo. |

Las particiones observed, oracle y outcomes conservan separaciones explícitas. DecisionExample y TaskEligibility siguen las versiones de contrato vigentes; esta propuesta no añade campos runtime por sí sola. El estado de inferencia sólo admite información legal hasta T0; oracle y resultados posteriores pueden servir para etiquetas o evaluación, nunca para completar esa entrada. Las listas de campos permitidos son por tarea y un dato crítico ausente vuelve inelegible el ejemplo.

No se mezclan filas temporales, episodios y partidas como si fueran observaciones independientes. Cada estimación declara soporte por acción, partidas y jugadores, contexto comparable y limitaciones. Eliminar fuga futura no identifica causalidad: los resultados y la selección de acciones también dependen de habilidad, compañeros, intención y otras variables no observadas. La evaluación off-policy, si se usa, es evidencia secundaria en contextos con soporte suficiente; no elimina esos límites.

## Memoria y misiones desde el prototipo

La memoria empieza con estadísticas sencillas que se acercan a una referencia global cuando hay poco historial. La capacidad personal interviene **antes** de comparar acciones; la recurrencia y la prioridad se agregan después. Con una primera partida puede ofrecerse una práctica vinculada a un hallazgo respaldado, sin afirmar todavía un hábito estable.

Cada misión debe tener una habilidad concreta, razón de selección, ejercicio o conducta, dificultad ajustable, criterio observable de seguimiento y condiciones para revisarla. Se distinguen realización declarada, realización verificada y transferencia a partidas. Las respuestas del usuario y las métricas automáticas conservan su fuente; si una actividad externa no es observable, no se inventa su cumplimiento.

El progreso se mide sobre oportunidades comparables, con volumen e incertidumbre, no sólo por victorias, kills o una racha corta. Si faltan oportunidades se informa que aún no puede evaluarse. Para atribuir mejora a una misión se necesita un diseño de evaluación adicional; observar una mejora posterior sólo acredita una asociación.

## Evolución de la solución

| Opción | Función y condición para incorporarla |
|---|---|
| **Reglas, cálculos y casos revisados** | Base del prototipo y posible solución permanente. Los expertos ayudan a diseñar alternativas, comparar decisiones y seleccionar práctica. La ayuda humana se identifica y conserva su procedencia. |
| **Frecuencias y regresión regularizada** | Primera comparación estadística frente a reglas y referencias contextuales, con soporte e incertidumbre. |
| **Modelos tabulares, como CatBoost** | Alternativa para relaciones complejas si mejora consejos evaluados, calibración y cobertura frente al método sencillo. No se exige entrenar para avanzar una capacidad resuelta con reglas. |
| **Modelos de preferencias** | Comparar candidatas cuando haya decisiones completas revisadas. Se contrastan con consecuencias observadas y una fórmula auditable; el desacuerdo debe poder activar revisión o abstención. |
| **Modelos temporales** | Sólo si el orden de contactos, movimientos o coordinación cambia el significado y mejora de forma estable frente a un resumen del historial. Se prueba primero una solución pequeña; no se presupone un modelo de partida completa. |

Los hechos espaciales parciales y los formatos candidatos son instrumentos de desarrollo. No se suplen huecos con datos de otro lugar, otro instante o un mapa distinto. La revisión humana puede comparar una decisión cuyo contexto esté acreditado; no convierte geometría ausente en un hecho. La referencia sencilla se mantiene mientras otra opción no demuestre una mejora útil.

## Comparación, evaluación y explicación

El comparador conserva consecuencias separadas —objetivo, daño, supervivencia, apoyo, espacio, recursos, tiempo y dificultad— y sus limitaciones. Usa una fórmula revisable antes de necesitar un ranking aprendido. Si variar supuestos razonables cambia la opción ganadora, presenta condiciones, equivalencia o abstención. Ningún especialista declara un error sólo por perder una ronda.

La evaluación humana muestra información legal en T0, acción realizada y alternativas físicamente filtradas, sin revelar primero el desenlace. Permite preferencia, equivalencia, rechazo de todas las opciones o contexto insuficiente. La evaluación final incluye revisión independiente, desacuerdos y adjudicación; la ayuda experta cotidiana no sustituye una validación independiente.

Los ejemplos, reglas, modelos, calibradores, protocolos y assets se versionan. Se separan desarrollo, calibración y test final por partidas, se evalúa transferencia temporal y por jugador y se controlan ventanas solapadas. La calibración y los umbrales se fijan fuera del test final. Un benchmark usado repetidamente para seleccionar versiones pertenece a desarrollo: no conserva la condición de test final sellado. Los gates concretos y su orden se mantienen únicamente en el plan.

El Evidence Pack une hechos de T0, alternativas y filtros, resultados posteriores identificados, estimaciones o preferencias con su soporte, decisión del comparador, historial previo aplicable y referencias de replay. Las misiones enlazan esa evidencia y su criterio de seguimiento. El LLM consulta herramientas que entregan esta información controlada; distingue hechos, estimaciones y conocimiento general, conserva abstenciones y no cambia cifras ni decisiones. Si aparece contexto nuevo, solicita una revisión del motor en lugar de sustituir el juicio por texto libre. Una plantilla es un fallback válido.

## Implementación y recursos

Go produce hechos canónicos; Python bajo ai_coach/ construye ejemplos, compara opciones, mantiene memoria, evalúa práctica y, cuando proceda, entrena y sirve modelos. Node mantiene la orquestación y FastAPI expone resultados. Se priorizan módulos y almacenamiento locales; cada servicio adicional debe justificar su coste de mantenimiento. Los cambios de contratos, corpus y productores siguen los controles de compatibilidad y verificación del plan.

La CPU es el punto de partida. GPU, más anotación, adquisición u otros recursos pueden proponerse si una medición justifica mejores recomendaciones o menor coste total. El presupuesto flexible permite evaluar escenarios; no autoriza compras, servicios, descargas, entrenamientos ni cambios del corpus. Se contabilizan gasto externo, horas de revisión y desarrollo, almacenamiento, extracción e inferencia. Los límites, decisiones de inversión y permisos vigentes pertenecen al plan.
