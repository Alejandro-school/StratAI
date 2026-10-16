# Evaluación del intérprete temporal: protocolo de selección y revisión

Protocolo v0.16 del 28 de septiembre de 2026. El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md)
mantiene el estado y la siguiente acción. Este protocolo concreta TP10; no
sustituye los mínimos del piloto, los gates de publicación ni la evaluación de
recomendaciones. El usuario acepta «Hazlo» en respuesta a evaluar primero otros
jugadores del mismo periodo. El [recibo](../evidence/AI_COACH_TP10_SCOPE_ACCEPTANCE_2026-09-24.json)
conserva literal y contexto. La [selección de tres partidas](../../outputs/stratai-tp10-pilot-20260924/frozen-selection.json)
queda congelada antes de parsing, predicciones y etiquetas nuevas.

## 1. Alcance y condiciones comprobadas

Evaluar `collective_c4_approach@2-temporal-provisional`, con filtro de contradicción
de 1 s, sobre otras partidas reales de Mirage. Se mantienen IC19 e IC24:
organización compatible con preparar una entrada, funciones posibles justificadas
por hechos propios/aliados, posibilidad conservada durante pausas con datos
actuales y revisión ante contradicción. No se vuelve a consultar R1/R2/K1/K2.

La [auditoría de metadatos](../../outputs/stratai-independent-evaluation-20260924/cohort-audit.json)
comprueba los 44 bundles aprobados. Contiene 311 participantes con identificadores
válidos, representados mediante aliases locales sin conservar su correspondencia
de identidad. Las siete partidas/84 episodios ya utilizados son desarrollo.

| Propiedad de las otras 37 partidas | Resultado |
|---|---:|
| Sin participantes compartidos con las siete de desarrollo | 13 |
| Con uno o más participantes compartidos | 24 |
| Posteriores a la última partida de desarrollo | 0 |
| Parejas de candidatas que comparten participantes | 9 |

El corpus abarca del 24/08/2026 a las 02:56:51 UTC al 25/08/2026 a las 09:01:42 UTC.
La última partida también pertenece a desarrollo. Las fechas proceden de los
metadatos observados originales de FACEIT y están vinculadas por checksum; no
se han vuelto a consultar al proveedor. No se usan fechas de parsing como
fechas de partida. Se verifica la existencia de las 44 demos originales; sus
bytes no se han releído para calcular SHA en esta auditoría de metadatos.

Las 44 partidas aparecen en una auditoría agregada anterior. Esto no demuestra
que alguien haya revisado tácticamente sus episodios. La exposición privada no
registrada sigue siendo desconocida. Cero cruces de jugadores tampoco acredita
separación de organizaciones, coaches o estilos: esa información no está
verificada. No se declara independencia absoluta.

La consulta recibida distingue dos alcances:

- **Otros jugadores del mismo periodo, aceptado:** un piloto retrospectivo, separado por
  partida y participante respecto al desarrollo conocido, con las limitaciones
  de exposición registradas. Puede detectar errores fuera de las siete partidas;
  no acredita generalización hacia partidas posteriores.
- **Partidas posteriores, validación futura pendiente:** no hay candidatas en las fuentes
  aprobadas actuales. Mantener el protocolo preparado y concretar la fuente
  admisible posterior; esta propuesta no autoriza adquirir ni descargar demos.

## 2. Selección técnica antes de etiquetas y predicciones

Se acepta el alcance del mismo periodo con **tres partidas**, procesadas
secuencialmente en destinos TEMP nuevos. La extracción y materialización siguen
siendo secuenciales; una [enmienda técnica de planificación](../../outputs/stratai-tp10-pilot-20260924/validation-scheduling.json)
permite solapar un proceso adicional de validación de sólo lectura, con un
máximo inicial de dos procesos. Terminada toda la generación, una
[segunda enmienda](../../outputs/stratai-tp10-pilot-20260924/validation-scheduling-2.json)
permite hasta tres verificaciones de sólo lectura simultáneas, tras comprobar
la memoria disponible. No cambia la selección, los parámetros ni los artefactos
congelados. No es una estimación representativa de
toda la población ni un mínimo suficiente para promoción. El procedimiento es:

1. Excluir las siete partidas conocidas y cualquier partida con participantes
   compartidos con ellas. Ordenar las 13 restantes por SHA-256 ascendente.
