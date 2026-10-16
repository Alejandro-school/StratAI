# E02: defensa de bomba y recuento de vivos — 25/09/2026

En el corte final de E02 hay **2 T vivos frente a 1 CT vivo**. La bomba está
plantada y conserva **39,265625 s**. Han transcurrido 69 s desde el fin de freeze.
El recuento se ha comprobado en las capturas originales y posteriores del mismo
corte y se ha contrastado con las siete bajas públicas anteriores: tres T y
cuatro CT. No se deduce una muerte de posiciones ausentes.

## Lectura recibida

> Se ve un claro ejemplo de bomba plantada y defendiendo la bomba, en este caso paece que hay un jugador en A y otro debajo de palacio. Es una buena posición. No se cuantos enemigos quedan vivos. Estaría bien saberlo.

El [literal](../../../outputs/stratai-e02-review-20260925/response.json) se conserva
sin corregirlo. La normalización del agente es `defend_planted_bomb`, una etiqueta
de revisión admitida por el contrato existente. **No es un plan recién implementado
en el detector**. A y debajo de palacio son ubicaciones tentativas señaladas por
el experto; no se transforman en destinos ni en zonas verificadas automáticamente.
La valoración «Es una buena posición» también procede del experto. No se asignan
funciones individuales ni una superioridad táctica comprobada.

La lectura se recibió con la ficha v2 y antes de mostrar la predicción en esta
conversación. La conversación ya había aportado contexto factual de plantado y
reloj. Se registra revisión asistida, exposición privada desconocida y consulta
opcional de exposición pendiente. No se presupone una respuesta ni ceguera absoluta.

## Contraste de la predicción congelada

La predicción original se abstuvo, con `sustained_carrier_retreat` y
`unique_current_c4_carrier_unavailable`. No emitió la defensa tras plantar que
describe el experto. El [contraste v2](../../../outputs/stratai-e02-review-20260925/comparison-v2.json)
lo registra como `compatible_plan_not_emitted`. El detector vigente busca
preparación de entrada y no tiene una regla de defensa posplantado. La ausencia
de portador actual después de plantar no equivale a una captura incompleta de C4.
Este caso señala una carencia de cobertura; no establece por sí solo una tasa
de error ni una recomendación válida.

La revisión usa el episodio y contexto **originales**, enlazados desde el
[recibo de entrega v2](../../../outputs/stratai-round-context-20260925/E02-delivery.json).
No se recalcula ni ajusta la predicción. El [registro activo](../../../outputs/stratai-e02-review-20260925/active-review-scope.json)
mantiene 36 casos históricos, E01 excluida, E02 contrastada descriptivamente,
34 pendientes y cero puntuaciones de precisión. La comparación histórica incluye
E01, pero su revisión sigue fuera de la puntuación activa.

## Comprobación y alcance

La [auditoría factual](../../../outputs/stratai-e02-review-20260925/E02-diagnosis.json)
enlaza hashes y corte `sequence=2222`, `frame=15703`, `tick=15683`, `epoch=0`,
`instance=3`, ronda 2 de `demo-0beb9110ec408f80c39b`. Los diez jugadores tienen
identidad, conexión y estado vivo/muerto disponibles; ninguno se cuenta como
muerto por omisión. Los campos coinciden entre ambas capturas. El recuento
independiente de bajas usa sólo eventos hasta ese tick. No se conservan posiciones
ni recursos ocultos de rivales en esta evidencia.

Se validó el schema de revisión, la procedencia de las 36 predicciones y ambos
contrastes. Los 44 manifests, las fichas originales, E02 v2 y el código del
intérprete permanecen intactos. El Excel v19 añade únicamente IC28 en A33:G33
de «Contexto colectivo» y conserva las otras celdas, las 14 hojas, fórmulas,
estilos y controles. Se revisó su representación antes y después de editar.

Este cierre registra la revisión y responde a una consulta sobre datos ya
capturados. No cambia productores, runtime, contratos ni fichas; no requiere
repetir el smoke. Cero demos reprocesadas, descargas, promociones, limpieza o
entrenamiento. El uso accidental inicial de un Python sin `jsonschema` no produjo
artefactos de revisión; la ejecución posterior con el entorno del proyecto pasó.

## Siguiente acción

Incorporar a la ficha un contador de aliados/rivales vivos en el instante
seleccionado. Esta mejora factual está solicitada y no requiere volver a pedir
el criterio experto de E02. Debe preservar desconocidos, fronteras de ronda y
fuentes causales, sin añadir posiciones o recursos ocultos de rivales. Antes del
cierre funcional corresponderán pruebas y el smoke de una demo aprobado por
AGENTS.md. El recuento comprobado aquí **todavía no es ese contador automático**.

Después se puede continuar con E03, conservando las limitaciones de su ficha v1.
La nueva interpretación de defensa tras plantar necesita su propio encuadre
de fase; esta lectura no autoriza inferir funciones, calidad de posiciones o
consejos generales. No volver a solicitar esta lectura de E02.

[Evidencia](../../evidence/AI_COACH_E02_EXPERT_READING_2026-09-25.json).
