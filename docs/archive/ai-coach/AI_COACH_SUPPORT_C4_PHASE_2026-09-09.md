# Cierre 3E.5 — tenencia causal de C4 propia o aliada

Fecha: 9 de septiembre de 2026. **Entrega factual implementada y validada; capacidad completa de apoyo S2–S3 pendiente.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene la siguiente acción. [Contrato C4](../../contracts/support/canonical-support-c4-v1.md) y [evidencia completa](../../evidence/AI_COACH_SUPPORT_C4_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Resultado funcional

El selector optativo `--ai-coach-support-c4-candidate` añade inventario nativo y propietario actuales en el mismo corte support/objetivo. Comprueba handle completo, serie, entidad actual, vida del objeto y pawn; identifica C4 por clase `CC4` y definición nativa `49`. Cada corte se lee de nuevo. Los payloads idénticos comparten almacenamiento sin cruzar instancia/época ni completar el pasado. No usa inventario SDK, último portador o posición global de bomba.

El materializador y lector distinguen prueba bilateral positiva, ausencia con inventario completo y abstención por cobertura insuficiente o conflicto. Dos schemas propios llevan el registro a **70 schemas/19 razones**, conservando catálogo `1.7.0`, parser `v16`, export `3.8.0` y quality `12`. El acceso inicial propio/aliado conserva el prefijo exacto y excluye identidad como feature, enemigos ocultos y desenlaces.

Con C4 confirmada se enlaza la posición exacta del portador ya verificada. **Esta coordenada describe al jugador; la posición física de C4 sigue no acreditada.** No se infieren sitio de planta, responsabilidad táctica, comunicación ni intención.

## Cobertura real

- **30716 cortes**, **857 estados almacenados** y **306740 filas de jugador**, con 29859 cortes que comparten un payload recién leído e idéntico.
- **32340 anclajes y 998 filas de matriz** conservados; 30678 anclajes en intervalos canónicos y 1662 preliminares separados.
- Tenencia propia: **2057 positivas, 30283 ausencias acreditadas y 0 insuficientes**.
- Tenencia del aliado: **2057 positivas, 30283 ausencias acreditadas y 0 insuficientes**.
- Inventario observado en **213897 muestras de jugador**; 92843 mantienen pawn no utilizable. Las formas, referencias, tipos y motivos medidos están desglosados en la evidencia.
- **25 vidas de C4**, 20 con cambios observados en su referencia de propietario. No se atribuye causa a esos cambios.
- **Cero decisiones elegibles, comparaciones, recomendaciones, entrenamiento o soporte de producto añadido.** Las filas correlacionadas por partida/par no son decisiones independientes.

## Validación

`npm run check:all` pasó en **20.418 s**. La suite AI Coach pasó **860 pruebas**, 48 nuevas, en **120.43 s** medidos por el wrapper. Seis funciones Go nuevas cubren captura fresca, orden dentro del tick, longitud cero, tipos ausentes, serial completo, destrucción, reutilización de vida, fronteras, destino optativo y serialización opaca. Ruff, vet, módulos fijados, contrato global y `git diff --check` correctos.

Se procesó una sola vez `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, después de las suites. Build: `working-tree-support-c4-3e5-20260909-r1`. Parser: **83.208 s**; materialización completa: **1012.457 s**; relectura y reconstrucción: **771.071 s**; validación canónica: **402.824 s**, cero errores.

El auditor independiente contrasta **194040 hechos** de tenencia y posiciones, recomputa la matriz y los denominadores desde las referencias nativas, y comprueba vidas, propietarios y cortes. Otro control conserva los hechos de objetivo, físicos, anclajes y NAV anteriores. Las mutaciones sobre prefijos reales y futuro añadido no cambian las features iniciales; las posiciones e inventarios de enemigos ajenos al par permanecen excluidos.

La demo acredita las variantes enumeradas y sus conteos en la evidencia. Referencias malformadas, inventarios parciales, destrucción todavía presente en el mapa nativo, cero handle resuelto, reutilización del mismo handle con nueva vida, duplicados, rewind y cambios de dueño antes/después de un corte del mismo tick se verifican mediante fixtures; no se inventan ejemplos reales de esas variantes. El grafo devolvió `Transport closed`; se usaron fuentes localizadas y hashes exactos del SDK/build. La generación inicial de schemas se reejecutó con el venv del proyecto tras detectar falta de `jsonschema` en el Python del sistema; un aviso Ruff del fixture se corrigió antes de las suites. No hubo correcciones tras el parseo ni segundo intento real.

## Conservación y retención

Los **72 artefactos canónicos** conservan SHA-256 y conteos. Las observaciones y hechos 3E.1–3E.4, selección, IDs, T0, instancias preliminares, matrices y consultas/grafo/resultados NAV coinciden con el temporal aprobado de 3E.4. Se conservan también sus 794 conflictos preliminares de fase pública; observar tenencia no los corrige ni acredita la ronda.

Inventario protegido: **4356 archivos/181 hashes**, 44 bundles y 50 fuentes intactos. Catálogo, `AGENTS.md`, interfaz temporal, `package.json` y las 34 evidencias JSON anteriores sin cambios.

Salida: `C:\Users\kerch\AppData\Local\Temp\stratai-3e5-smoke-oanfdbop`. **83 archivos / 1591228791 bytes**, retenidos con inventario y hashes. El directorio de trabajo y todos los temporales anteriores se conservan. No hubo borrados de datos ni temporales, solicitudes de limpieza, adquisiciones, reproceso por lote, migración 44/44, entrenamiento, promoción o publicación.

## Piezas pendientes y continuación

3E.5 cierra la prueba factual de tenencia propia/aliada y la referencia a la posición del portador. No implementa ubicación física legal de bomba caída/plantada, cuenta atrás efectiva ni progreso de intentos; no se declara ausencia de sus fuentes. La siguiente entrega **3E.6** comprobará relojes nativos del objetivo y su dominio temporal, pausas, sincronización y disponibilidad antes de derivar un margen efectivo.

La conexión física NAV, entrada/compromiso previo, LOS/contacto, cooperación, intención, velocidad instantánea y readiness completa conservan sus límites. Economía mantiene sus dependencias de transacciones itemizadas y assets exactos. No se cierra S2–S3 ni se habilitan consejos por este avance factual.
