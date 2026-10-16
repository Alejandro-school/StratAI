# Lote TP10: preparación técnica de la revisión

Fecha: 24 de septiembre de 2026. La preparación técnica de tres partidas y 36
fichas está terminada. **Hay cero lecturas humanas nuevas; la evaluación táctica
sigue pendiente.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md)
mantiene el estado y la siguiente acción. El [recibo](../../evidence/AI_COACH_TP10_SCOPE_ACCEPTANCE_2026-09-24.json)
conserva «Hazlo» y la interpretación contextual del agente: evaluar primero
otros jugadores del mismo periodo. IC19 e IC24 no cambian.

La selección congelada usa tres demos aprobadas del 24/08/2026 con 30
participantes distintos entre sí y de las siete partidas de desarrollo. Sus
checksums completos y fechas están en la [evidencia](../../evidence/AI_COACH_TP10_PILOT_2026-09-24.json).
Las tres partidas y las cuatro reservas que comparten jugadores con ellas quedan
fuera de nuevos ajustes. No hay separación temporal respecto a todo desarrollo;
la exposición privada y la independencia entre organizaciones no se acreditan.

Se seleccionaron por reloj 12 cortes por partida antes de ejecutar el detector,
sin consultar posiciones, desenlaces o salidas automáticas. Cada corte se amplió
al prefijo continuo disponible de su ronda: **26.195 muestras en 36 rondas**,
máximo 1.820 por ficha frente al límite de 4.097. No hubo truncado, sustitución
ni selección posterior por resultado. No hay dos lecturas humanas sucesivas de
una misma ronda en esta muestra; verificar evolución interna no cubre ese contraste.

Las 36 predicciones principales de `collective_c4_approach@2-temporal-provisional`
se sellaron con filtro de 1 s antes de las sondas secundarias de 0,5/2 s y de
cualquier respuesta nueva. Estas sondas no calibran el resultado principal.
Predicciones y recuentos interpretativos se retienen separados de las fichas;
los 36 casos permanecen sin puntuar y la exactitud táctica es desconocida.

## Verificación realizada

Los tres validadores canónicos y los lectores de las 36 interpretaciones y
fichas pasan. La auditoría independiente de las fuentes recalcula 328.109
referencias causales, 130.975 estados de vida, 98.567 posiciones, 148.263
comprobaciones de posesión/cantidad de granadas y 16.831 tenencias positivas de
C4. Comprueba 26.195 estados del objetivo, 147 eventos temporales, 16 señales de
aproximación y 21 contadores sobre 1.358 pasos de movimiento. Son controles
factuales y de ejecución, no valoraciones de estrategia.

| Partida | Parsing (s) | Interpretación (s) | Fichas (s) | Lector interpretación (s) | Lector fichas (s) | Canónico (s) |
|---|---:|---:|---:|---:|---:|---:|
| `demo-0beb9110ec408f80c39b` | 83.201 | 188.429 | 289.632 | 337.513 | 308.691 | 319.838 |
| `demo-1028fce95708dda309ae` | 105.691 | 224.116 | 259.671 | 253.024 | 257.421 | 527.144 |
| `demo-5224a45934f2d5b94c01` | 138.888 | 255.698 | 321.2 | 358.804 | 330.44 | 646.211 |

La auditoría factual duró 1006.693 s. Comandos exactos,
cwd, versiones, salidas y hashes se conservan en la evidencia. Se reutiliza el
binario `working-tree-contextual-temporal-persistence-20260924` (SHA `eea54eb43375b50d4e1485241b129eace342f0776820784a87741704625f9ce3`), parser v16,
contrato canónico 3.8.0, 112 schemas y 19 razones. No se modificó código runtime,
productores, contratos ni parámetros: este lote de evaluación autorizado usa
el productor del [cierre funcional anterior](AI_COACH_TEMPORAL_PERSISTENCE_2026-09-24.md),
no es un nuevo smoke de tres demos. No se declara una nueva ejecución de toda
la suite; la comprobación presente se concentra en los nuevos artefactos reales.

Extracción y materialización secuenciales; después se solaparon hasta tres
verificaciones de sólo lectura, con memoria disponible comprobada y enmiendas
retenidas. No se sobrescribió ningún artefacto congelado.

E01 se revisó visualmente en escritorio (1.440 × 1.050) y móvil (390 × 844):
reproducción, control temporal, mapa, estados y recursos. Sin desbordamiento
horizontal ni errores JavaScript de la aplicación; el servidor de vista previa
devolvió un 404 de favicon. Los aliases próximos pueden superponerse en el mapa
y los nombres son pequeños en móvil; el listado por color sigue disponible.
Esta inspección visual cubre E01; lectores y auditoría factual cubren las 36.
El renderer congelado permanece intacto.

El operador inspeccionó un fragmento de progreso de la primera demo con metadatos
de final de rondas 18–20 y la vista factual de E01. No cambió la selección ni
produjo etiquetas. La exposición del revisor a esos logs, al futuro o a salidas
del sistema se debe registrar; no se presupone ceguera a partir del formato.

## Conservación y entrega

Se verifican por hash 161 archivos sellados, el binario, los 44 manifiestos,
las tres demos originales y nueve artefactos principales de predicción/contexto.
Se retienen 330 archivos TEMP, 4,894,102,031 bytes,
con [inventario](../../../outputs/stratai-tp10-pilot-20260924/retained-inventory.json).
Los originales y los outputs anteriores permanecen de sólo lectura; sin
adquisición, promoción, modificación de CURRENT, migración 44/44 ni limpieza.

El [Excel v15](../../../outputs/stratai-tp10-pilot-20260924/StratAI_Plan_del_coach_TP10_v15.xlsx)
incorpora IC25 sólo en A30:G30 de Contexto colectivo. Comparación estructural y
visual verifican las 14 hojas, valores, fórmulas, estilos y controles previos;
v14 se conserva. El acuerdo de alcance no es una etiqueta táctica.

Se entrega [E01](../../../outputs/stratai-tp10-pilot-20260924/review/E01.html) con
[instrucciones para responder por chat](../../../outputs/stratai-tp10-pilot-20260924/review/Instrucciones.md).
Su hash coincide con la ficha congelada. El siguiente paso es conservar exposición
y lectura literal, normalizar y contrastar por separado, y continuar con E02.
Los nombres de zonas mantienen el límite de asset/build no acreditado. No se
atribuyen conocimientos rivales, no se han generado recomendaciones y no hay
entrenamiento automático. Perfil, práctica y seguimiento conservan sus dependencias.
