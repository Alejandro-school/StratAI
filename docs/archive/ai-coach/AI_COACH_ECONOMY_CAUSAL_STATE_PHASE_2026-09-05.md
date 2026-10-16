# Paso 3A — Estado económico causal

Fecha: 5 de septiembre de 2026. **Implementado, validado y cerrado operativamente; ausencia de la salida temporal verificada a las 13:29 UTC.** El [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y siguiente acción. La [evidencia máquina](../../evidence/AI_COACH_ECONOMY_CAUSAL_STATE_SINGLE_DEMO_SMOKE_2026-09-05.json) conserva comandos, versiones, conteos, abstenciones e inventario de hashes.

## Resultado implementado

El parser de Go dispone de un modo optativo de observación económica. Captura propiedades nativas antes de la consolidación de fin de ronda y separa tres compartimentos: estado inicial disponible, ejecución durante freeze y reconciliación posterior. Conserva cero, `false` y ausencias explícitas, con origen, tick de observación y tick de disponibilidad. No rellena el estado inicial con compras, devoluciones, resultados o dinero futuro.

El [contrato económico v1](../../contracts/economy/canonical-economy-candidate-v1.md) define `1.0.0-candidate.1`, sus dos schemas, alias opacos y restricciones. El [validador independiente](../../../ai_coach/contracts/validate_economy_candidate.py) comprueba identidad, hashes, fuentes, tiempos, motivos de disponibilidad, particiones y joins; recalcula los diagnósticos contra el artefacto económico canónico verificado. El acceso inicial se limita al equipo propio y exige cinco actores con precompra acreditada; no habilita consejos ni decisiones económicas.

La exportación normal conserva `3.8.0`, parser `v16` y manifest `stratai.canonical_manifest@3`. El modo económico procesa una sola vez la demo, añade dos archivos candidatos al bundle temporal y no activa productores espaciales. Catálogo `1.6.0`, `DecisionExample`, ventanas de decisión y elegibilidad histórica permanecen sin cambios. Se mantienen `training_allowed=false` y `product_supported=false`.

## Verificación ejecutada en esta fase

- `npm run check:all`: correcto. Frontend: 88 pruebas en 23 archivos; Node: 14 aprobadas y 3 omitidas; Go: todos los paquetes; backend Python: 43 pruebas; build frontend correcto.
- AI Coach: 276 pruebas aprobadas en 54,64 s. Los dos schemas nuevos elevan a 35 el registro runtime; se actualizaron sus dos comprobaciones de conteo.
- Tras corregir el lector de dinero canónico del validador independiente: 39 pruebas relevantes aprobadas en 3,95 s; Ruff de los archivos Python nuevos correcto. La primera validación real había supuesto un escalar donde `3.8.0` usa un objeto con `amount`, `status` y `source`. Se corrigieron lector y fixture; el productor y los artefactos generados no cambiaron. Se validó de nuevo la misma salida, sin otro parseo.
- Validador económico: correcto en 1,254 s. Validación semántica canónica completa: **0 errores** en 361,232 s.
- Verificación documental final: 48 documentos, 207 enlaces locales sin destinos ausentes, un único prompt activo y `git diff --check` correcto. El inventario inicial de 805 archivos no detecta cambios ajenos a los previstos para esta fase; se conserva la interfaz temporal.

Las pruebas cubren causalidad y futuro, dinero cero frente a ausente, primera muestra tardía, actividad anterior/mismo tick, desconexión, inventario desconocido, reinicios, referencias de drops, integridad, unión por actor, acceso propio frente a rival y protección de salida temporal. La demo real cubre pickups y refunds; no contiene callbacks de drop capturados por este observador. Las referencias de drop aliado se prueban con fixtures y no se presentan como evidencia real de esta partida.

## Smoke de una única demo aprobada

Alias: `demo-a11eb0d600ee702c9edb`. SHA-256: `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Se verificó aprobación y hash de origen antes de parsear. Build: `working-tree-economy-3a-20260905-r1`. Una pasada del productor, 88,355 s; dos intentos del validador económico sobre esa misma salida. Es una partida de desarrollo, sin condición de test sellado.

| Medida | Resultado |
|---|---:|
| Rondas canónicas / instancias observadas | 24 / 25; el número 1 aparece dos veces |
| Participantes | 10 |
| Snapshots iniciales / freeze / fin | 25 / 25 / 25 |
| Estados iniciales de jugador | 250 |
| Precompra acreditada | 77 |
| Excluidos por actividad anterior a la captura | 173 |
| Observaciones monetarias / dinero cero observado | 748 / 47 |
| Inventarios observados | 748 |
| Callbacks en ejecución estricta / fuera o en límites | 1.044 / 547 |
| Pickups sin atribución de compra / refunds | 1.547 / 44 |
| Drops capturados / enlaces de drop aliado | 0 / 0 |
| Comparaciones recomputadas independientemente | 460; todas con delta cero |
| Estados de equipo con cinco actores precompra | 5; cuatro con número de ronda repetido y uno no ambiguo |
| Entidades atribuidas a dos inventarios del mismo snapshot | 0 |

Los dos estados de participante ausente conservan los ocho campos como no disponibles. `mp_maxmoney` está ausente en las 75 capturas de contexto; no se sustituye por un valor de configuración externa. Las otras cinco variables y ambos marcadores se observan en las 75. Las versiones de precios y reglas heredados se conservan sólo como referencias no aplicadas, sin compatibilidad acreditada para inferir gastos o ingresos.

Los 77 estados acreditan una frontera factual; no son 77 compras correctas, ni 77 consejos, ni ejemplos elegibles. Los cinco estados completos tampoco son cinco oportunidades competitivas acreditadas: cuatro pertenecen a las dos instancias numeradas como ronda 1. La unión a agregados por ese número se bloquea; sus comparaciones no se fabrican. Sólo la instancia 20, ronda visible 19, lado CT, ofrece un equipo completo con número no ambiguo en esta comprobación.

## Preservación e historial del cierre operativo

Se compararon antes y después los paths opacos, tamaños y tiempos de modificación de **4.356 archivos / 42.372.398.910 bytes**, incluidos los 44 bundles aprobados, 50 fuentes, exports de StratAI y releases. Los SHA-256 de 181 archivos de control —manifests, demo usada y todos los archivos de release— coinciden. No se afirma haber recalculado después del smoke el contenido completo de todos los demás artefactos. Las 15 evidencias anteriores y el catálogo conservan sus hashes. No hubo descarga, reproceso por lote, publicación, promoción ni cambio de `CURRENT`.

La salida generada tenía **75 archivos / 736.832.581 bytes**, con todos sus hashes conservados en la evidencia antes del intento de limpieza:

`C:/Users/kerch/AppData/Local/Temp/stratai-3a-smoke-04feac4db16b4a529ce3d804247665f8`

La revisión automática rechazó inicialmente el comando de borrado con el motivo literal `blocked by policy`; el comando no llegó a ejecutarse. En esa comprobación la carpeta y el inventario seguían presentes, por lo que se conservó el cierre como pendiente. Su resolución posterior se registra a continuación, sin modificar los resultados funcionales.

Los temporales de 1B.12 permanecen intactos y conservan su deuda independiente. No se intentó su limpieza en esta fase.

### Recomprobación del cierre operativo — 5 de septiembre, 13:25 UTC

A petición del usuario se ejecutó el siguiente paso registrado. Se resolvió la carpeta exacta bajo TEMP, se descartaron enlaces en ancestros y descendientes y se contrastaron correctamente los 75 hashes con la evidencia conservada. Un comando PowerShell con la ruta literal volvió a ser rechazado por la revisión automática con `blocked by policy`, antes de ejecutarse. La comprobación posterior conservaba 75 archivos / 736.832.581 bytes. Se añadió el intento a `temporary_output_cleanup.rechecks` sin alterar resultados del productor, smoke o validadores. El cierre permaneció pendiente en ese momento; no se repitió ninguna ejecución funcional ni se abrió 3B.

### Cierre verificado — 5 de septiembre, 13:29 UTC

Al comprobar el estado para el usuario, la carpeta exacta ya no existía. Se verificaron tanto su ausencia como cero entradas con ese nombre en el directorio TEMP padre accesible. La eliminación ocurrió fuera de las acciones ejecutadas por el agente; no se acredita su autor ni el instante de borrado. Se registra el cierre operativo con la evidencia pequeña intacta, sin repetir parser, smoke o suites ni atribuir al agente una eliminación que sus herramientas habían rechazado. Se retira el prompt de limpieza y se prepara el de 3B; esa fase funcional no se ha iniciado.

## Continuidad propuesta

La siguiente pieza funcional es **3B: mejorar la cobertura precompra y acreditar la correspondencia entre instancias y rondas competitivas**. Debe investigar una captura anterior respaldada por propiedades nativas, sin ignorar actividad o reconstruir el pasado desde cambios posteriores. Después seguirán contratos de ventana/acción económica, reglas compatibles y alternativas viables. La revisión humana, el comparador, el perfil, las misiones y el producto continúan pendientes según el plan.
