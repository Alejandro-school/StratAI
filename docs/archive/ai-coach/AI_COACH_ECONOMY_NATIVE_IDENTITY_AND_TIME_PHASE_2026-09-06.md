# Identidad y relojes económicos 3D.5 — 6 de septiembre de 2026

Entrega funcional validada; limpieza pendiente por rechazo automático. [Evidencia completa](../../evidence/AI_COACH_ECONOMY_NATIVE_IDENTITY_SINGLE_DEMO_SMOKE_2026-09-06.json), [contrato](../../contracts/economy/canonical-economy-native-identity-v1.md) y [estado vigente](../../AI_COACH_IMPLEMENTATION_PLAN.md).

El selector optativo **8** añade `economy_native_identity@1`, un productor Go y un lector Python que rechaza procedencia, serie, secuencia o fronteras inválidas. Conserva separados el tick de la cabecera de demo, las muestras `CNETMsg_Tick` y el `server_tick` del evento. Las referencias a entidades se resuelven con índice, serie completa, clase y disponibilidad en ese instante. No se completan identidades desde el futuro ni se incorporan estos datos al estado inicial.

Se investigaron las fuentes primarias del SDK instalado `demoinfocs-golang/v5@v5.2.0` y su despachador `godispatch@v1.4.1`; sus hashes y enlaces están en la evidencia. El grafo de código devolvió `Transport closed`, por lo que se usó lectura localizada. No se repitió la investigación de codecs, defaults o paquetes de assets.

| Comprobación en la única partida aprobada | Resultado verificable | Límite |
|---|---|---|
| Reloj de red | 158.371 muestras y un segmento observado | No hay reset ni ausencia reales; esas variantes tienen fixtures. |
| Evento frente a red, mismo fotograma y tick de demo | 1.642 coincidencias; 41 eventos con un tick menos | La diferencia no acredita su causa ni sincronización universal. |
| Vida de entidades | 1.208 creaciones observadas: 1.196 equipos y 12 pawns; 1.051 callbacks de destrucción | 617 reutilizaciones de índice, 109 sustituciones sin destrucción observada y 48 entidades vivas al final; no se inventa el cierre de vidas. |
| Referencias de dueño e inventario | 2.567 entregas a pawn y 1.173 a equipo resueltas | Son entregas, no objetos únicos ni transacciones; reconciliación bilateral pendiente. |
| Referencias descartadas o ausentes | 67 series de equipo distintas, 863 equipos no disponibles, cuatro pawns no disponibles y 563 sentinels inválidos | No se unen por índice. Otros 1.516 slots ausentes quedan fuera de la longitud activa observada. |
| Enlaces con la traza | 3.410 enlaces exactos a eventos/callbacks previos de la misma ejecución | 1.460 callbacks `ItemPickup` tienen entidad observada y 223 no. La asociación por tipo del SDK no prueba objeto exclusivo del evento. |
| Mensajes pickup/drop y envoltorio económico | Cero entregas en los cuatro canales consultados | Ausencia acotada a esta sonda y demo; no prueba ausencia de compras o drops. Sin cobertura real de sus cuerpos. |
| Fronteras | 75 entregas; 73 enlazadas por instancia y tick | Dos sin instancia competitiva; no se enlazan por número de ronda repetido. |

La diferencia red–demo es 11.232 ticks en la primera muestra y 11.233 en las otras 158.370. Se conserva como distribución observada, sin aplicar offset. El SDK ordena mensajes por prioridad y puede entregar un evento antes de aplicar las entidades de su paquete. La coincidencia de relojes tampoco acredita subtick de ejecución o conocimiento del jugador.

El SDK reenvía las propiedades registradas al crear entidades. Hay **3.172 lecturas base, 3.172 callbacks del fotograma de creación y 4.618 posteriores**. El grupo de creación se conserva como potencial reenvío, sin llamarlo nuevas modificaciones nativas. Esta precisión también se añade al contrato de la traza 3D.3; sus archivos y las 26 evidencias anteriores no se reescriben.

El primer smoke encontró un defecto real de orden: `godispatch` inicializa la caché recorriendo mapas, aunque su comentario promete orden de registro. Faltaban las 25 sondas de inicio del contexto y el validador rechazó la salida. Se corrigió **sólo la ruta del selector 8**, agrupando consumidores del proyecto por tipo concreto y ejecutándolos en orden explícito. Dos pruebas nuevas reproducen un despachador invertido y la retirada de un consumidor durante una entrega. El orden relativo a la lógica interna del SDK sigue sin garantizarse; las secuencias no son IDs estables entre builds.

Después de la corrección pasaron `npm run check:all`, **549 pruebas del AI Coach** —22 nuevas—, las seis pruebas Go añadidas, Ruff y `go mod verify`. Se procesó **la misma única partida aprobada** en otro TEMP vacío: dos pasadas, ningún segundo match ni sobrescritura. Parser final **77,585 s**, reconstrucción **5,604 s**, lector de identidad CLI **35,386 s** y validación canónica completa **263,920 s, cero errores**. Alias `demo-a11eb0d600ee702c9edb`; SHA-256 completo, comandos, build `working-tree-economy-3d5-20260906-r2` e inventarios en la evidencia.

El recálculo independiente, sin importar el lector de identidad, confirma relojes, segmentos, referencias, series, vidas, longitud de inventario, fronteras y enlaces de traza. Las seis salidas de control v2/3D.1–3D.3 conservan exactamente sus observaciones —incluido el orden de la traza en esta ejecución— salvo build y referencias de procedencia. Se mantienen 77/250 estados iniciales acreditados, 173 ambiguos, tres equipos completos, 50 oportunidades, 60 diferencias numéricas, 15 transiciones y 48 oportunidades con traza. Las 1.467 capturas nativas, 1.215 intervalos, 930 entradas añadidas, 54 retiradas y 463 referencias del manifiesto se conservan. Los 4.356 archivos protegidos y sus 181 hashes de control no cambian.

El registro tiene **53 schemas y 19 razones**, plan `2.16` y contrato humano `1.23`; catálogo `1.7.0`, export base `3.8.0`, parser `v16`, quality `12` y manifest `@3` conservados. No se cambian elegibilidad, runtime, corpus, release o CURRENT. No hay compras, devoluciones o drops voluntarios itemizados acreditados: el saldo y los 44 refunds heurísticos siguen sin demostrar causa o importe por objeto. Reglas efectivas completas, assets exactos y cooperación permanecen sin acreditar; comparación, entrenamiento y producto siguen deshabilitados.

La siguiente acción es **3D.6: reconciliar tenencia por objeto durante freeze**: comprobar cuándo dueño y slots activos describen el mismo hecho, con corte temporal verificable, conflicto y ausencia explícitos. No es otra auditoría de conteos ni una promesa de consejo. La dependencia exacta de assets se mantiene separada.

La revisión automática de aprobación rechazó `Remove-Item` sobre las dos nuevas salidas con `blocked by policy`, antes de iniciar el proceso. Se conservan **171 archivos / 1.536.981.476 bytes**, recontrastados por hash después del rechazo. No se repitió ni eludió el intento, ni se tocaron temporales anteriores. La deuda de limpieza es independiente de la validación funcional.
