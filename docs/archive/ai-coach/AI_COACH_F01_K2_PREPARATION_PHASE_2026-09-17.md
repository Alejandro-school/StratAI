# Preparación de F01-K2 v0.2

17 de septiembre de 2026. Preparación de un único corte posterior de la misma ronda y partida, tras la instrucción «Hazlo». La [ficha neutral](../../ai-coach/AI_COACH_F01_K2_REVIEW_v0.2.md) y el [minimapa comparativo](../../../outputs/stratai-f01-k2-review-20260917-75f20cf/F01-K2_v0.2_mapa.png) quedan preparados para recibir la interpretación del usuario. No se ha recibido una respuesta de K2 ni se ha reconstruido K3.

## Corte y selección

K1 se conserva: tick 8524, frame 8542, secuencia 1389. K2: tick **8636**, frame **8654**, secuencia **1417**, instancia **2**, época **0**, ronda **1** de `demo-a11eb0d600ee702c9edb`. Diferencia de **1,75 segundos**; quedan **101,25 segundos**, frente a 103 en K1. Continúan 0–0, 5v5 y los cinco T con 100 de vida.

Antes de inspeccionar el prefijo posterior se registró en TEMP una política de selección: nueva acción propia, cambio material de despliegue o información disponible, sin usar desenlace, muerte o confirmación del plan. Se recorrieron dos prefijos acotados (hasta 8588 y hasta 8652). No había cambio de inventario/arma ni evento propio nuevo en esos registros. Se conservaron las trayectorias y giros observados.

La decisión concreta fue presentar la **primera interrupción del avance horizontal de T1 corroborada por repetición de posiciones del replay**: x=−930/y=605 en 8628, 8632 y 8636, tras desacelerar. Dos intervalos entre esas muestras suman 0,125 segundos. El corte termina en la tercera muestra. Es un criterio de presentación de este episodio, no un umbral táctico, velocidad cero demostrada o espera intencionada.

Hay cambios de orientación sustanciales anteriores: T5 eleva la mirada y T4 gira su yaw mientras avanza. Se incluyen expresamente en la ficha. No se clasifican como jitter, no se declaran irrelevantes tácticamente y no se afirma que la interrupción sea el primer cambio significativo de cualquier clase. La política general estaba registrada antes; la elección operacional de confirmar dos intervalos se documentó al inspeccionar el prefijo. No se presenta como una regla táctica prerregistrada.

Se inspeccionaron muestras propias hasta 8652, dieciséis ticks posteriores al corte elegido. La ficha, el JSON del corte y el gráfico excluyen ese futuro; esta preparación no es ciega. `replay.events` está vacío hasta K2 y no hay nuevos registros `combat_shots` entre K1 y K2; sí existen nueve registros de disparos anteriores a K1, sin atribuirlos aquí a actores o conocimiento. No se afirma ausencia global de acciones. No se usó una muerte, resultado, lanzamiento futuro o site final para escoger el corte.

## Material presentado

- Mismos alias asignados en K1 y conciliados por identidad; no se renumeran por posición en K2.
- Comparación de los cinco jugadores, orientación y movimiento registrado. Trayectoria de K2 acotada a K1–K2, con fuentes anteriores conservadas.
- T1 repite posición horizontal; T2, T3 y T4 siguen avanzando. T4 conserva C4. T5 sigue en la misma posición horizontal con humo equipado y flash en inventario; su pitch pasa de 75,27° hacia abajo a −7,89°.
- Recursos y alturas por jugador, reloj e intervalo, sin posiciones CT ocultas, munición exacta o efectos de utilidad supuestos.
- Una sola pregunta: qué se mantiene/cambia del plan y las funciones respecto a K1 y qué señales nuevas lo justifican.

## Comprobaciones

La [evidencia de preparación](../../evidence/AI_COACH_F01_K2_REVIEW_PREPARATION_2026-09-17.json) detalla los resultados y hashes. Se comprueban replay/support, continuidad de alias y vida, inventario, reloj nativo, tenencia C4, prefijo sin futuro y gráfico. Reloj y C4 se contrastan también con los contextos retenidos de 3E.16 y 3E.15 mediante un comprobador independiente.

