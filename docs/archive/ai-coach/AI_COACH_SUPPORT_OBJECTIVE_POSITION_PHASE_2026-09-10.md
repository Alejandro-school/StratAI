# Cierre 3E.7: posición nativa del objetivo y disponibilidad legal

10 de septiembre de 2026. Implementación del 9 de septiembre y cierre tras completar la relectura el día 10. Alcance definido en el [contrato](../../contracts/support/canonical-support-objective-position-v1.md) y datos reproducibles en la [evidencia](../../evidence/AI_COACH_SUPPORT_OBJECTIVE_POSITION_SINGLE_DEMO_SMOKE_2026-09-10.json). El [plan activo](../../AI_COACH_IMPLEMENTATION_PLAN.md) decide la siguiente acción.

## Resultado y alcance

Se conserva la coordenada nativa de cada `CC4`/`CPlantedC4` y se acredita por separado su sistema mundial y su disponibilidad propia/aliada. La demo proporciona **426 posiciones legales por rol**: **401** por plantado propio corroborado y **25** por contacto actual de desactivación. Son registros correlacionados de una partida y **11 vidas plantadas**, no 852 oportunidades independientes. Cero decisiones tácticas elegibles, comparaciones, recomendaciones, entrenamiento o soporte de producto.

La extensión optativa añade dos schemas `1.0.0-candidate.1`, productor Go, consumidor Python, materializador, lector, acceso a prefijo y matriz. Runtime **76 schemas/19 razones**; catálogo **1.7.0**, parser **v16**, export **3.8.0** y quality **12** conservados. El selector es `--ai-coach-support-objective-position-candidate`.

## Fuente y cobertura medida

| Unidad | Resultado |
|---|---|
| Entregas | 13633: 2196 created, 1678 creation_replay, 9732 update y 27 destroyed |
| Nacimientos | 36: 25 CC4 y 11 CPlantedC4 |
| Estados / cortes | 658 / 30716 |
| Registros geométricos por estado | 657; 547 raíces mundiales y 110 parentados |
| Raíces CC4 sin dueño | 525 registros, 20 vidas; sin atribuir automáticamente conocimiento |
| Raíces plantadas | 22 registros de 11 vidas: 14 activos y 8 resueltos |
| Anclajes / matriz | 32340 / 998; 30678 con intervalo competitivo y 1662 preliminares |
| Posición legal por rol | 426 observadas, 2056 sin transformación del parent, 794 sin fase acreditada y 29064 sin observación legal |

Las 110 C4 transportadas tienen el mismo parent nativo que el pawn propietario actual y token de attachment **973653956**. Sus coordenadas locales y offsets de hueso raíz son cero. Ese cero es válido como dato local y no es una posición mundial del objeto. Faltan transformación del attachment y modelo/pose con procedencia compatible; los campos y handles presentes no bastan para obtenerla. Las sondas de transformación mundial absoluta consultadas están ausentes, sin afirmar ausencia de todas las fuentes posibles.

El recálculo usa `celda × 512 − 16384 + desplazamiento`, según el SDK fijado. El paso de celda no limita el desplazamiento: se verifican **1081 componentes por encima de 512**, con máximo CC4 **1023.75**. Se preservan **657 rotaciones `[3]float32`**. La geometría global permanece fuera del acceso inicial.

Los **11 plantados** tienen intento nativo continuo, el mismo pawn, destrucción de CC4 y creación de CPlantedC4 en el mismo frame, evento público y reloj concordantes. Su creación ocurre **0.125 s después del deadline** en esta demo. Se registra la terminación observada, sin inventar simultaneidad con el plazo ni una tolerancia máxima. Los **2201 frames** activos previos se revisan independientemente. Una actualización corporal relevante, incluso no decodificable, o cambio de dueño/vida/época revoca la posición retenida. Contacto actual puede acreditar una nueva disponibilidad.

## Verificación

