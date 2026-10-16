# Fuentes económicas nativas y reconstrucción por objeto — 3D.1

Fecha: 6 de septiembre de 2026. Estado: implementación y validación completadas; limpieza pendiente por rechazo automático. **La fase 3D completa sigue pendiente.** Estado y siguiente acción en el [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md); cifras, comandos, versiones, dos intentos e inventarios en la [evidencia JSON](../../evidence/AI_COACH_ECONOMY_NATIVE_SOURCES_SINGLE_DEMO_SMOKE_2026-09-06.json).

## Entrega y alcance

Se añadió un observador optativo en Go y un lector/reconstructor en Python, con tres schemas propios. El modo económico `3` conserva v2 y añade dos artefactos de fuentes; el informe por objeto se materializa en otro destino TEMP vacío. Registro de 43 schemas; catálogo `1.7.0`, export `3.8.0`, parser `v16`, quality `12` y manifest base `@3` mantienen su significado. El [contrato de fuentes](../../contracts/economy/canonical-economy-native-sources-v1.md) describe presencia, disponibilidad, procedencia, versiones y comandos reproducibles.

Las listas se leen por nombre de propiedad presente en la demo. Los [nombres extraídos por cs2-dumper](https://github.com/a2x/cs2-dumper/blob/main/output/client_dll.hpp) orientan su localización; la versión actual y sus offsets no prueban compatibilidad histórica. La revisión de [demoinfocs v5.2.0](https://github.com/markus-wa/demoinfocs-golang/blob/v5.2.0/pkg/demoinfocs/datatables.go) confirmó que `ItemRefund` usa una deducción de entidad destruida, zona de compra y saldo creciente en el mismo tick. No se transforma ese callback en una devolución nativa con importe.

## Resultado real

Se utilizó exactamente una partida aprobada, alias `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Fue procesada dos veces por una corrección, siempre hacia destinos aislados nuevos. Se mantiene como desarrollo, no test final sellado.

| Medida | Resultado |
|---|---|
| Identidad nativa | Build 10896; patch y protocolo 14177 |
| Muestreo | 355.360 comprobaciones de actor por fotograma; 1.467 capturas retenidas |
| Listas de devolución | 1.465 completas y dos ausentes por desconexión |
| Contadores por ronda/partida | Ausentes en las propiedades consultadas de las 1.467 capturas |
| Evento nativo `item_purchase` | Cero observados |
| Intervalos de lista recalculados | 1.215 |
| Entradas añadidas / retiradas | 930 / 54 |
| Interior competitivo / límites o sin enlace | 896 / 571 registros |
| Acciones clasificadas / paquetes de reglas compatibles | 0 / 0 |

Las entradas conservan el coste registrado para ese objeto. Su suma o conteo no prueba un historial completo: cambios de metadatos o de elegibilidad también modifican la lista. **930 adiciones no son 930 compras completas acreditadas; 54 retiradas no son 54 reembolsos.** El valor especial `m_nPrevArmor=-1` se observó 2.779 veces y se conserva literalmente; no se interpreta como armadura física ni importe.

Se recalcularon además v2 y 3C sobre la misma salida: 77/250 estados precompra acreditados, 173 ambiguos, 23 enlaces competitivos, tres equipos completos, 50 fichas, 60 diferencias numéricas y 15 transiciones de inventario. Los datos posteriores no amplían el estado inicial.

## Verificación y correcciones

`npm run check:all` pasó: 88 pruebas frontend, 14 Node —tres omitidas—, 43 Python backend, paquetes Go y build frontend. Las 376 pruebas del AI Coach incluyen 29 nuevas de fuentes nativas. Ruff y registro global también pasaron.

El primer smoke produjo 27 capturas ligadas a la ronda anterior cuando ya empezaba la preparación de la siguiente. El validador rechazó la salida antes de reconstruirla. Se cerró el límite usando el fin de freeze observado, se añadió su prueba y se preservó el signo del metadato de armadura. Tras repetir las suites, la misma demo pasó en el segundo intento: parser 83,826 s; reconstrucción 5,482 s. El validador independiente recalculó los 1.215 intervalos desde las fuentes sin llamar al constructor. La revisión canónica completa dio cero errores en 353.503 s. Los inventarios y hashes de ambos intentos quedan retenidos.

El corpus permanece protegido: 44 bundles, 50 demos fuente, 4.356 archivos con metadatos iguales y 181 controles con hash idéntico. No se modificaron el catálogo ni los datos aprobados. De 18 evidencias previas, 15 conservan su SHA-256 y tres sólo registran el cierre de limpieza tras verificar ausencia de los cuatro temporales comunicados por el usuario. Los rechazos históricos permanecen.

## Pendiente

El cierre documental verificó 255 enlaces locales en 45 documentos, un único prompt activo, catálogo intacto y fuentes funcionales iguales a la build validada. Se conservaron sin cambios 905 de los 925 archivos previos inventariados; los 20 restantes están dentro del alcance registrado. `git diff --check` pasó.

Falta acreditar assets y reglas aplicables a la versión observada: precios generales, recompensas/bonus previo, loadout/límites y refunds. Tampoco está demostrado que las fuentes actuales recojan todas las operaciones intermedias o todos los drops. La aceptación de un compañero no se inventa. Todas las puertas de comparación, entrenamiento y producto permanecen cerradas. El plan selecciona la continuación; este informe no autoriza cestas ni recomendaciones.

Limpieza: limpieza pendiente por rechazo automático. Los destinos exactos y su estado están en `cleanup` de la evidencia. El directorio auxiliar pequeño contiene binario, scripts y logs privados; no es corpus ni evidencia publicada y no forma parte del borrado de salidas.
