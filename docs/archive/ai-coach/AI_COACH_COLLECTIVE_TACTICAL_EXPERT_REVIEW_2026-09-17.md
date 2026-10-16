# Recepción de revisión experta ICT-D1

17 de septiembre de 2026. Se reciben 18 interpretaciones libres y 18 entradas de información faltante en «Fichas tácticas» del [Excel actualizado por el usuario](../../../outputs/stratai-tactical-review-20260917-7ed3f25/StratAI_Plan_del_coach_ICT-D1_v2.xlsx). Registro de desarrollo, sin prueba ciega, ground truth final ni ejecución de un modelo. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y siguiente acción.

## Fuente y alcance

SHA-256 recibido: `9DA2162C83FF0A43A33925F51135117D8926C7DBD2AD1DAD6466797BE401560C`. Hoja «Fichas tácticas», F6:G23. Se preserva una copia de recepción y su inventario en la evidencia. El libro fuente no se modifica.

El usuario simplificó «Fichas tácticas» de doce a nueve columnas. Se respeta esa estructura; no se reponen las columnas retiradas ni se suponen valores de exposición. «Contexto colectivo» sigue sin anotaciones y las 18 valoraciones de «Hipótesis de diseño» siguen en «Pendiente». Las diez hojas anteriores no presentan cambios de contenido frente a la versión entregada. Las respuestas libres no completan automáticamente esos otros campos.

La exposición a hipótesis y cortes posteriores no consta. El material fue entregado completo y se trata como revisión abierta. No se inventan orden de respuesta, fecha individual de cada anotación ni revisión independiente.

## Resultado principal

Antes de puntuar interpretación táctica, debe corregirse la información que recibe el experto. Las fichas omitían localización individual, utilizaban alias abreviados y accesos genéricos, dejaban relojes y continuidad incompletos y describían estados de conocimiento con una exclusividad no acreditada. Dos secuencias fueron objetadas expresamente por su realismo. No se considera al revisor equivocado por interpretar una premisa ambigua.

## Tratamiento de cada respuesta

| Ficha | Aportación del experto, resumida | Tratamiento en el diseño |
|---|---|---|
| F01-K1 | Interpretación limitada por jugadores no localizados. | Enumerar todos los actores. El rango «T1–T5» se leyó como extremos y no expresa posiciones exactas. |
| F01-K2 | Lectura de ejecución B, utilidad y posible cobertura trasera; falta T3. | Conservar esa lectura como revisión contextual y aportar posición de T3 por corte. |
| F01-K3 | Ejecución B interpretada y propuesta de que T4 ayude; se supone que sólo han pasado 20 segundos. | El consejo sobre T4 queda condicionado al reloj y amenazas supuestos. K3 no fijaba tiempo actual. |
| F02-K1 | Defensa estándar según el experto; pide resultados de los ataques previos para decidir si reforzar. | Añadir desarrollo y resultado causal de rondas previas; no etiquetar tres CT automáticamente como refuerzo. |
| F02-K2 | Contacto parcial, relevancia de ver C4 y funciones de apartamentos/corta/conector. | Concretar posiciones y cobertura; separar deber de comunicar de comunicación observada. |
| F02-K3 | Lectura favorable, pero «acceso» y posición de CT2 no se entienden. | Sustituir ubicaciones genéricas por posición y acceso inequívocos antes de validar funciones. |
| F03-K1 | Defensa estándar considerada razonable. | No exigir una explicación de refuerzo para una distribución que también admite despliegue estándar. |
| F03-K2 | Mantener sería normal si CT1 no comunica. | Conservar el condicional: la ausencia de comunicación no está acreditada. |
| F03-K3 | Se remite a F02-K3 y pregunta por T2. | Conservar el alias literal T2; no corregirlo a CT2 sin base. Nuevo formato explicitará identidades y perspectivas, sin revelar posiciones enemigas desconocidas. |
| F04-K1 | Presión e información distribuida, denominada SPLIT por el experto. | Conservar su vocabulario y concretar su significado con ejemplos; no fijar aún una taxonomía universal. |
| F04-K2 | Presión de medio que puede desequilibrar defensa; riesgo de avanzar solo sin cobertura. | La valoración depende de posiciones, ángulos, cobertura e información aún no descritos. |
| F04-K3 | Secuencia objetada por poco realista y no evaluable objetivamente. | Excluir esta premisa como prueba táctica hasta reconstruirla o sustituirla; «99%» es expresión del revisor, no estadística medida. |
| F05-K1 | Contacto/granada insuficientes para cambiar toda la defensa; cuestiona conocimiento exclusivo de CT1. | Corregir: contacto registrado para CT1, comunicación y demás receptores desconocidos. |
| F05-K2 | Mantener conector para preservar control de medio. | Registrar criterio condicionado al reparto y coberturas que faltan, sin regla universal de posición. |
| F05-K3 | Secuencia de plantado objetada como imposible con las premisas mostradas. | Excluir esta secuencia como prueba táctica hasta reconstruir la continuidad. No declarar imposibilidad física general demostrada. |
| F06-K1 | La flash y quien la porta condicionan la coordinación; considera posible cambiar de punto con 50 segundos. | Concretar posición de T2 y rutas. El juicio no acredita cualquier llegada ni una ETA física. |
| F06-K2 | Falta lugar de la muerte; plantea reagrupar y usar potencia de fuego tras perder utilidad. | La ficha sólo acredita pérdida de la única flash del grupo. Completar inventario antes de afirmar ausencia de toda utilidad. |
| F06-K3 | Guardar puede ser razonable por tiempo, recursos y marcador; pide economía del equipo. | Añadir economía e inventario actuales y escenarios de la ronda siguiente sin usar compras futuras. |