2. Recorrer ese orden, aceptar una partida sólo si no comparte participantes
   con las ya aceptadas y detener el piloto en tres. No optimizar el grupo según
   salidas del detector, resultado de la ronda, bajas o lectura humana.
3. Antes del procesamiento, registrar la respuesta de alcance y congelar el
   manifiesto de selección, demos y hashes de código/configuración/mapa. Si una
   demo falla, retener el fallo y repetir esa misma entrada tras resolverlo;
   no sustituirla por una que produzca resultados favorables.
4. Mantener fuera de nuevos ajustes todas las partidas seleccionadas y aquellas
   de la reserva que compartan participantes con ellas. Si se usa su evaluación
   para modificar reglas, umbrales, presentación o referencias, registrar la
   exposición y reservar otro test para evaluar la nueva versión.

La selección congelada resultante es:

| Orden | Alias opaco | Fecha UTC |
|---|---|---|
| 1 | `demo-0beb9110ec408f80c39b` | 24/08/2026 21:38:54 |
| 2 | `demo-1028fce95708dda309ae` | 24/08/2026 15:35:43 |
| 3 | `demo-5224a45934f2d5b94c01` | 24/08/2026 21:14:11 |

Sus 30 participantes son distintos entre sí y de los de desarrollo. La
recorrida completa admite ocho partidas mutuamente disjuntas, sin afirmar que
sea el máximo posible. Las otras cinco no se incorporan automáticamente al lote.
Los checksums completos y el censo de cruces están en la auditoría.

Para seleccionar episodios, reutilizar la política técnica reproducible de
12 cortes T por partida: ventanas de 65 muestras continuas de fase `playing`,
no solapadas, espaciadas uniformemente en el listado de ventanas elegibles.
La elegibilidad usa sólo reloj, secuencia, lado, fase, ronda e identidad de
instancia; no planes, destinos, resultado, posiciones o aciertos. Congelar los
cortes antes de ejecutar el detector. Con menos de 12 ventanas elegibles,
conservar y comunicar la insuficiencia, sin cambiar la partida.

Para evaluar persistencia, cada corte incluirá el prefijo disponible desde el
inicio continuo de su ronda/fase de juego hasta ese instante, no sólo cuatro
segundos de movimiento. Registrar explícitamente reinicios y límites de
continuidad. Si se supera el límite contractual de 4.097 muestras, no truncar
silenciosamente ni alterar la selección: registrar el caso no materializable y
resolver la limitación antes de evaluarlo. Los prefijos de una misma ronda se
solapan y nunca cuentan como observaciones estadísticamente independientes.

### Enmienda de alcance recibida el 25/09: rondas de cuchillos

El usuario identifica E01 como ronda de cuchillos y pide omitir ese tipo de ronda
del análisis táctico. [Literal y procedencia](../evidence/AI_COACH_KNIFE_ROUND_EXCLUSION_2026-09-25.json).
La [cola activa ajustada](../../outputs/stratai-knife-exclusion-20260925/active-review-scope.json)
excluye E01 de revisión/puntuación tácticas. Se conservan los 36 casos, sus hashes,
predicciones y primera lectura: no se borra ni se reemplaza el episodio.
Recuento al aplicar IC27: 36 seleccionados originalmente, 1 excluido por alcance, 35 pendientes
de contexto y revisión, 0 lecturas activas puntuadas. Los 35 no se declaran
automáticamente rondas competitivas válidas.

La decisión llega después de la selección, predicción y primera lectura, y tras
mostrar la abstención y contexto adicional al experto. Es un ajuste de alcance
declarado, no una selección prefijada ni una mejora demostrada de precisión. La
clasificación de E01 procede del experto, no de un campo nativo de ronda. Las
respuestas de plan originales permanecen intactas y fuera del detector.

El criterio prospectivo excluye rondas identificadas fiablemente como cuchillos.
Inventario desconocido, ausencia de C4, cuchillo equipado o ronda inicial no
bastan por sí solos. La clasificación conservadora y el bloqueo de desconocidos están implementados;
la identificación positiva de cuchillos en replay real sigue pendiente de evidencia suficiente;
conservar motivo, evidencia y casos de tipo no acreditado, sin fabricar etiquetas.
No usar la exclusión como acierto de abstención ni incluir estos episodios en
objetivos tácticos, recomendaciones o práctica. No se autoriza entrenamiento.

## 3. Ficha ciega y respuesta humana

