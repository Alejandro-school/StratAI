# Corrección de la distribución de las fichas — 28/09/2026

El usuario pide tres columnas alineadas y señala que no ve el drop de T4 en E04.
La captura aportada ya contiene «C4 soltado por T4 a 00:02», debajo del mapa.
El defecto es de visibilidad y jerarquía, no de extracción. Se corrige dentro del
alcance de interfaz acordado; no abre una fase táctica ni necesita otra revisión
experta o fila del Excel. IC32 y el libro v23 se mantienen.

El instrumento @7 sitúa mapa, estado/eventos y formulario en la misma fila desde
1100 px. Las columnas largas tienen desplazamiento independiente. En pantallas
estrechas se adapta a dos columnas y, hasta 700 px, se apila mapa, información y
formulario. Se compactan títulos y ayuda; los tres campos siguen siendo alias,
interpretación e información que falta. El enlace de cabecera salta al formulario.

El bloque central destaca el último evento C4 disponible, conserva el estado
actual y permite desplegar el historial. E04 muestra el drop de T4 a 00:02. No se
infiere que siga en el suelo a partir de ese evento ni de la ausencia de portador.
Recuentos de ambos equipos, tiempo restante, recursos y bajas se reúnen en esa
columna. Los detalles de marcas CT se mantienen fuera del radar. Se corrige también
el texto heredado que decía que no se mostraban posiciones rivales.

Build y 23 pruebas existentes correctos. Los cinco bloques factuales por ficha
se conservan exactamente; 106 entradas protegidas mantienen sus hashes. No se
modifican productores, contratos de datos ni el detector: no aplica repetir el
smoke de una demo para esta corrección visual. El smoke de @6 queda como evidencia
histórica de esos datos, sin atribuir una nueva ejecución a @7.

Playwright local: inspección de capturas a 1440×1000, 1366×768 y 390×844; tres
columnas a y=182,296875 px en escritorio, C4 y botón de envío visibles, sin
desbordamiento horizontal. Se comprueban clic en marca CT, historial C4, bajas,
salto a un instante anterior y retorno al corte fijo; sin errores de consola.
En móvil se inspeccionan mapa y formulario, con enlace directo a la respuesta.

Publicado como versión 9 del mismo sitio. Comprobación HTTP 34/34 contra hashes
del build; invitación sin cuenta y aislamiento conservados. Las 34 versiones
anteriores mantienen su hash; se añaden sólo las pruebas técnicas 35 y 36, ocultas
por defecto y excluidas de evaluación. Respuestas históricas mantienen instrumento
y hash; pestañas anteriores piden recarga antes de enviar y recuperan su borrador.
No se modifica el esquema D1 ni las claves de invitación, organizador o sesión.

Sigue pendiente recoger lecturas humanas y separar diferencias de información
respecto al detector congelado. No hay entrenamiento automático.

[Evidencia](../../evidence/AI_COACH_THREE_COLUMN_REVIEW_2026-09-28.json).
