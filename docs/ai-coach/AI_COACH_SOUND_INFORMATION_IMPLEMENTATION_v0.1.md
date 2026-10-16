# Información sonora estimada y perfiles fijos de CS2

Fecha: 15/10/2026. Contrato de la implementación local de sonido. El estado de
verificación y la siguiente acción pertenecen al [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md).
Conserva el [encuadre recibido](AI_COACH_SOUND_INFORMATION_SCOPE_v0.2.md) y la
[auditoría anterior](AI_COACH_SOUND_NATIVE_SOURCE_AUDIT_v0.1.md).

El usuario pide buscar los valores fijos del alcance que representa el círculo
del radar e implementarlos para pasos, recargas, zoom y las demás emisiones.
Esta instrucción resuelve la revisión de perfiles que había quedado pendiente;
no requiere volver a pedir que el usuario proporcione radios.

Los archivos extraídos del juego publicados por SteamTracking proporcionan las
definiciones. Se fija el commit `df666dcbaa6f0d0cd713426537d9f71fddc442b3`,
publicado el 15/10/2026. Se conservan archivos originales, URL, SHA-256 y
generador en [la evidencia de esta entrega](../../outputs/stratai-sound-ranges-20261015/).
La tabla compartida está en [cs2_profiles.json](../../backend/go-service/pkg/sound/cs2_profiles.json):
1312 eventos, 548 perfiles utilizables. Los demás conservan sus valores y
procedencia, pero no conceden recepción con este contrato.
No es una calibración acústica realizada sobre el ejecutable de la demo.

Se extraen eventos de `game_sounds_footsteps`, `game_sounds_player`,
`game_sounds_weapons`, `game_sounds_world` y `game_sounds_physics`. Cada nombre
conserva su propio radio y fuente. Se usa el `broadcast_distance_override`
positivo cuando existe; en su ausencia, el extremo de ganancia cero de una
`distance_volume_mapping_curve` explícitamente habilitada. No se sustituye un
radio ausente por una distancia universal. `display_broadcast` debe estar
habilitado y la señal no puede ser local, silenciosa o de volumen cero.
La selección es un modelo conservador de alcance utilizable; no demuestra que
el motor dibuje exactamente ese círculo en todas las versiones y condiciones.
La ayuda nativa de Sound Tools confirma que la curva relaciona distancia y
volumen, y describe un descarte de voces de un solo disparo por debajo de
0,001 de atenuación. Ese descarte y la curva de ganancia efectiva no se
simulan: el radio elegido es una referencia geométrica máxima, no una
medición exacta del círculo renderizado ni de audición. Se conserva la
[ayuda y su hash](../../outputs/stratai-sound-ranges-20261015/native-help-source.json).

| Ejemplo de evento nativo | Radio, unidades del juego | Fuente del valor |
|---|---:|---|
| `Base.Footstep`, `Base.Land` | 1100 | Final de curva |
| `Weapon_AK47.Clipout` | 1100 | Final de curva del componente de recarga |
| `Weapon_AWP.Zoom`, `Weapon_SSG08.Zoom` | 597.142883 | Final de curva |
| `C4.PlantSound` | 1300 | Final de curva |
| `c4.disarmstart`, `c4.disarmfinish` | 2000 | Final de curva |
| `Weapon_AK47.Single` | 9000 | Override de difusión |
| `Weapon_USP.SilencedShot`, `Weapon_M4A1.Silenced` | 1400 | Final de curva |
| `Grenade.PreThrow.Gear` | 1100 | Final de curva |
| `Default.WalkJump` | 98 | Override de difusión, anterior al final de curva |

Los ejemplos no reemplazan la tabla: ciertos materiales, armas y componentes
tienen otros valores. No se presenta «todas las recargas = 550» ni «todos los
pasos = 800» a partir de tablas web incompatibles con los archivos retenidos.
Tampoco se inventan pasos audibles a partir de movimiento, velocidad, pulsar
Shift o un cambio de estado de zoom.

## Fuente y disponibilidad causal

`ParseDemoWithReplaySoundInformation` activa el nuevo productor de forma
explícita, junto a los productores factuales y visuales anteriores. Las rutas
históricas conservan su selección y contratos. No cambia el bundle base 3.8
ni se promueve el candidato sobre el corpus aprobado.

