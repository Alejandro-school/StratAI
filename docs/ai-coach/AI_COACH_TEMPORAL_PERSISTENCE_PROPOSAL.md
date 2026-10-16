# Persistencia y revisión temporal — criterio y alcance acordados

Fecha: 24 de septiembre de 2026. Encuadre recibido e implementación verificada; se conserva la versión
`collective_c4_approach@1-provisional`. El plan canónico conserva el estado de
entrega. Esta propuesta desarrolla el siguiente paso solicitado por el usuario.

[Resultado y verificaciones del alcance implementado](../archive/ai-coach/AI_COACH_TEMPORAL_PERSISTENCE_2026-09-24.md).
La evaluación independiente TP10 permanece pendiente en el plan.

## Problema y alcance concreto

R2 demuestra un veto demasiado sensible al último movimiento del portador: se
descarta A tras un ajuste pequeño pese a conservarse la aproximación global,
la C4 y la trayectoria de los demás. La inspección está cerrada y no se repite.

La siguiente versión separaría dos cuestiones: si existe evidencia suficiente
para proponer por primera vez una preparación y si una lectura anterior aún
puede sostenerse ante señales nuevas. Mantener una posibilidad no equivale a
confirmar intención o ejecución. El cambio se limita al intérprete propio/aliado,
su actualización temporal y la explicación concreta de sus abstenciones.

Quedan fuera: inferir qué vio el fallecido, reconstruir comunicación, afirmar
que A era mejor, acreditar rutas o zonas/build, recomendar y entrenar. No se
añaden respuestas humanas al detector ni se cambia la salida congelada anterior.

## Criterios ya recibidos

- IC19 permite organizaciones compatibles con preparar una entrada, pausas
  mientras la trayectoria siga disponible, revisión ante dispersión/retroceso
  y abstención ante información insuficiente.
- R2 permite interpretar ejecución hacia A pese al pequeño reajuste final de
  T1. La posible reunión de T5 es una función hipotética. No se conoce el motivo
  de elegir A ni lo que vio o transmitió el compañero muerto.
- Muerte y dato ausente se distinguen. La muerte de un actor no borra las
  trayectorias válidas de los supervivientes; obliga a revisar el grupo disponible.

Fuentes: [aceptación IC19](../evidence/AI_COACH_CONTEXTUAL_CRITERION_ACCEPTANCE_2026-09-23.json),
[lectura R2](../evidence/AI_COACH_R2_EXPERT_READING_2026-09-24.json) y
[diagnóstico técnico](../archive/ai-coach/AI_COACH_R2_DETECTOR_DIAGNOSIS_2026-09-24.md).

Estos criterios no necesitan otra confirmación. No fijan una tolerancia universal
en unidades del mapa o segundos ni resuelven la duración de una hipótesis sin
señales nuevas de preparación.

## Comportamiento propuesto

| Situación | Lectura propuesta | Evidencia que debe conservarse |
|---|---|---|
| Aproximación conjunta inicial con C4 acreditada | Proponer preparación posible hacia la referencia trazable | Participantes, trayectoria causal y C4; sin recurrir a una lectura humana |
| Reajuste breve del portador con avance global y organización aún compatibles | Mantener la posibilidad, dejando constancia del reajuste | Avance acumulado, desplazamiento contrario y continuidad; no basta mirar el último signo |
| Pausa mientras otros aliados mantienen señales compatibles | Mantener la posibilidad con estado actualizado | Posiciones y recursos vigentes, qué aliados aportan señales y qué falta |
| Alejamiento sostenido del portador o separación contradictoria del grupo | Pasar a revisión; abstenerse de una preparación vigente si pierde soporte | Señales contradictorias por actor y momento; no conservar por inercia el destino anterior |
| Nueva dirección común suficientemente respaldada | Sustituir la lectura previa | Evidencia positiva del nuevo destino; perder A no demuestra ir a B |
| Identidad/C4 ambiguas, hueco temporal o cambio de ronda | Cortar la continuidad dependiente y abstenerse | Motivo factual concreto; nunca completar con el último dato conocido |
| Muerte de un integrante con supervivientes trazables | Recalcular la organización de los vivos | Muerte explícita, composición actual y C4; sin suponer qué información transmitió |
| Grupo completamente parado, sin nuevas señales ni contradicciones | **Aceptado:** mantener la posibilidad hasta contradicción | Datos actuales completos y evidencia previa enlazada; no inventar abandono ni preparación activa |

Para el último caso el usuario responde «Mantener posibilidad hasta contradicción».
Se conserva la posibilidad respaldada, sin caducarla sólo por falta de movimiento.
No se crea una preparación desde una imagen estática ni se conservan recursos o
roles históricos como actuales. [Literal y alcance](../evidence/AI_COACH_TEMPORAL_PERSISTENCE_CRITERION_2026-09-24.json).

