# Interpretación del contexto y del plan colectivo

Diseño ICT-D1 v0.6, 17 de septiembre de 2026. Preparación documental autorizada por el usuario tras revisar el enfoque. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado, orden y siguiente acción. Esta especificación no crea un contrato runtime ni acredita una capacidad funcional.

## 1. Problema que debe resolver

El coach debe interpretar qué está intentando hacer un equipo a partir de su distribución y de la secuencia de acciones, relacionándolo con el contexto de la partida. Después podrá valorar las decisiones individuales dentro de ese plan probable cuando haya evidencia suficiente.

Un número de jugadores o una distancia no define por sí solo un acierto. Tres defensores en una zona pueden constituir un refuerzo coherente con patrones rivales previos, una apuesta o un reparto discutible. Un grupo que se dirige a B ofrece señales de preparación hacia B; los hechos posteriores permiten precisar si presiona, ejecuta, espera o cambia de objetivo.

La IA debe realizar esa interpretación. No se exige que el jugador declare primero un plan ni que complete todo el Excel. Una declaración, cuando exista, se registra como una fuente adicional atribuida y se contrasta con la conducta.

### Enfoque expresado por el usuario

- La valoración depende del contexto completo disponible de la partida y de la estrategia de ambos equipos.
- La distribución y el movimiento permiten inferir el plan que se prepara. El análisis debe incorporar esas señales.
- El historial puede explicar adaptaciones del reparto: reforzar una zona ante ataques rivales repetidos es un ejemplo contextual, no una regla numérica universal.
- La entrada puede desarrollarse en una zona o recorrido; no se exige un punto de cruce fijo universal.
- El ground truth táctico debe reflejar subjetividad contextual y alternativas defendibles.
- Revisar una parte concreta con el experto al llegar a su desarrollo y conservar sus anotaciones.

Las anotaciones originales de C026–C028 añaden explícitamente tiempo de ronda e información del equipo, con el ejemplo de conocer un reparto rival 2 en A / 3 en B. Esa información sólo se utilizará como conocida cuando su procedencia y disponibilidad lo permitan. La autorización «Hazlo» cubre esta especificación y las primeras fichas; no aprueba las respuestas tácticas propuestas en ellas.

## 2. Entrega y alcance

Esta preparación entrega:

1. Definiciones y campos conceptuales del contexto y las hipótesis de plan.
2. Método inicial para construir y actualizar interpretaciones.
3. Seis situaciones ficticias de Mirage, con tres cortes cada una: 18 fichas de revisión de desarrollo.
4. Preguntas y criterios de verificación separados de las respuestas humanas. La entrega v0.1 dejó las respuestas vacías; se han recibido después las 18 interpretaciones de desarrollo, documentadas en la sección 12, y las lecturas reales de F01-K1 v0.2 y F01-K2 v0.2, en las secciones 14 y 16.
5. Nueva versión del Excel con anotaciones previas preservadas y cambios localizados en entrada/apoyo, parámetros y desarrollo asociados.

El primer ámbito funcional que pueda derivarse de esta revisión será acotado por datos acreditados y por el enfoque experto. No se implementa un detector, clasificador, modelo, generador de recomendaciones ni entrenamiento en esta entrega. S2–S3 siguen abiertos. No se modifican IDs de capacidades, schemas, corpus, bundles, releases ni CURRENT.

## 3. Ubicación en el motor

Se integra en las [seis partes existentes](AI_COACH_MODEL_ARCHITECTURE.md):

~~~text
Observador: hechos con fuente, momento y perspectiva
  -> interpretación colectiva dentro de los especialistas
     contexto + hipótesis de plan + funciones + alternativas + incertidumbre
  -> especialistas de entrada, apoyo, rotación, objetivo y otras familias pertinentes
  -> Comparador: viabilidad, coherencia y preferencia, cuando estén acreditadas
  -> explicación y evidencia controladas
~~~

La interpretación es una hipótesis derivada. `FLD-INTENT` mantiene su condición de intención mental no observable. Inferir un plan no convierte intención, comunicación o atención en hechos extraídos. Tampoco crea un séptimo servicio ni exige un modelo separado por capacidad.

