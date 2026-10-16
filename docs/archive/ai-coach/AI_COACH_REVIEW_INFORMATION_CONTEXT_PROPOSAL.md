# Contexto de bajas y rivales para revisar las jugadas

Alcance del 26/09/2026, aceptado con «Implementalo por favor.» tras recomendar la perspectiva del equipo. La petición identifica una carencia del instrumento: conocer sólo los vivos y los movimientos aliados no permite explicar adecuadamente decisiones tras perder compañeros.

El usuario acepta implementar la opción recomendada de información del equipo. Se registra la aceptación contextual, sin inventar una respuesta literal distinta ni reabrir criterios de radar/contacto visual ya aceptados el 22/09. La vista global de espectador no forma parte de esta ampliación. La simplificación del formulario es independiente y ya está implementada. Este encuadre satisface la elección de perspectiva de AGENTS.md; no acredita por sí mismo nuevas fuentes de contactos.

Alcance propuesto para información del equipo:

1. Historial de bajas aliadas: quién, cuándo y lugar respaldado por el evento de muerte y una referencia espacial válida. Marcarlo al avanzar la secuencia; ocultarlo al retroceder antes del evento. Si sólo existe una muestra anterior, indicar «última posición registrada antes de morir» y su momento, sin convertirla en lugar exacto de muerte.
2. Contactos rivales: identidad local, observador, instante y ubicación de la señal, únicamente con fuente y atribución acreditadas. No incorporar posiciones globales ocultas como conocimiento del equipo.
3. Contacto perdido: conservar como antecedente con antigüedad, sin afirmar que el rival sigue allí ni inventar tiempo de caducidad del radar. No duplicar al mismo rival observado por varias personas.
4. Información ausente: voz, comunicación, atención o traslado de información al resto del equipo siguen desconocidos cuando no hay evidencia. Una baja no acredita por sí misma que la víctima viera a su atacante.

Auditoría de las 34 fichas publicadas: 63 transiciones vivo→muerto dentro de los intervalos mostrados, cada una con muestra anterior, identidad continua y `adjacent_samples`. Las 63 tienen posición anterior disponible y ninguna tiene posición en la primera muestra muerta. Son ocurrencias en las fichas, no 63 eventos independientes. Los JSON incluyen sólo actores aliados, objetivo y recuentos agregados; no incluyen contactos rivales o un registro de bajas con localización. El renderer descarta al actor al morir. [Auditoría](../../../outputs/stratai-review-simple-20260926/context-availability.json).

Fuentes localizadas: `registerAtomicCombatHandlers` registra eventos Kill con tiempo e identidades; `captureRawKillEvent` conserva posiciones víctima/atacante, pero excluye suicidio, fuego amigo y víctimas sin killer. Sus campos de zonas y visibilidad no acreditan por sí mismos información recibida. Debe contrastarse con la fuente completa; no usar ese subconjunto como registro universal de bajas ni sus coordenadas de atacante como avistamiento.

El vínculo de spotting nativo con observador/receptor, coordenada comunicada y presentación sigue pendiente, según [la dependencia de radar](../../ai-coach/AI_COACH_ICT_I1B_RADAR_STATE_DEPENDENCY.md). Una máscara no prueba por sí sola qué vio el jugador, qué apareció en su radar o qué comunicó. Se mantienen las fuentes exclusivamente de replays reales; no se solicitan grabaciones personales ni se repite la sonda sin evidencia nueva.

Verificación prevista tras encuadre: coincidencia de demo, reloj, identidad/vida, ronda/lado y posición; cortes antes/después de baja/contacto, huecos, muertes simultáneas, fuego amigo, suicidio, contacto perdido, varios observadores y retroceso del cursor. Sólo hechos disponibles hasta cada corte. Si cambia un productor/contrato de replay, aplicar el smoke de exactamente una demo aprobada en TEMP, después de las pruebas; no reprocesar los 44 bundles. Conservar las fichas/predicciones y etiquetar el nuevo instrumento sin alterar las lecturas anteriores.


## Resultado del alcance disponible

Historial y posiciones aliadas anteriores implementados y publicados como
`stratai.review_web@3`; criterio IC30 en libro v21. Once pruebas, una demo real
y verificación alojada de las 34 fichas. Contactos, observadores y contacto
perdido siguen sin fuente atribuida; no se declara completada esa capacidad.
[Cierre y límites](AI_COACH_REVIEW_INFORMATION_CONTEXT_2026-09-26.md).


## Corrección posterior: información por bajas

El 26/09 el usuario pide enlazar la feed, atacante, posición y arma. Esa fuente
canónica ya existía y ha quedado conectada en instrumento @4; no dependía de
resolver los contactos visuales. Las limitaciones históricas del apartado
anterior describen @3, no la ficha vigente. [Alcance y significado del dato](AI_COACH_KILL_CONTEXT_SCOPE.md)
y [cierre](AI_COACH_KILL_CONTEXT_2026-09-26.md). Avistamiento
continuo, percepción y comunicaciones siguen diferenciados de los eventos Kill.
