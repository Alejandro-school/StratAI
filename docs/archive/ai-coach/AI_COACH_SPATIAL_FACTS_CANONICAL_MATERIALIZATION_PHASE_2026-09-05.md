# Fase 1B.12 — materialización canónica candidata de hechos espaciales

Fecha: 5 de septiembre de 2026  
Estado: **implementada, validada y cerrada operativamente**

## Resultado

El candidato `3.9.0-candidate.1` incorpora los tres hechos espaciales `@2` a
`DecisionExample@2` y a la cobertura consumida por `TaskEligibility@1`. Su
manifest compone explícitamente el bundle base inmutable `3.8.0`, los tres
reportes/sidecars, un puente opaco de participantes y los artefactos de joins y
ejemplos. No modifica el manifest, las filas ni el lineage del bundle base.

El [contrato candidato](../../canonical-spatial-candidate-v3.9.md) versiona manifest,
descriptores, loader, validator y joins. La cadena ejecutable es:

`canonical-export --ai-coach-spatial-candidate` → tres productores factuales
desde el mismo replay → reportes/JSONL `@2` y puente opaco → manifest candidato
→ loader/verificación independiente → join exacto por decisión/T0 →
`DecisionExample@2` → cobertura → `TaskEligibility@1`.

Los ejemplos separan valores de referencias, procedencia, unidades y causas de
ausencia. No contienen `tactical_assessment`, respuestas humanas ni identidad
cruda; oracle y outcomes permanecen fuera de observed. Los ocho schemas humanos
v1/v2 siguen rechazados como ground truth. Los tres IDs mixtos legados siguen
fuera de prerrequisitos.

## Cobertura real y límite del candidato

La demo produjo 918 decisiones: 246 de peek y 672 de trade. Se persistieron
918 ejemplos y 2.754 joins, tres por decisión. El join exige igualdad de partida,
ronda, participante y tick; no usa muestras anteriores ni futuras como si
describieran T0.

Los conteos siguientes se aplican **a cada uno de los tres campos**:

| Familia | Decisiones | Hecho presente | Muestra exacta con ausencia física | Sin muestra exacta |
|---|---:|---:|---:|---:|
| Peek/hold/reposición | 246 | 0 | 5 | 241 |
| Trade/spacing | 672 | 1 | 4 | 667 |
| Total | 918 | 1 | 9 | 908 |

Las 9 ausencias físicas corresponden a `outside_declared_sample_coverage`.
Las 908 ausencias de join se identifican como `no_exact_t0_sample`. El muestreo
de los sidecars permanece en 1 Hz; su baja coincidencia con los ticks de decisión
es un déficit medido, independiente de la cobertura geométrica del paquete.

La auditoría distingue `present`, `missing`, `null/unavailable`, motivos físicos,
motivos de join y violaciones temporales. Los sidecars sin join nunca acreditan
un campo. La elegibilidad del candidato sigue en **0 elegibles, 3 inelegibles y
14 no materializadas**; no hay evidencia espacial presente para peek en esta
demo. Una sola demo no promueve soporte ni abre entrenamiento.

## Verificación

- **256 pruebas del AI Coach pasadas**, incluidas 45 nuevas del candidato.
  Cubren cada hecho, negativos, unavailable, antes/igual/después de T0,
  futuro/oracle invariantes, source/map/package/hash mismatch, duplicados,
  desorden, joins manipulados, privacidad, campos sin join y compatibilidad.
- **`npm run check:all` pasado**: 88 pruebas frontend, 14 Node pasadas y
  3 omitidas, todos los paquetes Go, 43 pruebas backend Python y build Vite.
- El catálogo `1.6.0` valida 17 capacidades, 15 decisiones, 51 acciones,
  18 outcomes y 58 campos. Los contratos runtime validan 33 schemas y
  19 reason codes. Se corrigió una expectativa antigua de 28 schemas antes
  de ejecutar el smoke.
- El lector antiguo sigue leyendo `3.8.0` y declarando estos campos como
  `missing`, incluso con sidecars adyacentes. El nuevo lector exige el candidato
  explícito. El rollback consiste en volver a seleccionar el mismo bundle base,
  sin cambios de archivos ni de `CURRENT`.

