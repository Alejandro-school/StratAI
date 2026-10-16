# Actualización temporal del intérprete contextual

Fecha: 24 de septiembre de 2026. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md)
conserva el estado de cierre. [Criterio recibido](../../evidence/AI_COACH_TEMPORAL_PERSISTENCE_CRITERION_2026-09-24.json)
y [contrato](../../contracts/context/canonical-contextual-interpreter-v1.md#actualización-temporal-v2).

## Resultado y alcance

El usuario responde «Mantener posibilidad hasta contradicción». IC24/Excel v14
conserva el literal, límites y procedencia. Sólo cambia A29:G29 de Contexto
colectivo: las 14 hojas, anotaciones, valores, fórmulas, estilos y controles
anteriores se preservan y se verifica visualmente la fila nueva.

`collective_c4_approach@2-temporal-provisional` mantiene una posibilidad respaldada
durante las pausas, incluso de todo el grupo. Sigue comprobando posiciones,
identidad, C4 y continuidad. Una imagen estática sin soporte anterior no crea
una preparación. El recurso o la función de un jugador se recalcula en el corte
actual; la muerte excluye al fallecido y revisa el soporte de los supervivientes.

Una contradicción revisa o retira la lectura. Se conservan los hechos de aliados
no afectados y una nueva lectura tras revisión exige movimiento fresco del
portador. La versión anterior y sus predicciones permanecen disponibles.
El materializador offline usa v2 por defecto; `--configuration` permite elegir
v1 y la validación recompone la versión registrada en cada manifiesto.

El nuevo schema `stratai.contextual_interpretation@2` registra origen, soporte
actual, eventos y estado causal. El registro pasa a **112 schemas/19 razones**,
con catálogo `1.7.0` conservado. No cambia la interfaz temporal, no añade un
endpoint ni habilita consejos, perfil, práctica o entrenamiento.

## Contraste real de desarrollo

Se reproducen exactamente las **84 predicciones anteriores** en siete partidas
retenidas. La nueva versión produce **19 lecturas compatibles y 65 abstenciones**:

| V1 → V2 | Episodios |
|---|---:|
| Abstención → abstención | 63 |
| Abstención → compatible | 9 |
| Compatible → compatible | 10 |
| Compatible → abstención | 2 |

Estos cambios requieren revisión: una emisión nueva no es automáticamente un
acierto y una abstención nueva no es automáticamente un error. Son partidas ya
expuestas, sin precisión independiente ni nuevas etiquetas humanas.

R1 sigue absteniéndose: no tiene un portador actual único acreditado. Se conserva
la lectura humana de estrategia indeterminada y B posible, sin convertirla en
una regla universal para dos supervivientes.

R2 emite **preparación posible hacia A**. Registra la retirada de un aliado en la
muestra 6354 y reconoce soporte renovado en 6355. La posibilidad se mantiene en
los últimos cortes, incluidos 6378–6382, pese al pequeño reajuste del portador.
La muerte no borra las trayectorias de los vivos ni acredita lo que el fallecido
vio o transmitió. El motivo de elegir A sigue desconocido.

Se contrastan **65 prefijos**: recomputar el prefijo completo y actualizar desde
la lectura anterior produce los mismos destinos. La versión de diagnóstico
intermedia, conservada como `real-regression.json`, no es la salida vigente;
la comparación final está en `real-regression-final.json`.

## Parámetro y límites

El filtro técnico inicial es 1 segundo de movimiento contrario acumulado sin
avance intermedio. Las muestras quietas no suman tiempo ni borran lo observado.
Deshacer toda la aproximación del portador retira la lectura inmediatamente.
Este valor fue declarado antes de ejecutar R2 con v2; no procede de una respuesta
experta ni define cuánto dura una retirada en Counter-Strike.

En la comprobación de sensibilidad, R2 se abstiene con 0,5 s y propone A con 1 y
2 s. **La calibración sigue siendo provisional.** El resultado no demuestra que
el filtro generalice ni que todas las retiradas o dispersiones estén cubiertas.
Los valores alternativos son sondas documentadas, no versiones publicadas ni
ajustes elegidos automáticamente.

TP01–TP09 tienen controles de software y contraste real donde existen variantes
positivas. TP10, evaluación independiente, permanece pendiente. Las pausas largas
de todo el grupo, pérdida de C4/identidad, retirada sostenida del portador y cambio
de destino se verifican mediante controles construidos; no se presentan como
hechos observados en el smoke ni como etiquetas expertas.

## Verificación

- `npm run check:all`: correcto, 113,742 s.
- Suite completa del coach: 1.994 pruebas correctas y un fallo de recuento que
  aún esperaba 111 schemas. Se actualiza esa aserción a 112; la repetición de las
  **320 pruebas afectadas** pasa en 43,321 s. Se conserva el log fallido de la
  ejecución completa y no se presenta como una ejecución completa limpia.
- 26 controles nuevos cubren pausa, retirada, dispersión, cambio de destino,
  muerte, continuidad, procedencia, recursos actuales, causalidad y validación
  offline de ambas configuraciones. Ruff correcto.
- Registro contractual: 112 schemas y 19 razones, sin cambios del catálogo.
- Smoke de una demo aprobada: parser 79 s, materialización 25,375 s, lector
  24,105 s y validación canónica 266,166 s, sin errores. Auditoría independiente
  2,048 s: 942 referencias causales, ocho señales de aproximación, 200 pasos de
  movimiento y ocho contadores recalculados. Dos lecturas compatibles A y dos
  abstenciones prescriptivas. [Evidencia de cierre](../../evidence/AI_COACH_TEMPORAL_PERSISTENCE_2026-09-24.json).

Las 65 muestras factuales finales coinciden con la captura anterior, salvo el
ID del registro ligado a la nueva build. La salida contiene 83 archivos y
1.113.810.560 bytes retenidos. Los 77 archivos protegidos y las fuentes
funcionales selladas antes del smoke conservan sus hashes.

La prueba real ejecuta parser y productor contextual hacia un TEMP nuevo, después
de las pruebas. Rechaza destinos fuera de TEMP, no vacíos o sobrescrituras antes
del parsing. La fuente y los 44 bundles aprobados se conservan como sólo lectura.
Se retienen todos los artefactos temporales, incluido staging, con inventario y
hashes; no hay limpieza ni promoción.

## Siguiente dependencia

Preparar la evaluación independiente con separación por partida, participantes,
fecha y exposición; congelar versión/selección y obtener revisiones ciegas sin
mostrar la predicción. R1/R2 y las siete partidas usadas aquí siguen siendo
desarrollo. Mapas/build, información rival legal, alternativas viables y comparación
respaldada mantienen sus dependencias para consejo, perfil, práctica y seguimiento.
