# Traza económica 3D.3 — 6 de septiembre de 2026

Entrega funcional validada. [Evidencia completa](../../evidence/AI_COACH_ECONOMY_TRANSACTION_TRACE_SINGLE_DEMO_SMOKE_2026-09-06.json), [contrato](../../contracts/economy/canonical-economy-transaction-trace-v1.md) y [estado vigente](../../AI_COACH_IMPLEMENTATION_PLAN.md).

El selector 7 conserva 7.918 registros de canales separados. Se observan 1.564 actualizaciones de cuenta, 1.235 del contador de gasto, 26 lecturas base al crear controladores, 1.683 eventos nativos `item_pickup`, 1.683 controles genéricos, 1.683 callbacks `ItemPickup` y 44 callbacks heurísticos `ItemRefund`. La sonda recibe una lista nativa de descriptores y no deja eventos sin descriptor. Los 1.683 eventos tienen actor mapeado con el campo `ValShort` del userid tipo 9 y alias local. Esto no acredita un identificador exclusivo de transacción ni de objeto nativo.

La lectura del SDK fijado en v5.2.0 demuestra que recoge compra/recogida bajo el mismo callback y deduce refunds por coincidencia de aumento de dinero, destrucción y zona de compra. No se convierten en etiquetas verificadas. No aparecen `item_purchase`, `item_remove`, `item_refund` ni `item_pickup_slerp` en la sonda de cinco nombres; los otros mensajes de pickup/drop inventariados quedan fuera de esta captura. No es una ausencia global de compras o drops.

Se preservan tipos, campos repetidos, cero/falso/ausencia, secuencia de entrega, fotograma, tick, identidad local y fronteras estrictas. El `server_tick` nativo difiere del tick del parser en todos los eventos capturados; su relación queda pendiente. El enlace a oportunidad sólo clasifica la entrega retrospectiva. No se observan grupos reales con varias actualizaciones de saldo en el mismo fotograma/tick: los casos compensados están probados mediante fixtures, sin atribuirles cobertura real.

`npm run check:all` y **527 pruebas del AI Coach** pasan, junto con Ruff y `go mod verify`. Se corrigió durante el smoke un nombre de campo incorrecto del lector (`participant_alias` en v2) y se añadió la variante userid tipo 9 observada, siguiendo el acceso del SDK. Se repitieron pruebas y **la misma única partida aprobada**, en otro TEMP nuevo: dos pasadas, nunca dos partidas ni sobrescritura. Parser final 84.765 s; lector CLI 9.513 s; validación canónica completa 268.779 s, cero errores.

El recálculo independiente confirma conteos y enlaces por instancia/roster/ventana. Las observaciones v2 y 3D.1–3D.2D son iguales a las de la build anterior salvo build y referencias por hash: 77/250 estados iniciales, 173 ambiguos, tres equipos completos, 50 fichas, 60 diferencias numéricas, 15 transiciones y 463 referencias del manifiesto se conservan. Los 4.356 archivos protegidos y 181 hashes de control no cambian.

Limpieza pendiente por rechazo automático; se conservan inventarios y no se repite ni elude el rechazo. Los dos destinos de esta fase y sus archivos están inventariados en la evidencia; los temporales previos no se borraron ni sus rechazos se reintentaron.

La entrega permite examinar canales y sus carencias. Las compras, drops voluntarios y devoluciones itemizadas siguen sin acreditar; no se habilitan cestas, consejos, entrenamiento ni producto. La revisión conjunta posterior queda en el [informe 3D.4](AI_COACH_ECONOMY_OPPORTUNITY_SUFFICIENCY_PHASE_2026-09-06.md).