Un plan inferido puede ser incoherente o ejecutarse mal. La capa de interpretación debe conservar esas posibilidades; no inventar un motivo favorable para cada acción. En los juicios donde el plan no sea decisivo, no convertir su falta de identificación en un bloqueo global de otras capacidades.

## 4. Cuatro niveles que deben permanecer separados

| Nivel | Ejemplo | Qué lo respalda |
|---|---|---|
| Hecho | Tres CT ocupan posiciones de la zona B en el corte. | Observaciones válidas y clasificación espacial acreditada para esa afirmación. |
| Interpretación | El reparto es compatible con reforzar B. | Distribución y secuencia, con otras explicaciones conservadas. |
| Razón plausible | Puede ser adaptación a contactos tempranos previos en B. | Historial anterior disponible desde la perspectiva, sin afirmar el motivo mental real. |
| Valoración y consejo | Mantener o cambiar el reparto merece una recomendación. | Contexto, alternativas realizables, revisión y gates del Comparador. No se deriva automáticamente de identificar un plan. |

Reconocer conducta, explicar una posible razón y demostrar que una alternativa es mejor son afirmaciones distintas. No se usan bajas, victoria ni desenlace para aprobar retrospectivamente una interpretación inicial.

## 5. Contexto y perspectiva

### Tres vistas explícitas

1. **Equipo analizado:** distribución y acciones propias/aliadas. Puede describirse su conducta colectiva con fuentes válidas.
2. **Información para una decisión:** datos disponibles al actor o a los receptores registrados, con antigüedad y vía de disponibilidad. No asumir que cada jugador conoce automáticamente todo lo que supo cualquier compañero.
3. **Revisión retrospectiva completa:** la demo puede permitir describir ambos equipos. Esa visión se mantiene separada de las entradas que justifican una decisión del jugador.

Que un contacto esté registrado para CT1 no demuestra que sólo CT1 lo conociera. Sin una traza de transmisión, la comunicación y otros receptores quedan desconocidos; no se infiere silencio. Una premisa ficticia tampoco debe afirmar exclusividad sin una justificación mostrada.

La hipótesis del plan rival usada por el coach se construye desde la segunda vista. Tener posiciones rivales completas en la tercera no autoriza introducirlas en el contexto legal del actor. Tampoco es suficiente que un hecho ocurriera en una ronda anterior: se conserva qué pudo saberse de él y desde qué perspectiva.

### Variables de contexto propuestas

| Grupo | Contenido | Límite |
|---|---|---|
| Partida | Mapa/build, lado, marcador causal, fase, economía y horizonte. | Sin marcador final ni recursos futuros. |
| Historia dentro de la partida | Patrones previos por zona, ritmo, recursos y respuesta, con denominador y condiciones. | No generalizar automáticamente a partir de pocas rondas ni revelar información rival oculta. |
| Estado actual | Vivos, salud, equipamiento, utilidad, C4, relojes y disponibilidad. | Conservar ausencia, procedencia y resolución real. |
| Conducta colectiva | Distribución, trayectorias, orientación, ritmo, pausas y orden de acciones. | Separar dirección observada, destino probable y ruta física acreditada. |
| Información | Contactos/eventos, receptores, tiempo del hecho y disponibilidad. | Conocimiento agregado no prueba comunicación ni atención individual. |
| Plan probable | Objetivo local, zona/recorrido, organización y funciones posibles. | Ninguna cantidad fija de jugadores determina acierto o error. |
| Evolución | Cambios por contacto, baja, recursos, tiempo u objetivo. | La nueva hipótesis se registra en un corte nuevo. |

El historial previo entre partidas del jugador sigue bajo el contrato de memoria. Los patrones del rival en rondas anteriores de la partida actual pertenecen al contexto causal de esa partida; no se confunden ambos historiales.

## 6. Registro conceptual de interpretación

Los nombres siguientes son campos de diseño, no un schema ejecutable ni IDs nuevos del catálogo.

