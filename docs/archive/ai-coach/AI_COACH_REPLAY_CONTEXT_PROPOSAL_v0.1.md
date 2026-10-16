# Interpretación contextual desde replay — ICT-I1.R2

23 de septiembre de 2026. Propuesta concreta y aplicación documental a K1/K2.
Usa los criterios ya recibidos; no añade respuestas del experto ni implementa
un clasificador. El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado.

## Resultado que debe producir

Explicar qué plan sugiere la organización propia, qué función parece asumir cada
jugador y por qué mantener o revisar esa lectura al llegar otro corte. Cada frase
debe distinguir hechos, interpretación y desconocidos. No valorar todavía si la
jugada es buena ni recomendar una alternativa.

La referencia es la [lectura experta de K1](../../evidence/AI_COACH_F01_K1_EXPERT_REVIEW_2026-09-17.json)
y su [continuidad en K2](../../evidence/AI_COACH_F01_K2_EXPERT_RESPONSE_2026-09-17.json).
Esta aplicación conoce esas respuestas: no es una predicción independiente ni
demuestra que un modelo haya aprendido a reconocer entradas.

## Aplicación al caso real

### K1 — tick 8524, secuencia 1389

**Lectura contextual:** el despliegue sugiere un plan de entrada en B. T1–T4
avanzan en el recorrido mostrado; T4 porta C4. T5 permanece detrás con humo
equipado y utilidad disponible, compatible con la función de preparar utilidad
que le atribuyó el experto. El grupo se organiza para entrar con el portador.

**Fundamento:** movimiento propio mostrado durante el prefijo, distribución,
recursos y presencia del objetivo. «B» conserva su procedencia en la localización
de la ficha y la lectura humana; no procede de un clasificador espacial del
registro temporal. C4 y humo equipado sí están en los registros exactos.

**Límite:** «cubrir a T4» es una interpretación de organización aportada por el
experto. No prueba ángulos cubiertos, contacto, trade viable ni protección física
efectiva. No se asignan automáticamente los roles de entry, segundo entry o lurker
a T1–T3 por su orden. Tampoco se afirma que el asalto ya se haya consumado.

### K2 — tick 8636, secuencia 1417

**Actualización:** se mantiene la lectura de entrada/push hacia B. T1–T4 han
avanzado entre ambos cortes y T4 conserva C4. T5 mantiene su posición horizontal
y el humo equipado, cambiando la orientación de la mirada. La breve repetición
de posición de T1 al final no basta para convertir la secuencia en abandono o
en un plan distinto. Esto coincide con la continuidad expresada por el experto.

**Precisión factual:** no se afirma que los cinco se estén desplazando
simultáneamente. La ficha K2 acredita tres posiciones horizontales repetidas de
T1 durante 0,125 s; T5 no tiene desplazamiento horizontal resuelto entre cortes.
Esa duración describe este caso, no fija un umbral táctico para detectar pausas.
Los deltas del informe R1 son adyacentes: no se usan como si abarcaran K1→K2.

**Límite:** no se declara un lanzamiento de T5, un humo efectivo, una espera
deliberada de T1, defensa débil, ausencia de contactos ni decisión correcta.
«No vemos motivo acreditado para cambiar esta lectura» no significa «sabemos que
el equipo recibió cero información nueva».

## Frontera entre evidencia e interpretación

| Afirmación | Base disponible | Tratamiento |
|---|---|---|
| T4 lleva C4 | Tenencia propia/aliada en ambos cortes | Hecho con referencia exacta. |
| T1–T4 avanzan entre los cortes | Posiciones de las fichas y registros contrastados | Descripción del desplazamiento; no prueba tránsito seguro. |
| T5 tiene humo equipado | Estado físico exacto | Hecho; inventario completo procede de la ficha, no del registro R1. |
| El plan es entrar en B | Conjunto de señales y lectura experta localizada | Hipótesis contextual, sin probabilidad inventada ni intención mental confirmada. |
| T5 prepara utilidad para el grupo | Recursos, distribución y papel atribuido por el experto | Función posible en este episodio; no rol permanente ni regla basada sólo en arma equipada. |
| El grupo cubre eficazmente a T4 | La lectura humana habla de cubrirlo | Eficacia física no acreditada; no convertirla en hecho. |
| Hay que continuar, rotar o corregir la jugada | Faltan percepción, alternativas y comparación suficientes | No emitir recomendación. |

La munición exacta se excluye de esta interpretación: la ficha K1 ya recoge la
diferencia entre `raw_clip=20` y el exportado `19`. Que R1 reproduzca ese valor
exportado no resuelve su semántica ni acredita capacidad concreta de disparo.
No se modifica retrospectivamente el informe retenido ni se usa ese número para
valorar disponibilidad del arma.