- `CMsgSosStartSoundEvent`: nombre resuelto desde el hash MurmurHash2 del evento,
  con seed `0x53524332`, sin adivinar por proximidad temporal. Los parámetros
  usan claves con seed `0x31415926`. Se decodifican registros tipados de
  posición, volumen, pitch y el entero nativo `source_soundscape`. Este último
  describe el entorno sonoro y no se convierte en radio, bando o identidad.
  Formatos, claves o números no válidos abstienen. Las señales físicas con
  `impact_speed_input` se reconocen, pero quedan abstendidas: su ganancia
  dinámica requiere otra curva y no se presume positiva por tener radio.
- `CCSUsrMsg_WeaponSound`: nombre y origen explícitos del mensaje nativo. Estos
  mensajes incluyen componentes de recarga, manejo, cuchillo y granadas.
- `CMsgTEFireBullets`: definición de arma y `WeaponSound_t` nativos. La tabla de
  `weapons.vdata` y `items_game.txt` distingue sonido normal y silenciado,
  incluidos tipos 2, 9 y 18 cuando tienen correspondencia explícita. Un
  `WeaponFire` genérico o el arma activa posterior no eligen ese perfil.

La disponibilidad es el tick del callback, dentro de los límites de la ronda.
Las posiciones y vida se capturan en ese callback. No se toman muestras futuras
ni se reconstruye el origen oculto desde una trayectoria posterior.

## Recepción y memoria

Para una emisión positiva con perfil utilizable, se calcula
`sqrt(dx² + dy² + dz²) < radio`. Se excluye el extremo, cuya ganancia es cero.
Se admiten todas las direcciones y alturas. No se utiliza FOV, raycast visual,
pared, suelo o humo como bloqueo acústico. No se modelan materiales, ecos,
enmascaramiento, ajustes del usuario, atención ni audición efectiva.

El observador directo debe estar vivo y pertenecer al bando receptor. Una
muerte en ese mismo tick impide nueva recepción directa. Las emisiones
propias no se convierten en señales rivales. Si el origen nativo no acredita
bando, se conserva desconocido. La procedencia CT para T, o T para CT, es una
estimación derivada de demo autorizada por el usuario, sin reconocimiento real.

La proyección `team_replay_projection@11`, activada con `soundInformation`,
añade `replay_sound_information@1`. Recalcula la distancia desde los puntos
nativos y valida catálogo, familia, fuente y radio. Publica una región compatible
estimada, fecha y receptores posibles. Elimina identidad del emisor, puntos
nativos, capturas de observadores y parámetros empaquetados. No crea un marcador
visual de enemigo. La identidad percibida, el número de rivales, las coordenadas
percibidas exactas y la posición actual siguen desconocidos.

Se mantiene comunicación inmediata simulada y posible con los compañeros
presentes, incluidos muertos. La memoria conserva por jugador el último
registro individual y el último compartido por categoría audible, región y bando
estimado. Variantes nativas de material/arma no multiplican categorías que el
contexto presenta como iguales. Esta agrupación no usa identidad enemiga ni cuenta emisores. El
registro nativo mantiene las emisiones completas para auditar el cálculo.

El lector Python `replay_context_input@9` comprueba independientemente catálogo,
radios, límites temporales, vida, campos admitidos, receptores y memoria. Su
texto se construye desde referencias verificadas. El contexto para el modelo
omite distancias exactas y hashes/nombres nativos que pudieran confundirse con
conocimiento del jugador. Esta frontera no autentica por sí sola los puntos
ocultos del productor; el smoke los contrasta por separado.

Ejemplos:

- Un paso con perfil de 1100 unidades, a 1000 de distancia horizontal y 300 de
  altura, queda a unas 1044 unidades: recepción posible. A 1000 horizontal y
  500 vertical queda a unas 1118: fuera del alcance de ese perfil.
- Oír una recarga compatible con B Apps no demuestra que sea seguro salir.
  La fecha, el posible enmascaramiento y la posición actual siguen siendo
  relevantes y desconocidos cuando no existe otra evidencia.
- Tres emisiones de pasos no son tres CT. La regla experta visual de tres
  lanzamientos del mismo tipo en cinco segundos, con flashes excluidas, no se
  extiende al sonido.

La compatibilidad exacta entre esta referencia actual del juego y la build
histórica de la demo sigue sin verificar. Se conserva explícita en la entrada
y en el texto. No equivale a afirmar que el emisor estuviera físicamente en una
zona percibida con precisión ni que el receptor reconociera al emisor.

Qwen, sus presupuestos, lecturas anteriores, veinte notas y análisis originales
se conservan. Esta fase incorpora información; no prueba una mejora táctica ni
adelanta la revisión experta de interpretación general.