| Campo | Contenido requerido |
|---|---|
| Identificación | Caso/episodio, equipo analizado, actor cuando proceda, versión y corte. |
| Corte y continuidad | Tiempo/frame/tick disponibles y eventos hasta el corte; identidad/lado/vida y huecos relevantes. |
| Perspectiva y procedencia | Fuente, receptor, canal y disponibilidad de cada hecho o patrón utilizado. |
| Resumen factual | Distribución, movimientos, acciones, recursos e información directamente respaldados. |
| Hipótesis principales | Una o varias lecturas compatibles del objetivo y la organización colectiva. |
| Señales | Hechos que respaldan cada lectura y hechos que la cuestionan. |
| Funciones posibles | Papel local de cada jugador, alternativas y función indeterminada cuando corresponda. |
| Razones contextuales | Relación posible con historial, recursos y presión, sin atribución mental automática. |
| Desconocidos | Dato ausente y afirmaciones concretas que impide sostener. |
| Actualización | Hipótesis anterior, observación nueva y motivo de mantener, precisar, descartar o sustituir. |
| Revisión | Respuesta experta, razones, desacuerdo, versión y exposición a hipótesis o cortes posteriores. |
| Límites de uso | Descriptivo, hipótesis revisable o pendiente de valoración. Ninguna salida se transforma directamente en corrección. |

No publicar porcentajes de probabilidad hasta disponer de una calibración adecuada. Expresar las señales y ambigüedades en lenguaje claro, sin porcentajes inventados.

## 7. Desarrollo técnico propuesto después de la revisión

1. **Preparar prefijos de evidencia.** Leer sólo hechos disponibles hasta el corte, con trazabilidad y perspectiva. Los prefijos reales sólo se materializarán cuando sus fuentes estén acreditadas.
2. **Describir la organización observable.** Resumir distribución, dirección, ritmo, recursos y orden de acciones. Mantener vacíos donde falten fuentes.
3. **Proponer hipótesis limitadas.** Construir interpretaciones que expliquen señales concretas y conservar alternativas incompatibles entre sí cuando la evidencia no las resuelva. Las fichas sirven para acordar vocabulario y reglas iniciales, sin imponer una taxonomía cerrada.
4. **Relacionar antecedentes.** Identificar razones compatibles con historial y recursos. No afirmar una relación causal mental sólo por correlación.
5. **Actualizar por nueva evidencia.** Registrar cada versión. Permitir que aparezcan funciones o planes distintos al inicio de la ronda. Un futuro alternativo no cambia el registro ya emitido para el mismo prefijo.
6. **Revisar comprensión antes de prescribir.** Contrastar interpretación, razones y actualización con el experto. Después delimitar qué juicios podrían consumirla.

El baseline futuro será pequeño y auditable: resúmenes factuales y candidatos basados en criterios revisados. Un LLM podrá redactar o proponer hipótesis en desarrollo con hechos identificados, pero sus salidas deberán superar los mismos controles y no actuarán como juez táctico por sí solas. Los ejemplos redactados ahora por el agente no constituyen una ejecución de ese baseline ni una medición de un modelo. Decidir si hace falta aprendizaje corresponde a una fase posterior con soporte y autorización.

## 8. Primeras fichas

El [cuaderno de fichas](AI_COACH_COLLECTIVE_TACTICAL_REVIEW_CARDS.md) y las nuevas hojas del Excel contienen F01–F06. Cada caso tiene tres cortes K1–K3. Son **ficticios, de desarrollo y sin resultados reales**. Los estados estipulados no resuelven ninguna dependencia técnica.

El reloj de la base corresponde a K1. Los cortes posteriores sin reloj explícito indican tiempo actual no especificado; «continúa K1/K2» conserva antecedentes, no congela el reloj. F06-K3 sí fija 0:25 como nueva premisa ficticia. No inferir llegada o margen temporal a partir de un reloj anterior.

