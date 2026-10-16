# Cierre 3E.6B — origen, duración y progreso nativo del plantado

Fecha: 9 de septiembre de 2026. **Implementado y validado el origen nativo del intento y su progreso temporal en los 63 anclajes de plantado de cada rol.** Se resuelve la dependencia factual de 3E.6; la capacidad táctica completa conserva las dependencias del [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md). [Contrato](../../contracts/support/canonical-support-attempt-origin-v1.md) y [evidencia completa](../../evidence/AI_COACH_SUPPORT_ATTEMPT_ORIGIN_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Qué cambió y por qué

El estado activo y el plazo final no bastaban para calcular un porcentaje: faltaba un inicio o duración acreditados. La extensión `--ai-coach-support-attempt-origin-candidate` registra las transiciones de `m_bStartedArming`, asignaciones de `m_fArmedTime` y cambios de propietario. Sella cada `FrameDone` activo, además de los frames con entregas o fin de actividad, antes del muestreo de apoyo a 16 Hz.

El SDK fijado emite propiedades iniciales otra vez después de `OnCreateFinished`. Se separan esas emisiones de una transición nativa posterior a creación. El origen exige un cambio observado `false → true`, asignación de deadline en el mismo frame, ambos ticks nativos concordantes, intervalo y prueba causal de dominio ya disponible. Vida C4/pawn, propietario e inventario se corroboran; pausas, huecos, cambios de dominio o de deadline invalidan la continuidad. No se usa un default de duración ni el primer sample tardío.

La duración y fracción describen el horizonte entre la **transición replicada, con resolución de tick nativo**, y su plazo. No se presentan como instante de pulsación, subtick físico, barra exacta del cliente o desenlace. La fuente nativa `m_flArmingLength` sigue ausente en la sonda anterior; ahora existe una prueba positiva independiente mediante inicio y deadline.

## Cobertura medida

- **277 entregas**, 2331 frames retenidos y **30716 cortes support**.
- 25 vidas C4: 75 valores de creación, 75 emisiones iniciales separadas, 106 actualizaciones posteriores y 21 destrucciones. No se observó una creación ya activa; esa variante se prueba con fixtures.
- **12 transiciones nativas de inicio acreditadas**, todas con la asignación del deadline después del flag en su frame. En las doce, `deadline − inicio_nativo = 3.0 s`; es un resultado medido de esta demo, no un parámetro del código.
- Una transición nativa `true → false` queda separada de la destrucción. Se conserva la terminación de actividad sin adjudicar éxito por el contador.
- **32340 anclajes/998 filas de matriz**; 30678 anclajes con intervalo canónico y 1662 preliminares separados.
- Por rol propio y aliado: **63 orígenes, duraciones, tiempos transcurridos y progresos observados**. Son muestras correlacionadas de **11 orígenes** en los anclajes, no 126 intentos independientes.
- Progreso observado por rol: 2.08–99.48 %; transcurrido: 0.0625–2.984375 s.
- Por rol: 31.483 no aplicables fuera del intento de plantado y **794 abstenciones** por los conflictos preliminares de fase pública ya documentados. Cero decisiones elegibles.

El informe nuevo añade sus cuatro hechos por rol. El artefacto 3E.6 conserva su contrato y sus abstenciones históricas sobre duración de plantado; sus cuentas atrás y progreso/margen de defuse permanecen idénticos. La extensión nueva aporta la prueba que faltaba sin reescribir aquella capa.

## Verificaciones

`npm run check:all`: **16.189 s**. Suite AI Coach: **944 pruebas —40 nuevas—**, **166.994 s** de wrapper. Cinco funciones Go nuevas; Ruff, vet, módulos y los **74 schemas/19 razones** correctos. Los ajustes iniciales de schema y fixtures se resolvieron antes de la suite final y de la única pasada real.

Una única partida aprobada y una pasada: `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Build `working-tree-support-attempt-origin-3e6b-20260909-r1`. Parser **94.802 s**, materialización completa **1962.097 s**, relectura **1643.246 s**, canónico **384.472 s**, cero errores.

La auditoría independiente contrasta **258720 hechos/estados**, recompone **378 valores numéricos** y recorre **1614 frames activos distintos** entre inicio y T0. Verifica transiciones, ticks, asignaciones de plazo, dominio, propiedad e inventario sin importar funciones de derivación de la aplicación. Matriz y denominadores coinciden. Las auditorías anteriores de objetivo, C4 y relojes también pasan sobre la nueva salida.

Los controles reales de acceso inicial reproducen el informe y mantienen sus valores al modificar relojes ajenos, timers enemigos o añadir futuro malformado. Las pruebas de creación activa, cancelación/reinicio en el mismo frame, asignación de plazo anterior al flag, duración cero/negativa, deadline heredado o modificado, plazo expirado, pausa competitiva, huecos, referencias defectuosas y reutilización de handle se distinguen de las variantes realmente observadas.

## Conservación y continuación

Los **72 artefactos canónicos** mantienen hashes y conteos. Las cinco fuentes support/físico/objetivo/C4/relojes y los siete artefactos de anclajes/contextos/NAV anteriores conservan todos sus hechos, IDs, T0, matrices y selección. No se promueven anclajes preliminares.

Inventario protegido: **4356 archivos/181 hashes, 44 bundles y 50 fuentes**, intactos. Catálogo `1.7.0`, parser `v16`, export `3.8.0`, quality `12`, interfaz temporal, `AGENTS.md`, `package.json`, evidencias anteriores y cambios ajenos conservados. Salida: `C:\Users\kerch\AppData\Local\Temp\stratai-3e6b-smoke-00qni8in`; **87 archivos/1925297865 bytes**, retenidos con inventario. Trabajo y temporales previos también conservados; sin limpieza, adquisición, migración, entrenamiento, promoción ni publicación.

Siguiente fase: **3E.7, posición física nativa del objetivo y disponibilidad legal**. Separar la posición del portador, la del objeto C4 vivo, la bomba caída y la plantada; acreditar presencia y vida, coordenadas/sistema espacial y fuente legal propia/aliada en T0. La posición global o cacheada del SDK no prueba conocimiento del jugador. NAV físico, entrada/compromiso, LOS/contacto, cooperación, intención, readiness y economía mantienen dependencias distintas; no se cierra S2–S3 por esta entrega.
