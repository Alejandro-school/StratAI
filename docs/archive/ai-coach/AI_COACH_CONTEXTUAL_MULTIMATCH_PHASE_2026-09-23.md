# Evaluación técnica contextual en siete partidas — 23 de septiembre de 2026

Se ejecuta después del [cierre de una demo](AI_COACH_CONTEXTUAL_INTERPRETER_PHASE_2026-09-23.md)
con el detector congelado. [Evidencia](../../evidence/AI_COACH_CONTEXTUAL_MULTIMATCH_EVALUATION_2026-09-23.json).

| Partida opaca | Episodios | Emisiones | Abstenciones | Carga, generación y escritura, s |
|---|---:|---:|---:|---:|
| `demo-a11eb0d600ee702c9edb` | 12 | 2 | 10 | 39.273 |
| `demo-5bdb2dac2277dd9463ac` | 12 | 2 | 10 | 26.825 |
| `demo-381bbf14c9ccd5693047` | 12 | 1 | 11 | 31.651 |
| `demo-310597a1cb8f27f26c55` | 12 | 1 | 11 | 33.036 |
| `demo-6e4a3660d53931747fa8` | 12 | 1 | 11 | 48.687 |
| `demo-045df2521c790a7d8a69` | 12 | 2 | 10 | 32.457 |
| `demo-066944f5acb806ba5f81` | 12 | 3 | 9 | 29.089 |

Son 84 episodios de siete partidas: cinco capturas de desarrollo
ya validadas y dos partidas distintas de la reserva. Se eligieron las dos por el
menor SHA-256 de la reserva antes de producir sus salidas. Por partida se fijaron
12 ventanas T no solapadas de 65 muestras, uniformes entre ventanas jugando, antes
de inferir. Los cortes no se eligieron por acierto, incertidumbre ni respuesta.
No hay intervalos correlacionados tratados como partidas distintas.

**No se ha medido precisión táctica:** 84 casos sin revisar,
aciertos y errores humanos pendientes, `agreement=null`. Las abstenciones no son
aciertos automáticos. Las dos partidas de reserva son nuevas para desarrollar
esta regla, pero la exposición histórica, el cruce de participantes/fechas y la
revisión ciega siguen sin acreditar una evaluación independiente. Las cinco de
desarrollo no se presentan como test. No se ajustó la regla tras ver este lote.

Cada paquete conserva su entrada causal, predicción separada, plantilla vacía
de revisión, conteos agrupados por partida, fuentes y hashes. Las fichas neutrales
[R1](../../../outputs/stratai-contextual-20260923/R1.md) y
[R2](../../../outputs/stratai-contextual-20260923/R2.md) presentan la cuarta ventana
preseleccionada de cada partida de reserva, sin enseñar la predicción. El mapa
local es orientación de desarrollo; no certifica zonas ni build.

El auditor factual independiente recompone 19669
posiciones observadas, 66487 referencias y
39 señales emitidas desde las fuentes
nativas. Conserva 7480 posiciones no disponibles.
Este recálculo verifica hechos y geometría de las señales, no su corrección táctica.

Dos `.dem` locales aprobadas se procesaron con el mismo binario, checksum y
directorio TEMP vacío exclusivo. Las cinco capturas anteriores se leyeron sin
reescribirlas. Se conserva un máximo de dos procesos de partida simultáneos,
sin adquisición, promoción, corpus-wide reprocesado, reemplazo de CURRENT ni
entrenamiento. Los 44 manifests de corpus conservan sus hashes.

Retención del lote: 195 archivos / 2288309270 bytes,
con inventario SHA-256. Las recomendaciones, prioridades y misiones siguen
abstendidas por sus requisitos factuales y de evaluación, según el
[contrato](../../contracts/context/canonical-contextual-interpreter-v1.md).
