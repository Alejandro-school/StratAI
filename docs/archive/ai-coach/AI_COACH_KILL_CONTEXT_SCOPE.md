# Contexto rival a partir de bajas

Encuadre recibido el 26/09/2026. El usuario pide enlazar la feed con la identidad y
posición del atacante, además del arma que causa la baja, porque esa información
puede cambiar la interpretación táctica. La instrucción concreta autoriza esta
ampliación; no se vuelve a pedir una elección de perspectiva ya resuelta.

Se conserva su afirmación «siempre si matan a un compañero identificas al enemigo
desde donde lo han matado» como criterio experto, sin convertir la garantía del
100 % en un hecho técnico verificado. La feed permite enlazar atacante y arma;
la replay aporta la posición registrada del evento. La entrega presenta ambas
fuentes y no afirma percepción o comunicación universal del punto exacto.

## Alcance y ejemplos

- Mostrar víctima, atacante, arma causante, tiempo de ronda y las posiciones
  registradas en el evento Kill. Ejemplo de formato: «CT2 → T3 · AWP · 00:48».
- Marcar al atacante sólo desde ese evento y conservar su última posición con
  antigüedad, sin actualizarla con trayectoria global oculta. Otra baja causada
  por él puede actualizar el punto; una baja suya retira su marcador.
- Posición de víctima desde Kill cuando está disponible; la muestra anterior con
  vida sigue siendo fallback explícito. No sustituir un origen de lanzamiento por
  la posición del atacante cuando una granada mata más tarde.
- Arma de la baja desde `callback_weapon`, sin inferir la que sigue equipada.
  AWP, M4, escopeta o utilidad son hechos para la revisión; no generan todavía
  una regla táctica nueva, recomendación o etiqueta de entrenamiento.
- Conservar suicidio/mundo/atacante ausente como tales; no inventar enemigos.
  Respetar cuchillos, calentamiento, discontinuidades, lado, identidad y vida.

## Fuentes y verificación

Fuente disponible: `canonical/events/combat_events.jsonl`, contrato
`stratai.combat_event@2`. `registerAtomicCombatHandlers` usa `combatPlayerRef`
para capturar posiciones de atacante y víctima en el callback, y el arma procede
del mismo Kill. Esta fuente ya existe y no es el subconjunto filtrado de
`captureRawKillEvent`. La falta de contactos en la proyección de las fichas no
significaba que el registro canónico careciera de bajas y posiciones.

Nuevo suplemento `stratai.review_kill_context@1`, generado por
`ai_coach.interpretation.kill_context`: verifica hash de soporte/manifiesto/
combate, checksum de demo, identidades/vidas, ronda, reloj y correspondencia con
los cortes ya mostrados. Sólo publica aliases locales. Los 102 bloques previos
de las 34 fichas permanecen intactos; el nuevo contexto se identifica mediante
instrumento web @4 y hash de ficha, sin reetiquetar respuestas anteriores.

Pruebas: prefijos y retroceso, varios eventos/observaciones del mismo atacante,
baja del atacante, daño póstumo, ausencia de arma/posición, vida distinta, hueco,
suicidio y mundo, fuente/identidad discordantes. Tras pruebas, ejecutar exactamente
una demo aprobada en TEMP nuevo y el productor del suplemento sobre esa captura.
Recomputar los valores del nuevo suplemento independientemente. No regenerar
paquetes v3 inalterados ni el corpus; conservar capturas, outputs y predicciones.

El avistamiento continuo por visión/radar sigue siendo otra dependencia. Este
encuadre permite entregar información útil por bajas sin esperar su resolución.