Las fichas originales usan `review_packet` v1. Preparar nuevas vistas con
`review_packet_v2`, conservando las originales y su vínculo de IDs, una vez
congelados los cortes: secuencia reproducible, minimapa rotulado, vivos/muertos,
posiciones disponibles, arma/munición, utilidad vigente y cantidades cuando
estén acreditadas, portador o estado conocido del C4 y ausencias explícitas.
Las etiquetas espaciales conservan el límite de asset/build pendiente; no
acreditan por sí solas una ruta ni la percepción rival.

Cada ficha termina en su corte. No incluye futuro, desenlace, posiciones rivales
ocultas, nombres identificables o predicción del detector. El revisor recibe los
cortes en orden temporal, cerrando su lectura de un corte antes de abrir el
siguiente de esa ronda. Las respuestas anteriores no se reescriben al ver más
contexto. El hash de la ficha presentada se conserva con la respuesta.

Se puede responder por chat con el [formulario breve](../../outputs/stratai-independent-evaluation-20260924/Ficha_de_revision_ciega.md).
Registrar primero si el revisor conoce la partida/episodio, ha visto el futuro
o conoce una predicción. Después conservar literalmente su lectura y separar:

- Estrategia interpretable o indeterminada y planes compatibles, si los hay.
- Destinos posibles, respaldados o rechazados; varias opciones son admisibles.
- Funciones posibles de cada alias, señales que las sostienen y ausencias.
- Continuidad o cambio respecto al corte anterior, únicamente si ya fue visto.

Normalizar después en `stratai.contextual_review@2`, con autoría, exposición y
contexto exactos. La interpretación del agente al normalizar debe identificarse
como tal y no inventar una respuesta del revisor. Una orientación sugerida, una
predicción vista o un desenlace conocido impiden clasificar esa lectura como
ciega; se conserva por separado. Un único experto y exposición desconocida
permiten contraste asistido, no acreditan por sí solos evaluación independiente.

Ejemplos y excepciones que ya forman parte del alcance:

- Pausa con C4/posiciones verificadas después de un avance: comprobar si se
  conserva la posibilidad, sin dar por correcta cualquier persistencia.
- Retroceso sostenido, dispersión o cambio de portador: revisar con la evidencia
  disponible; no equiparar automáticamente cualquier movimiento a cambio de plan.
- Sólo dos supervivientes: puede haber un destino posible sin estrategia
  identificable. No imponer abstención universal por el número de vivos.
- Utilidad equipada: función de preparación posible, sin atribuir lanzamiento,
  cobertura o eficacia no observados.
- Datos insuficientes: conservar el motivo técnico y la indeterminación humana
  por separado. Una abstención no se considera un acierto sin contraste.

La muestra sistemática puede no contener todas estas situaciones. Informar su
ausencia; no sustituir episodios después de ver las predicciones. Los controles
construidos de desarrollo no rellenan celdas de evaluación con replays reales.

## 4. Comparación y análisis de errores

Congelar las predicciones antes de incorporar respuestas; no cargarlas en las
fichas. Mantener las respuestas fuera de la entrada del detector. Comparar con
`review.compare`, preservando los enlaces de episodio/contexto/configuración.
Si el contexto del mapa y el de la ficha tienen IDs distintos, registrar el
enlace factual exacto y su diferencia; no forzar igualdad ni copiar un ID.

Informar por partida y como total descriptivo, siempre con denominadores:
episodios previstos/materializados/revisados, emisiones, abstenciones y motivos,
planes compatibles no emitidos, planes fuera de la lectura revisada, destinos
respaldados/posibles/rechazados, funciones revisadas y desacuerdos. Los casos sin
revisión permanecen sin puntuar. Destino «posible» no equivale a plan confirmado.

Para actualización temporal, conservar una línea por corte con lectura humana,
lectura automática y cambio respecto al anterior. Distinguir pérdidas de
datos, retirada de una posibilidad y cambio de destino. Las funciones se
contrastan cualitativamente: el comparador existente no proporciona una métrica
de exactitud de roles. Registrar esa limitación sin fabricar un porcentaje.

El resultado principal usa **1 s**, fijado antes de las nuevas etiquetas. Las
sondas de 0,5 s y 2 s son análisis secundarios de sensibilidad, marcados como
tales y ejecutados después de congelar el resultado principal. No elegir el
umbral que mejor encaje en estos mismos episodios y llamarlo validado. Cualquier
ajuste requiere una versión nueva y otro grupo de evaluación.

