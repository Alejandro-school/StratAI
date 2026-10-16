# Estado de ronda causal del replay v1

Extensión optativa `stratai.replay_round_context@1` dentro de cada frame del productor de contactos. Conserva replay base 5, bundle 3.8, catálogo 1.7.0 y 116 schemas runtime. La proyección de equipo @6/input @4 añade zonas y C4 previamente cerrados; los instrumentos anteriores conservan su semántica.

El número pertenece a la instancia actual del replay. `phase` distingue `freezetime`, `preplant`, `planted`, `ended` y `unknown`. `RoundEnd` acredita el final, independientemente de seguir muestreando hasta `RoundEndOfficial`. `end_tick` y `winner` sólo aparecen en snapshots posteriores al final. Nunca se consultan para anticipar la victoria desde el resumen final de la ronda.

`elapsed_seconds` cuenta ticks de demo desde freeze, con tickrate observado. No acredita segundos activos durante pausas. Las cuentas de ronda y C4 son campos separados y nullable. El legacy `time_remaining`, que usa defaults, no repara sus ausencias.

Un reloj observado exige controles de pausa completos y pausa general descartada; declaraciones nativas `GameTime_t`; mensajes NET y packet del frame actual, coincidentes; intervalo nativo válido y testigo de origen en el frame exacto de fin de freeze. Un reinicio, rewind, cambio de entidad, intervalo u origen invalida ese testigo. No se ajusta un offset con resultados futuros. La cuenta es deadline menos tiempo nativo actual, limitada inferiormente a cero. Más de una bomba activa provoca abstención. `clock_evidence` conserva fuentes, tipos, testigo y ambos mensajes para recálculo independiente.

La proyección sólo expone contexto fechado. Entre muestras conserva fase histórica y abstiene el reloj. `eligibility` separa evidencia causal del objetivo C4 de `unknown`; nunca clasifica cuchillos por el arma activa o por falta de C4. El clasificador positivo de cuchillos conserva sus dependencias previas. Estado resuelto no demuestra por sí solo explosión o desactivación.

Ejemplo N01: 15 s transcurridos, 100 s de reloj nativo preplant. N04/N05/N10: ronda terminada, sin cuenta activa. Los cortes históricos no se sustituyen. La verificación y el estado de entrega se registran en el [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md).