| Caso | Situación visible | Contraste buscado |
|---|---|---|
| F01 | Grupo T hacia apartamentos B. | Preparación, pausa y avance con utilidad; actualización progresiva. |
| F02 | Reparto CT con tres jugadores cerca de B e historial de contactos. | Distribución y razón contextual plausible. |
| F03 | Mismo reparto y secuencia con historial ausente. | Conservar descripción y retirar motivos no sustentados. |
| F04 | T repartidos entre A, medio y B. | Funciones posibles del jugador separado y convergencia. |
| F05 | Contacto parcial en A y nueva información del objetivo. | Perspectiva, motivo de una rotación y actualización sin reconstruir intenciones con el futuro. |
| F06 | Grupo hacia B con pérdida de jugador/recurso. | Pausa, retirada local, reorganización y margen temporal incierto. |

Los títulos sólo describen la situación. No contienen la intención correcta ni el desenlace. No se presentan guiones de «error» o de «acierto» ya resueltos.

### Proceso de revisión, una ficha cada vez

1. Presentar únicamente base y hechos del corte asignado, empezando por F01-K1. No mostrar cortes posteriores ni las hipótesis propuestas antes de la primera lectura del experto.
2. Recoger plan(s) plausible(s), señales, funciones posibles y desconocidos en palabras del usuario. No exigir un número fijo de alternativas.
3. Mostrar las hipótesis de diseño del agente y recoger correcciones, omisiones y equivalencias. Se puede rechazar la propuesta o considerar insuficiente la información.
4. Mostrar el corte siguiente y preguntar qué se conserva y qué cambia, con el motivo. No sobrescribir las respuestas previas.
5. Registrar qué información había visto el revisor y qué faltaba. No convertir una respuesta exploratoria en aprobación de todos los casos ni en test independiente.

El libro y el documento completos son materiales abiertos de desarrollo: permiten leer casos y propuestas futuras. Si el revisor ya los vio, registrar esa exposición. No describir esta revisión como ciega. Para una evaluación ciega futura habría que entregar únicamente el prefijo asignado, ocultando físicamente hipótesis, identidad de variantes y cortes posteriores.

## 9. Cómo verificarlo

| Comprobación | Procedimiento previsto | Resultado exigido |
|---|---|---|
| Hechos y trazabilidad | Contrastar cada hecho con fuente y disponibilidad; recálculo independiente donde sea práctico. | No rellenar ausencias con una historia táctica plausible. |
| Causalidad | Truncar o alterar el futuro manteniendo el prefijo; comparar el registro previo bajo configuración fijada. | La entrada y registro para ese corte no dependen del futuro. |
| Perspectiva | Ocultar datos rivales/compartidos no disponibles y conservar receptores. | Ningún consejo usa omnisciencia o comunicación supuesta como hecho. |
| Sensibilidad contextual | Comparar F02 y F03 con idéntico despliegue y distinto historial. | El motivo inferido cambia o pierde apoyo; la distribución observada se conserva. |
| Comprensión táctica | Revisión del experto sin desenlace, con razones y alternativas. | Lecturas defendibles y discrepancias explícitas; no se exige una única respuesta. |
| Actualización | Revisar los tres cortes y el cambio de hipótesis. | Cambios vinculados a hechos nuevos, sin reescribir el pasado. |
| Función y apoyo | Revisar jugador separado, utilidad, recursos y reparto. | Función probable separada de posibilidad física de ayudar. |
| Valoración | Exigir alternativa viable y mejor respaldada antes de corregir. | Identificar plan no demuestra conveniencia ni error. |
| Robustez | Casos ambiguos, contradictorios, sin historial o con huecos. | Hipótesis revisables y abstención localizada cuando sea necesaria. |

En este diseño no se ejecutan esas pruebas contra un motor inexistente ni se fijan umbrales tácticos nuevos. La revisión experta deberá concretar una rúbrica antes de implementar el evaluador. La evaluación formal mantendrá los conjuntos independientes, soporte y gates existentes del plan, separada de estas seis situaciones de desarrollo.

## 10. Datos existentes y dependencias

La [conciliación C0 de 3E.22](../archive/ai-coach/AI_COACH_SUPPORT_PREPARATION_JOINT_COVERAGE_PHASE_2026-09-16.md) conserva nueve contextos factuales en los cortes candidatos. Su cobertura no acredita plan, entrada táctica, contacto completo, apoyo o superioridad de alternativas.

