# Cierre 3E.6 — relojes causales nativos del objetivo

Fecha: 9 de septiembre de 2026. **Cuentas atrás y márgenes nativos implementados y validados en los cortes acreditados.** El porcentaje de plantado conserva una dependencia de inicio/duración; apoyo completo S2–S3 sigue pendiente. [Plan](../../AI_COACH_IMPLEMENTATION_PLAN.md), [contrato](../../contracts/support/canonical-support-objective-clock-v1.md) y [evidencia completa](../../evidence/AI_COACH_SUPPORT_OBJECTIVE_CLOCK_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Resultado y correspondencia temporal

El selector `--ai-coach-support-objective-clock-candidate` captura dos ticks nativos de servidor, intervalo explícito, controles de pausa, deadlines y duraciones nativas, tipos declarados desde el serializer, vidas y referencias actuales. Los cortes support/objetivo/C4 se conservan. Dos schemas nuevos elevan el registro a **72 schemas/19 razones**; catálogo `1.7.0`, parser `v16`, export `3.8.0` y quality `12` permanecen iguales.

El primer intento real mostró que el tiempo de simulación del pawn estaba ausente en todas las muestras. El consumidor inicial, limitado a esa correspondencia, no acreditaba los relojes. Se añadió una prueba positiva causal: en **25 inicios de ronda**, `m_fRoundStartTime`, declarado `GameTime_t`, coincide exactamente con ambos ticks nativos multiplicados por su intervalo en el corte público de fin de freeze. La prueba sólo sirve desde ese corte y se invalida por cambios de origen, vida, instancia, época, intervalo/sesión, ausencia/desacuerdo de ticks o retroceso. No se ajusta un offset usando el desenlace ni se convierte `Parser.CurrentTime()` en tiempo de servidor.

La demo contiene estados nativos de pausa general, técnica, de equipos, espera, freeze, warmup y reinicio. No contiene `CSVCMsg_SetPause` en las sondas retenidas; esa ausencia no se convierte en `false`. La alternativa positiva es el conjunto de propiedades nativas. El camino de simulación propia exacta sigue cubierto por fixtures, sin atribuirle cobertura real.

## Cobertura real

- **30716 cortes, 32340 anclajes y 998 filas de matriz**; 30678 anclajes con intervalo canónico y 1662 preliminares separados.
- Por cada rol, propio y aliado: **29802 cuentas atrás de ronda** y **1744 de bomba**. Ronda: 3.6875–119.875 s; bomba: 0–39.984375 s.
- **88 tiempos restantes de intento por rol**: 63 anclajes de plantado y 25 de defuse. Son observaciones correlacionadas, no 88 intentos independientes: agrupar por vida nativa y deadline da 11 grupos de plantado y uno de defuse por rol, sin adjudicar sus desenlaces.
- **25 progresos y 25 márgenes de defuse por rol**, recalculados desde duración y deadlines nativos de la misma bomba y el defusor propio/aliado. El margen observado es 23.234375 s; no acredita éxito ni viabilidad táctica.
- **14 anclajes por rol con reloj de bomba expirado y valor cero**; los otros 1730 mantienen plazo activo. No se infiere explosión desde el reloj vencido.
- **794 anclajes preliminares** conservan el conflicto público previo y abstención de relojes; sus flags de warmup/pausa no los promueven. En los otros 31546 hay correspondencia temporal acreditada. Cero decisiones elegibles.

El porcentaje de plantado queda `unavailable:native_attempt_duration_unavailable` en 63 anclajes por rol: se observan estado de arming y deadline, pero `m_flArmingLength` no tiene valor ni declaración. No se aplica un default ni se toma el primer sample como inicio exacto. La siguiente entrega busca una fuente positiva del inicio/duración.

## Validación

`npm run check:all`: **17.153 s**. Suite AI Coach: **904 pruebas —44 nuevas—**, **144.036 s** de wrapper. Cinco funciones Go nuevas, Ruff, vet, módulos fijados y contrato global correctos. Las pruebas cubren presencia/tipos/ceros, dominio nativo distinto del tick de demo, pausas, deadline vencido, intento interrumpido, referencias contradictorias, destrucción/reutilización, rondas/épocas, orden dentro del tick, cambio de sesión/intervalo, prueba futura y contaminaciones enemigas.

Se procesó **una única partida distinta**, dos pasadas de la misma fuente aprobada: `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. La segunda siguió a las suites completas tras corregir la correspondencia temporal. Build final: `working-tree-support-objective-clock-3e6-20260909-r2`. Parser final **88.437 s**, materialización **1466.326 s**, relectura **1340.571 s**, canónico **385.914 s**, cero errores.

El auditor independiente verifica **646800 hechos/estados**, matrices y denominadores; recompone cada valor numérico observado desde ticks nativos, pruebas de origen, deadlines y duración, sin importar las funciones de derivación. Comprueba 25 pruebas de origen y los prefijos que las mantienen vigentes. Los controles de acceso inicial sobre casos reales no cambian al modificar relojes de jugadores ajenos, timers de defuse enemigos o añadir futuro malformado.

Las pausas competitivas durante intentos, interrupciones en anclajes, márgenes negativos, referencias defectuosas, cero handle, vida reutilizada y cambios de intervalo/dominio se verifican con fixtures; no se inventan ejemplos reales. La pausa observada en los anclajes reales pertenece a los 794 conflictos preliminares y no acredita una cuenta atrás durante esa pausa. Las variantes reales y tipos/ausencias están desglosados en la evidencia.

## Conservación y retención

Los **72 artefactos canónicos** mantienen hash y conteos. Observaciones, selección, IDs/T0, instancias preliminares, contextos y matrices **3E.1–3E.5** son iguales al control retenido. Las comprobaciones independientes de objetivo y C4 previas también pasan.

Se conservan **4356 archivos protegidos/181 hashes**, 44 bundles y 50 fuentes; catálogo, `AGENTS.md`, interfaz temporal, `package.json`, evidencias anteriores y cambios ajenos intactos. Salida final: `C:\Users\kerch\AppData\Local\Temp\stratai-3e6-smoke-wxg1rdhw`; **85 archivos/1848790531 bytes**, con inventario. La primera salida conserva 78 archivos/1297856714 bytes, también inventariados. Trabajo y temporales anteriores retenidos; ningún borrado de datos, solicitud de limpieza, adquisición, reproceso por lote, migración, promoción, entrenamiento o publicación.

## Continuación

**3E.6B: origen nativo y duración del intento de plantado.** Acreditar la transición y su tiempo de disponibilidad, sincronizada con el deadline, o una duración nativa positiva. Un flag inicial de baseline, primer sample tardío o duración convencional no basta. Preservar las cuentas atrás y el progreso/margen de defuse verificados aquí.

Ubicación física legal del objetivo, conexión NAV física, entrada/compromiso previo, LOS/contacto, cooperación, intención, velocidad instantánea, readiness completa y transacciones económicas conservan dependencias distintas. La entrega no habilita recomendaciones ni cierra S2–S3.
