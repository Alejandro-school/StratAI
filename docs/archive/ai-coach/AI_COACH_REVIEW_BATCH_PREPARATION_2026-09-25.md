# Lote de revisión TP10: E03–E36 — 25 de septiembre de 2026

Quedan preparados y verificados los **34 casos pendientes**, agrupados en 10,
12 y 12 de las tres partidas congeladas. El usuario pide recibirlos juntos;
se sustituye la entrega de una ficha por turno. E01 permanece excluida por
criterio experto y E02 conserva su lectura. No se han recibido respuestas nuevas.

- [Índice del lote](../../../outputs/stratai-review-batch-20260925/paquete/index.html).
- [Paquete portable](../../../outputs/stratai-review-batch-20260925/StratAI_34_casos_TP10.zip).
- [Hoja de respuestas](../../../outputs/stratai-review-batch-20260925/paquete/Respuestas_TP10_34_casos.xlsx).
- [Evidencia detallada](../../evidence/AI_COACH_REVIEW_BATCH_PREPARATION_2026-09-25.json).

## Alcance e instrumento

Se conservan los mismos cortes, participantes propios/aliados, recursos y C4
de las 34 fichas originales. Las vistas v3 incorporan los relojes ya verificados
y el recuento agregado de vivos de ambos equipos. No se muestran posiciones
ocultas rivales, predicciones ni desenlaces posteriores al corte. Cada vista
nueva enlaza con los IDs y hashes congelados. Los originales permanecen intactos.
Las copias portables añaden E03–E36 y partida al título y cabecera para identificar
cada pestaña; sus datos y scripts incrustados son idénticos a la ficha validada.

Se reutiliza una captura completa y se recapturan las otras dos demos aprobadas
con el mismo binario. No cambia el runtime ni se abre un cierre funcional con
nuevo smoke: es ejecución del lote previamente seleccionado. No se adquieren
partidas, se promueve salida ni se modifica el corpus de 44 bundles.

## Verificación

| Etapa | Segundos | Salida |
|---|---:|---:|
| materialize-1 | 404.808 | 0 |
| validate-1 | 516.843 | 0 |
| producer-2 | 122.168 | 0 |
| materialize-2 | 472.094 | 0 |
| validate-2 | 464.786 | 0 |
| producer-3 | 138.176 | 0 |
| materialize-3 | 609.915 | 0 |
| validate-3 | 560.932 | 0 |

Los 34 materializados y validadores pasan. Auditoría independiente de
25,025 cortes: 49,457 recuentos
nativos recalculados y 49,457 contrastes
contra bajas públicas. Los estados y ausencias de cada reloj están en la
evidencia por caso; un reloj no aplicable no se convierte en dato observado.
Se recalculan además 593 recuentos parciales reales: desconocidos conservados,
sin inferirlos muertos ni atribuir un total de vivos exacto.
Los relojes observados se recalculan desde sus plazos y reloj de red nativos.
Se cotejan los hechos originales, sin abrir las predicciones congeladas.

La hoja contiene 34 filas y 136 celdas de respuesta vacías, sin fórmulas ni
respuestas sugeridas; conserva selector de exposición y cabecera fija.
Se revisaron visualmente la hoja, el índice en escritorio/móvil y una ficha de
cada partida. Se verifican filtros, enlaces, reproducción y entrega HTTP de
la hoja. El ZIP conserva los 38 archivos por hash y pasa comprobación CRC.

## Retención y siguiente paso

Se conservan 242 archivos temporales nuevos y
3,953,307,245 bytes, con
[inventario](../../../outputs/stratai-review-batch-20260925/retained-inventory.json).
La captura reutilizada conserva su inventario anterior. Fuentes, runtime,
fichas originales, predicciones y los 44 manifiestos mantienen sus hashes.

Recibir respuestas **agrupadas**, conservar literalmente cada lectura y su
exposición, y contrastar sólo los casos respondidos con la predicción congelada.
Las celdas sin respuesta siguen pendientes. El cambio de instrumento posterior
a E01/E02 está declarado en TP10 v0.7; estas lecturas no acreditan por sí solas
generalización ni entrenan automáticamente el detector. Las reglas nuevas
mantienen su revisión de fase y dependencias.
