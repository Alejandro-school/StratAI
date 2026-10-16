# Temporización nativa propia y aliada en C0 — cierre 3E.20

15 de septiembre de 2026. **Plazo nativo del servicio del pawn unido al corte C0 exacto; ticks/fracciones del arma conservan abstención. Cero decisiones elegibles.** [Plan](../../AI_COACH_IMPLEMENTATION_PLAN.md), [contrato](../../contracts/support/canonical-support-preparation-weapon-timing-v1.md) y [evidencia](../../evidence/AI_COACH_SUPPORT_PREPARATION_WEAPON_TIMING_SINGLE_DEMO_SMOKE_2026-09-15.json). Primera fase del bloque actual; 3E.19 estaba cerrada antes de iniciar esta implementación.

## Entrega y pruebas

`ai_coach.support.preparation_weapon_timing` añade acceso inicial, materializador y lector, con ocho hechos y dos schemas estrictos: **102 schemas/19 razones**. Reutiliza productor y semántica 3E.9, sin cambios Go. Recorta siete entradas antes de validar. Exige corte común, entregas/estados, pawn y arma actuales, vida nativa/handle completo/serie, corte físico por source_sequence, propietario bilateral y continuidad actor/lado/pawn/instancia/época. Copias de features y procedencia/testigos separados; ausencia no elimina candidatos.

Pasan `npm run check:all`, **1759 pruebas AI Coach, 94 nuevas**, Ruff del alcance y contrato. Cobertura: primera muestra, claves exactas, futuro malformado incluso del mismo tick/FrameDone, identidades/vidas/series/dominios, propiedad/arma física actual, GameTime_t/pausa/cero/ausencia/tipos/no finitos, intervalos/origen/simulación, pending/elapsed/paused, no aplicabilidad y abstenciones GameTick_t/fracciones, copias/invariancia ajena y corrupción del lector. Durante desarrollo se corrigieron expectativas de fixtures (clave heredada de movimiento y campos frame/tick inexistentes en físico); las suites finales pasan después de esas correcciones. Ruff general conserva dos avisos preexistentes en spatial_candidate y su test, fuera del alcance.

## Demo y recálculo

Una pasada de `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`, después de las suites. Build `working-tree-support-preparation-weapon-timing-3e20-20260915-r1`, TEMP nuevo/vacío, comandos exactos y binding privado retenidos. Productor **105.131 s**, materializador **559.931 s**, lector **656.053 s**, canónico **360.35 s**.

| Comprobación | Resultado |
|---|---:|
| Cortes / actor-cortes / candidatos | 30716 / 199454 / 660686 |
| Hechos y abstenciones recalculados | 1595632 |
| Plazos del pawn recalculados numéricamente | 193225 |
| Estados del plazo por actor-corte | {"elapsed": 152161, "pending": 41064} |
| Testigos causales de reloj | 25 |
| Campos de snapshots contrastados | 2349080 |
| Matriz / artefactos canónicos iguales | 1040 / 72 |

Auditor independiente sin consumidores de aplicación: replay de entregas/predecesores/vidas, corte físico actual y propiedad, testigos causales de origen, resta GameTime_t, estados y abstenciones, referencias y matriz completa. **24547597** muestras comprueban continuidad; causas retenidas: **{}**. Los conteos por actor-corte y por candidato/rol se separan en evidencia. Positivos reales pending/elapsed se distinguen de variantes sintéticas de pausa/tipos/corrupción; no se afirma cobertura real de todos los bordes.

Siete auditores anteriores recalculan físico/objetivo/C4/reloj/origen/posición/movimiento frente a captura nueva y conservan C0, candidatos/IDs/T0/exclusiones/ejecución. Fuente de temporización comparada con 3E.10. Lector completo y artefactos antes/después iguales. Tres controles reales incluyen primera muestra, cambio de estado del plazo y último corte; los dos primeros prueban futuro malformado, geometría ajena y copias.

## Conservación y límites

44 bundles/50 demos: 4356 archivos por tamaño/mtime y 181 hashes seleccionados, sin cambios. **993 archivos previos** comprobados por SHA-256, intactos. Salida `C:\Users\kerch\AppData\Local\Temp\stratai-3e20-smoke-eorwgea0` y trabajo `C:\Users\kerch\AppData\Local\Temp\stratai-3e20-work-1jk4js93` retenidos con inventarios. Producto, interfaz temporal, catálogo, DecisionExample, DEC-TRADE-001 y cambios existentes preservados. Sin limpieza, descarga/adquisición, migración por lote/44–44, promoción, publicación, entrenamiento ni cambios de CURRENT.

Un plazo vencido no acredita can_fire/readiness. Tick/fracción del arma conserva dependencia de codec/dominio/sentinels. Entrada/compromiso/contacto/NAV, velocidad instantánea, attachment transportado/C4 caída, intención/cooperación y economía mantienen sus dependencias. Una demo y ventanas correlacionadas no acreditan generalización. **S2–S3 siguen abiertos; cero decisiones elegibles.**

## Siguiente acción

**3E.21: entradas visuales propias y aliadas en C0.** 3E.10 acredita offset nativo de ojos y FOV estable de referencia (32340 offsets y 370 FOV por rol en sus anclajes), aún sin unión al T0 prospectivo. Unir sus cinco hechos y medir cobertura C0 conserva flash/humo/contacto unavailable, sin nueva investigación de proxies ni contacto inventado. Se prepara el único prompt; la segunda fase comienza sólo después de este cierre y su sello.
