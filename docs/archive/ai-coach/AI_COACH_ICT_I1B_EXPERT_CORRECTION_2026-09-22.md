# ICT-I1.B — corrección experta del radar

22 de septiembre de 2026. Cierre documental de recepción de criterio.
El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y
la siguiente acción.

El usuario corrigió la premisa de la pregunta: la aparición en radar deriva de
contacto visual, con punto rojo durante el contacto e interrogante durante unos
segundos tras perder visibilidad. El [recibo](../../evidence/AI_COACH_ICT_I1B_EXPERT_CORRECTION_2026-09-22.json)
conserva sus palabras exactas y separa la síntesis del agente.

Se retira la alternativa «radar o contacto visual» y el ejemplo que no explicaba
el origen visual. La propuesta distingue observador/receptor y contacto actual
o perdido. No duplica un rival por sus representaciones ni confirma su posición
actual a partir de la persistencia. «Unos segundos» no fija un valor numérico.
La advertencia LOS/FOV del SDK no demuestra una mecánica sin origen visual.

IC17 del Excel v7 recoge literal, síntesis, límites y fuente S22. Se conservan
v6 y las versiones anteriores; comparación de las 14 hojas con valores,
fórmulas, estilos, anotaciones y validaciones previas preservados. Evidencia de
la comparación en `outputs/stratai-ict-i1b-feedback-20260922/verification.json`.

La respuesta no verifica receptores, datos reales del canal, codec, transiciones
del HUD, duración ni zona. No aprueba una regla táctica nueva. No se ha cambiado
parser, contratos o runtime, ni procesado demos o entrenado modelos. No procede
smoke funcional para esta recepción documental. La preparación anterior queda
como evidencia histórica, con su pregunta sustituida por la corrección recibida.
