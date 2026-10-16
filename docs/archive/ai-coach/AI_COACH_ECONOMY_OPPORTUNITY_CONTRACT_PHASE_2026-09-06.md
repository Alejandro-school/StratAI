# Paso 3C: oportunidades económicas y ejecución factual

Fecha: 6 de septiembre de 2026. **Contrato implementado, validado y cerrado operativamente.** El usuario pidió expresamente implementar 3C mientras la deuda de limpieza 3B seguía registrada. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) determina la siguiente acción. [Evidencia máquina](../../evidence/AI_COACH_ECONOMY_OPPORTUNITY_CONTRACT_SINGLE_DEMO_SMOKE_2026-09-06.json).

## Resultado

Existe un productor ejecutable que convierte el candidato económico v2 en fichas por equipo y ronda. Conserva el estado conocido antes de decidir, la ejecución durante todo el freeze y las razones por las que todavía no pueden compararse compras. Su validador independiente vuelve a calcular los cambios desde el origen.

| Medida en una demo aprobada | Resultado |
|---|---:|
| Instancias observadas / rondas canónicas | 25 / 24 |
| Fichas por instancia y lado | 50 |
| Equipos iniciales completos | 3 |
| Fichas sin equipo inicial completo | 47: 4 por correspondencia competitiva y 43 por estado inicial incompleto |
| Ventanas de freeze observadas | 50 |
| Ejecuciones parcialmente reconstruidas | 3 |
| Actores con diferencias entre capturas | 15 |
| Diferencias numéricas recalculadas | 60 |
| Transiciones de pertenencia de inventario recalculadas | 15 |
| Callbacks de ejecución / en ticks límite | 1.044 / 354 |
| Acciones clasificadas / reglas compatibles para comparar | 0 / 0 |

Las tres oportunidades conservan la cobertura de 3B: CT y T en ronda 1, y CT en ronda 19. Los callbacks se contabilizan también en filas no disponibles para conservar el denominador; en los tres equipos completos hay 28 callbacks: 27 recogidas y un refund. Los 77/250 estados individuales acreditados y los 173 ambiguos no cambian.

| Equipo/ronda | T0 → fin de freeze, ticks | Dinero conjunto inicial → final | Variación del contador nativo de gasto |
|---|---|---|---:|
| CT, ronda 1 | 6061 → 7756 | 4.000 → 1.400 | 2.600 |
| T, ronda 1 | 6061 → 7756 | 4.000 → 700 | 3.300 |
| CT, ronda 19 | 116279 → 117559 | 36.100 → 31.200 | 4.900 |

Estas diferencias describen capturas; no son cestas de compra itemizadas. En CT ronda 1 el valor nativo de equipo aumenta 2.650, mientras dinero/gasto varían 2.600: no se fuerza igualdad ni se transforma ese diagnóstico en transacciones inventadas. Las entidades añadidas tampoco prueban cantidades de granadas o munición.

## Implementación y límites

El [contrato de oportunidades v1](../../contracts/economy/canonical-economy-opportunities-v1.md) registra protocolo, datos y manifest nuevos. El catálogo pasa a `1.7.0` para vincular economía al protocolo y sustituir sólo su ventana genérica por todo el freeze observado. Hay 40 schemas runtime y 19 códigos de motivo, con los mismos 17 IDs de capacidad, 15 de decisión, 51 de acción, 18 de outcome y 58 de campo.

El productor Python consume v2 verificado por hash y valida el destino vacío bajo TEMP antes de leerlo. Se conservan Go, export `3.8.0`, parser `v16`, quality `12`, manifest base `@3`, lectores v1/v2 y los 44 bundles. El binario Go recompilado tiene el mismo SHA-256 que en 3B; el productor cambiado es el materializador de oportunidades Python.

La ejecución usa estrictamente `T0 < tick < fin de freeze`, con eventos límite separados y sólo captura final exacta. El fin de freeze no acredita fin de tienda. Se respetan pausas: la primera ronda competitiva tiene 1.695 ticks, sin recorte a dos segundos ni a una duración nominal. El acceso inicial excluye rival, ejecución, resultados y duración futura. Se comprobaron tres casos reales de invariancia de particiones y otros tres de invariancia frente a futuro/rival en la fuente.

