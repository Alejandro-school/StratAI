# Feed, arma causante y posiciones del evento de baja

26/09/2026. Entrega publicada en las 34 fichas existentes. El usuario pide usar
la feed para identificar al atacante, situarlo en el mapa y conocer el arma que
causó la baja. [Alcance acordado](AI_COACH_KILL_CONTEXT_SCOPE.md). IC31 en el
libro TP10 v22 conserva su literal y la afirmación del 100 % como opinión experta,
sin convertirla en una garantía técnica de percepción universal.

La fuente canónica ya contenía los hechos: `combat_events.jsonl`, contrato @2,
registra atacante, víctima, arma del callback y posiciones de ambos. El cierre
anterior describía correctamente la carencia en los tres bloques de la ficha,
pero no había conectado esta fuente disponible. El impedimento de validar
avistamientos no impedía mostrar la información de las bajas.

Nuevo productor `ai_coach.interpretation.kill_context` y suplemento con schema
`stratai.review_kill_context@1`: verifica checksum de demo, hashes, identidad,
lado/vida, ronda y corte. Sólo publica aliases locales; el arma procede del Kill,
no de inventario posterior. El tiempo usa la observación previa del reloj y tick
nativo. No hay subtick observado ni afirmación de percepción/voz. Se conservan
suicidios, mundo y datos ausentes sin inventar atacantes.

La web @4 muestra 118 ocurrencias de bajas (63 aliadas) de tres partidas ya
capturadas, sin reprocesarlas. La feed incluye ambos equipos, arma y tiempo.
Cruces en posición de víctima del evento, con fallback anterior explícito;
rombos para la última posición del atacante en una baja, con antigüedad. Nueva
baja causada por él actualiza la marca; su muerte la retira, incluido daño
póstumo. El retroceso elimina los eventos posteriores. La posición cuando una
granada mata no se llama origen de lanzamiento. Se conservan tres campos,
mm:ss, invitaciones y respuestas antiguas con su instrumento/hash originales.

Pruebas: 8 del nuevo productor, 6 de contrato, 43 de backend, lint público y
lint focal aprobados; 15 de la web aprobadas. Un smoke posterior sobre una sola
demo aprobada: `demo-0beb9110ec408f80c39b`. Parser/binario Go sin cambios y sellados
contra el build registrado. 41 bajas, 21 aliadas y 8.255 cortes en diez
intervalos de esa misma partida, contrastados independientemente con el registro
canónico en identidad, arma, posiciones, tiempo, disponibilidad y marcadores.
Contratos y hashes verificados; cero bajas omitidas en ese smoke. No se volvió a
generar el paquete v3 inalterado. Cuarenta y cuatro manifiestos y 160 archivos
protegidos intactos. Comandos, tiempos, SHA y temporales quedan retenidos.

Verificación alojada: 34/34 fichas coinciden con el build, conservan los tres
bloques originales e incorporan el suplemento de bajas. Guardado, revisiones,
reintento y aislamiento comprobados; diez registros anteriores preservados y
dos nuevos registros de prueba ocultos por defecto. Cero lecturas expertas
inventadas. No hubo prueba visual/interactiva de navegador.

El libro v22 conserva las 14 hojas, valores, fórmulas, controles y estilos;
únicamente A36:G36 añade IC31, comprobado por comparación y render. No se
modificó el detector ni se entrenó. La evaluación debe distinguir instrumentos:
los revisores de @4 reciben contexto que el detector congelado todavía no usa.
La dependencia de avistamiento continuo sigue abierta, pero no bloquea revisar
los casos con esta información por bajas. [Evidencia](../../evidence/AI_COACH_KILL_CONTEXT_2026-09-26.json).
