# Representación contextual y criterio de suficiencia

Versión documental 1.0, 04/10/2026. Fase CS1. El [plan canónico](AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado, orden y siguiente acción. Este documento fija el encuadre humano; no es un schema runtime, una regla nueva ni una evaluación terminada.

## Criterio experto recibido

El usuario respondió **«Si, me parece bien.»** al criterio presentado el 04/10:

> Una lectura resulta insuficiente si acierta el destino pero omite información disponible que cambia el significado de la situación —por ejemplo, bomba plantada, pérdida de compañeros o recursos relevantes—. Puede mantener varias hipótesis si las justifica.

El [recibo literal](evidence/AI_COACH_CONTEXTUAL_SUFFICIENCY_ACCEPTANCE_2026-10-04.json) conserva respuesta, propuesta, hashes y alcance. Se mantiene el objetivo recibido el 03/10: integrar el contexto observable y evitar enseñar exhaustivamente cada combinación. Los textos originales de Alex no se convierten automáticamente en etiquetas.

## Presentación de una lectura

| Parte mostrada | Contenido | Condición de soporte |
|---|---|---|
| Situación | Objetivo, fase, jugadores, distribución, recursos e información disponible pertinentes | Hechos actuales y antecedentes separados; perspectiva y ausencias explícitas |
| Evolución | Qué cambió y qué acontecimientos afectan la interpretación | Prefijo causal, sin desenlace posterior como entrada; continuidad y cambios con referencia |
| Interpretación | Organizaciones o planes compatibles y su fundamento; preferencia cuando se sostenga | Inferencias identificadas, alternativas conservadas y señales a favor/en contra |
| Incertidumbre | Qué no puede afirmarse, qué alternativas siguen abiertas y qué información falta | Incertidumbre local no borra hechos acreditados ni fuerza un destino |

La presentación se puede resumir, pero conserva acceso a la evidencia y a las limitaciones. No debe convertirse en una lista de todos los valores extraídos. La interpretación responde qué ocurre y por qué una lectura es compatible; todavía no elige la acción mejor ni genera una misión.

Las nueve dimensiones de la auditoría se organizan técnicamente en esas cuatro partes:

| Dimensión técnica | Parte principal | Aplicación |
|---|---|---|
| Marco | Situación | Lado, mapa/build, ronda/instancia/época, corte, fase y elegibilidad |
| Objetivo | Situación | C4 portado/caído/plantado y relojes disponibles, sin deducir un estado por ausencia |
| Equipo | Situación | Identidad, vivos, posición y recursos; recuentos parciales permanecen parciales |
| Información | Situación e incertidumbre | Perspectiva, eventos fechados, antigüedad y percepción no acreditada |
| Evolución | Evolución | Cambios respaldados del prefijo; movimiento y tiempo son evidencia contextual |
| Restricciones | Situación e incertidumbre | Limitaciones de recursos/tiempo/fuentes; una opción no se declara físicamente viable por plausibilidad |
| Organización | Interpretación | Relaciones/funciones posibles sustentadas, sin intención o cooperación confirmadas |
| Hipótesis | Interpretación e incertidumbre | Compatibles, preferidas o indeterminadas, con alternativas y fundamento |
| Revisión | Evolución e incertidumbre | Qué evidencia mantiene, debilita o revisa una lectura; qué observación nueva sería necesaria |

Esta correspondencia es diseño técnico del agente para aplicar la propuesta aceptada. No son nueve nuevas declaraciones literales del usuario, nueve inferencias ya implementadas ni etiquetas tácticas de casos.

## Rúbrica operativa

Evaluar primero afirmaciones y después la suficiencia del episodio. Cada afirmación necesita referencia, corte, perspectiva y condición de soporte. La revisión debe distinguir:

| Resultado de revisión | Criterio | Consecuencia |
|---|---|---|
| Hecho sustentado | El dato es correcto y disponible bajo esa perspectiva y corte | Puede formar parte de la situación/evolución |
| Inferencia compatible | Los hechos citados la respaldan sin convertirla en intención confirmada | Puede conservarse junto a otras hipótesis |
| Preferencia sustentada | Se justifica el orden entre alternativas manteniendo lo que no se excluye | Preferencia ordinal; no porcentaje inventado |
| Omitido relevante | El dato estaba disponible y cambia el significado de la lectura, pero se omitió | Lectura insuficiente aunque acierte el destino: criterio experto aceptado |
| Información ausente | La fuente no acredita el dato necesario y se declara el límite | Limitar la afirmación dependiente; no contarlo automáticamente como error de toda la lectura |
| Afirmación contradicha | La lectura choca con un hecho acreditado o usa estado futuro/otra perspectiva | Identificar la afirmación y su contradicción; no suplirla con una explicación plausible |
| Fuera de alcance específico | Una regla acotada no trata ese problema | No convierte la situación completa en indescriptible ni su abstención en error |

La distinción entre omisión pertinente, dato ausente y dato irrelevante es la aplicación operativa del agente al criterio recibido. No se exige mencionar cada dato disponible. La importancia se justifica por cómo cambia el significado en el episodio; no se decide por una lista universal de campos obligatorios para cualquier ronda.

No hay tasa global en esta fase. Las revisiones por familia, independencia, igualdad de información y exposición siguen pendientes. Una respuesta con información añadida no evalúa de forma equivalente la predicción anterior. Varias muestras del mismo episodio no se cuentan como observaciones independientes.

## Ejemplos recibidos y excepciones

| Referencia de desarrollo | Criterio que ya se puede aplicar | Límite que debe mantenerse |
|---|---|---|
| E06 | Conservar A más probable y B posible, con distribución y posible reunión/apoyo como fundamento experto | C4 no fundamenta esa preferencia. Los nombres nativos añadidos después no se atribuyen a la ficha original; apoyo eficaz e intención no están acreditados |
| E17 | Describir la movilización inicial según la lectura recibida y mantener insuficiente la afirmación fuerte de preparación | No pedir un mínimo universal de segundos ni dar por resuelto el caso porque una regla de dos vivos se abstenga |
| E24 | Integrar planta acreditada: omitirla cambia el significado de la situación | Recuento CT parcial no certifica 1v1; abstención de preparación no demuestra reconocer posplantado |
| E13/E25/E30 | Conservar elegibilidad desconocida y describir sólo lo que esté respaldado | No etiquetar cuchillos por ausencia de C4 ni convertir unknown en una clasificación positiva |
| K1/K2/R2 | Mantener una lectura sustentada durante pausas o correcciones locales compatibles | Persistencia no se acredita sólo por un filtro de un segundo; un movimiento nuevo tampoco elimina automáticamente la organización |
| T/CT | Compartir hechos, estructura y separación de evidencia/inferencia | No invertir etiquetas T para fabricar criterios CT. Defensa/rotación/retake específicos requieren material causal y revisión propia |

Estos ejemplos son referencias para diseño y regresión, no nuevas etiquetas expertas ni nuevas predicciones. Fuente: [revisión de Alex](../outputs/stratai-alex-review-20261001/REVISION_ALEX_TP10.md), [auditoría contextual](../outputs/stratai-context-strategy-audit-20261003/AUDITORIA_Y_PLAN.md) y recibos de criterio ya conservados.

## Datos y verificación del encuadre

Datos necesarios por afirmación: marco y corte exactos; hechos propios/aliados; objetivo y recursos; historial previo pertinente; suplementos e instrumentos identificados; perspectiva, disponibilidad y ausencias. Las posiciones rivales de eventos son históricas, no ubicación actual ni percepción individual acreditada. Se conserva la separación entre descripción retrospectiva e información legal para valorar una decisión.

Comprobaciones documentales de CS1:

1. El literal y el alcance coinciden con la pregunta aceptada, sin aprobar por extensión proveedor, arquitectura semántica o ejecución de otra fase.
2. El esquema de lectura puede representar contexto sin destino único, varias hipótesis y desconocidos locales.
3. Ejemplos, excepciones, datos necesarios y verificación conservan criterios ya recibidos, sin preguntas repetidas ni etiquetas automáticas.
4. La omisión de un dato disponible y pertinente se distingue de una ausencia factual y de información irrelevante.
5. Los documentos del producto mantienen el comparador prescriptivo y el LLM conversacional en sus alcances vigentes.
6. El nuevo libro añade sólo IC39 y conserva contenido, estilos, objetos, validaciones y hojas anteriores; originales y código mantienen hashes.

CS1 cierra únicamente la representación y el criterio de suficiencia documentales. No acredita interpretación ejecutable, comprensión táctica, validación independiente o coaching. El [cierre](../outputs/stratai-context-criterion-20261004/RESULTADO.md) registra lo comprobado.

## Propuesta preparada para la siguiente fase: CS2

La propuesta preparada el 04/10 reúne una **entrada factual común**, independiente del detector y de las respuestas humanas, usando fuentes ya acreditadas. El usuario encuadra su alcance el 05/10; [alcance aceptado, foco CT y verificación](ai-coach/AI_COACH_CONTEXT_INPUT_PHASE.md). Preparación y orientación congeladas permanecen intactas.

Alcance concreto propuesto:

- Reunir marco, objetivo, equipo, recursos, información disponible e historial causal mediante referencias verificables; mostrar ausencias sin rellenarlas.
- Organizar situación y evolución observadas, e incertidumbre factual. Dejar interpretación táctica sin emitir cuando no exista un consumidor revisado. No confundir una entrada completa con una lectura correcta.
- Reutilizar los contextos @5/@9 y tratar el nombre nativo posterior como suplemento separado. No incorporar a E03 los hechos de @9 como si Alex los hubiera recibido en @5.
- Usar la misma estructura para T/CT cuando las fuentes y su perspectiva lo permitan. Si sólo hay material T acreditado para una variante, declarar cobertura pendiente; no crear casos CT ficticios.
- No agregar aún clustering espacial, umbrales de grupos, apoyo eficaz, información rival mental ni una clasificación automática de movilización/execute/fake.

Verificación funcional posterior propuesta: origen/hashes, causalidad e identidad, preservación de hechos y desconocidos, retiro de un campo limitando sólo dependencias, conflictos, versiones de instrumento y separación de datos de respuestas/predicciones. Si cambia productor, hechos derivados o contrato runtime, pruebas pertinentes y exactamente una demo aprobada en TEMP aislado, con recomputación independiente y retención. Un lote posterior queda separado y no modifica los 44 bundles.

El encuadre CS2 se recibe el 05/10, conforme a la [forma de trabajo por fase](AI_COACH_IMPLEMENTATION_PLAN.md#forma-de-trabajo-por-fase); su implementación y cierre funcional siguen pendientes. El usuario aporta el foco general CT: patrones T, eficacia y adaptación defensiva explicadas. Casos/criterios CT operativos, especialista semántico, modelos/proveedores y presupuesto experimental se resolverán en sus fases pertinentes; no se amplía retrospectivamente el cierre documental CS1.

## Continuación contextual aceptada — 16/10/2026

El usuario pide evitar que la interpretación se reduzca a reglas «si pasa X, significa Y» y acepta recuperar comprensión de situación/evolución, hipótesis justificadas y posterior comparación de acciones bajo incertidumbre. Esta dirección desarrolla el criterio CS1 ya recibido: varias explicaciones pueden ser compatibles y acertar el destino no basta. No modifica retrospectivamente los cierres anteriores; el [plan canónico](AI_COACH_IMPLEMENTATION_PLAN.md) contiene el estado vigente.

El [alcance contextual v0.1](ai-coach/AI_COACH_CONTEXTUAL_INTERPRETATION_SCOPE_v0.1.md) concreta perspectiva T, antecedentes causales, protecciones factuales y revisión de dos casos ilustrativos con igual fotografía final y evolución diferente. La región B Apps es una medición de apoyo, no la frontera de todo lo interpretable. Las lecturas A/B aún son propuestas pendientes de encuadre experto; ni las probabilidades numéricas ni las recomendaciones se declaran verificadas o implementadas.
