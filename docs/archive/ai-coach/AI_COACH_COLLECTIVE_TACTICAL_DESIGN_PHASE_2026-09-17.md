# Preparación documental ICT-D1: interpretación táctica colectiva

17 de septiembre de 2026. Cierre de la preparación documental autorizada por el usuario. No es un cierre funcional de S2–S3 ni una nueva fase de parsing. El [plan v2.44](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la próxima acción.

## Resultado

- [Especificación ICT-D1 v0.1](../../ai-coach/AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md): contexto, perspectivas, hipótesis de plan, razones plausibles, funciones, actualización y verificación.
- [Seis situaciones ficticias y 18 cortes](../../ai-coach/AI_COACH_COLLECTIVE_TACTICAL_REVIEW_CARDS.md), con propuestas del agente separadas de respuestas humanas pendientes.
- [Excel versionado](../../../outputs/stratai-tactical-review-20260917-7ed3f25/StratAI_Plan_del_coach_ICT-D1_v2.xlsx): conserva las diez hojas anteriores y añade Contexto colectivo, Fichas tácticas e Hipótesis de diseño. Cambios acotados en C021–C030, P03/P13/P14/P27, D03, guía y referencias.
- Arquitectura v2.1: la interpretación se integra en los especialistas existentes, entre hechos del Observador y valoración del Comparador. No crea otro servicio ni transforma `FLD-INTENT` en un hecho observable.
- Nota de alcance en dependencias C0: la frontera geométrica heredada no se impone como definición universal de entrada. Se conservan las pruebas físicas que cada afirmación requiera.
- Un único prompt vigente para revisar F01-K1 con el usuario antes de delimitar una implementación.

## Aportación humana y conservación

Se incorporan las instrucciones de conversación sobre contexto de partida, patrones del rival, inferencia del plan desde la distribución y revisión por fases. Se preservan literalmente las anotaciones de la versión de Excel recibida, incluidas C026–C028 sobre tiempo e información disponible.

El original anotado mantiene su SHA-256. Sus 75 valoraciones —71 «De acuerdo» y cuatro «Matizar»— permanecen intactas, junto con notas, prioridades e IDs. Esas valoraciones no se extienden a las nuevas hipótesis. Las respuestas de las 18 fichas están vacías y sus propuestas pendientes. No se inventaron respuestas en nombre del usuario.

## Verificación de la entrega

- Revisión independiente del diseño: corregidas presentaciones que anticipaban eventos de K2/K3, una función táctica expresada como hecho, el estado pendiente del enfoque general y la herencia implícita de relojes entre cortes.
- Inspección visual de las tres hojas nuevas y vistas afectadas del libro anterior. Conservados formatos, tablas, validaciones y dimensiones de paneles inmovilizados. La exportación puede restablecer la posición de desplazamiento de la vista.
- Comparación de las diez hojas originales en memoria y del XLSX guardado: sin cambios de valores/fórmulas fuera de las celdas previstas y de las nuevas filas de fuentes. Estilos equivalentes y controles conservados.
- Las cuatro fórmulas de resumen conservan su significado. Se entrecomilló la referencia a la hoja con tilde para que el evaluador de importación resolviera `COUNTIF`; comprobadas las cantidades 0 / 71 / 4 / 0. No se atribuyó el error de importación al archivo original.
- Enlaces locales, un único prompt, whitespace y hashes protegidos comprobados. Conteos, comandos, hashes e inventario retenido en la [evidencia documental](../../evidence/AI_COACH_COLLECTIVE_TACTICAL_DESIGN_2026-09-17.json).
- No se ejecutaron suites funcionales ni smoke: la entrega modifica documentos y una copia del Excel, sin cambiar parsing, exportadores, hechos derivados, schemas ni runtime. No sustituye ningún gate funcional posterior.

## Límites y siguiente acción

Las fichas son material abierto de desarrollo. El documento completo y el Excel permiten ver cortes futuros y propuestas; la revisión no se declarará ciega. Se registrará la exposición del revisor y, en una evaluación formal futura, cada prefijo deberá entregarse separado.

No se ha ejecutado un modelo de interpretación, validado un plan táctico en una demo real ni aprobado un consejo. Las carencias de visión, rutas, movimiento, arma, C4 y economía conservan sus límites. S2–S3, elegibilidad, entrenamiento y migración 44/44 siguen pendientes. Corpus, bundles, temporales anteriores, catálogo, interfaz temporal y CURRENT no se modificaron.

La siguiente acción es **revisar F01-K1**: presentar exclusivamente contexto y hechos, recoger la interpretación del usuario y sus señales, y después contrastarla con la propuesta del agente. El enfoque general ya fue aportado; las respuestas específicas y el alcance funcional se concretarán durante esa revisión.
