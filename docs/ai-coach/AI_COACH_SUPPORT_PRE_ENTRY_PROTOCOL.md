# Protocolo de preparación de apoyo anterior a la ejecución

Diseño congelado `support_preparation_census/0.1.0-design.1`, fase 3E.11. Este documento especifica una población prospectiva de **cortes de preparación candidatos** para S2–S3; no acredita una entrada física, un compromiso táctico ni una nueva decisión elegible. El [plan](../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y siguiente acción. Se conservan el [piloto](AI_COACH_INITIAL_PILOT_DESIGN.md), los [anclajes 3E.1](../contracts/support/canonical-support-anchors-v1.md) y todas las capas posteriores.

Es un diseño documental, fuera del registro runtime. No incorpora schemas, IDs estables al catálogo, productores, lectores, `DecisionExample` ni features nuevas al producto. Los ensayos sobre fuentes retenidas comprueban selección y consistencia del diseño; no constituyen una nueva extracción ni una validación de entrada física.

## Decisión temporal

El selector 3E.1 reconoce `d²[q-1] <= d²[q-2]` y `d²[q] > d²[q-1]`. Su T0 es `q`, cuando el aumento ya se observó. Cambiarlo por `q-1` conservando ese disparador seguiría seleccionando por información posterior al nuevo T0. Elegir una muestra antes de la próxima muerte, cruce o disparo tiene el mismo problema.

Se congela **C0: censar cada corte support actual**, sin exigir un movimiento anterior ni posterior. En cada corte se conservan los pares propios/aliados utilizables, incluida la permanencia sin desplazamiento muestreado. T0 precede estrictamente a la ventana de ejecución posterior; no se afirma que preceda al inicio físico de un movimiento que podría estar en curso. La unidad es un punto de observación de preparación, no un instante mental de decisión.

La conducta observable que se estudiará después de T0 es la **continuación o cambio de las posiciones muestreadas del actor y del aliado**, junto con sus límites de continuidad. Avanzar hacia una zona, seguir, reagruparse, esperar o mantener son interpretaciones/alternativas que requieren parámetros y contexto adicional. La sola geometría del par no las etiqueta.

| Concepto | Prueba necesaria | Alcance de C0 |
|---|---|---|
| Cambio de posición muestreada | Dos posiciones presentes, mismo pawn/vida/lado y frontera temporal acreditada | Puede describirse después de T0; no acredita tecla, inicio instantáneo ni intención |
| Inicio físico del movimiento | Fuente actual y codec/reloj/resolución que permitan identificar el inicio y excluir movimiento ya iniciado | Pendiente; velocidad media o posiciones iguales cuantizadas no bastan |
| Entrada física | Región/frontera orientada definida antes de seleccionar, coordenadas/procedencia exactas, estados a ambos lados y prueba temporal/continuidad suficiente | Pendiente; no se inventa una frontera con el evento futuro, NAV parcial o un callout |
| Compromiso táctico | Situación, función, alternativas viables, información legal y criterio de revisión contextual | Pendiente; una trayectoria o muerte no prueba compromiso mental |

## Población, claves y cortes

1. **Fuente:** `support_observations.json` acreditado, `fresh_replay_16hz_FrameDone_with_full_pawn_life@1`. El candidato runtime futuro debe verificar schema, procedencia y correspondencia de la captura; esta fase reutiliza los bytes retenidos y sus referencias originales.
2. **Corte:** una fila `sequence/frame/tick/instance/epoch` ya entregada. Se censan todos los cortes, incluidos los excluidos. Sólo se selecciona en `phase=playing`, con tick rate numérico finito positivo y `sample_step` entero igual a `int(tick_rate/16)>=1`. Sequence/frame/tick/instance/epoch deben ser enteros; sequence/frame positivos e instance/epoch no negativos. Un bool no se acepta como número. `playing` es la fase del capturador: no acredita por sí sola fin de freeze, reglas competitivas ni ronda oficial. No se filtra con el enlace competitivo calculado al final.
3. **Actor:** alias único del corte, lado CT/T, conexión positiva, identidad nativa observada con handle entero de 32 bits sin índice sentinel `16383` y vida local presente, estado vivo observado positivo y posición XYZ observada con las tres componentes numéricas finitas. Se rechazan colisiones de alias y referencias de pawn entre participantes del mismo lado. La precedencia de causa es alias ambiguo, pawn compartido, lado no soportado, desconexión, identidad, disponibilidad de estado vivo, no vivo y posición. Las ausencias se registran por causa, no se rellenan con otra muestra.
4. **Aliado:** otro actor del mismo lado que supera exactamente esos requisitos. No se exige distancia, orientación, arma, contacto, función táctica ni movimiento. Todos los aliados utilizables se conservan; no se elige el que posteriormente muere o resulta más cercano.
5. **Unidad:** `partida / versión de protocolo / época / instancia / sequence / actor → aliado`. Frame y tick son referencias exactas del mismo corte. No se usa un índice ordinal de las filas seleccionadas, el hash del contenido futuro, un desenlace ni la cantidad final de episodios para construir la clave.
6. **Deduplicación:** una fila por clave; tick repetido con otro FrameDone es otro corte. Ordenar el roster de otra manera no cambia claves ni hechos. Ambos sentidos del par y cortes contiguos son observaciones correlacionadas, no decisiones independientes.
7. **Cambios y vacíos:** la primera muestra de una vida/instancia nueva puede dar un candidato actual. La historia y la ejecución nunca cruzan una vida, serie, lado, instancia o época incompatibles. Una desconexión o un hueco posterior censura la ejecución, sin borrar el T0 anterior.

No se supone que el roster esté completo ni que siempre contenga cinco jugadores por lado. El censo es de **lo capturado**, no de todas las oportunidades físicas de la partida. `sin aliado utilizable en el corte` no equivale a `sin apoyo físicamente posible`.

## Regla ejecutable de selección

```text
para cada corte q ya recibido:
    registrar el corte y el roster observado
    si fase/reloj de q no son admisibles:
        registrar exclusión del corte; continuar
    resolver aliases únicos y referencias de pawn por lado en q
    para cada actor del roster:
        registrar disponibilidad o causa de exclusión del actor
        para cada otro alias del mismo lado conocido:
            registrar un par examinado
            si actor no es utilizable: excluir por actor
            si aliado no es utilizable: excluir por aliado
            en otro caso:
                emitir C0(q, actor, aliado) con T0=q
                copiar sólo el estado actual propio/aliado disponible
```

Los datos de posiciones/armas/vidas enemigas, humo/flash/rayos internos, kills, eventos posteriores, rondas finales, puntuaciones y juicios humanos no intervienen. Integridad de los identificadores y metadatos de la fuente es procedencia, nunca una feature. Añadir un campo desconocido no puede introducir un predicado de selección.

Para identidad de partida y protocolo fijadas, la obligación es:

`selección(prefijo hasta q) == selección(stream completo) restringida a cortes <= q`.

La selección utiliza sólo el corte actual, por lo que la igualdad se aplica también al truncar el final, cambiar toda la ejecución posterior o añadir un futuro malformado que el acceso inicial no consulta. El lector completo podrá rechazar un artefacto corrupto; el acceso inicial debe recortar el prefijo antes de validar datos fuera de T0. Una mutación de identidad que invalide el propio corte no es un contrafactual de futuro válido.

## Estado inicial, ejecución y censura

El estado inicial mínimo contiene copias de posición y estado vivo observados propios/aliados, con disponibilidad en el mismo corte. Alias, handles, vidas, hashes, referencias de ronda y conteos agregados quedan en procedencia. Las coordenadas aliadas conservan el alcance de proxy del equipo; no prueban atención, voz o conocimiento subjetivo. Las capas 3E.2–3E.10 sólo podrán incorporarse en un trabajo posterior mediante un acceso exacto a este nuevo T0; no se arrastra el contexto de un anclaje 3E.1 cercano.

Se congela para la comprobación de ejecución una ventana **abierta en T0 y cerrada en `T0+ceil(2*tick_rate)`**, heredando la duración de investigación de 3E.1. No es un tiempo de reacción humano validado ni un plazo táctico óptimo. Se exige muestra final exacta y secuencia completa a la cadencia declarada, con continuidad de fase/reloj/identidad/lado. Los estados posteriores son siempre ejecución/evaluación; jamás deciden si C0 existió.

Por rol, la ventana completa distingue `desplazamiento_muestreado` de `sin_desplazamiento_muestreado`: cualquier diferencia exacta entre posiciones exportadas consecutivas cuenta como cambio. No se añade un umbral táctico para eliminar jitter; la precisión X/Y cuantizada y Z exportada se declara como limitación. Si falta horizonte, hay hueco/frontera, identidad/lado ausentes o cambiados, desconexión, posición ausente o una muestra no viva, se conserva una censura específica y no un cero. Se comprueba primero la existencia del horizonte exacto; después se registra el primer fallo temporal de la ventana, con precedencia de frontera/reloj, identidad/lado y disponibilidad del actor en esa muestra. La ausencia de desplazamiento muestreado no demuestra inmovilidad continua, espera deliberada ni ausencia de entrada entre muestras.

Las muertes pueden cruzarse después, por clave y ventana canónica acreditadas, para describir casos con/sin muerte registrada. No se infiere una muerte de una muestra no viva, ni ausencia de muerte cuando falta cobertura de eventos. No se eliminan ejemplos sin muerte. Los hechos de entrada física y compromiso táctico permanecen `no acreditados` en todos los candidatos de este diseño.

## Denominadores y función de mantener

Se publicarán por separado cortes totales/admisibles/excluidos, filas de roster, aliases actor/corte con sus causas, pares del mismo lado examinados, pares seleccionados/excluidos y actor/cortes con al menos un aliado utilizable. Deben cuadrar:

- `cortes totales = admisibles + excluidos por fase + excluidos por reloj`;
- `aliases en cortes admisibles = actores utilizables + actores no utilizables`;
- `pares examinados = excluidos por actor + excluidos por aliado + seleccionados`;
- por rol, `seleccionados = ventanas completas con cambio + completas sin cambio + censuradas`.

Una matriz por actor/aliado/instancia/época conserva ceros, lado y denominadores. Los motivos por corte, actor y par no se suman entre sí. Un análisis de cobertura distingue actor/cortes y pares/cortes; no transforma los pares o las ventanas superpuestas en tamaño efectivo de muestra. La agrupación por partida/actor/episodio y cualquier submuestreo de revisión se diseñarán y versionarán antes de entrenar o evaluar un modelo. C0 no impone una cuota basada en una futura etiqueta positiva.

Un actor que no se desplaza continúa en el censo. Plantar/desactivar, sostener otro ángulo, conservar una posición por objetivo, esperar una utilidad o evitar una ruta inviable pueden justificar mantener; esas funciones **no se asignan automáticamente**. «Sin entrada», «detención» y «espera» son casos obligatorios de revisión o abstención, no etiquetas mentales deducidas de coordenadas. La misión del piloto no premia acercarse siempre ni convierte una ausencia de acción en un fallo.

## Matriz de aceptación del diseño

Los [31 casos sintéticos y sus resultados esperados](../references/support-preparation-selection-cases-v1.json) conservan el generador y el verificador de referencia identificados por hash. Son pruebas del diseño documental, fuera de las suites y schemas runtime.

| Caso | Selección C0 | Ejecución / interpretación |
|---|---|---|
| Par vivo en la primera muestra, sin historia | Sí | Historia anterior no disponible; no prueba inicio físico previo |
| Ambos permanecen en las mismas posiciones muestreadas | Sí | Sin cambio si la ventana está completa; espera e inmovilidad continua desconocidas |
| Un aliado avanza después de T0 | Sí, igual que en la rama sin avance | Cambio posterior separado; T0 no se elige por ese avance |
| El movimiento ya estaba en curso antes de T0 | Sí | Continuación posible; no llamar al corte anterior al inicio del movimiento |
| Separación aumenta, disminuye o permanece igual | Mismo criterio de disponibilidad | No etiqueta calidad de spacing ni recomienda juntarse |
| Plantado/defuse o una función que puede justificar mantener | Sí si el par es utilizable | Contexto/valor de mantener requieren acreditación propia |
| Ventana sin muerte registrada | Sí | Resultado posterior; ausencia de kill sólo con cobertura acreditada |
| Desconexión, muerte muestreada o cambio de lado posteriores | Se conserva el T0 seleccionado | Censura; no continuidad de vida ni acción inventada |
| Actor o aliado ya no vivo/desconectado/sin posición en T0 | No para ese par; exclusión contada | No rellenar el estado desde antes o después |
| Pawn compartido por aliases del mismo lado o alias duplicado | Exclusión por identidad ambigua | No reasignar dueño mediante geometría o futuro |
| Geometría enemiga o una kill futura cambiadas | Invariante | No consulta esos valores |
| Instancia preliminar, freeze no acreditado o enlace competitivo posterior ausente | No se promueve; censo de investigación si el corte cumple C0 | Nunca declarar ronda competitiva/elegibilidad a partir de `playing` |
| Dos FrameDone distintos con el mismo tick | Claves distintas | No prueba avance físico entre ellos; continuidad posterior se comprueba aparte |
| Hueco anterior o posterior al corte actual | C0 actual no se retrocede ni se elimina | Historia/ejecución correspondiente se censura |

## Material positivo pendiente y siguiente implementación

No se ha acreditado un disparador específico anterior a entrada física con las fuentes existentes. Para ese disparador falta **una definición espacial orientada de entrada fijada antes de los eventos**, con geometría y coordenadas vinculadas a la demo, y una fuente temporal/de continuidad que permita probar estar antes del cruce o inicio en la resolución declarada. Si se pretende anticipar una orden de movimiento, hace falta la entrega nativa de esa orden y su codec/reloj; una resta retroactiva o una velocidad media no la sustituye. Un comienzo de fase pública puede proporcionar un corte anterior a ejecución posible, pero no prueba una oportunidad de entrada de un actor concreto.

El compromiso táctico exige un criterio contextual con alternativas acreditadas; no se registra como intención observada. Contacto legal, NAV físico, velocidad instantánea, C4 transportada/caída, temporización completa y cooperación conservan las dependencias ya documentadas. No se reabren sus inventarios sin material positivo.

La intervención siguiente puede implementar **únicamente el censo C0**, sus exclusiones, referencias, acceso por prefijo y descriptores posteriores separados, con nombres que conserven su carácter de preparación candidata. Debe mantener intacto 3E.1 y demostrar que sus anclajes quedan localizables dentro de la nueva población, sin reescribirlos. La implementación requerirá contratos runtime explícitos, pruebas y el gate de una demo aprobada a TEMP nuevo; no está ejecutada por este documento. Comparaciones, recomendaciones, entrenamiento, revisión táctica y producto continúan deshabilitados.
