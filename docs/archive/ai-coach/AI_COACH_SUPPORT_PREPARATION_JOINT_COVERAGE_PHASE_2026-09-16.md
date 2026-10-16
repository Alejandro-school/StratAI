# Cobertura conjunta y dependencias C0 — cierre 3E.22

16 de septiembre de 2026. **Conciliación de sólo lectura completada; S2–S3 siguen abiertos y no hay una segunda implementación justificada sin material positivo nuevo.** [Plan](../../AI_COACH_IMPLEMENTATION_PLAN.md), [dependencias y pruebas de aceptación](../../ai-coach/AI_COACH_SUPPORT_C0_DEPENDENCIES.md) y [evidencia reproducible](../../evidence/AI_COACH_SUPPORT_PREPARATION_JOINT_COVERAGE_AUDIT_2026-09-16.json).

## Alcance y procedencia

Se leen C0 original 3E.12 y los nueve contextos 3E.13–3E.21, cada uno desde su salida y procedencia originales. Se comprueban contra las evidencias de cierre **810 archivos** de sus inventarios originales, y las referencias de cada manifest contra el inventario SHA-256 recalculado en esta fase. Todas las observaciones support completas coinciden al serializar su contenido de forma determinista; builds, hashes de archivo y manifests originales se conservan separados. Esto no crea un bundle conjunto ni acredita migración 44/44.