## Operacionalización y dependencias

Registrar por actor la aproximación neta, los tramos de avance/alejamiento,
la magnitud acumulada del movimiento contrario, su continuidad y el soporte
del grupo. La C4 no debe tener un veto basado únicamente en un último incremento.
Una lectura conservada debe enlazar su evidencia previa y describir qué cambió.

La distinción entre ajuste y retirada no se resolverá eligiendo un número que
haga pasar R2. Cualquier parámetro técnico se documentará y contrastará frente
a los controles de abajo, antes de congelar la nueva configuración. La falta
de criterio sobre una situación debe producir revisión/abstención, no una
persistencia ilimitada elegida por defecto.

No se promete una ruta física a partir de distancia a un landmark. Las señales
espaciales siguen limitadas a referencias de desarrollo sin zonas/build
acreditadas. Inventario y hechos propios/aliados conservan su procedencia causal.

## Casos y verificación prevista

| ID | Control | Resultado o condición a verificar |
|---|---|---|
| TP01 | R2 completo, con sus prefijos reales | La lectura compatible A no debe desaparecer sólo por el último pequeño reajuste de T1 |
| TP02 | Pausa con soporte previo y avance de compañeros | Conservar la posibilidad con evidencia trazable, sin afirmar ejecución |
| TP03 | Retroceso que continúa y deshace la organización previa | Revisar/retirar la lectura; impedir persistencia automática de A |
| TP04 | Grupo parado sin señales nuevas y datos completos | Mantener la posibilidad previa hasta contradicción; sin señal previa, abstenerse |
| TP05 | Destino alternativo con aproximación conjunta acreditada | Revisar hacia el destino nuevo; distinguirlo de simple pérdida de soporte anterior |
| TP06 | Salto de muestras, identidad, C4 o ronda | Interrumpir el soporte afectado; no puentear discontinuidades |
| TP07 | Muerte de un compañero con trayectoria válida de los vivos | Conservar los hechos de los supervivientes y reevaluar su organización |
| TP08 | Mismo prefijo con futuro, rivales ocultos o respuesta humana alterados | Salida idéntica; ausencia de filtración |
| TP09 | Ventanas cercanas al mismo movimiento | Describir sensibilidad y evitar cambios espurios debidos sólo al último signo |
| TP10 | Casos reales ajenos al ajuste de reglas | Revisiones independientes y exposición registrada; no convertir ejemplos de desarrollo en test |

Los controles construidos para probar el software no se presentarán como
partidas reales ni como etiquetas expertas. R2 y cualquier caso usado para
ajustar reglas son desarrollo/exploración. Los 84 episodios ya seleccionados no
adquieren independencia por repetir la ejecución.

Antes del cierre funcional: pruebas del alcance, contratos, verificaciones
públicas apropiadas y exactamente una demo aprobada en TEMP nuevo después de
las pruebas, con recomputación de señales y conteos, hashes y retención.
La versión previa y sus predicciones se conservan para comparación. La evaluación
independiente requiere un protocolo y separación por partida/jugador/fecha y
exposición, además de revisiones humanas; no se acredita sólo con pruebas.

## Encuadre recibido y condición de implementación

En una pausa de todo el grupo, con C4 e identidades verificadas y sin señales
nuevas ni contradicciones: ¿se conserva sólo la lectura histórica y se pasa a
revisión, o se mantiene «posible entrada A» mientras no aparezca contradicción?

Consulta presentada y respondida el 24/09: «Mantener posibilidad hasta contradicción».
IC24 en el libro v14 conserva el literal y límites, con v13 intacto. El encuadre
autoriza la implementación dependiente y las decisiones técnicas rutinarias.

Parámetro técnico inicial, declarado antes de ejecutar R2 con la nueva regla:
1 segundo de movimiento contrario acumulado en un tramo sin avance hacia la
referencia confirma una contradicción provisional. Las muestras estáticas no
suman tiempo ni borran lo observado. Si el portador deshace toda la aproximación
que respaldó la hipótesis, se retira sin esperar ese segundo. La dispersión usa
la dirección de los pasos hacia otra referencia, con la misma confirmación.
Es un filtro temporal de ingeniería, no una duración universal de retirada ni
un valor aportado por el experto. Se contrastan también 0,5 y 2 segundos como
análisis de sensibilidad separado; no se ajusta el valor para hacer pasar R2.

[AGENTS.md](../../AGENTS.md) exige «ask the user for their expert framing and wait
for that input before dependent implementation». La [forma de trabajo por fase](../AI_COACH_IMPLEMENTATION_PLAN.md#forma-de-trabajo-por-fase)
concreta su aplicación; no exige repetir acuerdos recibidos ni pedir permiso
para decisiones técnicas dentro de ese encuadre.
