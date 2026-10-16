# Contexto económico nativo — 3D.2B

Fecha: 6 de septiembre de 2026. Selector, lector e informe implementados y probados. La compatibilidad económica completa continúa pendiente. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) selecciona la siguiente acción; la [evidencia](../../evidence/AI_COACH_ECONOMY_NATIVE_RULE_CONTEXT_SINGLE_DEMO_SMOKE_2026-09-06.json) conserva comandos, conteos, hashes, validación y cierre operativo.

## Entrega y avance

El nuevo selector económico 4 conserva las salidas v2 y fuentes nativas @1, y añade el [contexto económico por mensaje y frontera](../../contracts/economy/canonical-economy-rule-context-v1.md). Go captura directamente asignaciones de 56 nombres permitidos, su orden, presencia y tick. Python verifica las fronteras contra v2 y reconstruye el prefijo disponible. Registra también cuatro propiedades de `GameRules` y presencia/tamaño/hash de datos opacos de sesión. Tres schemas llevan el registro runtime a 49; el catálogo `1.7.0` y los contratos anteriores se conservan.

El SDK `ConVarsUpdated` consolida en un mapa las asignaciones de cada mensaje y utiliza getters que pierden presencia. Leer `CNETMsg_SetConVar` directamente permite conservar repeticiones, ausencia, vacío, cero y booleanos. No se completa una variable con su default, ni se atribuye al jugador conocimiento de parámetros del servidor.

## Resultado en una partida

Demo de desarrollo aprobada `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Una pasada del productor cambiado en TEMP nuevo y vacío, después de los tests.

| Comprobación | Resultado |
|---|---|
| Mensajes nativos / asignaciones económicas retenidas | 19 / 29 |
| Fronteras de inicio y fin de freeze | 50, en 25 instancias |
| Variables con valor observado | 22 de las 56 consultadas; antes se capturaban cinco valores |
| Valores iniciales / de fin de freeze | 550 / 550 |
| Variables no vistas en el prefijo consultado | 34; 1.700 comprobaciones entre ambas fronteras |
| Propiedades nativas observadas | 200 de 200 comprobaciones |
| Contador CT de pérdidas | Valores crudos entre 0 y 5 |
| Contador T de pérdidas | Valores crudos entre 0 y 3 |
| Restricciones nativas CT/T consultadas | `false` en las 50 fronteras |
| Paquetes económicos aprobados | 0 |

Entre los datos nuevos están límites de granadas, parámetros de premios, armadura gratuita, premios individuales y pérdidas iniciales. Se conservan cambios de configuración reales: por ejemplo, límite de flashbang `0`/`2`, dinero inicial `0`/`800` y premios individuales `false`/`true`. Estos valores incluyen la instancia preliminar no enlazada; no se presentan todos como rondas competitivas. El informe mantiene la identidad de instancia y su origen v2.

La agrupación de armadura/casco observa cinco combinaciones: objeto 50 a coste 650 con armadura previa 0, 82 o 91; objeto 51 a coste 350 con armadura previa 100 y sin casco, en 101 ocurrencias; y objeto 51 a coste 1.000 con armadura previa 0 y sin casco, en 534 ocurrencias. Se cuentan entradas repetidas en capturas, no compras distintas. La asociación observada no acredita una fórmula universal ni errores del jugador.

`game_session_manifest` está presente con **7.160 bytes**, hash `6f2854ab5b7551145c8fec8e8f9ae1db942e8d1f96c91595c4c3a90e8d1539b2`. `game_session_config.data` contiene **570 bytes**, hash `b5992036fd330fc6de0c6c23c7a29ff5be13847cd5c3cd4b13a55cf675e5b57d`. Se observaron en tick 0. Sólo se retuvieron sus fingerprints; el contenido no fue interpretado ni publicado. Son una fuente concreta para investigar a continuación, sin afirmar que contengan las reglas o identidad buscadas.

## Verificación y límites

Pasaron `npm run check:all`, 453 pruebas del AI Coach —37 nuevas—, Ruff y contrato global. Las pruebas cubren presencia, asignaciones repetidas, cambios posteriores en el mismo tick, integridad, tipos, datos opacos, falsificación de informes, selección del productor y rechazo de destinos inseguros antes del parser. Parser real: 103,497 s; materializador del contexto: 8,154 s. Se recalcularon 2.800 contrastes de configuración y 200 correspondencias de propiedades; el recuento de armadura es independiente del constructor del informe. La validación semántica canónica completa y su duración quedan registradas en la evidencia de cierre.

Se mantienen 77/250 actores inicialmente acreditados, 173 ambiguos, tres equipos competitivos completos, 50 fichas de oportunidad, 60 diferencias numéricas y 15 transiciones. El nuevo registro no rellena retrospectivamente el estado inicial ni amplía elegibilidad. Se verificó el inventario protegido de 4.356 archivos y los 181 hashes de control; catálogo y 20 evidencias previas permanecen intactos.

Los contadores nativos de pérdidas aportan estado, pero no acreditan el cálculo ni importe del bonus. `mp_maxmoney`, el premio base por baja y otras variables no aparecen en el canal consultado; no se concluye que falten en toda la demo. La identidad exacta de assets, precios condicionados, recompensas, límites completos, loadout y transacciones continúan pendientes. Todas las puertas de recomendaciones, entrenamiento y producto permanecen cerradas.

La validación semántica canónica completa pasó con **cero errores en 354,686 s**. La continuación 3D.2C debe interpretar los bloques de sesión con fuentes verificables. Si no contienen los campos buscados, debe documentar el resultado y seleccionar una dependencia concreta. La limpieza de esta salida se registra en la evidencia después de retener inventario y hashes. No se reintentan ni eluden los borrados rechazados de fases anteriores.

La revisión automática rechazó la retirada de `C:/Users/kerch/AppData/Local/Temp/stratai-3d2b-smoke-cpq92n1q` con `Remove-Item -LiteralPath … -Recurse -Force`: `blocked by policy`, antes de crear el proceso. Los 85 archivos / 745.825.698 bytes se inventariaron y sus hashes se verificaron antes del intento. Se comprobó después que el directorio sigue presente. La limpieza permanece pendiente; no hubo reintento ni alternativa para eludir el rechazo.

El cierre documental comprobó 291 enlaces locales de 49 documentos sin destinos ausentes, un único prompt vigente y `git diff --check` correcto. Las 14 modificaciones de archivos previos pertenecen al alcance registrado; los otros 936 conservan su hash. Las fuentes funcionales siguen iguales a las de la build validada.
