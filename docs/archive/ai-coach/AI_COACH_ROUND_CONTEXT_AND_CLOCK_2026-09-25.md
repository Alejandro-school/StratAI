# Contexto de ronda y reloj nativo de las fichas — 25/09/2026

Implementados el observador de configuración pública, el filtro conservador de elegibilidad y las fichas v2 con reloj nativo. E01 sigue excluida por el criterio experto IC27. **No se acredita identificación automática positiva de cuchillos en esta replay**: faltan seis convars requeridas y no hay prohibición de compra. El clasificador devuelve desconocido; esa abstención no cuenta como acierto táctico.

La petición «Hazlo» autoriza la ejecución del criterio ya recibido. [Contrato](../../contracts/context/canonical-round-context-v1.md), [evidencia](../../evidence/AI_COACH_ROUND_CONTEXT_AND_CLOCK_2026-09-25.json), [Excel v18](../../../outputs/stratai-round-context-20260925/StratAI_Plan_del_coach_TP10_v18.xlsx).

## Implementación y alcance

- El opt-in de relojes añade un sidecar de nueve convars y tres propiedades nativas, con presencia/nulidad, orden, sesión y fronteras causales. No rellena defaults. Normaliza únicamente las dos representaciones observables de booleano desactivado, `0` y `false`.
- Una clasificación positiva de cuchillos exige configuración completa corroborada por inventarios propios/aliados completos. Desconocidos y contradicciones bloquean el paso al detector. C4/plantado público en el prefijo permite identificar objetivo C4; el antecedente no se pierde por un hueco ni cruza ronda/sesión. No equivale a certificar ranked/competitivo.
- `review_packet_v2` separa tiempo de tramo, transcurrido desde freeze y plazo nativo de ronda/bomba. Reutiliza alineación y pausas verificadas. `analyze` sólo interpreta tramos elegibles; los demás conservan interpretación nula.
- Las plantillas v1, el detector temporal y las predicciones congeladas no cambian. El nuevo lector recalcula JSON/HTML y hashes. Registro runtime: **114 schemas/19 razones**, catálogo 1.7.0.

## Verificación

`check:all` pasó. La suite completa pasó **2.014 pruebas** antes de dos ajustes finales: conservar la prueba de C4 durante contradicciones y aceptar `false` nativo. La repetición final de todo el alcance afectado pasó **141 pruebas**, incluidas 22 del módulo nuevo; contrato y lint específico también pasaron. No se presenta aquella suite completa como una segunda ejecución posterior a los ajustes.

El smoke utilizó exactamente una partida aprobada, `demo-0beb9110ec408f80c39b`, SHA-256 `0beb9110ec408f80c39ba19f90729773705108e21d88d0df20b60267a82f91c7`. Tras detectar la representación booleana real se repitió **la misma demo** después de las pruebas, en otro TEMP vacío. Son dos pasadas de una partida, sin selección alternativa. Parser v16, canonical 3.8.0, quality 12, build `working-tree-round-context-20260925`; comandos, duración, sello de fuentes y binario constan en la evidencia. El overlay sólo desactiva el borrado automático del staging aislado.

Se validaron schema, semántica y bundle canónico. La fuente contiene **23.777 cortes** nativos de contexto y reloj, cuatro mensajes con seis asignaciones a tres convars. La [auditoría independiente](../../../outputs/stratai-round-context-20260925/factual-audit.json) recalcula **1.170 transcurridos, 1.158 cuentas atrás de ronda y 12 de bomba**. Los prefijos de E01/E02 conservan exactamente posiciones, identidades, recursos, inventario, C4 y referencias anteriores, descontando sólo IDs derivados de versión. E01 produce 65 estados desconocidos; E02 produce 1.105 estados con objetivo C4. La rama positiva de cuchillos sólo tiene verificación sintética; no se amplía la regla para encajar E01.

E01: reloj nativo 2:00 → 1:56. E02: inicia en 1:55 de ronda y, al corte final de 1:09 transcurridos, la ronda ya está plantada y la bomba conserva 39,265625 s; no se reutiliza el reloj de ronda como reloj de bomba. La ficha se revisó a 1440×1050 y 390×844, sin desbordamiento horizontal, con Home, reproducción y pausa. Los bytes del HTML revisado coinciden con la salida final.

El libro conserva 14 hojas, todas las respuestas literales, valores, fórmulas, estilos y controles; sólo cambia F32/G32 de IC27. Los 44 manifests y el SHA de la demo original permanecen intactos. Ambos TEMP quedan retenidos con [inventario](../../../outputs/stratai-round-context-20260925/retained-inventory.json). No hubo limpieza, descarga, promoción, CURRENT, migración 44/44 ni entrenamiento.

## Evaluación y siguiente acción

La [ficha E02 v2](../../../outputs/stratai-round-context-20260925/review/E02.html) está preparada sin predicción ni desenlace posterior al corte. El [recibo de entrega](../../../outputs/stratai-round-context-20260925/E02-delivery.json) enlaza la nueva vista con los IDs originales; cualquier respuesta se vincula al episodio/contexto congelados. El operador vio el prefijo aliado durante QA. La exposición privada del revisor sigue desconocida.

El instrumento cambia después de E01: queda declarado en TP10 v0.5, sin recalcular predicciones primarias ni declarar un nuevo test independiente. Se mantienen **36 casos históricos, E01 excluida, 35 pendientes y cero lecturas activas puntuadas**. Siguiente acción: recibir la lectura de E02 y contrastarla con su predicción congelada. La identificación nativa positiva de cuchillos queda como limitación de cobertura que exige evidencia real completa; no bloquea revisar E02, cuyo objetivo C4 sí está observado. Consejos, perfil, práctica y seguimiento conservan sus dependencias.
