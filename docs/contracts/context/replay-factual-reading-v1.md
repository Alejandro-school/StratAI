# Lectura factual del replay con referencias fechadas

Extensión optativa de desarrollo: entrada `stratai.replay_context_input@6`,
proyección `stratai.team_replay_projection@8`, ronda
`stratai.replay_round_context@2`, inventario `stratai.replay_inventory@1`,
vida por callback `stratai.replay_life_context@1` y salida
`stratai.replay_factual_reading@1`. Se mantienen el bundle 3.8, el registro
canónico de 116 schemas y las versiones históricas de este instrumento.
El plan canónico determina estado, autorización y siguiente fase.

## Frontera entre datos y texto

El modelo devuelve referencias para situación, evolución e incertidumbre; para
interpretación selecciona únicamente un principio del banco existente y sus
referencias. No suministra texto, sujetos, valores ni grados de certeza.
La aplicación redacta cada afirmación mediante una descripción cerrada y conserva
el hecho exacto, con ID, sujeto, instante, valor y estado. El verificador vuelve
a construir el texto y exige igualdad completa: una cita válida no basta.

Los hechos actuales esenciales, el estado inicial de ronda/objetivo y sus
acontecimientos se incluyen aunque el modelo los omita. El inicio S0 entregado
es citable sin consulta redundante. Las muestras intermedias exigen consulta
causal dentro de los límites existentes. Una referencia actual no sustituye
a una histórica; un dato conocido no puede declararse desconocido.

Los criterios tácticos son condicionales. Seleccionarlos no acredita su premisa,
su pertinencia ni una conclusión táctica. El indicador
`factual_integrity=checked_reference_rendering` acredita fidelidad de la
representación a los hechos admitidos. `semantic_truth` y `tactical_quality`
siguen `not_evaluated`: no declara que la demo, las estimaciones o la percepción
humana hayan sido verificadas por un oráculo externo.

## C4, vida, inventarios y fechas

El portador actual procede exclusivamente del objetivo y de su conciliación
con un único jugador vivo. C4 en una lista histórica de equipo no acredita
posesión actual; no se repite como portador en el texto del inventario.
Soltar C4, empezar un intento de plantado y completar el plantado conservan
tipos distintos. Resuelto no determina por sí solo desactivación o explosión.

Go observa el inventario nativo antes de aplicar la caché visual. Una observación
actual lleva tick, disponibilidad, fuente, arma activa, lista y casco. Al morir
o faltar una nueva observación válida, se conserva el último registro como
`last_observed`, con sus valores y fecha originales. Sin observación previa
se entrega `unavailable`, con valores nulos/lista vacía. Los nombres pertenecen
al vocabulario `Equipment.String` del parser fijado demoinfocs v5.2.0; no son
un campo de prosa libre.

El inventario histórico se muestra como límite de vigencia actual, con su fecha
y condición histórica explícitas. No afirma equipo utilizable por un muerto.
Si un kill público llega entre dos muestras, la vida se actualiza mediante
`life_context` ligado a ese acontecimiento, mientras posición, recursos e
inventario conservan sus fechas. Python separa `own_life` de `own_player`.
Retroceder el replay no modifica los registros ni anticipa la muerte.

Se rechazan IDs duplicados, más de cinco aliados, valores propios incoherentes,
salud no entera/fuera de 0–100, recursos negativos, fechas/procedencias inválidas
y posesiones contradictorias. El daño reportado y la salud reportada se conservan
por separado; no se sustituye la salud nativa por una resta.

## Zonas, ronda y acontecimientos

Se conserva la pertenencia a volúmenes 3D locales y su incertidumbre de build:
una zona estimada no se convierte en observación confirmada. Los proyectiles
simulados no reciben nombres de zonas atravesadas sin un hecho que lo acredite.
Lanzamiento, inicio y final de efecto tampoco se intercambian.

Las cuentas atrás observadas se recalculan desde los mensajes nativos, intervalo,
origen presenciado al fin de freeze, duración, pausa y plazo de bomba. Una
cuenta plausible pero distinta del cálculo también se rechaza. La evidencia
proyectada tiene campos cerrados; quedan fuera progreso de defuse privado y
cualquier estructura ajena al reloj público. Un fin futuro no revela ganador.
Una ronda cerrada no puede reabrirse ni cambiar de resultado en su prefijo.

Entre muestras no se interpola una cuenta atrás. El estado conserva fecha de
observación, y la fase se concilia con el objetivo cuando comparten instante.
Las posiciones de acontecimientos deben corresponder a su callback o a su
observación de contacto; no pueden traer una fecha posterior.
En daño recibido, las coordenadas corresponden al jugador herido en el callback;
la observación del atacante conserva su fecha independiente. Un fin de fuego
sin coordenadas nativas conserva el acontecimiento, sin inventar posición ni
fecha de posición. Las proyecciones históricas mantienen sus campos originales.

Contactos calculados permanecen estimados. Trayectorias/efectos propios y puntos
de muerte comunicados permanecen simulados. No se acreditan intención, apoyo
eficaz, trayectoria futura, audición real ni recursos rivales ocultos.

## Historia y revisión

Las entradas @1–@5 mantienen vistas, recibos y reproducción offline. El adaptador
rechaza nuevas generaciones de replay por esas versiones anteriores antes de
contactar con el modelo; la generación protegida exige @6. Los originales y sus
fallos se conservan, sin regeneraciones selectivas.

El visor distingue hechos registrados, límites/estimaciones y criterios para
revisar. F1 es una lectura determinista sin llamada a modelo; C-F1, cuando existe,
añade la selección local del modelo bajo el contrato cerrado. Las variantes
históricas mantienen texto, códigos, referencias y vinculaciones de notas.

El cierre exige suites pertinentes y exactamente una demo aprobada en TEMP
aislado, vacío y retenido después de pruebas, con contraste de causalidad,
inventario, relojes, geometría y acontecimientos. Los intentos fallidos se
conservan; repetir un smoke sólo puede usar la misma partida. La revisión
experta de nuevas fases y la evaluación independiente siguen siendo gates
distintos. Este contrato no habilita entrenamiento ni promoción.

