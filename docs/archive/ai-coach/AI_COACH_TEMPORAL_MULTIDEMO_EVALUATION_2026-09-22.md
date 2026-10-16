# Primera evaluación técnica en varias demos — 22 de septiembre de 2026

Cinco partidas aprobadas completan extracción, generación y validación del registro descriptivo. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y orden. [Evidencia completa](../../evidence/AI_COACH_TEMPORAL_MULTIDEMO_EVALUATION_2026-09-22.json); [protocolo](../../ai-coach/AI_COACH_TECHNICAL_EVALUATION_PROTOCOL.md).

## Rendimiento medido

| Partida opaca | Registros | Parser, s | Generación, s | Lector, s | Auditor, s |
|---|---:|---:|---:|---:|---:|
| `demo-a11eb0d600ee702c9edb` | 61432 | 75.3 | 64.7 | 66.5 | 18.6 |
| `demo-5bdb2dac2277dd9463ac` | 34926 | 44.9 | 38.0 | 38.8 | 11.8 |
| `demo-381bbf14c9ccd5693047` | 46716 | 60.3 | 49.5 | 51.3 | 15.9 |
| `demo-310597a1cb8f27f26c55` | 49282 | 63.5 | 53.4 | 54.0 | 16.4 |
| `demo-6e4a3660d53931747fa8` | 79650 | 101.9 | 84.1 | 85.5 | 26.2 |

La primera partida es el smoke ya cerrado de la [optimización](AI_COACH_TEMPORAL_PERFORMANCE_2026-09-22.md). Las otras cuatro se ejecutan después de ese cierre, con máximo dos pipelines simultáneos. No sumar tiempos paralelos para calcular duración del lote ni atribuir variación del parser al cambio Python. Lector y auditor son comprobaciones de desarrollo separadas del tiempo de generación. La misma captura de referencia pasó de 1237,647 s a 65,509 s con JSONL idéntico; la versión final conserva esa equivalencia mediante comparación de ambos smokes de la misma partida.

## Corrección y cobertura observada

- 272006 registros de ambos lados; 1354505 estados actor/corte.
- 887551 posiciones y 886310 desplazamientos contrastados independientemente.
- 5245131 hechos físicos contrastados; 65691 tenencias C4 positivas y 821860 negativas corroboradas.
- 1000 registros espaciados de las cuatro partidas adicionales contrastados además con el validador Python original. La partida de referencia conserva equivalencia completa con la salida anterior.
- 1088024 hipótesis explícitamente indisponibles. Son cuatro estados contractuales por registro; no se interpretan como precisión, fallos tácticos ni ejemplos supervisados.

Los cinco lectores recomponen todos sus registros. La auditoría recalcula desde fuentes originales, sin llamar al productor de diferencias, y comprueba procedencia, identidades, recursos y C4. La muestra se congeló por tamaño antes de ejecutar los nuevos outputs; no se escogieron partidas para obtener resultados positivos. No demuestra cobertura de todas las builds, niveles, mapas o variantes posibles.

## Datos y recorrido pendiente

Las cinco partidas quedan identificadas como desarrollo conocido. Las otras 39 son una reserva de selección, no un test independiente ya acreditado. Antes de ajustar cualquier modelo se debe cerrar exposición previa, separación temporal/por partida, cruces de jugadores, etiquetado y evaluación independiente conforme al plan. No hay aprendizaje automático al procesar estas demos.

El primer alcance táctico conserva ICT-I1.B y E01–E04. El protocolo concreta señal, receptor, tiempo, vínculo espacial, alternativas y abstención; faltan fuentes positivas para completar esas afirmaciones. No se ha convertido un evento global en percepción ni solicitado de nuevo criterios ya recibidos. No se ha implementado ni entrenado un clasificador táctico.

## Retención y conservación

Los 44 manifests permanecen iguales a la fotografía previa y los SHA-256 de las 44 demos originales coinciden con sus checksums aprobados. El lote adicional retiene 308 archivos / 7055023388 bytes, con inventario completo. Benchmark y ambos smokes permanecen inventariados por el cierre previo. Sin limpieza, descarga de demos, promoción, migración 44/44, cambios de CURRENT o entrenamiento. El único paquete instalado para esta optimización es la dependencia Python fijada en requirements.
