# Cierre 3E.9: temporización nativa del arma propia y aliada

11 de septiembre de 2026. Extensión factual implementada y validada con una demo real. El [contrato](../../contracts/support/canonical-support-weapon-timing-v1.md) fija los campos y sus límites; la [evidencia](../../evidence/AI_COACH_SUPPORT_WEAPON_TIMING_SINGLE_DEMO_SMOKE_2026-09-11.json) conserva comandos, resultados e inventarios. El [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene orden y dependencias.

## Resultado

Se acredita el plazo nativo `GameTime_t` del servicio de armas del pawn en **31546 anclajes por rol**: **5952 pendientes y 25594 vencidos**, con **794 abstenciones** por reloj no acreditado. Un plazo vencido sólo describe ese plazo. **No hay un `can_fire=true`, readiness completa ni apoyo viable acreditados.**

Los ticks y fracciones del arma se capturan con tipos y procedencia, pero **no se convierten a segundos**. El SDK fijado expone los `GameTick_t` consultados como `uint32` por su decoder por defecto `VarUint32`; no aporta una prueba del codec semántico del wrapper, su referencia temporal, composición de fracciones o sentinels. No se aplican divisiones ajustadas a los datos, reinterpretaciones de signo ni fórmulas de otra build. El valor `4294967294` de aplazamiento queda nativo y no se interpreta como espera real o plazo vencido.

Se añaden productor Go, dos schemas `1.0.0-candidate.1`, consumidor Python, acceso inicial causal, materializador y lector con recálculo completo. Selector `--ai-coach-support-weapon-timing-candidate`. Runtime **80 schemas/19 razones**; parser **v16**, export **3.8.0**, quality **12** y catálogo **1.7.0** conservados. Cero decisiones elegibles, comparaciones, recomendaciones, entrenamiento o soporte de producto.

## Cobertura real

| Unidad | Resultado |
|---|---|
| Entregas / estados / cortes | **67162 / 12640 / 30716** |
| Estados por objeto | **4142 de pawn y 8498 de arma**, en 22 clases capturadas |
| Vidas | **1208 creadas en el prefijo, 946 capturadas, 1051 destrucciones** |
| Referencias por corte | **213356 armas con identidad/tenencia contrastadas**, **541 armas actuales no resueltas**, **92843 pawns no utilizables** |
| Cambios observados | **3728 cambios de vida de arma activa**, un cambio de vida de pawn y 24 cambios de dominio de corte |
| Anclajes / matriz | **32340 / 998**; **30678 competitivos y 1662 preliminares** conservados |
| Plazo del pawn por rol | **31546 observados**, **794 abstenciones**; 25 testigos causales de origen nativo |
| Plazos primario/secundario por rol | **20092 abstenciones por codec/dominio/composición pendientes**, **12218 no aplicables** y **30 sin arma actual** en cada hecho |
| Aplazamiento por rol | **20092 abstenciones por valor grande de semántica no acreditada**, más los mismos **12218 no aplicables/30 no resueltos** |

Son conteos correlacionados de una sola partida; los estados compartidos no son oportunidades independientes. Las referencias de clase, vida y handle son procedencia, no features. Los plazos positivos del pawn en los anclajes van de **0.0009765625 a 3.051025390625 segundos nativos**; no representan resolución de input humano, HUD ni tiempo de pared.

Se reutilizó la cabecera retenida de 3E.8: **151 combinaciones clase/ruta** coinciden con sus declaraciones, sin nuevas lecturas de cabecera ni nuevas fuentes. El plazo del pawn está presente en sus 4142 estados; simulación y tick base siguen ausentes. Las fracciones primarias/secundarias contienen cero y valores positivos menores que uno. El aplazamiento contiene `4294967294` y fracción cero en los 8498 estados. Estas observaciones no prueban el significado temporal de esos enteros.

## Verificación

- `npm run check:all`: pasa en **62.545 s**. **1092 pruebas AI Coach**, incluidas **43 nuevas**, pasan en **261.002 s**; cinco funciones Go nuevas, Ruff, `go vet ./...` y `go mod verify` correctos. Suites finales antes del smoke.
- Dos pasadas de la misma única demo `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, cada una hacia su TEMP nuevo. Primer productor R1 **136.807 s**; productor final R2, tras las suites finales, **139.626 s**. Materializador R1 **4732.174 s**, lector completo R1 **3220.993 s**; canónico final R2 **388.996 s** y validación completa de sus fuentes **545.96 s**, cero errores. Son tiempos de validación aislada, con recálculo de capas previas; no miden latencia de producto.
- Auditoría sin importar derivaciones de producción: **2349080 campos** reconstruidos desde las entregas, **414613 enlaces a estados retenidos**, todos los cortes, identidades y matriz. Reconstruye independientemente los 25 testigos de reloj y **63092 plazos numéricos**; contrasta los 517440 hechos/estados iniciales.
- **Cuatro controles reales** cubren plazo pendiente/vencido en ambos roles y coinciden con el informe final. Cambiar sólo el plazo nativo de un enemigo e introducir futuro malformado con localizador temporal no cambia las features; identidades y `can_fire` quedan fuera.
- Fixtures cubren cero/ausencia/sentinel, tipos/finitud, ratios discordantes/no acreditados, relojes ausentes o incompatibles, pausa, mismo tick con distinto orden, cambios de dominio, creación/reemisión, destrucción, sustitución de pawn/arma, serie y reutilización de handle, propiedad, categoría no aplicable, futuros y recálculo del lector. En los anclajes validados no aparece pausa positiva. Tampoco se observa reutilización del mismo handle completo entre las vidas capturadas; esas dos variantes conservan cobertura sólo por fixtures.
- Los **72 artefactos canónicos** conservan SHA-256 y conteos. **Ocho fuentes del parser y diez artefactos de contexto anteriores** conservan hechos, selección, IDs/T0 y matrices, excluyendo únicamente build/referencias actualizadas. La relectura no modifica los archivos sellados. Los cuatro controles y el recuento preliminar coinciden con el informe final.

Tres fallos de helpers de auditoría se corrigieron sin cambiar el productor ni repetir la demo por esos fallos: un futuro sintético carecía de localizador `frame`, un serializer sin campos se representaba con `null` y los contratos anteriores almacenaban referencias hash en campos distintos del contenedor `references`. Se conservan los helpers originales y corregidos. Son independientes del defecto de identidad descrito a continuación.

El wrapper PowerShell devolvió 1 al redirigir el aviso de inicio del intérprete, incluso en un programa mínimo con salida nativa 0. La validación final de fuentes llegó a su resultado exitoso, idéntico en JSON y al final del log, sin traceback; el diagnóstico del wrapper queda en la evidencia.

La revisión final reprodujo un defecto de identidad: una vida física anterior podía aceptarse si otra entidad reutilizaba exactamente el handle completo. La prueba falla con la condición anterior y pasa tras exigir vida y vigencia física de la entidad nativa actual. R2 corrige sólo el productor Go y amplía su prueba; vuelve a pasar las suites antes de procesar la misma demo. R1 queda superado como productor y se retiene sin modificaciones.

Las **nueve fuentes R2 conservan exactamente todos los payloads y el resto de metadatos de R1**, salvo `build_identifier` y referencias recalculadas. Los 440 archivos restantes del inventario de compilación no cambian, incluido todo el consumidor Python. Por esa igualdad explícita, la materialización y relectura completas de R1 acreditan esos mismos datos: se conservan con su identificador y procedencia originales, sin copiarlas ni cambiar sus referencias a R2. El cargador completo valida de nuevo hashes, schemas y semántica causal de R2; la auditoría independiente recalcula desde R2 los 63092 plazos y contrasta su matriz con el informe retenido. Los cuatro controles iniciales también se repiten con R2.

El grafo estuvo disponible en la implementación; al cerrar su transporte durante la revisión final se usaron los archivos ya identificados y el SDK local. Los inventarios fijan el código compilado y validado.

## Conservación y dependencia restante

Salida materializada R1: **93 archivos / 2510908218 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3e9-smoke-z11om8sh`. Extracción final R2: **82 archivos / 1615082741 bytes** en `C:/Users/kerch/AppData/Local/Temp/stratai-3e9-r2-smoke-41akowpl`. Ambas se retienen con trabajos e inventarios: **175 archivos / 4125990959 bytes** de salida en total. La fuente, los 44 bundles, 50 demos, exports, releases, salida/trabajo/evidencia 3E.8, catálogo, interfaz temporal y cambios ajenos se conservan. Sin limpieza, adquisición, migración, promoción ni publicación.

La temporización derivada primaria/secundaria y el aplazamiento quedan pendientes de una **fuente positiva de la build correspondiente que acredite codec `GameTick_t`, dominio/referencia temporal, composición y actualización de fracciones, sentinels y aplicación por arma**, disponible en el prefijo. No reabrir el mismo inventario sin esa fuente. La semántica de cero del plazo del pawn tampoco se acredita con esta demo; el consumidor se abstiene en ese caso.

La siguiente pieza del plan es **3E.10: contacto visual legal propio/aliado en el corte support**, contrastando el evaluador candidato existente con la resolución y procedencia del corte actual. Se conservan las dependencias de entrada/compromiso, velocidad instantánea, precisión, readiness completa, transformación del objetivo transportado, observación legal de C4 caída, NAV físico, cooperación/intención y economía.
