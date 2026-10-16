# Hipótesis de ataque B — contrato local acotado v2

`stratai.replay_attack_hypothesis@2`, `BoundedAttackHypothesisSession` y adaptador optativo `1.5.0-local.1`. Conserva el criterio experto y los hechos del [contrato v1](replay-attack-hypothesis-v1.md). La revisión corrige la interfaz de generación tras los rechazos de la primera tanda; no añade una hipótesis de juego. Estado y siguiente acción: [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md).

El usuario eligió punto 2 de B Apps y grupo de 2–3 T. El punto corresponde a XY `(-1500, 750)` en el [mapa revisado](../../../outputs/stratai-attack-depth-framing-20261015/revision-b-apps.html). La cercanía sigue siendo una estimación del modelo sin radio táctico experto; la referencia de zonas mantiene compatibilidad de build no verificada. No se convierte Apartments entero en zona de ataque.

## Reparación de la interfaz

En v1, el generador admitía muchas filas donde el render sólo permitía una hipótesis de ataque. N01/W01 repitieron hipótesis y agotaron 8.192 tokens; sus resultados quedaron rechazados. El [diagnóstico con fuentes primarias](../../../outputs/stratai-attack-hypothesis-20261015/SCHEMA_DIAGNOSIS.md) conserva la primera tanda completa y explica por qué una propiedad de unicidad en JSON Schema no bastaba para guiar la generación.

La selección v2 admite **una hipótesis opcional o hasta ocho principios condicionales**, separados como alternativas completas de la sección. Una lista vacía sigue siendo válida. La hipótesis elige exactamente una combinación canónica de dos/tres referencias actuales de jugadores distintos. Estas combinaciones se suministran al modelo y al esquema; acreditan identidad distinta, no agrupación espacial o intención. El render verifica las mismas restricciones sin depender del esquema.

Terminar exige `action: finish`, `indices: []` y lectura completa. Consultar exige `action: consult`, índices no vacíos dentro del límite causal y `reading: null`. Se mantienen 32 consultas, 512.000 bytes y ocho llamadas máximas; no se amplía contexto, salida ni se reparan respuestas automáticamente.

## Datos y compatibilidad

La sesión deriva sólo combinaciones de los jugadores ya elegibles en v1. Los hechos, distancias, soportes de zona/ronda/C4, información posible y restricciones de sonido permanecen intactos. El recibo identifica @2 y el hash de la nueva entrada, manteniendo el hash del paquete original. El render conserva los hechos protegidos y declara @2 en su verificación.

El adaptador @1.4 y la sesión @1 se conservan para reproducir la primera tanda; los formatos anteriores al cambio y el comportamiento por defecto del cliente siguen intactos. El experimento final selecciona explícitamente @2/1.5.0; no cambia los pesos del Qwen3.5:9b original ni habilita una ruta de producto.

La elegibilidad excluye muertos, posiciones antiguas o ambiguas, ronda cerrada, plantado, objetivo ausente y pausa acreditada. Estas exclusiones limitan el alcance del instrumento; no demuestran ausencia de ataque en el mundo. Posible ataque B no confirma plantado, apoyo desde Medio, defensa reactiva, eficacia o recomendación.

## Comprobación y lectura de resultados

Las regresiones cubren las repeticiones observadas, cuentas duplicadas, tamaños, orden canónico, combinaciones inválidas de acción/índices/lectura, abstención, los ocho principios y consultas causales. Una mutación que vuelve a permitir dos hipótesis debe fallar. Se repiten suites y la misma demo aprobada después de la reparación, conservando ambas capturas.

El smoke final cubre cortes reales con cero, dos y tres jugadores elegibles, su historia causal y el render del productor nuevo. El smoke inicial completo de once entradas permanece conservado. La nueva comparación vuelve a ejecutar N01/N04/N10/W01 como lote completo, sin reintentos selectivos. Los tres N conservan las entradas congeladas originales; W usa la nueva captura de la misma situación geométrica, sin forzar identidad de IDs nativos entre extracciones.

La integridad mecánica se verifica; que Qwen lea correctamente la cercanía y la posibilidad de ataque requiere revisión táctica. Una respuesta válida puede contener una mala hipótesis; una abstención no prueba ausencia de ataque. No se presenta esa revisión como evaluación independiente ni se deriva una tasa de acierto de cuatro ejemplos de desarrollo.