## Criterios trasladables y excepciones

1. **Combinar señales propias.** Examinar distribución, trayectoria, C4, recursos
   y acciones disponibles hasta el corte. No convertir «cuatro jugadores», una
   distancia, un humo equipado o un tiempo fijo en una regla universal.
2. **Separar plan y ejecución.** Un plan de entrada puede estar preparándose;
   inferirlo no acredita que se haya cruzado una frontera física ni ganado espacio.
3. **Mantener continuidad con razones.** En K2 la organización sigue siendo
   compatible con K1. Una pausa breve o un jugador de utilidad retrasado no obligan
   por sí solos a cambiar de plan. No conservar toda hipótesis indefinidamente:
   si el contexto anterior deja de estar disponible, declarar información insuficiente.
4. **Conservar alternativas sin inventar señales.** Un fake no queda descartado
   como posibilidad mental, pero no recibe respaldo positivo aquí por el mero
   hecho de que exista como alternativa. E03 requiere presión/utilidad y contexto
   que no se incorporan como hechos observados a estos cortes.
5. **Aplicar E01–E04 sólo con sus premisas.** Más defensa percibida, oportunidad
   en medio, fake o pérdida de quien buscaba información no se activan con datos
   enemigos omniscientes, ausencia de señal o etiquetas de este ejemplo. E02
   mantiene la excepción contextual de todo el equipo en B.

Q1 aporta avance/disparos/C4 como señales contextuales; aquí no se añade un
disparo que no figure en las entradas. Q18 y Q26 conservan sus respuestas, pero
este episodio no juzga el efecto de una flash ni un postplant.

## Alcance propuesto para el primer prototipo

Entrada: prefijo propio/aliado con identidad, reloj, trayectoria, recursos,
objetivo y referencias. Una referencia espacial puede utilizarse sólo con fuente
y alcance explícitos. Las respuestas humanas, el nombre K1/K2, el resultado de la
ronda y los datos rivales ocultos quedan fuera del generador de hipótesis.

Salida propuesta: lectura compatible, señales que la respaldan, funciones
posibles, alternativas con sus razones, desconocidos y cambio frente al corte
anterior. Sin porcentajes, puntuación táctica ni consejo. Esta es una propuesta de
interfaz de dominio, no un nuevo schema, endpoint o modelo entrenado.

Hay dos dependencias concretas antes de una emisión automática «entrada en B»:

- **Contexto temporal suficiente:** R1 muestra dos cortes y cambios adyacentes;
  no entrega por sí solo la trayectoria de cuatro segundos ni todos los eventos
  utilizados en las fichas. El prototipo debe consumir el prefijo correspondiente
  con sus referencias, sin rellenarlo desde el texto experto.
- **Referencia espacial trazable:** el registro temporal declara zona semántica
  no disponible. La localización humana aceptada para estas fichas no constituye
  automáticamente un clasificador general de coordenadas a B. Sin esa referencia,
  conservar la descripción de avance propio con C4 y dejar el destino sin determinar.

Estas dependencias no exigen vídeo, otra demo, reabrir la sonda de spotting ni
resolver todos los iconos del radar. Tampoco requieren volver a preguntar cuál
es el plan de K1/K2. El siguiente trabajo técnico debe resolver el contrato de
entrada y el alcance de esas referencias antes de conectar un generador.

## Verificación prevista

- Misma evidencia y configuración: lectura previa invariante al cambiar futuro,
  resultado, identidad del caso, respuestas del experto o rivales ocultos.
- Toda señal cita un hecho disponible hasta su corte; una salida «B» requiere
  referencia espacial admisible. No usar los propios resultados esperados como entrada.
- Cambios de vida/lado/ronda y huecos conservan sus límites. Cero y falso no se
  transforman en ausencia. Pausas no se traducen automáticamente en retirada.
- Contrastar desarrollo con K1/K2 sin llamar a ese acuerdo precisión del modelo.
  Evaluar generalización después en episodios independientes revisados; no basta
  contar cortes de una misma partida como ejemplos independientes.
- Si cambia un productor, hecho derivado, schema o contrato runtime: pruebas y
  exactamente una demo aprobada en TEMP nuevo, con recálculo y retención. Esta
  preparación documental no cambia esos componentes ni requiere repetir el smoke.

## Resultado de esta fase

Queda concretada la salida contextual esperada y su trazabilidad en dos cortes
reales, junto con excepciones, dependencias y verificación. No hay criterio nuevo
que registrar en el Excel ni pregunta experta que repetir para esta aplicación.
No se ha ejecutado un detector de intención ni acreditado recomendación o aprendizaje.
