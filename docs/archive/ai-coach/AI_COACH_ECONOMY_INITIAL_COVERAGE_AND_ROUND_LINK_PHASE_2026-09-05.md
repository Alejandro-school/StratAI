# Paso 3B: frontera inicial y correspondencia de rondas

Fecha: 5 de septiembre de 2026. **Implementado, validado y cerrado operativamente.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina la siguiente acción. [Evidencia máquina](../../evidence/AI_COACH_ECONOMY_INITIAL_COVERAGE_AND_ROUND_LINK_SINGLE_DEMO_SMOKE_2026-09-05.json).

## Resultado medido

La captura dentro de `RoundStart` no recupera estados precompra en esta demo: sus 2.000 propiedades de actor coinciden con las del primer `FrameDone`. Demoinfocs `v5.2.0` entrega progresión de ronda y eventos aplazados al terminar el mismo fotograma nativo. Un callback anterior no prueba una observación anterior a la modificación de propiedades.

| Medida | Resultado |
|---|---:|
| Rondas canónicas / instancias observadas | 24 / 25 |
| Sondas iniciales de actor | 250 |
| Estados precompra del control v1 | 77, sin cambio respecto a 3A |
| Estados bloqueados por actividad del tick inicial | 173 |
| Recogidas en el tick inicial posteriores a la sonda | 353 |
| Correspondencias competitivas verificadas | 23 |
| Instancias excluidas por ausencia de final oficial | 2 |
| Actores iniciales acreditados en rondas verificadas | 64 |
| Estados acreditados fuera de rondas verificadas | 13 |
| Equipos completos accesibles con correspondencia verificada | 3 |

Las tres oportunidades son CT y T en la ronda 1 —instancia 2, T0 6061— y CT en la ronda 19 —instancia 20, T0 116279—. En 3A había cinco equipos completos según observación inicial: cuatro con número de ronda repetido y uno con número no ambiguo. 3B verifica la pertenencia competitiva de tres equipos; no genera tres recomendaciones ni acredita tres decisiones económicas elegibles.

Se excluye la instancia inicial con número 1 y T0 1, y la última ronda, número 24 y T0 153469. Ambas carecen de `RoundEndOfficial`. El último frame del replay no se utiliza para fabricar ese callback. Los flags nativos de partida iniciada y freeze son verdaderos en las 25 sondas: por sí solos no distinguen el inicio preliminar.

## Cambios entregados

El [contrato v2](../../contracts/economy/canonical-economy-candidate-v2.md), `1.0.0-candidate.2`, conserva los compartimentos y lector v1, añade sondas iniciales, flags nativos con presencia explícita y callbacks previos al inicio. Mantiene la exportación normal `3.8.0` y el catálogo `1.6.0`; registra dos schemas nuevos, 37 en total.

El productor enlaza por identidad, inicio, final oficial y participantes. El validador verifica los artefactos canónicos por hash y recalcula los 25 enlaces. El lector inicial exige cinco aliados, observaciones completas en T0 y ausencia de actividad del tick inicial; copia los hechos propios y excluye economía rival y consecuencias posteriores. Los callbacks de refund previos a un reinicio no desaparecen al abrir otra instancia. No se fusionan propiedades observadas en momentos distintos para completar una captura.

Se conserva `--ai-coach-economy-candidate` como modo aislado; `--economy-candidate-version 2` selecciona la nueva versión. El valor por defecto sigue siendo 1. Las dos versiones rechazan destinos fuera de TEMP, no vacíos, enlaces y modos incompatibles antes del parsing.

## Verificación