## 5. Dependencias y verificación de ejecución

La respuesta de alcance y el sello de selección están registrados. Antes de
preparar las fichas se verifica la disponibilidad de las fuentes actuales del parser.
No completar inventario vigente a partir de listas históricas ni interpretar
datos ausentes como muerte. La disponibilidad actual de la demo no acredita que
los sidecars necesarios estén ya materializados para estas tres partidas.

En la ejecución posterior, validar contratos, tiempo causal, identidad y origen
de las fuentes; verificar SHA de la demo antes y después y recalcular señales y
hechos afectados. Validar las fichas con su lector y comprobar visualmente mapa,
controles y recursos en escritorio/móvil. Verificar que no contienen respuestas
del sistema ni datos posteriores. Rechazar destinos no temporales, ocupados o
enlaces antes de parsing; conservar comandos, duraciones, versiones, inventarios,
fallos y hashes. Los bundles originales permanecen de sólo lectura.

Si cambia un productor, hecho derivado, esquema o contrato runtime, cumplir
primero las pruebas pertinentes y el smoke de exactamente una demo aprobada de
AGENTS.md. Un lote posterior se registra por separado, sin disfrazarlo de smoke
ni de migración 44/44. El censo previo no procesó demos. El lote aceptado
posterior sí procesa las tres entradas con el productor ya verificado, sin
cambios runtime: [cierre técnico](../archive/ai-coach/AI_COACH_TP10_PILOT_2026-09-24.md)
y [evidencia](../evidence/AI_COACH_TP10_PILOT_2026-09-24.json). Son 36 prefijos
verificados en 36 rondas distintas, con 26.195 muestras. No hay lecturas
humanas sucesivas de una misma ronda en este lote; la auditoría de evolución
interna no sustituye ese contraste. La revisión visual cubre E01 en escritorio
y móvil; los lectores y la auditoría factual cubren las 36 fichas.

La primera ficha se entrega sin predicción. El operador vio durante el progreso
metadatos de final de las rondas 18–20 en el log de la primera demo y revisó
visualmente E01; esto no cambió la selección ni produjo etiquetas. Registrar
si el revisor accedió a esos logs, al desenlace o a predicciones antes de
clasificar su lectura como ciega. La exposición privada sigue desconocida.

La evaluación descriptiva no resuelve percepción rival legal, límites de zonas
y build, alternativas físicamente viables ni comparación para aconsejar. Perfil,
práctica y seguimiento conservan sus dependencias. No se entrena ni se habilita
aprendizaje automático al incorporar partidas o revisiones.

## Enmienda de instrumento v0.5 — reloj y elegibilidad

El [cierre técnico](../archive/ai-coach/AI_COACH_ROUND_CONTEXT_AND_CLOCK_2026-09-25.md)
añade contexto nativo y reloj por muestra, después de la revisión de E01.
El filtro conservador exige prueba positiva; los tipos desconocidos no se
envían al intérprete nuevo. E01 sigue excluida por el experto y no se cuenta
como abstención acertada: su clasificación nativa continúa desconocida.
La rama positiva de cuchillos está probada sólo sintéticamente.

E02 se entrega mediante [ficha v2](../../outputs/stratai-round-context-20260925/review/E02.html).
El [recibo de enlace](../../outputs/stratai-round-context-20260925/E02-delivery.json)
verifica identidad factual con la vista anterior y conserva los IDs originales
para la etiqueta/predicción. En aquella entrega todavía no había respuesta E02 ni predicción nueva.
Se registra exposición del operador a la secuencia aliada durante QA; la
exposición privada del revisor debe recogerse al recibir su lectura.
Las otras fichas originales no se sustituyen ni se presumen enriquecidas.
El cambio de presentación impide presentar esta versión como un test nuevo
congelado antes de toda revisión. Recuento en aquella entrega: 35 pendientes, cero puntuadas.

## Registro posterior de E02 — 25/09/2026

La [lectura E02](../archive/ai-coach/AI_COACH_E02_EXPERT_READING_2026-09-25.md)
está recibida y enlazada a los IDs originales. La predicción congelada de entrada
se abstuvo frente a la defensa de bomba que describe el experto. Se registra
`compatible_plan_not_emitted` como diferencia descriptiva y carencia de cobertura.
No se recalibra el detector ni se declara precisión. La exposición privada sigue
desconocida; la revisión se registra como asistida por el contexto factual previo.

