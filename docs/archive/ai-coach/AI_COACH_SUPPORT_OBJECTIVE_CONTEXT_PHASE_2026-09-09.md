# Cierre 3E.4 — contexto causal del objetivo

Fecha: 9 de septiembre de 2026. **Entrega factual implementada y validada; capacidad completa de apoyo S2–S3 pendiente.** El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene la única continuación autorizada. [Contrato de objetivo](../../contracts/support/canonical-support-objective-v1.md) y [evidencia completa](../../evidence/AI_COACH_SUPPORT_OBJECTIVE_CONTEXT_SINGLE_DEMO_SMOKE_2026-09-09.json).

## Resultado funcional

El selector optativo `--ai-coach-support-objective-candidate` conserva base 3.8.0 y observaciones previas; añade reglas Source 2 frescas, señales del pawn y eventos públicos con orden de entrega dentro del corte support/replay. Materializador, lector y dos schemas propios conservan selección, IDs, T0, fuente, instancia, época y vidas. Registro: **68 schemas/19 razones**; catálogo `1.7.0`, parser `v16`, quality `12`, `DecisionExample` y `DEC-TRADE-001` conservados.

El acceso inicial devuelve fase pública, banderas de bomba/freeze/warmup, duración nominal observada y tiempos transcurridos en ticks del parser, además de zona de bomba, desactivación y kit propios/aliados. No incluye actor/vida como features, enemigo oculto, sitio, coordenadas de bomba o desenlace posterior. Cada evento se aplica sólo después de su entrega, aunque comparta tick con un corte anterior. Se conserva el tipo y la presencia nativos; faltas y conflictos no se rellenan con defaults.

## Cobertura real

- **30.716 muestras**, **94 eventos públicos** y **32.340 anclajes/998 filas de matriz** conservados.
- **31.546 fases públicas observadas:** 29.802 preplant y 1.744 plantadas. **794 conflictos** entre fase terminal y replay que vuelve a jugar en la misma instancia preliminar quedan como información insuficiente.
- En intervalos canónicos: **30.678 anclajes**, con 28.934 preplant y 1.744 plantados. Los **1.662 preliminares** se mantienen separados: 868 preplant y 794 conflictos; no se promueven.
- Las cuatro reglas tienen presencia y tipo correcto en las **30.716 muestras**. Los tiempos transcurridos de ronda se recalculan en 32.340 anclajes; los de bomba en 1.744. Los otros estados de reloj de bomba son 29.802 no aplicables y 794 indisponibles por conflicto de fase.
- **Cero decisiones elegibles, comparaciones, recomendaciones, entrenamiento o ampliación del producto.** Observar una fase no acredita responsabilidad táctica, cuenta atrás efectiva o apoyo viable.

Los conteos por campo, presencia, valor y tipo originales, actor/aliado/instancia y acreditación competitiva están en el informe temporal y la evidencia. Cada dirección del par sigue correlacionada; no se presenta como una decisión independiente.

## Validación y correcciones

`npm run check:all` pasó en **17.265 s**. La suite AI Coach pasó **812 pruebas**, 37 nuevas, en **115.939 s** de ejecución medida por el wrapper. Cinco funciones Go nuevas comprueban captura/orden, identidades, opt-in, destino y serialización. Ruff, vet, módulos fijados, contrato global y `git diff --check` correctos. La comprobación final del grafo devolvió `Transport closed`; se conserva su respuesta y se contrastan directamente los archivos y sus hashes de build.

Sólo se procesó `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, en **tres pasadas de la misma partida**, cada una a un destino nuevo:

1. El control de datos reales detectó que el prefijo antiguo `cs_gamerules_data` devolvía ausencias falsas. Se corrigió a `m_pGameRules`, contrastado con `bindGameRules` del SDK Source 2, y se añadió regresión con ambos prefijos presentes.
2. La inspección del prefijo real detectó 794 anclajes preliminares tras un fin de ronda y otro fin de freeze sin nueva instancia. Se corrigió el acceso inicial para abstenerse ante el conflicto, sin modificar anclajes ni acreditación competitiva. Se añadió regresión del caso real.
3. Después de las pruebas finales, la ejecución aceptada usa build `working-tree-support-objective-3e4-20260909-r3`. Parser: **72.355 s**; materialización: **611.697 s**; relectura y reconstrucción: **545.353 s**; validación canónica completa: **251.746 s**, **cero errores**.

Los dos materializadores iniciales se detuvieron al detectar el defecto; sus salidas no son evidencia de una entrega válida. Ambos intentos se conservan, junto con sus builds y logs. Un fixture inicial de combinación de modos omitía el argumento que pretendía comprobar; se corrigió. Un log de suites generales solapadas se descartó; las pruebas finales proceden de ejecuciones completas sin solapamiento de la misma suite.

El auditor independiente recalcula fases desde las entregas originales, fronteras de eventos, tiempos, datos nativos y matriz, sin utilizar la función de reconstrucción de producción. Controles con datos reales comprueban que mutar enemigos o añadir futuro no cambia el acceso inicial; también se reconstruyen los controles NAV/físicos anteriores. Variantes sintéticas —ceros temporales en un anclaje, tipos incorrectos, referencias reutilizadas, rewind y ausencia exacta— se distinguen de las observadas en esta demo de desarrollo.

## Conservación y retención

Los **72 artefactos canónicos** conservan SHA-256 y conteos. Observaciones, anclajes, contextos físicos, consultas/grafo/resultados NAV y sus matrices/resúmenes coinciden con 3E.3. Inventario protegido: **4.356 archivos/181 hashes**, 44 bundles y 50 fuentes intactos. Catálogo, `AGENTS.md`, interfaz temporal y evidencias anteriores sin cambios.

Salida aceptada: `C:\Users\kerch\AppData\Local\Temp\stratai-3e4-smoke-bb9v9wqf`. **81 archivos / 1532803303 bytes**, con inventario y hashes en la evidencia. Los intentos rechazados y el directorio de trabajo permanecen retenidos. No hubo intentos de borrado ni solicitudes de limpieza, adquisición, migración 44/44, entrenamiento, promoción o publicación.

## Lo que sigue pendiente

Este contrato no implementa tenencia de C4, ubicación legal del objetivo, progreso completo de intentos ni cuenta atrás efectiva con pausas. No se afirma que sus fuentes falten en la demo. La siguiente entrega **3E.5** debe acreditar tenencia propia/aliada en el mismo corte, antes de usarla para interpretar responsabilidades.

La conexión física NAV, entrada previa, contacto, cooperación, intención, velocidad instantánea y readiness completa conservan sus límites. Economía mantiene sus dependencias transaccionales y de assets exactos. Esta entrega no cierra S2–S3 ni habilita consejos; la validación integral y táctica del coach sigue pendiente según el plan.