Única partida de origen, ya procesada en las fases históricas: `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Se conservan los diez build identifiers y versiones en `provenance.json`. Protocolo `support_preparation_census/1.0.0-candidate.1`, hash `424313f04d793a1a939880dc9bcf752693e0f413fe26df479994bedea42da524`. Parser v16, bundle 3.8.0, quality 12, catálogo 1.7.0 y registro de 104 schemas/19 razones permanecen intactos.

No se modifica ni rematerializa ningún contexto. El analizador documental independiente no importa consumidores de aplicación. Une por igualdad de corte completo —secuencia, frame, tick, época, instancia, reloj y paso—, candidato, actor/aliado, lado y referencias de pawn. No une por cercanía temporal ni por un anclaje 3E.1. Recompone pares a partir del roster/exclusiones, contrasta posiciones/vidas contra la captura, lee toda la ejecución retenida y reconstruye todas las matrices.

## Denominadores conciliados

| Unidad | Resultado |
|---|---:|
| Cortes totales / admisibles / excluidos por fase / por reloj | 30.716 / 27.948 / 2.768 / 0 |
| Filas de roster | 306.740 |
| Actor-cortes en cortes admisibles | 279.437 |
| Actores utilizables / no vivos | 205.324 / 74.113 |
| Actor-cortes con al menos un aliado utilizable | **199.454** |
| Pares examinados | 1.117.576 |
| Excluidos por actor / aliado | 296.452 / 160.438 |
| Candidatos dirigidos seleccionados | **660.686** |
| Filas de matriz por artefacto / total contrastado | 1.040 / **10.400** |

La diferencia entre 205.324 actores utilizables y 199.454 con aliado seleccionado se conserva; no se interpreta como ausencia de apoyo físicamente posible. Una segunda vía recalcula los **660.686 IDs únicos** desde partida/protocolo/época/instancia/secuencia/actor/aliado y obtiene SHA-256 de su secuencia `4a9f75b3c797da423ce6e5c409e6934fe65cf0ad96c80af7d98cc82428d77a50`.

Ejecución posterior por cada rol: **597.847** ventanas con cambio muestreado, **40.390** sin cambio y **22.449** censuradas. Censuras: 14.774 no vivo, 6.619 hueco/frontera y 1.056 identidad/lado. Hay **4.726** ventanas sin cambio en ambos roles. Los motivos restantes mantienen cero explícito en el manifest original. Estos resultados quedan separados del estado inicial y nunca deciden la selección. Sin cambio no significa espera, inmovilidad continua, ausencia de entrada ni una decisión correcta de mantener; no vivo no identifica por sí solo una muerte canónica.

## Cobertura marginal y conjunta

Se conservan **57 componentes**: 50 hechos por actor más siete globales del objetivo. `joint-coverage.json` contiene histogramas por hecho, motivo y rol, combinaciones de estados propio/aliado y conteos globales por corte con contexto. `status-patterns.json` guarda el orden de los campos, **42 patrones completos de estado por actor-corte** y **166 patrones de pares ordenados**. Se distinguen O=observado, N=no aplicable y U=ausente/conflicto; los motivos concretos se mantienen en los histogramas. No se cuenta N como observación positiva.

Para leer la coincidencia entre capas, la tabla usa subconjuntos explícitos de diagnóstico: físico básico con velocidad media; objetivo global y del actor; tenencia/posición del portador; diez hechos de reloj; cuatro de origen; tres de posición legal; suelo/referencia/modo; dos plazos del pawn; offset/FOV de referencia. Los campos de cada grupo están enumerados en la evidencia. **Disponible en esta tabla significa O o N**. No son requisitos de elegibilidad ni un filtro de oportunidades; los campos excluidos de estos grupos siguen presentes en el vector completo de 57 componentes.

| Grupo diagnóstico | Actor-cortes | Candidato/rol propio | Candidato/rol aliado | Mismo candidato, ambos roles |
|---|---:|---:|---:|---:|
| Físico básico y velocidad media | 196.823 | 652.973 | 652.973 | 646.386 |
| Objetivo público y del actor | 193.225 | 636.980 | 636.980 | 636.980 |
| Tenencia C4 y posición del portador | 199.454 | 660.686 | 660.686 | 660.686 |
| Relojes del objetivo | 192.675 | 635.608 | 635.608 | 634.236 |
| Origen de plantado | 193.225 | 636.980 | 636.980 | 636.980 |
| Posición legal del objetivo | 4.131 | 8.623 | 8.623 | **0** |
| Suelo y modo | 199.144 | 659.446 | 659.446 | 659.446 |
| Plazo del pawn | 193.225 | 636.980 | 636.980 | 636.980 |
| Offset y FOV de referencia | 3.625 | 10.263 | 10.263 | **214** |

Ejemplos que impiden sumar marginales:

- Físico básico completo y offset/FOV coinciden en **1.820 actor-cortes**, **5.346 candidatos por rol** y sólo **24 candidatos con ambos roles**. No implica contacto ni readiness.
- Posición legal y FOV coinciden en **23 actor-cortes**, **46 candidatos por rol**, pero ninguno en ambos roles. Las posiciones propias/aliadas no se propagan como conocimiento subjetivo.
- Los nueve grupos coinciden en **17 actor-cortes**, **34 candidatos por rol** y cero en ambos roles. El conjunto completo de 57 componentes tiene cero filas sin ausencia. Ninguna de estas intersecciones define elegibilidad: exigir todos los campos de todas las situaciones sería también un criterio incorrecto, y las dependencias físicas/tácticas siguen faltando incluso en los subconjuntos disponibles.
- `clock.attempt_progress` conserva 550 ausencias de duración en plantado. **Esos mismos 550 actor-cortes** tienen `origin.planting_progress` observado en 3E.17; otros 160 tienen progreso de defuse observado en la capa de reloj. Se documenta la coincidencia exacta sin sobrescribir el contexto antiguo ni crear una feature combinada.
- Las **4.131 posiciones legales** de 3E.18 coinciden con la abstención de `c4.physical_c4_position` de 3E.15. Son contratos de distinto alcance; la segunda capa acredita plantado propio estacionario/contacto de defuse. No es un conflicto que permita rellenar C4 transportada o caída.

Los detalles por estado/motivo, las 36 intersecciones entre pares de grupos para cada unidad y las intersecciones acumuladas quedan retenidos. El orden acumulado se declara en `group_order`; no expresa prioridad táctica. Las igualdad de marginales propio/aliado procede de conservar ambos sentidos del par, no de independencia estadística. Una sola demo repetida en smokes históricos es material de desarrollo; no aporta nuevas partidas, generalización ni tamaño efectivo de muestra.

## Verificación de esta fase

La pasada completa de unión y conciliación tarda **270,586 s**. Una segunda agregación desde los patrones conjuntos realiza **168 contrastes**, recompone marginales/máscaras y recalcula los IDs en **12,523 s**. Coinciden todas las matrices, sus ceros, estados/motivos, conteos y descriptores posteriores. Se comprueban contrato, enlaces locales, prompt único y `git diff --check`; resultados y comandos exactos en evidencia.

3E.22 es la fase de análisis de sólo lectura prescrita por su prompt: **cero productores, cero demos nuevas y cero rematerializaciones**. No se ejecutan ni se atribuyen a esta fase las suites funcionales o smokes de 3E.12–3E.21. Sus resultados históricos se conservan con procedencia; no se reutilizan como un smoke nuevo. No cambian parsing, exportadores, hechos, schemas ni contratos runtime. La validación propia es integridad, conciliación exhaustiva, segunda agregación y verificación documental/contractual.

Durante el desarrollo del script local se corrigieron un nombre `inspect.py` que interfería con el módulo estándar y el tratamiento de `global=null` en cortes sin candidatos. La comprobación de inventarios se adaptó a las tres claves históricas (`output_inventory`, `inventory`, `smoke_inventory`) y leyó el inventario referenciado, no sólo hashes embebidos en la evidencia. Fueron errores del analizador documental; las comprobaciones finales pasan, sin defectos funcionales encontrados ni cambios en los datos originales.

## Conservación

**1.170 archivos previos / 22.232.117.945 bytes** se comprueban antes/después por SHA-256, incluidas salidas históricas, el intento rechazado 3E.19 R1, fuentes, contratos y evidencias. Dentro de ellos se verifican separadamente los 810 archivos contra sus inventarios originales. Corpus protegido: **44 bundles, 50 demos, 4.356 archivos por tamaño/mtime y 181 hashes seleccionados**, sin cambios; no equivale a rehash completo del corpus.

Trabajo nuevo y resultados retenidos en `C:/Users/kerch/AppData/Local/Temp/stratai-3e22-work-5c399a2e7b694b32a729d5eb813b444d`, con scripts, inventario, copia del plan/prompt ejecutado y sello de cierre. Baseline de 1.268 archivos del workspace; sólo documentación/evidencia/prompt de esta fase pueden cambiar. Producto, interfaz temporal, catálogo y runtime conservan sus hashes. Sin limpieza, descargas, adquisiciones, migración por lote/44–44, promoción, publicación, entrenamiento ni cambios de CURRENT.

## Cierre y segunda fase bloqueada

La [matriz de dependencias](../../ai-coach/AI_COACH_SUPPORT_C0_DEPENDENCIES.md) identifica, para cada requisito S2–S3, evidencia actual, material positivo necesario, comprobación independiente y mínima acción útil. NAV 3E.3 y rayos siguen siendo diagnósticos internos: falta unión factual/acreditación física; trasladarlos a C0 no resolvería esa dependencia. Contacto legal, velocidad instantánea, readiness completa, attachment transportado/C4 caída, intención/cooperación y economía conservan sus abstenciones.

**3E.22 se cierra; la segunda fase del bloque no se inicia ni se declara completada.** No apareció un defecto que requiera corrección ni una fuente nueva que justifique otra extensión. El punto de bloqueo es la ausencia de material positivo verificable para intervenir sobre los requisitos restantes. Para la anterioridad de S2 hacen falta frontera orientada previamente definida, geometría/transformaciones vinculadas a la demo y prueba temporal suficiente; las otras vías de entrada están especificadas en la matriz. La disponibilidad de un paquete de assets sólo abriría su comprobación, no cerraría S2–S3.

Se deja un único prompt condicionado a recibir ese material y comprobar su procedencia; no se reserva artificialmente otra extensión ni se repite el inventario negativo. La autorización de dos fases no autoriza inventar alcance ni adquirir las fuentes faltantes. **Cero decisiones elegibles, comparaciones, recomendaciones o entrenamiento; ninguna tercera fase ejecutada.**