Las respuestas originales completas, con sus expresiones y errores de escritura, están en la [evidencia de recepción](../../evidence/AI_COACH_COLLECTIVE_TACTICAL_EXPERT_REVIEW_2026-09-17.json). Esta tabla resume y distingue la interpretación del agente; no sustituye el texto del experto.

## Criterios incorporados

- Presentar cada jugador por separado, con posición comprensible, orientación, desplazamiento y recursos relevantes. Evitar rangos como «T1–T5» o «T1–T3» en la ficha mostrada.
- Mostrar un reloj actual en cada corte y continuidad espacial entre cortes. Precisar el acceso, el lugar de una baja y qué cambios ocurrieron.
- Distinguir contacto registrado, información recibida, comunicación no registrada y conocimiento desconocido. No inferir silencio ni conocimiento exclusivo por ausencia de una traza.
- Relacionar patrones previos con resultados, desarrollo y condiciones disponibles de esas rondas. Haber ganado antes no prueba por sí solo una defensa óptima.
- Incorporar C4 observado, inventario de utilidad completo, zona que se abandona, economía actual y opciones futuras condicionadas.
- Separar la objeción al material de la preferencia táctica. F04-K3 y F05-K3 permanecen conservadas como historia y no se usarán como prueba táctica hasta reconstruirlas o sustituirlas.

## Pendientes y siguiente acción

Las opiniones sobre T4, conector, rotación, guardar o reagrupar quedan condicionadas a sus premisas. No se convierten porcentajes coloquiales en probabilidades ni vocabulario experto en categorías ejecutables sin concretarlo. No se declara aprobada ninguna capacidad ni resuelto S2–S3.

Rehacer **sólo F01-K1** en una nueva versión con actores enumerados, posiciones claras y estado completo para el juicio buscado. Priorizar un corte de material Mirage ya aprobado y existente si permite respaldar esos datos, sin adquirir ni reprocesar. Si faltan datos críticos, identificarlos antes de pedir otra interpretación. No pedir al usuario repetir las 18 respuestas. Tras comprobar juntos que la nueva presentación funciona se decidirá cómo continuar.

La [especificación v0.2](../../ai-coach/AI_COACH_COLLECTIVE_TACTICAL_INTERPRETATION_DESIGN.md) desarrolla el formato mínimo. Esta recepción sólo modifica documentación y añade evidencia. El Excel permanece intacto y no se ejecutan suites funcionales ni smoke porque no cambian productores, schemas o runtime.