| Componente | Punto de partida | Límite que conserva |
|---|---|---|
| Posiciones y acciones propias/aliadas | Observaciones publicadas y cortes C0 con procedencia. | Muestreo y disponibilidad específicos; la trayectoria no demuestra propósito. |
| Objetivo, C4 y relojes | Extensiones factuales acreditadas en sus variantes. | Transporte/caída, geometría legal y otros componentes conservan sus dependencias. |
| Movimiento y arma | Estados nativos parciales y plazos del pawn. | Velocidad media no reemplaza actual; plazo vencido no demuestra readiness. |
| Visión y rival conocido | Referencias visuales parciales y fuentes con límites. | FOV, humo, flash y contacto no se dan por completos. |
| Zonas, rutas y funciones | Callouts/proxies y reglas documentales propuestas. | NAV no acredita ruta física ni reparto táctico correcto. |
| Economía e historial | Hechos parciales y rondas previas que deberán filtrarse por perspectiva. | Operaciones itemizadas y reglas exactas pendientes; historial no es omnisciencia. |

La [matriz de dependencias](AI_COACH_SUPPORT_C0_DEPENDENCIES.md) y el plan rigen la prueba factual. La frontera orientada es un requisito del diseño heredado que debe revisarse al formalizar episodios contextuales; no se impone como definición universal de entrada. Cambiar esa definición exigirá su propia revisión versionada de contratos y validación, sin retocar evidencia histórica.

## 11. Cierre de esta preparación

La entrega documental se considera completa al existir especificación, seis situaciones con 18 cortes, preguntas y espacios de respuesta, Excel versionado con anotaciones preservadas, enlaces válidos y un único prompt de continuación. La evidencia de ese cierre se registra en el plan.

Se conservan las 18 interpretaciones libres de las fichas v0.1 y se ha recibido la lectura del corte real F01-K1 v0.2. El usuario confirma que puede localizar a todos los jugadores en este caso. Quedan pendientes la reparación de premisas y formato del conjunto, las 18 valoraciones de Hipótesis de diseño v0.1, el vocabulario/rúbrica final y el alcance de cualquier implementación. No se pide volver a aprobar el enfoque general ni repetir respuestas. La propuesta de F01-K2 se conserva en la sección 14 como antecedente. La recepción Q1 de la sección 15 aporta nuevos criterios; el plan canónico fija la siguiente acción.


## 12. Revisión experta recibida y formato v0.2

El [registro de recepción](../archive/ai-coach/AI_COACH_COLLECTIVE_TACTICAL_EXPERT_REVIEW_2026-09-17.md) conserva el significado y procedencia de las 18 respuestas y sus límites. F04-K3 y F05-K3 fueron objetadas por el experto como secuencias poco realistas; se conservan en el cuaderno v0.1 como historia y quedan excluidas como prueba táctica hasta reconstrucción o sustitución. El acuerdo previo sobre casuísticas y las interpretaciones libres no acreditan automáticamente las 18 hipótesis del agente.

### Presentación mínima de un corte

| Parte | Qué debe mostrarse | Ausencia y criterio de uso |
|---|---|---|
| Identidad y perspectiva | Mapa, ronda, lado, actor/equipo analizado y fuente del corte. | Nunca mezclar información rival omnisciente con la usada para juzgar al jugador. |
| Tiempo y continuidad | Reloj actual, orden de eventos y duración/transición desde el corte anterior. | No reutilizar el reloj de K1. Sin tiempo suficiente, no juzgar urgencia o rotación. |
| Jugadores propios/aliados | Una fila por actor: alias explícito, posición comprensible, orientación, movimiento, vida y recursos pertinentes. | Evitar rangos de alias, «cerca de B» o «acceso» cuando no permitan situar al jugador. Posición concreta puede ser un lugar reconocible o un minimapa respaldado; no se exige siempre una coordenada numérica. |
| Rival conocido | Contacto, posición observada o última conocida, antigüedad, C4 cuando se observó y receptor registrado. | «No consta comunicación» deja comunicación y demás receptores desconocidos. No afirma que el dato sea exclusivo. |
| Inventario | C4, flash y otras granadas, arma y munición relevantes para la pregunta. | Perder la única flash no acredita perder toda la utilidad. No completar inventario ausente. |
| Historia relevante | Frecuencia, desarrollo y resultado de rondas previas con condiciones y origen disponibles causalmente. | Un resultado anterior contextualiza la adaptación; no prueba una decisión óptima. |
| Economía y horizonte | Recursos actuales propios/aliados y escenarios para la siguiente ronda cuando se evalúe guardar o continuar. | No inventar compras futuras ni usar el resultado de la ronda actual. |
| Espacio y función | Acceso concreto, ángulos, zona abandonada y posibles funciones. | Separar posición/orientación observadas, función inferida y apoyo físicamente acreditado. |
| Pregunta y límites | Una pregunta de interpretación, señales visibles y datos que faltan para ese juicio. | Reparar material insuficiente antes de pedir una respuesta forzada. |

