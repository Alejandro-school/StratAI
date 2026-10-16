# ICT-I1.B1 — comprobación real del canal de spotting

22 de septiembre de 2026. Se ha comprobado una demo aprobada con una sonda
aislada. **Hay estado nativo de spotting; no está acreditada la distinción
entre punto rojo e interrogante para un jugador.**

El [plan](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva el estado y la siguiente
acción. El [recibo experto](../../evidence/AI_COACH_ICT_I1B_EXPERT_CORRECTION_2026-09-22.json)
y el Excel v7 se mantienen sin cambios. La [evidencia del smoke](../../evidence/AI_COACH_ICT_I1B_SPOTTING_PROBE_2026-09-22.json)
incluye comandos, hashes, conteos y temporales retenidos.

## Resultado observado

| Comprobación | Resultado |
|---|---|
| Partida | `demo-a11eb0d600ee702c9edb`; una sola partida, una pasada |
| Duración | 21,177 s de proceso; 20,805 s de parsing/captura interna |
| Perspectiva declarada | `is_hltv=true`, slot 10; build 10896, patch 14177 |
| Cobertura | 158.386 fotogramas; 4.205 registros diagnósticos |
| Propiedades | 12 entidades pawn, 72 consultas; las tres rutas directas presentes y las tres anidadas ausentes |
| Entregas del SDK | 1.762 callbacks de propiedades; 970 callbacks `PlayerSpottersChanged` |
| Instantáneas cambiadas | 1.372 cortes, 15.047 instancias de actor, incluidos espectadores/ausencias de pawn |
| Mensajes de radar | Cero `ProcessSpottedEntityUpdate` y cero sobres de ese tipo en los caminos instrumentados |
| Recálculo | 27.352 máscaras recalculadas; 41.028 valores de instantánea contrastados con callbacks previos |

Las declaraciones capturadas son `m_bSpotted: bool` y
`m_bSpottedByMask: uint32[2]`. El SDK entrega los valores de máscara como
`uint64`; se rechazan valores fuera de 32 bits por palabra. La segunda palabra
se observa a cero. La decodificación Python independiente usa representación
binaria, separada del bucle de bits del productor Go.

Se observan 2.496 muestras con máscara no nula, incluidas 410 con varios bits.
Los 3.103 bits activos encuentran un único candidato de slot en su corte. Eso
no demuestra entrega del radar a ese jugador ni al resto del equipo.

Entre cortes compatibles por alias, entidad, lado, ronda y época de reloj hay
390 transiciones de cero a no cero, 379 de no cero a cero y 174 entre valores
no nulos distintos. **Son cambios de máscara, no contactos visuales etiquetados.**
Hay dos reasignaciones de índice de controlador. La identidad de objeto/serie
se conserva; no se reasignan retrospectivamente los callbacks a un jugador.

En las 13.676 muestras con datos directos completos, el booleano es falso con
las máscaras a cero en 11.180, y verdadero con algún bit activo en 2.496.
No aparece una distinción adicional que por sí sola identifique interrogante
persistente frente a marcador retirado. La sonda no interpreta esa correlación
como prueba del estado visual. No hubo callbacks de registro ni repeticiones
del mismo valor entre callbacks sucesivos de la misma propiedad/entidad.

## Implementación y verificación

Nuevo comando independiente `backend/go-service/cmd/spotting-probe`, sin integrar
estos datos en el coach ni modificar el parser de producción o los contratos
canónicos. Escribe exclusivamente bajo TEMP vacío, rechaza enlaces/reparse,
destinos no temporales y sobrescrituras antes de abrir la demo; verifica el
checksum antes de parsear. Conserva las salidas fallidas.

Formato diagnóstico `stratai.spotting_probe@1`, con schema de observaciones
retenido y comprobación independiente de todas las filas, identidad, orden,
épocas de reloj, tipos, presencia, bits y conteos. Las instantáneas se toman al
final del fotograma y no se atribuyen a un callback anterior. El tick es el
reloj del SDK en la entrega, no una prueba de ejecución subtick ni un timestamp
independiente del emisor. Los callbacks de creación tampoco se convierten en
nuevos avistamientos.

Pasaron `npm run check:all` en 71,541 s y ocho pruebas específicas de la sonda.
La prueba de symlink no pudo crear el enlace por privilegios de Windows y se
omitió; una comprobación adicional con una **junction real** confirmó el rechazo
antes de abrir un input inexistente. Se conserva también ese directorio vacío
de prueba y su enlace. Cuatro alteraciones del schema fueron rechazadas.

El productor no consulta posiciones globales, nombres, Steam IDs, desenlaces ni
etiquetas. Los actores llevan aliases locales. Todas las features tácticas,
estados del HUD y atribuciones de receptor continúan sin habilitar.

Se verificaron sin cambios 599 archivos protegidos y los 44 hashes de manifests
aprobados. El checksum de la demo original coincide antes y después. No se ha
revalidado el contenido completo de los 44 bundles, ni se han reprocesado,
promovido, entrenado o eliminado artefactos. El TEMP del smoke conserva cuatro
archivos y 367.520.337 bytes, con inventario y SHA-256.

## Dependencia que queda delimitada

Se entrega el [contraste independiente necesario](../../ai-coach/AI_COACH_ICT_I1B_RADAR_STATE_DEPENDENCY.md):
vincular los campos nativos al radar realmente mostrado, con perspectiva y
reloj acreditados. El criterio experto ya está recibido; la carencia es factual.
No volver a procesar otras demos para buscar un positivo ni inventar duración
del interrogante. ICT-I1.B1 cierra la sonda; ICT-I1.B completa permanece pendiente.