La selección descubre una diferencia de muestreo: replay/support tienen pasos de cuatro ticks, pero el JSONL canónico de `player_states` tiene pasos de 32. No hay una fila exacta en 8636; la última anterior es 8620. La primera versión del extractor documental suponía una fila exacta y se detuvo por esa comprobación. Se corrigió esa suposición local y se conservó la ausencia explícita, sin modificar un productor, elegir un corte posterior para ocultarla ni trasladar valores de 8620 a K2. Los estados actuales mostrados proceden del replay/sidecars exactos disponibles.

La reproducción de x/y es a la resolución del replay. Se conserva z y la limitación de planta/radar; no se infiere inmovilidad absoluta, disponibilidad para disparar, colisiones o contacto. No se reutiliza un smoke antiguo como si fuera una ejecución nueva.

El comprobador independiente corrigió una condición inicial demasiado amplia que exigía ausencia de disparos en toda la ronda anterior al corte. El control correcto distingue el prefijo histórico de las novedades entre K1 y K2. La corrección y los nueve registros anteriores se conservan en su resultado; no cambian el corte ni añaden interpretación táctica.

## Aplicación documental separada de R1–R7

Esta sección es análisis del agente para comprobar límites de la presentación, no respuesta humana ni salida de un modelo. La ficha neutral no incorpora estas propuestas como conclusiones.

| Criterio | Lo que aporta K2 | Qué sigue sin acreditarse |
|---|---|---|
| R1, plan | El grupo sigue desplegado en la misma zona y dirección general, con cambios de orientación/avance documentados. Puede mantenerse o matizarse la lectura de K1; corresponde recoger la interpretación nueva. | Plan comunicado, intención mental o compromiso irreversible. No se impone que haya cambiado el plan por presentar otro corte. |
| R2, continuidad/control | T1 repite x/y mientras otros avanzan; se conoce exactamente el prefijo mostrado. | Espera deliberada, final de entrada, consolidación, control de ángulos o detención física exacta. |
| R3, funciones | T4 sigue siendo portador; T5 conserva utilidad equipada y posición detrás. | Nueva función cierta, error de separación o cobertura física del portador. La lectura de funciones de K1 se preserva como interpretación. |
| R4, información | Reloj, estado y recursos propios visibles en registros; voz y contacto siguen desconocidos. | Ausencia de amenazas o comunicación; conocimiento individual de todo lo mostrado al revisor. |
| R5, apoyo | Posiciones y orientación actuales permiten situar a cada actor. | Trade viable, líneas de visión, ruta o alternativa superior. No se prescribe acercarse o esperar. |
| R6, utilidad | Humo equipado y flash en inventario; cambio de pitch de T5. No aparece lanzamiento propio hasta el corte. | Preparación mental de un lanzamiento, trayectoria, detonación, efecto sobre aliados/rivales o eficacia. |
| R7, corrección | Se puede discutir si la interpretación del plan debe actualizarse. | Error, acierto táctico o mejora prescriptiva respaldada. No se asigna misión correctiva. |

## Retención, alcance y continuación

Se retienen política previa, dos inspecciones, scripts y correcciones locales, corte anonimizado, corroboración, SVG/PNG e inventarios en destinos nuevos. Fuente, corpus, bundles, temporales anteriores, K1, respuestas, Excel v4 e interfaz temporal se mantienen sin escritura. No hay nueva adquisición, parsing de demo, entrenamiento, promoción, publicación o modificación de CURRENT.

Esta preparación no cambia código del producto, productores, schemas o runtime; no aplican suites funcionales ni smoke nuevo. S2–S3, cero capacidades elegibles, las 18 hipótesis pendientes y las objeciones a F04-K3/F05-K3 conservan su estado.

El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) deja como única siguiente acción recibir y registrar la lectura de K2. No se prepara otro corte ni una implementación antes de esa aportación dependiente.
