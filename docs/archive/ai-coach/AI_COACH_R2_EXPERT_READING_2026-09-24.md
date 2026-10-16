# Lectura experta de R2 — 24 de septiembre de 2026

Se recibe y conserva la respuesta del usuario sobre la ficha temporal de R2.
El [recibo literal](../../evidence/AI_COACH_R2_EXPERT_READING_2026-09-24.json)
separa sus palabras de la normalización del agente. IC23 del
[Excel v13](../../../outputs/stratai-r2-expert-review-20260924/StratAI_Plan_del_coach_ICT_I2_v13.xlsx)
conserva la respuesta completa y los matices. Las 14 hojas anteriores se mantienen.

## Lectura y límites

El experto interpreta que el equipo se dirige a A para ejecutar la jugada.
Propone que T5 podría regresar desde medio para reunirse con el equipo después
de la muerte del compañero. La posible muerte en medio y su relación causal con
el regreso de T5 permanecen como hipótesis.

El motivo de elegir A queda desconocido. Que el fallecido hubiera visto tres
rivales en B o en medio son ejemplos de información que podría cambiar esa
justificación, no observaciones de la replay. Tampoco se acredita qué recibió el
equipo ni que A fuera la mejor alternativa. La muerte no completa por sí sola
la percepción, el número de rivales, su posición o la comunicación del jugador.

La normalización a `prepare_entry` representa compatibilidad con preparar una
ejecución en A: el experto menciona expresamente ejecutar la jugada, además del
destino. No afirma intención interna cierta ni entrada consumada. T5 conserva
`regroup_with_team` como función posible, sin asignar funciones a los demás.

## Contraste exploratorio

La predicción congelada se **abstuvo** con
`no_joint_continuous_approach_with_c4`. El evaluador v2 devuelve
`compatible_plan_not_emitted`: existe una lectura compatible aportada por el
experto que la regla no emitió. No se presenta como tasa de error, fallo
independientemente acreditado ni motivo suficiente para relajar umbrales.

El informe técnico permite estudiar posteriormente qué condición excluyó la
lectura y cuánto contexto haría falta. Esta recepción no demuestra todavía que
la causa del desacuerdo sea la duración del intervalo, las rutas distintas, la
muerte o la información rival; son puntos para un diagnóstico posterior.

La ficha presentada añade inventario que no consumía el detector congelado y
usa una captura con distinta compilación. Se verifican nuevamente sus hashes
y la igualdad de los 65 estados factuales anteriores, excluyendo los IDs de
registro dependientes de procedencia. Se conservan ambos IDs de contexto, sin
presentar la ficha ampliada como entrada idéntica del detector.

La predicción no se mostró en la conversación antes de recibir la respuesta.
La exposición privada es desconocida y se mantiene la ayuda visual del mapa.
Los solapamientos y fechas ya auditados de R1/R2 siguen impidiendo acreditar
generalización independiente. Hay dos lecturas humanas contrastadas, R1 y R2,
sin precisión global. Los 84 templates v1 y las predicciones quedan intactos.

## Comprobaciones y continuación

Se ejecutan las validaciones existentes de esquema de revisión, manifiesto,
configuración, hash de predicción y vínculo de contexto, sin cambios de runtime.
El libro se revisa visualmente, se comprueban valores, fórmulas, estilos y
controles de las 14 hojas y se verifica la conservación de 71 archivos protegidos.
Sólo cambian A28:G28 y los resúmenes F26/F27 del libro nuevo. El recibo y la
[verificación reproducible](../../../outputs/stratai-r2-expert-review-20260924/comparison-verification.json)
conservan los comandos y los enlaces a las fuentes.

Cero parsing nuevo, entrenamiento, modificación del detector o promoción.
Este cierre registra una revisión humana; no cambia productores, hechos
derivados ni contratos runtime y no requiere repetir el smoke anterior.

La siguiente acción del plan es preparar la revisión del protocolo independiente
y el diagnóstico de cobertura contextual, manteniendo estos casos como material
exploratorio. Antes de una ampliación del detector se necesitan alcance concreto,
casos de contraste, criterios de abstención y encuadre experto de las decisiones
nuevas. No volver a pedir R1/R2 ni tratar las hipótesis sobre rivales como hechos.
