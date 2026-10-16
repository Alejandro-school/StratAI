# Contrato candidato de oportunidades económicas v1

Versión `1.0.0-candidate.1`, 6 de septiembre de 2026. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado, evidencia y siguiente acción. Este contrato define fichas factuales de desarrollo para `CAP-ECONOMY-001` y `DEC-ECONOMY-001`; no acredita compras correctas ni habilita consejos.

## Qué representa una ficha

Una ficha reúne el dinero y equipo que podían conocerse antes de decidir, los cambios observados durante la preparación de la ronda y los motivos que impiden comparar compras. Se conserva una fila por instancia de ronda observada y lado, también cuando faltan datos. El número visible de ronda no sirve como identidad única.

El estado inicial exige el [lector económico v2](canonical-economy-candidate-v2.md): cinco aliados reconciliados, propiedades completas en el tick inicial, ninguna actividad de objetos que vuelva ambiguo ese tick y correspondencia competitiva verificada. Si falla, `initial_state=null` y se conserva el motivo. No se completan huecos con rondas siguientes ni con información rival. Los enlaces competitivos usan el cierre posterior sólo para validar pertenencia, nunca como señal predictiva inicial.

Una fila, un equipo inicial acreditado y una decisión elegible son tres cantidades distintas. La demo de desarrollo produce 50 filas, tres equipos iniciales completos y cero decisiones económicas elegibles. Los 77/250 estados individuales acreditados por 3B no aumentan con este contrato.

## Ventana y separación temporal

| Parte | Regla |
|---|---|
| Estado inicial | Captura propia y aliada disponible exactamente en `RoundStart` (T0), bajo las condiciones v2. |
| Ejecución | Todo el intervalo observado `T0 < tick < RoundFreezetimeEnd`. Sustituye la ventana genérica de 0,25–2 segundos únicamente para economía. |
| Límites | Callbacks exactamente en T0 o fin de freeze, y callbacks previos a la sonda, en compartimentos separados. El orden de callbacks no acredita orden de subtick. |
| Punto de comparación final | Sólo la captura del fotograma exacto de fin de freeze. Una captura tardía no rellena esa posición. |
| Después del freeze | Excluido de esta acción. No se afirma que el fin del freeze sea el cierre de la tienda. |
| Duración nominal | Las variables del servidor se conservan con presencia y valor nativos. No sustituyen los ticks observados ni se les asignan unidades inferidas. |

La ventana puede incluir una pausa real. En la primera ronda competitiva de la demo dura 1.695 ticks; recortarla a dos segundos perdería ejecución. `initial_features(row)` devuelve una copia del estado inicial de una ficha previamente validada; no devuelve ejecución, diferencias, reglas auxiliares ni duración futura del freeze. Los IDs sirven para enlazar registros, no como señales predictivas.

## Qué reconstruimos de la ejecución

Para los cinco actores iniciales se conservan valores antes/después y diferencias de dinero, contador nativo de gasto, armadura y valor nativo del equipo. Casco, kit y zona de compra conservan valores booleanos, sin restarlos. Cada diferencia declara cuándo estuvo disponible. Actor desconectado, cambiado de lado o ausente en la captura exacta produce valores finales no disponibles.

El inventario compara pertenencia de entidades: conservadas, añadidas y retiradas. No acredita cantidades de munición/granadas ni transacciones de tienda. Una entidad reutilizada con un arma distinta invalida esa transición. Los eventos originales de recogida, drop y refund conservan procedencia y alias opacos; una recogida no se convierte en compra y un refund no acredita su importe.

`observed_action.status` sólo puede ser `partial` o `unavailable`; `action_id` y `basket` permanecen `null`. Diferencia monetaria cero no significa eco o ahorro, y la suma de diferencias no demuestra el coste de una cesta. Incluso las filas sin estado inicial pueden conservar callbacks de su lado para diagnóstico; no son ejecución plenamente reconstruida ni oportunidades elegibles.

## Comprobación de reglas

La comprobación informa presencia de `mp_buytime`, `mp_freezetime`, `mp_maxmoney`, `mp_startmoney`, `mp_maxrounds` y `mp_overtime_enable`, y si los cinco aliados estaban en zona de compra en T0. En los tres equipos acreditados falta `mp_maxmoney`.

Los hashes de precios y reglas heredados se retienen como referencias. No prueban compatibilidad con la versión del juego de la demo. Continúan sin acreditarse el vínculo de precios con esa versión, recompensas y bonus anterior, equipamiento comprable y límites de objetos, reglas de devolución y aceptación de cooperación por compañeros. El comprobador devuelve `unverified_for_comparison` incluso si todas las variables nativas están presentes. Esta versión especifica cómo bloquear la comparación; no contiene todavía un paquete de reglas económicas compatible ni un simulador de compras.

Se mantienen las cinco acciones futuras del catálogo —compra completa, parcial, eco, conservar equipo y drop— y sus seis parámetros físicos existentes. La evidencia de esta fase no los rellena ni clasifica acciones.

## Archivos y uso

| Archivo | Contrato |
|---|---|
| [Protocolo](../../../ai_coach/contracts/economy_opportunity_protocol.json) y [schema](../../../ai_coach/contracts/economy_opportunity_protocol.schema.json) | `stratai.economy_opportunity_protocol@1`, versión `1.0.0` |
| `economy_opportunities.json` | [stratai.economy_opportunities@1](../../../ai_coach/contracts/economy_opportunities.schema.json) |
| `economy_opportunities_manifest.json` | [stratai.economy_opportunities_manifest@1](../../../ai_coach/contracts/economy_opportunities_manifest.schema.json) |

El materializador Python consume exclusivamente un candidato económico v2 y verifica su manifest, artefactos canónicos, hashes y correspondencias. El manifest nuevo vincula por SHA-256 datos y manifest de origen, protocolo y catálogo `1.7.0`. El registro runtime contiene 40 schemas. Se preservan exportación base `3.8.0`, parser `v16`, quality `12`, manifest base `@3` y lectores económicos v1/v2.

Desde la raíz, con `$sourceCandidateManifest` apuntando al manifest v2 ya generado en un smoke aislado y `$opportunitiesRoot` a una carpeta dedicada, nueva y vacía dentro de TEMP:

```powershell
backend/venv/Scripts/python.exe -B -m ai_coach.economy.candidate_io --source $sourceCandidateManifest --output $opportunitiesRoot
backend/venv/Scripts/python.exe -B -m ai_coach.economy.validate_opportunities "$opportunitiesRoot/economy_opportunities_manifest.json" --source $sourceCandidateManifest --require-observed
```

El productor rechaza destinos no temporales, no vacíos, enlaces y sobrescrituras antes de leer el candidato. El validador independiente recalcula diferencias y pertenencia de inventario desde el origen, verifica ventanas, compartimentos, razones de abstención y vínculos de contrato, y rechaza alteraciones incluso si se han rehecho sus hashes. `--require-observed` exige registros afectados no vacíos, incluidos equipos iniciales, diferencias, inventario y callbacks.

`comparison_allowed=false`, `training_allowed=false` y `product_supported=false` son obligatorios. No se modifica `DecisionExample` ni `TaskEligibility`; los 44 bundles aprobados no incorporan este candidato. La comparación de cestas, reglas plenamente acreditadas, acción itemizada, recomendaciones, perfil y misiones requieren trabajo posterior con sus propias comprobaciones.