El recuento final 2 T/1 CT se comprobó después de recibir la lectura, sin modificar
la ficha entregada. No atribuir ese dato a la información mostrada originalmente
al revisor. La [cola actual](../../outputs/stratai-e02-review-20260925/active-review-scope.json)
conserva 36 casos históricos, E01 excluida, E02 contrastada, 34 pendientes y cero
puntuaciones de precisión. El siguiente cambio de presentación deberá declararse
y conservar las versiones anteriores; el protocolo no autoriza una nueva regla posplantado.

## Enmienda de instrumento v0.6 — recuento de ambos equipos

Tras la lectura y petición E02/IC29 se añade el contador causal 5v5 y resumen de
hechos en la [ficha E02 v3](../../outputs/stratai-team-status-20260925/review/E02.html).
El [recibo](../../outputs/stratai-team-status-20260925/E02-delivery.json) vincula la
vista nueva con el episodio/contexto originales. La información agregada de
vivos rivales es posterior a la lectura recibida: no reetiquetar esa lectura ni
atribuirle retrospectivamente esta información. No cambia la predicción primaria.

E03 y las otras vistas originales conservan su versión v1 hasta enriquecimiento
explícito. Las etiquetas siguientes deben registrar qué versión se mostró.
El cambio no acredita independencia nueva ni precisión: permanecen E01 excluida,
E02 contrastada descriptivamente y 34 pendientes. La lectura posplantado humana
no se incorpora como criterio del detector ni se utiliza para entrenamiento.

## Enmienda de entrega v0.7 — revisión conjunta de los 34 pendientes

El usuario solicita «Preparame todos los que hagan falta para revisarlos a la vez.
En vez de ir 1 a 1». El [literal](../../outputs/stratai-review-batch-20260925/request.json)
sustituye la entrega por turnos: preparar E03–E36 juntos, con índice y hoja de
respuestas. Son 34 rondas distintas de las tres partidas ya seleccionadas
(10 + 12 + 12). E01 queda excluido y E02 no se repite. No se altera la selección
ni la predicción congelada; las lecturas recibidas permanecerán inmutables.

Se homogeneiza la presentación a v3 con vivos/plantado/relojes. La captura nativa
completa existe para la primera demo. Las otras dos demos locales aprobadas se
recapturan con el productor verificado, en TEMP aislado; no cambia runtime ni
se adquieren partidas. Se trata de preparación del lote, no de un smoke nuevo.
Los cortes originales y hechos propios/aliados se cotejan y enlazan con las
nuevas vistas; no se muestra la predicción antes de las respuestas.

Las respuestas se pueden devolver agrupadas por chat o en una hoja. Exposición,
interpretación, señales y ausencias se conservan por caso; no se rellenan
respuestas no recibidas. El cambio de instrumento posterior a E01/E02 sigue
declarado y no constituye un nuevo test independiente ni entrenamiento.

Preparación v0.7 completada: [34 fichas v3 y hoja de respuestas](../../outputs/stratai-review-batch-20260925/paquete/index.html).
El [manifiesto de entrega](../../outputs/stratai-review-batch-20260925/delivery-manifest.json)
enlaza cada vista con el episodio/contexto y hashes originales. Se conservan las
vistas v1, sin atribuirles retrospectivamente el nuevo instrumento. No se han
recibido lecturas nuevas ni abierto predicciones durante la preparación. La
revisión visual del operador está declarada; exposición privada del experto
pendiente de registrar por caso. [Cierre y verificación](../archive/ai-coach/AI_COACH_REVIEW_BATCH_PREPARATION_2026-09-25.md).

## Enmienda v0.8 — varios revisores y acceso restringido