La comprobación de reglas identifica carencias, pero **ninguna ficha acredita aún reglas completas para comparar compras**. Falta `mp_maxmoney` en los tres estados iniciales completos; los hashes heredados no vinculan precios a la versión del juego de esta demo. Recompensas/bonus previo, loadout/límites de objetos, reglas de refund y cooperación continúan sin acreditarse. `observed_action` es parcial o no disponible, con `action_id=null` y `basket=null`. Gasto cero no etiqueta eco/ahorro. Se entrega el contrato y su comprobador; la reconstrucción itemizada y el paquete de reglas compatibles siguen pendientes.

## Verificación

- `npm run check:all`: correcto; 88 pruebas frontend, 14 Node y 3 omitidas, paquetes Go correctos, 43 Python backend y build frontend en 12,37 s.
- Suite AI Coach: **347 pruebas**, 76,56 s; 32 pruebas nuevas de oportunidades. Ruff y contrato global correctos.
- Una demo distinta, una pasada de Go —83,748 s— y una del materializador cambiado —4,548 s—, después de las pruebas automatizadas.
- Validación v2 correcta en 3,668 s: 460 comparaciones monetarias y 25 correspondencias recalculadas. Validación de oportunidades correcta en 4,752 s: 50 filas, 60 diferencias y 15 transiciones recalculadas independientemente.
- Validación semántica completa del bundle: **0 errores**, 306,392 s. El helper de inspección tuvo una errata de nombre de campo en su parte de reporte; se corrigió sin modificar productor ni repositorio y se reutilizaron los dos reportes ya correctos. La validación canónica se ejecutó una vez. PowerShell informó salida 1 del wrapper con el aviso de arranque de Python redirigido; se conservaron los resultados estructurados correctos y el inventario final completo, sin presentar ese código como éxito.
- Casos cubiertos: freeze largo, límites y post-freeze, captura tardía/ausente, desconexión/cambio de lado, entidades reutilizadas, gasto cero/refund, ausencia de reglas, protocolo o ventana incompatibles, alteraciones rehasheadas y rechazo de destinos inseguros antes de cargar entradas.
- Inventario protegido idéntico: 4.356 archivos / 42.372.398.910 bytes y 181 hashes de control. Se compararon rutas opacas, tamaños y fechas; no se rehashearon todos los cuerpos de los 44 bundles. Las 17 evidencias JSON anteriores se conservan sin cambios.
- Documentación: 242 enlaces locales de 51 documentos sin destinos ausentes, un único prompt activo y `git diff --check` correcto. De los 913 archivos previos inventariados, 901 mantienen su hash; los otros 12 son contratos/tests/documentos del alcance o el prompt retirado. Las fuentes del build no cambiaron después del smoke.

Demo `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, Mirage, 330.945.844 bytes. Build `working-tree-economy-3c-20260906-r1`, binario SHA-256 `74bf00db3e2f4bbe0e9d31804d30b767c6ebbe3c6f2d552d06390de6937a6f81`. Comandos, bindings, fuentes, versiones y hashes de artefactos están en la evidencia.

## Pendiente operativo y continuación

Se retuvieron evidencia e inventario de **77 archivos / 738.029.806 bytes** antes del borrado. Se verificaron los 77 hashes, destino absoluto literal bajo TEMP y ausencia de reparse points o escapes. La revisión automática de aprobación rechazó:

```powershell
Remove-Item -LiteralPath 'C:\Users\kerch\AppData\Local\Temp\stratai-3c-smoke-_v1_3msj' -Recurse -Force -ErrorAction Stop
```

Motivo literal: **`blocked by policy`**. No se intentó otro método. El prompt de limpieza entonces vigente limita la siguiente acción a esa salida o a acreditar su ausencia externa; no repetir parser, materializador ni suites pasadas. La carpeta y evidencia de 3B, así como la deuda espacial 1B.12, permanecen separadas y no se han modificado.

3D deberá acreditar reglas compatibles y compras/drops/refunds itemizados antes de generar cestas alternativas. La cobertura inicial también sigue pendiente. No hubo descarga, migración 44/44, promoción, modificación de release/CURRENT, entrenamiento, recomendaciones, perfil ni misiones runtime. `comparison_allowed=false`, `training_allowed=false`, `product_supported=false` y cero capacidades elegibles continúan vigentes.

## Cierre operativo verificado — 6 de septiembre de 2026

Tras comunicar el usuario que había terminado la limpieza, se verificó a las 2026-09-06T08:16:41.129992+00:00 que los destinos temporales de esta fase no existen ni figuran en su directorio padre accesible. La eliminación se realizó fuera de las acciones del agente; no se atribuye hora exacta. Se conservan inventarios, hashes y rechazos históricos. No se repitieron productores, demo ni suites para este cierre. El plan activo mantiene el siguiente trabajo funcional.