- `npm run check:all`: correcto; 88 pruebas frontend, 14 Node con 3 omitidas, todos los paquetes Go, 43 pruebas Python backend y build frontend.
- Suite AI Coach: 312 pruebas, 68,21 s. Después se reforzó el validador `--require-observed` para exigir equipos competitivos no vacíos y el acceso común para aplicar las condiciones v2 a datos nuevos: 78 pruebas pertinentes correctas, 10,98 s. El productor no cambió.
- Ruff y contrato global correctos: 17 capacidades, 15 decisiones, 51 acciones, 18 outcomes, 58 campos; 37 schemas y 19 códigos de motivo.
- Una demo distinta y una pasada del productor: 86,414 s. Tres verificaciones económicas del mismo artefacto, todas correctas; las posteriores comprueban el validador y acceso común reforzados. Ninguna repetición del parsing. Ambos accesos iniciales acreditan los mismos tres equipos.
- Validación semántica completa del bundle: 0 errores, 314,564 s. Recalculadas 460 comparaciones monetarias y 25 correspondencias; 1.044 callbacks en ejecución y 547 en otros compartimentos.
- Invariancia de hechos frente a cambios en rival, eventos futuros, reconciliación y rondas siguientes: comprobada en las tres oportunidades reales y en fixtures. Ningún inventario inicial con varios propietarios de una misma entidad.
- Fixtures cubren actividad antes/después de la sonda y después del primer frame en el mismo tick, refund previo al inicio, fallback sin adelantar disponibilidad, ausencia nativa, desconexión, cambio de lado, entidades duplicadas, fugas futuras, identidades, límites, joins duplicados y alteraciones rehasheadas.

El smoke usa `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, Mirage, 330.945.844 bytes. Build `working-tree-economy-3b-20260905-r1`, binario SHA-256 `74bf00db3e2f4bbe0e9d31804d30b767c6ebbe3c6f2d552d06390de6937a6f81`. Comandos exactos, inventarios, versiones, hashes y tiempos figuran en la evidencia.

Las 44 partidas aprobadas y las 50 fuentes permanecen protegidas. Se compararon rutas opacas, tamaños y fechas de 4.356 archivos y hashes de 181 controles; no se rehashearon todos los cuerpos de los 44 bundles. Catálogo y 16 evidencias previas sin cambios. No hubo descargas, promoción, migración 44/44, entrenamiento ni cambios a `CURRENT`, releases o temporales espaciales.

## Pendiente operativo

La evidencia pequeña se guardó antes del intento de limpieza. Se verificó el destino literal bajo TEMP, 75 archivos / 737.214.875 bytes y ausencia de reparse points. La revisión automática de aprobación rechazó:

```powershell
Remove-Item -LiteralPath 'C:\Users\kerch\AppData\Local\Temp\stratai-3b-smoke-h3466t8c' -Recurse -Force -ErrorAction Stop
```

Motivo: **`blocked by policy`**. Se registró el rechazo el 5 de septiembre a las 15:01 UTC y se confirmó que la salida seguía presente. No se intentó otro método de borrado. El prompt de limpieza de esta fase se retiró al pedir el usuario implementar 3C el 6 de septiembre; la deuda y los límites se conservan en el [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md). Resolverla exige retirar sólo esa salida cuando sea posible o acreditar su ausencia externa; no repetir demo, suites o validación canónica.

## Límite funcional

La cobertura sigue siendo pequeña: 173 estados no se arreglan cambiando el orden de callbacks. No hubo drops reales ni refunds antes de la sonda en el tick inicial; esas variantes se cubren con fixtures. `mp_maxmoney` sigue ausente y no se acreditan precios o recompensas heredados. La misma demo repetida es desarrollo, no un test final sellado.

Tras el cierre, 3C debe fijar oportunidad económica, ventana de freeze, reglas compatibles y reconstrucción de la acción observada. Comparar cestas, recomendar, revisión humana, perfil runtime y misiones siguen pendientes. `training_allowed=false`, `product_supported=false` y cero capacidades elegibles permanecen vigentes. La deuda operativa espacial 1B.12 sigue separada.

## Cierre operativo verificado — 6 de septiembre de 2026

Tras comunicar el usuario que había terminado la limpieza, se verificó a las 2026-09-06T08:16:41.129992+00:00 que los destinos temporales de esta fase no existen ni figuran en su directorio padre accesible. La eliminación se realizó fuera de las acciones del agente; no se atribuye hora exacta. Se conservan inventarios, hashes y rechazos históricos. No se repitieron productores, demo ni suites para este cierre. El plan activo mantiene el siguiente trabajo funcional.