El usuario solicita otros puntos de vista y elige un enlace web con acceso
restringido. El [sitio](https://stratai-revision-tp10.kerch0k.chatgpt.site) conserva las 34 fichas v3 por hash; cambia sólo
la guía de entrega y cabecera de la plantilla. El Excel vacío permanece intacto.
En el cierre v0.8 sólo tenía acceso el propietario, pendiente de correos; sustituido por v0.9.

Cada primera lectura se conserva por `caso + revisor`, con alias, fecha,
exposición declarada e instrumento mostrado. Escribir antes de consultar otras
opiniones; si ya hubo discusión, registrar esa exposición. Ninguna revisión
nueva sustituye la anterior. Conservar desacuerdos y ausencias, sin imponer
consenso ni usar una mayoría como verdad automática. Varias opiniones de un
mismo caso no se cuentan como nuevas partidas independientes.

En el cierre v0.8 la entrega era por archivo y no había captura web; sustituido por v0.9. El cambio no autoriza una regla táctica, ajuste del detector
o entrenamiento. [Cierre](../archive/ai-coach/AI_COACH_RESTRICTED_REVIEW_SITE_2026-09-25.md).

## Enmienda v0.9 — formulario web, invitación sin cuenta y mm:ss

Petición expresa del 26/09. Se reutiliza el sitio; cada caso E03–E36 incluye
formulario persistente. Instrumento `stratai.review_web@1`: conserva los bloques
factuales v3, añade la recogida de lecturas y presenta los relojes mm:ss truncados
sin cambiar los segundos originales. El registro guarda el SHA de la ficha
efectivamente mostrada y los vínculos al episodio/contexto congelados.

La invitación concede acceso por posesión del enlace; no requiere cuenta ni
email. El alias es declarado, la identidad técnica es la sesión del navegador.
No fusionar autores por alias ni considerar otro dispositivo como otra persona
independiente. La consulta global exige el enlace del organizador. Los revisores
no reciben lecturas ajenas a través del sitio. La primera lectura y cada corrección
se conservan; exposición previa declarada en cada envío. Registros `is_test=1`
son pruebas técnicas, nunca respuestas humanas ni casos evaluables.

El trabajo posterior sigue siendo contraste humano de los casos respondidos,
sin rellenar ausencias, imponer consenso o modificar el detector. Varias lecturas
del mismo caso no aumentan el número de partidas independientes. El despliegue
no cambia selección, predicciones ni gates de generalización/entrenamiento.
[Cierre](../archive/ai-coach/AI_COACH_WEB_REVIEW_COLLECTION_2026-09-26.md).

## Enmienda v0.10 — tres campos y carencia de información de combate

Por petición del usuario, `stratai.review_web@2` recoge sólo alias, interpretación
e información faltante. La exposición previa deja de preguntarse: `not_collected`
expresa que no se recabó, nunca ausencia de exposición. Lecturas anteriores
conservan lo declarado. Las señales antiguas permanecen en su historial; no hay
campo nuevo de señales. Cada versión mantiene fecha y vínculo a la ficha mostrada.

El usuario objeta que vivos y posiciones aliadas no bastan para explicar bajas
y decisiones sin información rival. Se ha auditado la carencia y preparado una
capa de eventos/contactos, pendiente de perspectiva acordada y acreditación de
fuentes. No se incorporan posiciones de espectador como información del equipo
ni se presenta una posición previa como lugar exacto de muerte. La simplificación
no cambia los 102 bloques factuales ni las predicciones del lote.
[Cierre](../archive/ai-coach/AI_COACH_SIMPLE_REVIEW_FORM_2026-09-26.md).

## Enmienda v0.11 — bajas causales, perspectiva de equipo y límites de información

Aceptada la perspectiva recomendada del equipo. `stratai.review_web@3` conserva
los tres campos y `not_collected` para exposición previa; añade historial de
bajas aliadas y últimas posiciones con vida. Cada respuesta mantiene el hash
de la ficha, instrumento y momento de recepción. No mezclar instrumentos como
si hubieran mostrado el mismo contexto; las lecturas anteriores no se reetiquetan.

Las marcas no son posiciones exactas de muerte. Sólo aparecen desde la primera
muestra que acredita la baja, sin usar futuro; huecos y cambios de vida/ronda
impiden arrastrar posiciones. No se infiere atacante, avistamiento o comunicación.
El recuento agregado de rivales vivos no aporta por sí solo ubicación ni percepción.

Los contactos de estas fichas permanecen no disponibles, por falta de fuente
atribuida. Las lecturas dependientes de información rival no pueden usarse como
prueba de generalización de capacidades aún no instrumentadas. Los desacuerdos
y carencias se conservan, sin convertir la opinión del revisor en entrada del
detector o entrenamiento. IC30 registra el encuadre; no añade etiqueta de caso.
[Cierre](../archive/ai-coach/AI_COACH_REVIEW_INFORMATION_CONTEXT_2026-09-26.md).

## Enmienda v0.12 — contexto por bajas, arma y posiciones registradas

El usuario pide enlazar feed y posición del atacante e incorporar el arma que
causa la muerte. IC31 registra el criterio literal, incluida su garantía del
100 % como opinión, sin validarla como percepción universal. El nuevo instrumento
@4 incorpora hechos de Kill, posiciones de callback y reloj causal. La fuente
canónica existía; su ausencia en la ficha no era ausencia en la replay.

Se conservan tres campos y exposición `not_collected`. El hash y el instrumento
identifican el contexto recibido. No reetiquetar respuestas antiguas ni mezclar
juicios con información diferente. El detector congelado no recibe este nuevo
suplemento: una discrepancia que depende del arma/posición añadida debe registrarse
como diferencia de información, no automáticamente como fallo de generalización.

Las posiciones quedan fechadas, envejecen y no se actualizan con trayectoria de
espectador. La feed que registra la muerte del atacante retira su marca. Arma
causante no significa arma todavía equipada; con utilidad, posición al matar no
significa origen del lanzamiento. No se infieren voz o comunicación ni visión
universal. Los avistamientos sin baja conservan su dependencia factual aparte.
[Cierre](../archive/ai-coach/AI_COACH_KILL_CONTEXT_2026-09-26.md).

## Enmienda v0.13 — situación fija y contexto anterior opcional

El usuario señala que revisar una secuencia completa confunde el objetivo de
interpretar una situación y pide corregir el HTML. @5 destaca el último frame
del caso, su ronda y su tiempo transcurrido mm:ss. La ficha abre pausada en ese
corte, con las trayectorias ocultas. Los controles se despliegan mediante
«Ver contexto anterior»; la feed puede llevar a un momento previo. Cerrar el
contexto o pulsar «Volver a la situación» detiene la reproducción y restaura el
corte objetivo. El formulario conserva el texto y no envía desde otro frame.

La unidad de revisión ya era el corte final: no cambia selección, duración de
prefijos, datos disponibles, JSON ni predicciones. La vinculación persistente
sigue siendo caso/episodio/contexto/hash, fijada en el servidor. El desplazamiento
del visor no cambia la situación asociada a la respuesta. Se conservan sólo
alias, interpretación e información faltante. Versionar la presentación impide
atribuir @5 a pestañas antiguas: sus envíos solicitan recarga, con borrador retenido.

Las respuestas históricas no se reetiquetan. El contexto por bajas continúa
ausente de las entradas del detector congelado; la corrección de interfaz no
resuelve esa diferencia ni autoriza medir como error suyo un juicio dependiente
de ese suplemento. No hay entrenamiento automático ni nuevo criterio táctico.
[Cierre](../archive/ai-coach/AI_COACH_SITUATION_REVIEW_2026-09-26.md).

## Enmienda v0.14 — formulario junto al mapa, intercambios y C4

El usuario pide que formulario y mapa estén a la misma altura, se vean las bajas
causadas por T, aparezca el CT en intercambios, las etiquetas no tapen el mapa y
se indiquen drops del C4 con jugador y tiempo. @6 aplica esa corrección: en
escritorio hay dos columnas y formulario persistente; en móvil se apilan. Las
marcas muestran el alias y trasladan arma/antigüedad al detalle fuera del mapa,
accesible por toque, clic o teclado. Una baja CT deja una cruz, sin conservarlo
como contacto vivo ni revivirlo por daño póstumo.

Para «se disparan entre ellos» se formuló una aclaración opcional sobre daño
registrado. Sin respuesta recibida durante este trabajo se explicitó y aplicó
esa interpretación acotada: callbacks `player_hurt` con daño positivo entre T y
CT, en ambas direcciones. Disparos sin impacto no aportan marcas nuevas. Las
posiciones pertenecen al evento, no prueban percepción ni comunicaciones ni
constituyen seguimiento continuo. Una interrupción de muestras invalida marcas
anteriores. La baja conserva el arma causante; dañar a CT no atribuye al CT el
arma usada por el atacante T.

El quinto bloque `stratai.review_extra_context@1` contiene 448 eventos de daño
y 59 eventos C4 en 34 fichas: 32 `bomb_drop` y 27 `bomb_pickup`. Seis drops coinciden
con el tick de una baja del portador y se muestran como caída al morir. El resto
se muestra como C4 soltado, sin afirmar que la acción fuera voluntaria. No se
infiere un drop desde ausencia de portador. Identidad, disponibilidad, relojes,
fuentes y hashes se enlazan al mismo caso. Los cuatro bloques anteriores no se
reescriben; los eventos futuros permanecen excluidos al retroceder.

Una demo aprobada ejecutada después de las pruebas y su materializador nuevo
acreditan 155 eventos de daño, 10 drops, 5 recogidas y 8.255 cortes conciliados.
Se preservan las 44 partidas aprobadas. Los envíos @5 piden recarga, sin alterar
lecturas guardadas ni sus hashes. Se verificaron las 32 versiones previas y se
añadieron sólo dos pruebas técnicas excluidas por `is_test=1`.

IC32/Excel v23 conserva la solicitud y la interpretación técnica adoptada, sin
inventar una respuesta experta. El detector sigue congelado y no consume estos
suplementos; no mezclar versiones como entradas equivalentes ni entrenar con
los envíos. [Cierre](../archive/ai-coach/AI_COACH_REVIEW_WORKSPACE_2026-09-28.md).

## Enmienda v0.15 — tres columnas y evento C4 destacado

@7 corrige la presentación a petición del usuario: mapa, información y formulario
comienzan en la misma fila en escritorio. El último evento C4 disponible queda
visible junto al estado actual; el historial completo se despliega en ese bloque.
La captura E04 ya contenía el drop de T4 a 00:02 bajo el mapa: se hace visible,
sin crear un hecho nuevo ni deducir la ubicación actual del objetivo. Los cinco
bloques de datos mantienen igualdad exacta con @6. Los contadores, relojes,
recursos y eventos se agrupan y se compacta la ayuda del formulario.

Se verifica visualmente escritorio/móvil en navegador local, las interacciones
de contexto y las 34 fichas alojadas por hash. Las 34 respuestas anteriores se
preservan; sólo se añaden dos registros técnicos excluidos. @7 queda identificado
en cada nuevo envío, sin reetiquetar los anteriores. No cambian las fuentes ni
el detector; no aplica un nuevo smoke de demo ni entrenamiento.
[Cierre](../archive/ai-coach/AI_COACH_THREE_COLUMN_REVIEW_2026-09-28.md).

Corrección posterior de presentación del mismo día: @8 amplía el mapa y elimina
el margen del contenedor de ancho máximo. No modifica los cinco bloques de datos
ni el objetivo de revisión. Se distingue de @7 para mantener el vínculo del envío
con el HTML visto; las pestañas anteriores solicitan recarga y retienen borrador.
36 versiones anteriores conservadas, sin crear nuevas lecturas alojadas durante
esta comprobación. [Cierre](../archive/ai-coach/AI_COACH_FULL_WIDTH_MAP_2026-09-28.md).

## Enmienda v0.16 — posición del C4 caído y acceso al contexto

@9 añade `bomb-data` como sexto bloque, preservando los cinco anteriores. En
fichas T elegibles, la posición del objeto C4 procede del snapshot nativo de
`stratai.replay_round@5` en el tick exacto de cada corte. Se exige dropped/preplant,
sin portador ni planta y posición observada de origen nativo. No hay interpolación,
arrastre de una muestra previa, uso de coordenadas del jugador que lo soltó ni
revelación de posición global del C4 a una ficha CT. Recogida, planta, exclusión
o dato incompatible retiran la marca. El marcador representa el instante mostrado.

«Ver contexto anterior» se sitúa arriba en Estado y eventos, junto al mapa. El
instante objetivo del formulario sigue fijo. IC33/Excel v24 registra el pedido.
34 fichas y 25.025 cortes, con 7.064 posiciones de bomba caída. Después de las
pruebas, una demo aprobada se procesó en TEMP nuevo y se contrastaron de forma
independiente los 8.255 cortes de diez casos, con 2.445 posiciones y 73 exclusiones.
Se verificaron también casos de recogida, planta y movimiento del objeto.

Publicación 12: los 34 HTML alojados coinciden por hash y las 36 respuestas
anteriores conservan sus valores. No se crean lecturas técnicas ni expertas durante
la comprobación alojada. @8 exige recarga antes de enviar. El detector permanece
congelado y no consume este suplemento. No atribuir diferencias debidas al nuevo
contexto a generalización del detector ni entrenar con las respuestas.
[Cierre](../archive/ai-coach/AI_COACH_GROUND_C4_REVIEW_2026-09-28.md).
