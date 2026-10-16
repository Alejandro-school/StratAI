# Corrección de las fichas de revisión R1/R2

El usuario señala que R1 muestra tres jugadores como desconocidos, exige interpretar
coordenadas, omite utilidad y usa la expresión confusa «vivo observado». Su respuesta
literal se conserva en IC20 del Excel v10 y en la
[evidencia](../../evidence/AI_COACH_CONTEXTUAL_REVIEW_PRESENTATION_2026-09-23.json).
Es una objeción a la información y presentación del caso, no una etiqueta táctica:
no se introduce como `insufficient_information` para puntuar al detector.

## Hechos y cambios

T1, T2 y T4 están muertos en el corte de R1. El contexto registra `alive=false`
con estado observado y omite su posición por `not_alive`. La replay del mismo
tick confirma ese estado. T3 y T5 están vivos.

La ficha anterior sólo mostraba el arma equipada. La lista `weapons` de la replay
contiene humo para T3 y humo/HE para T5. Sin embargo, `resolvePlayerEquipment` y
`mergePlayerEquipment` pueden mantener la lista anterior cuando el inventario
nativo llega vacío. El export no distingue una lectura actual de una lista
conservada. Se muestran los tipos listados con vigencia no acreditada, sin
certificar cantidades. El equipo de los muertos no se cuenta como recursos
utilizables por los supervivientes.

Las nuevas [R1](../../../outputs/stratai-review-presentation-20260923/R1.md) y
[R2](../../../outputs/stratai-review-presentation-20260923/R2.md) usan «Vivo/Muerto»,
tipos de utilidad y mapa rotulado, sin tabla de coordenadas. R1 incluye orientación
visual: T3 en el pasillo de apartamentos de B y T5 en conector por el lado de medio.
Es una lectura del radar local para ayudar al revisor, no una clasificación de
zona acreditada ni una nueva entrada del detector. Las etiquetas finales se
colocan sobre la geometría visible; los orígenes VMAP no se usan como centros de
regiones. Se conserva el primer render como material de trabajo.

## Verificación

- 650 estados de actor contrastados entre contextos y replays: cinco actores,
  65 muestras, dos casos. Igualdad de vida y posiciones disponibles.
- Hash del manifest de cada captura contra la evaluación congelada y hash de
  cada replay contra su manifest. Unión por identidad opaca, ronda y tick exacto,
  sin muestras posteriores para presentar hechos.
- Revisión visual de los dos mapas finales y de IC20. Excel v10 conserva las
  14 hojas, todas las celdas previas con valores/fórmulas/estilos y sus controles;
  sólo añade A25:G25. La respuesta literal coincide.
- Detector, corpus, fichas originales y Excel v9 conservados. No se abre la
  predicción de los casos ni se alteran los templates tácticos.

Es una corrección del material de revisión usando artefactos existentes, sin
cambios en parser, contrato runtime ni productor factual. No se vuelve a procesar
una demo ni corresponde repetir suites funcionales o smoke para esta presentación.

## Continuación

La próxima acción técnica es acreditar la vigencia del inventario antes de volver
a pedir una lectura táctica. Distinguir inventario nativo actual, última lista
conocida y ausencia, con identidad, corte, cantidades y procedencia. No rellenar
con futuro ni asumir que una lista vacía es cero granadas. Si requiere cambiar
el productor, aplicar pruebas y el gate de una demo aprobada a TEMP nuevo.
Esta reparación de información/legibilidad ya tiene encuadre experto.

Sigue pendiente la información legal sobre rivales y la compatibilidad espacial
exacta para afirmaciones que la requieran. No se pide un juicio concluyente con
esos huecos. Consejos y evaluación independiente conservan sus gates.