### Validación antes de volver a pedir una interpretación

1. Comprobar que se puede localizar a cada jugador mencionado y que el reloj pertenece al corte. Las expresiones «T1–T5» y «T1–T3» se sustituyen por enumeración individual explícita en nuevas fichas.
2. Verificar la continuidad de movimientos, contactos, bajas y objetivo. Un salto no explicado se trata como falta de datos, no como una prueba de comprensión táctica.
3. Mantener exactamente separados dato, desconocido e inferencia, especialmente en comunicación y roles.
4. Comprobar que historia, recursos y economía requeridos por la pregunta están presentes, o limitar la pregunta a lo que sí se puede interpretar.
5. Comprobar el formato con una sola ficha antes de reconstruir las siguientes. Este paso ya se cumplió para F01-K1 v0.2 con la respuesta de la sección 14; no implica que todas las fichas tengan datos suficientes. Mantener las respuestas v0.1 y registrar qué premisas cambian; una reevaluación no borra la primera lectura.

Priorizar material existente y aprobado de Mirage para evitar fabricar transiciones. Su disponibilidad concreta debe comprobarse en sólo lectura; la existencia de un replay no acredita automáticamente contacto legal, ruta o conocimiento individual. La preparación de ese corte no autoriza descargas, adquisición, nuevo parsing ni implementación runtime.

## 13. Primera ficha reparada: F01-K1 v0.2

La [ficha individual](AI_COACH_F01_K1_REVIEW_v0.2.md) sustituye el guion ficticio sólo para este corte. Usa la primera ronda de una partida aprobada, con mapa y cinco actores, reloj, inventario y cuatro segundos anteriores. No hay rondas competitivas previas que usar como patrón. Reloj y tenencia se corroboran en contextos ya retenidos; no se ejecuta el productor otra vez.

El radar ayuda a localizar actores. No certifica compatibilidad exacta de assets, líneas de visión ni contacto. Las etiquetas automáticas de zona ambiguas no se convierten en verdad. La diferencia entre valores de munición crudos y exportados se conserva sin resolver en esta revisión, que no juzga readiness ni apoyo físico. Estas limitaciones no se ocultan y no obligan a inventar otro caso para preguntar por organización observable.

La respuesta experta v0.2 está registrada en la sección siguiente. La ficha presentada se conserva: no se mostraron hipótesis del agente ni cortes posteriores en ella; el material de desarrollo previo ya fue abierto y esta revisión no se declara ciega. El [informe de preparación](../archive/ai-coach/AI_COACH_F01_K1_PREPARATION_PHASE_2026-09-17.md) registra cambios de premisas, procedencia y validaciones documentales.

## 14. Lectura experta de F01-K1 v0.2 y continuación acotada

Respuesta del usuario recibida el 17 de septiembre de 2026, conservada literalmente:

> Ahora puedo identificar cada jugador correctamente. Veo que se estan dirigiendo a B y que el plan es entrar en B. El T5 es el encargado de lanzar la utilidad del equipo y el resto de entrar, cubriendo a T4 que es el portador de la bomba. Mucho mejor esta información