- `npm run check:all`: pasa. **1001 pruebas AI Coach**, con **57 nuevas**, pasan tras los ajustes finales; seis funciones Go nuevas, Ruff, `go vet ./...` y `go mod verify` correctos.
- Parser final: **91.65 s**. Materializador: **9823.31 s**. Relectura completa: **1405.972 s**. Validación semántica canónica: **7433.934 s**, cero errores.
- Auditoría sin importar derivaciones de producción: **3612 componentes de coordenadas**, **1804807 campos contra el prefijo de entregas**, **852 posiciones legales**, **194040 hechos/estados**, matriz completa y pruebas nativas de plantado.
- **8 controles reales** del acceso inicial: igualdad con el informe, futuro malformado excluido, geometría de vidas ajenas modificada sin contaminar hechos y ausencia de identidades entre las features. Fixtures adicionales cubren cero/ausencia, tipos y finitud, parent/attachment contradictorio, arrays fijos, baseline, destrucción/reutilización de vida, cambios de propietario/pawn/época, mismo tick con orden distinto, movimiento de ida y vuelta y pérdida/recuperación de disponibilidad.
- 72 artefactos canónicos conservan SHA-256 y conteos. Las 14 fuentes/contextos anteriores conservan observaciones, selección, IDs, T0, instancias preliminares y hechos/matrices. El informe final coincide con el cálculo previo salvo referencias sustituidas por hashes reales, y la relectura no modifica los archivos sellados.

Las duraciones son tiempos transcurridos. El materializador y la validación canónica incluyen la suspensión del equipo del 9 de septiembre entre 18:49:42 y 20:47:18 UTC, corroborada por el registro de energía de Windows. La validación canónica informa 7433.934 s con esa pausa; no se interpreta ese valor como CPU ni como latencia de producto. La lectura completa comenzó después de reanudarse el equipo.

Las fixtures de C4 poseída pero sin parent, origen mundial cero, datos malformados, cambios adversariales, pausa o huecos no se presentan como variantes reales de esta partida. La vía `current_owned_c4` no tiene coordenadas mundiales acreditadas en la demo: el attachment obliga a abstenerse.

La primera relectura se interrumpió externamente el 9 de septiembre tras 793.407 s, con código 1073807364 y sin resultado de validación. Se conservan su resultado y logs. El día 10 se verificaron los hashes del código compilado y los 89 archivos sellados antes de repetir únicamente la lectura completa; no se volvió a parsear la demo ni se modificó la salida.

## Correcciones comprobadas con la demo

Se ejecutó siempre `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Hubo cuatro intentos sobre **una única partida distinta**:

1. Consulta de un descendiente inexistente de un handle escalar: excepción del SDK antes de exportar. Se añade resolución previa de declaración y regresión.
2. Recorrido de una colección de animación sin índice nativo: excepción antes de exportar. Se conservan sus miembros directos y su tipo no soportado, con regresión.
3. Extracción completa de control, sustituida para incorporar arrays fijos de rotación. El recálculo también detectó y corrigió dos límites inventados: desplazamiento máximo 512 y creación dentro de un tick del deadline. Se añadieron pruebas con valores y temporización reales.
4. Extracción final tras las suites: captura, semántica, recálculo, matriz y lector válidos.

Los comandos exactos, versiones, hashes de binarios/código/SDK, conteos, duraciones y todos los intentos quedan en la evidencia. El grafo devolvió `Transport closed`; el contraste estructural se hizo con lectura localizada del SDK fijado y código del proyecto.

## Conservación y pendientes

Salida final: **89 archivos / 2006632100 bytes**, en `C:/Users/kerch/AppData/Local/Temp/stratai-3e7-smoke-ltuoq21y`. Se conservan la extracción sustituida —80 archivos / 1326056412 bytes—, los dos destinos fallidos vacíos y el trabajo con inventarios. Los 44 bundles, 50 fuentes, releases, exports, temporales anteriores, catálogo, interfaz temporal, AGENTS y cambios ajenos permanecen intactos. No hubo borrado, petición de limpieza, adquisición, migración, promoción o publicación.

El alcance validado no cierra objetivo/apoyo completos: transformación del objetivo transportado, observación legal de C4 caída, NAV físico, entrada/compromiso, LOS/contacto, cinemática instantánea, readiness, cooperación/intención y economía conservan dependencias concretas. El plan selecciona **3E.8, cinemática nativa propia/aliada**, como pieza independiente; no reabre los inventarios de assets ni convierte hechos ausentes en conocidos.