## Smoke de una demo

Fuente aprobada: `demo-a11eb0d600ee702c9edb`, SHA-256
`a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`.

Se ejecutó **una sola pasada y un solo intento** del productor, después de las
pruebas, en un destino nuevo y vacío bajo `%TEMP%`. Build:
`working-tree-spatial-canonical-1b12-20260905-r1`. La demo tiene 24 rondas,
29.520 frames y 10 participantes reconciliados.

| Verificación | Resultado | Duración |
|---|---|---:|
| Productor combinado | Bundle base y tres sidecars desde un mismo replay | 124,091 s |
| Materialización y validación del candidato | 918 ejemplos recompuestos idénticos; 2.754 joins verificados | 181,430 s |
| Validador semántico canónico | 0 errores en el bundle base `3.8.0` | 467,622 s |
| Recomputación NAV desde el asset | 883 rutas, 0 discrepancias; 878 sumas de polilínea iguales | 2,544 s |

Cada sidecar contiene 12.802 filas. Exposure y cover tienen 883 hechos presentes
y 11.919 ausencias físicas; cover conserva 21.192 direcciones. Exit-route tiene
883 evaluaciones, 878 caminos disponibles y 5 ausencias NAV, además de las
11.919 posiciones fuera de cobertura. La validación independiente recomputa
sectores, oclusión, selección espacial, distancias y escenarios temporales.

Se reconciliaron 12.802 combinaciones frame/participante contra la posición,
yaw y postura del replay base, sin diferencias. Las disponibilidades fuente y
de los joins pasaron sin futuro ni contaminación oracle/outcomes.

Los 5.134 archivos protegidos conservaron tamaños y fechas de modificación;
149 hashes de manifests, assets y releases permanecieron iguales. La demo,
los 44 bundles y el estado Git de Faceit-Demos no cambiaron. `CURRENT` sigue en
`stratai-2c6db440231d463a` y sus releases conservan los hashes anteriores.

[Evidencia máquina completa](../../evidence/AI_COACH_SPATIAL_FACTS_CANONICAL_MATERIALIZATION_SINGLE_DEMO_SMOKE_2026-09-05.json).
Las rutas de los comandos se expresan mediante variables con resolución por
SHA-256 documentada en la evidencia; se conserva el orden de argumentos.

## Único pendiente operativo

La evidencia se escribió antes de intentar la limpieza. La revisión automática
rechazó tanto el borrado recursivo con preflight como el borrado individual de
los artefactos inventariados y comprobados por SHA-256. El motivo comunicado fue
`blocked by policy`, sin más detalle; ninguna de las dos órdenes se ejecutó.

Siguen retenidos **83 archivos, 746.944.298 bytes**, en:

`C:/Users/kerch/AppData/Local/Temp/stratai-1b12-smoke-hwelcj03`

Los logs, el inventario privado y la recomputación NAV están en:

`C:/Users/kerch/AppData/Local/Temp/stratai-1b12-control-qj5etbed`

Cuando se permita la limpieza o se retire la salida mediante una acción del
usuario, debe comprobarse su ausencia, actualizar la evidencia y marcar el
cierre. **No hace falta volver a procesar la demo.** El prompt de 1B.12 permanece
como única instrucción vigente y se limita al pendiente; todavía no se abre
una fase sucesora.

Tras ese cierre, el siguiente blocker medido es el muestreo espacial en T0 de
decisión: 908/918 decisiones carecen de muestra exacta. Su fase sucesora deberá
conservar las ausencias sin estado fuente acreditado y las versiones anteriores.
Gate 1B sigue pendiente y `training_allowed=false`; no se autoriza descarga,
entrenamiento, reprocesado 44/44, release, publicación ni promoción.

## Cierre operativo verificado — 6 de septiembre de 2026

Tras comunicar el usuario que había terminado la limpieza, se verificó a las 2026-09-06T08:16:41.129992+00:00 que los destinos temporales de esta fase no existen ni figuran en su directorio padre accesible. La eliminación se realizó fuera de las acciones del agente; no se atribuye hora exacta. Se conservan inventarios, hashes y rechazos históricos. No se repitieron productores, demo ni suites para este cierre. El plan activo mantiene el siguiente trabajo funcional.