El [informe de recepción](../archive/ai-coach/AI_COACH_F01_K1_EXPERT_REVIEW_2026-09-17.md) y su [evidencia](../evidence/AI_COACH_F01_K1_EXPERT_REVIEW_2026-09-17.json) fijan la versión y procedencia. La respuesta se incorpora en una fila nueva de [Excel v3](../../outputs/stratai-f01-k1-feedback-20260917-06ade91/StratAI_Plan_del_coach_ICT-D1_v3.xlsx), `Fichas tácticas` A24:I24, preservando el libro recibido y sus nueve columnas.

### Qué queda acordado

- La localización individual y el formato permiten a este usuario interpretar la organización de **este corte**. No es aprobación general de todas las fichas o de suficiencia para cualquier juicio.
- El experto interpreta un plan de entrada en B, T5 encargado de la utilidad y entrada del grupo protegiendo al portador T4. Son funciones e interpretación contextuales; no roles permanentes ni una plantilla obligatoria para otras entradas.
- La tenencia de C4 en T4 tiene su comprobación factual previa. «Cubriendo a T4» se conserva como lectura táctica del experto, sin convertirla en acreditación de visibilidad, comunicación o apoyo físico.
- La respuesta no juzga que la jugada sea superior, no demuestra que la entrada o el lanzamiento ocurran después y no completa las 18 valoraciones de Hipótesis de diseño v0.1, que siguen en Pendiente. No se infiere que no falte ningún otro dato por no haberlo enumerado.

### Propuesta posterior a K1, conservada como antecedente

La recepción de K1 propuso preparar sólo **F01-K2 real de la misma primera ronda y con los mismos alias**. El banco Q1 difirió su preparación; estos requisitos se conservan si una duda posterior necesita ese corte. Su finalidad es revisar qué se mantiene y qué cambia de la interpretación al aparecer nuevos hechos, no confirmar obligatoriamente la lectura inicial.

1. Leer registros existentes y seleccionar el primer cambio posterior relevante que esté respaldado —un lanzamiento propio si consta—. Registrar el criterio y el corte elegido. No saltar un cambio que contradiga la hipótesis para buscar después uno que la confirme ni presuponer que T5 lanzará.
2. Conservar referencia a K1, reloj actual, intervalo transcurrido, posiciones/alturas, estado e inventario de cada actor y C4. Separar los hechos anteriores al cambio, el cambio observado y los posteriores incluidos hasta K2. No mostrar el desenlace de la ronda ni información posterior a K2.
3. Mostrar sólo los datos necesarios para revisar la actualización, con perspectiva y desconocidos explícitos. Un lanzamiento no demuestra por sí solo efecto útil, coordinación acordada ni propósito mental previo. Si la continuidad o un hecho crítico faltan, señalar la carencia antes de pedir una lectura.
4. Preguntar qué se conserva y qué cambia de la interpretación de K1 y qué señales lo explican. Guardar esa futura respuesta vinculada a K2 sin reescribir K1.

Esta recepción sólo registra el acuerdo y ese siguiente paso. No se prepara K2, no se repara el resto de fichas ni se delimita una implementación. S2–S3, elegibilidad y dependencias factuales permanecen abiertas; no se ejecutan nuevos parses, runtime o entrenamiento.

## 15. Criterios Q1 y aclaraciones recibidas

El [informe Q1](../archive/ai-coach/AI_COACH_Q1_EXPERT_REVIEW_2026-09-17.md) y el [recibo literal](../evidence/AI_COACH_Q1_EXPERT_RESPONSES_2026-09-17.json) conservan 36 respuestas y tres aclaraciones posteriores. Se incorporan al diseño los siguientes criterios contextuales:

- **Plan dinámico:** los splits y ataques lentos pueden construirse con la información de la ronda. La continuidad habitual de un rush no obliga a toda ejecución a seguir un guion fijo. Historial rival, recursos y preferencias aportadas pueden explicar adaptaciones.
- **Entrada y control:** en el ejemplo de apartamentos B, avance colectivo, disparos y bomba apoyan una lectura de entrada; uno o dos mirando sin avanzar pueden estar buscando información. No son umbrales universales. Tomar una zona habitual rival aporta una posibilidad de control e información, con riesgo y cobertura defensiva que valorar. La línea imaginaria no define una frontera fija ni garantiza ausencia rival.
- **Funciones y cooperación:** posición, arma, utilidad y C4 contextualizan una función. Estar separado no acredita ser lurker. Una derrota puede deberse al equipamiento o al mérito rival. Revisar coordinación y alternativas antes de atribuir un error individual.
- **Información y rotación:** comunicación, cercanía, confirmación del ataque y antigüedad de la información influyen. Comunicación ausente en registros no equivale a silencio; cercanía no demuestra una orden ni el reparto correcto.
- **Utilidad y sincronización:** valorar posición de explosión, efecto sobre aliados/rivales y acción apoyada. Una flash posterior al asomo puede seguir siendo útil. Ni el orden temporal ni ganar el duelo prueban por sí solos calidad o causalidad. Duplicación requiere contexto espacial, temporal y de función.
- **Posplant T:** vigilar la bomba con distribución adaptada a los vivos; el usuario propone más cobertura con cinco vivos y sigilo/sorpresa en inferioridad. La respuesta inicial sobre desactivar se conserva separada de esta aclaración T.
- **Coaching:** priorizar impacto en la ronda y decisiones contextualizadas. Explicar con un ejemplo y proponer una misión ligada a una mejora respaldada. Recurrencia y progreso importan; niveles de dificultad y criterios de dominio siguen por concretar.
- **Alternativas y generalización:** conservar lecturas defendibles cuando no haya superioridad acreditada. Un porcentaje de éxito exige comparación válida e incertidumbre. Transferir fundamentos no acredita automáticamente su aplicación en otro mapa.

Las aclaraciones Q1, Q18 y Q26 están recibidas y no se vuelven a preguntar. Las respuestas sobre armadura, HE, equipamiento y roles se conservan como aportación experta: no sustituyen comprobación mecánica, datos causales o reglas exactas de la build. Las 18 hipótesis v0.1 siguen pendientes y F04-K3/F05-K3 siguen excluidas hasta reparar sus premisas. El Excel se versiona conservando sus anotaciones; no se implementa motor ni se cierra S2–S3.

## 16. Continuidad revisada en F01-K2 v0.2

Respuesta del usuario del 17 de septiembre de 2026 a la [comparación K1–K2](AI_COACH_F01_K2_REVIEW_v0.2.md), conservada literalmente:

> Mantengo la lectura anterior. Parece un push hacia B ya que se encuentran todos los jugadores avanzando

El [recibo](../evidence/AI_COACH_F01_K2_EXPERT_RESPONSE_2026-09-17.json) y el [informe](../archive/ai-coach/AI_COACH_F01_K2_EXPERT_REVIEW_2026-09-17.md) registran procedencia y contraste. Es continuidad de la interpretación de push/entrada B, con avance colectivo como señal. La lectura de K1 se mantiene por referencia; no se añaden funciones individuales ni nuevas razones para cada función.

El literal sobre todos los jugadores se conserva como lectura del conjunto. Los registros muestran avance de T1, T2, T3 y T4 en el intervalo; T1 repite XY al final y T5 no presenta desplazamiento horizontal resuelto. No se afirma movimiento simultáneo de los cinco ni se trata ese detalle como refutación automática del plan. No se exige que cambie la etiqueta por haber llegado otro corte.

El par aporta un caso de continuidad en desarrollo abierto, no una regla universal ante pausas, validación independiente o ejecución del baseline. Las fichas presentadas, las respuestas anteriores y las hipótesis pendientes permanecen versionadas. La respuesta no acredita intención mental, cobertura, efecto útil, corrección táctica o aprobación de toda la rúbrica.

El siguiente alcance propuesto es concretar el intérprete descriptivo temporal previsto en las secciones 6 y 7: prefijos comprobados, hipótesis con señales, posibles funciones, actualización y desconocidos. K1/K2 pueden ilustrar sus entradas y salidas, manteniendo la referencia humana separada de las entradas factuales. La preparación de alcance debe definir qué se necesita para contrastar ambigüedad y cambio antes de seleccionar otro corte. El plan conserva la decisión de ejecución; esta recepción no prepara K3 ni implementa un motor.
