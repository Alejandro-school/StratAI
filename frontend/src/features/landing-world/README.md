# Landing oficial — Mirage

Landing oficial de StratAI en la ruta pública `/`, implementada con React 18, React Three Fiber, Drei y Three.js existentes. `frontend/index.html` carga la aplicación y `App.jsx` importa este módulo de forma diferida mediante `index.js`, sustituyendo el landing anterior. `frontend/landing-world.html` se conserva como entrada independiente secundaria. [SURFACE.md](SURFACE.md) conserva la dirección y [EXPLORATION.md](EXPLORATION.md) documenta la procedencia visual. La sustitución en el código no implica un despliegue remoto.

## Ver la experiencia

Desde la raíz:

```powershell
npm --prefix frontend run dev -- --host 127.0.0.1
```

Abrir `http://127.0.0.1:3000/`. La vista aérea presenta «Tu partida, desde dentro.»; «Estoy listo» inicia un vuelo de 3,4 segundos hasta un encuadre abierto de A que incluye las cajas de CT. Al llegar queda disponible una cámara amortiguada y limitada al punto. La rueda actúa sobre la cámara y los paneles conservan su scroll nativo; no hay recorrido por scroll del documento.

Durante la carga, «Conectar Steam» conserva el enlace de autenticación y su transición existente, y «Ver funciones mientras carga» permite consultar el producto sin esperar al 3D. El escenario y la interfaz subyacente permanecen inertes hasta que están disponibles o se elige ese acceso; los controles de carga también quedan inertes durante el traspaso a Steam. Los errores de carga ofrecen reintento. El pie muestra «ANALIZA TU PARTIDA».

- Arrastrar desplaza; arrastrar con botón derecho o Mayús gira.
- Rueda y pellizco acercan o alejan. Con el escenario enfocado, las flechas desplazan, Mayús + flechas giran, +/− ajustan el zoom y R restablece.
- Los botones de cámara ofrecen acercar, alejar, restablecer A y volver al aire. El botón de la cabecera y la marca también controlan entrada o regreso.
- Los cinco puntos y el menú inferior abren replay, mapa, estadísticas, Coach y retos. Al cambiar de herramienta, la cámara parte enseguida hacia el destino y el panel saliente deja paso al nuevo. La última selección prevalece; pulsar fuera, Escape o cerrar cancela cualquier cambio pendiente y recupera la exploración anterior y el foco del control que abrió el panel. Los controles interiores siguen funcionando y seleccionar la misma herramienta conserva su replay.

Los límites absolutos de la cámara son X 0–4,8 y Z 22,5–26,8; distancia 19–26, elevación original −0,05 a +0,08 radianes, giro original −0,2 a +0,06 radianes y zoom óptico 0,8–2,1. Desplazamiento, giro, elevación y distancia adicional comparten una envolvente: sus extremos no se combinan y el recorrido efectivo es menor. La distancia parte de `17 / sin(elevación)` y tiene un radio adicional de 2,5 para mantener la altura mínima de 17 unidades. La comprobación de 972 combinaciones, incluidos los desplazamientos del cursor, no encontró cámaras debajo de tejados opacos con un margen de 0,6 unidades; es una comprobación de muestras, no un motor general de colisiones. Los puntos del mapa reservan una zona completa para las instrucciones y controles, también en pantallas estrechas.

El T está en escaleras y el CT en punto, con altura normalizada de 1,38 unidades y poses estáticas muestreadas de los clips existentes. El movimiento reducido ofrece viaje y ajustes inmediatos, mantiene los controles y simplifica los efectos decorativos.

El acceso normal a Steam en la misma pestaña inicia una salida de 1,1 segundos desde la cámara actual hacia el CT: el escenario se acerca mientras la interfaz desaparece y un velo oscuro presenta «StratAI» y «Conectando con Steam». Los puntos del escenario se ocultan durante la salida. Después se navega una sola vez al mismo enlace de autenticación existente. Escape cancela antes del traspaso y devuelve el foco al enlace; volver con el navegador restablece la experiencia. Si el navegador bloquea la navegación, quedan «Continuar a Steam» y «Volver a Mirage». Los clics con modificadores, el botón central y la apertura en otra pestaña mantienen su comportamiento nativo. Con movimiento reducido, 3D no disponible o un vuelo normal activo, se usa solo un fundido de 180 ms y la cámara permanece quieta.

Las herramientas coordinan una salida del panel de 120 ms, una entrada con perspectiva y opacidad de 360 ms y un arco de cámara de 560 ms. Conservan los destinos y el encuadre final de Mapa. Con movimiento reducido, el panel y su cámara cambian inmediatamente. Se reutilizan la cámara y el bucle bajo demanda existentes; no se añaden otro renderizador, recursos ni dependencias.

`landingContent.js` consume los textos trasladados sin cambios a [copy.es.json](copy.es.json); [copy.en.json](copy.en.json) conserva la copia inglesa y [replayDemoData.js](replayDemoData.js) el guion original. Los antiguos árboles `components/Landing` y `styles/Landing` se retiraron después de conservar una [instantánea acotada](../../../../output/landing-official-20261016/before-migration.zip). Los medios y modelos 3D permanecen. Los paneles amplios conservan sus capturas originales con ampliación nativa; pueden ocupar hasta `min(55svh, 620px)` en escritorio. La cabecera no lleva línea inferior. Cómo funciona, planes, facturación anual/mensual y siete FAQ permanecen accesibles desde ella. Steam utiliza el endpoint existente; no se han implementado cobros.

`index.js` importa los estilos en orden: `world.css`, `content.css`, `exploration.css`, `productPresentation.css` y `transitions.css`. Los selectores, reinicios y reglas de movimiento reducido se limitan a `.landing-world`, cuya raíz ocupa la pantalla sin bloquear globalmente el scroll de `html` o `body`. Así no afectan a otras rutas cuando el landing se desmonta. El tamaño de las etiquetas del replay queda fijado en 10,4 px para conservar la apariencia previa con la raíz móvil de 14 px de la aplicación. `frontend/index.html` añade Inter y Rajdhani a las fuentes existentes.

La raíz pública sigue sin comprobar sesión. El callback `SteamLoginSuccess` conserva el paso hacia `/history-code` o `/dashboard`, y `RequireAuth` mantiene la protección de las rutas privadas. La migración no cambia el backend, las sesiones ni los contratos del pipeline completo de demos.

Replay abre un panel ancho con el vídeo original de Dust2 y el Coach lado a lado en escritorio. Al abrirlo intenta reproducirlo silenciado y en línea. Cuando la actualización de tiempo alcanza cada momento de análisis, pausa y busca exactamente 1,3 o 6 segundos para congelar el fotograma calibrado. Elegir un capítulo utiliza la misma operación. Los comentarios originales aparecen progresivamente y la jugada continúa automáticamente tras sus tiempos de lectura. Los controles permiten pausar, continuar antes, reiniciar o elegir cualquiera de los dos momentos. Si el navegador rechaza la reproducción automática, queda disponible el botón de reproducir. Al terminar aparecen el resumen y las tres preguntas con respuestas preparadas; no son una conversación generada. Cerrar o reiniciar cancela los temporizadores pendientes y detiene la reproducción anterior.

El vídeo conserva el encuadre 16:9 para mantener la alineación de las anotaciones. El círculo rojo junto a «TÚ» y el amarillo del compañero mantienen sus etiquetas y pulso original definido en CSS. Sus coordenadas fijas se han recalibrado con los centros de las bolas azules del vídeo nativo de 1920 × 1080: a 1,3 s, corta usa (969, 652) y B (323, 480); a 6 s, corta usa (968, 625) y B (301, 641). Las [mediciones del vídeo](../../../../output/playwright/landing-world-replay-20261016/alignment/source-ball-measurements.json) conservan los centroides de referencia. Movimiento reducido conserva los círculos y etiquetas sin el pulso. En pantallas de hasta 760 px, la etiqueta del compañero parte del borde izquierdo del anclaje para caber dentro del vídeo; en escritorio sigue centrada. Son anotaciones de dos pausas guionizadas; no acreditan seguimiento continuo ni identidad durante el juego. En móvil, vídeo y análisis se apilan dentro del scroll normal del panel; el vídeo no queda fijo sobre los comentarios. La captura estática y la lista de funciones originales se conservan en «Captura original y funciones del replay». Movimiento reducido muestra el texto completo sin efecto de escritura y conserva las pausas y los controles. Esta demo sigue identificada como guionizada y el Coach continúa en desarrollo y evaluación.

## Comprobación

```powershell
npm run test:frontend
npm run build:frontend
```

Los resultados finales tras el ajuste de carga y la verificación de API/navegador se registran en [REVIEW.md](REVIEW.md) y en [la evidencia de la migración oficial](../../../../output/landing-official-20261016/). Esta documentación describe el código y no declara resultados finales pendientes, un despliegue remoto, una autenticación externa real de Steam ni pruebas de rendimiento en hardware modesto.

La entrada independiente secundaria sigue disponible con `npm --prefix frontend run build -- --config landing-world.vite.config.js`; escribe en `output/landing-world-build-20261015` y no sustituye `frontend/build`. Los resultados siguientes conservan la historia de la etapa independiente anterior a la migración oficial.

La [revisión de exploración](../../../../output/landing-world-exploration-20261015/) conserva la suite completa de 65 archivos y 402 pruebas anterior al ajuste acotado de cámara y etiquetas, seguida de 10 archivos y 46 pruebas enfocadas. El [refinamiento anterior de replay del 16 de octubre](../../../../output/landing-world-replay-20261016/) registra 11 archivos y 54 pruebas enfocadas aprobadas, más la compilación raíz y la independiente históricas. Aquella entrada independiente ocupa 853,10 kB (232,37 kB gzip).

La [restauración anterior del marcador](../../../../output/landing-world-replay-20261016/player-marker-tests.log), que incluye la regresión de cierre al pulsar fuera, pasa 11 archivos y 55 pruebas enfocadas. Su [validación retenida](../../../../output/landing-world-replay-20261016/player-marker-validation.json) conserva la compilación independiente aprobada: entrada de 853,39 kB (232,43 kB gzip) y CSS de 41,26 kB.

La [alineación anterior](../../../../output/landing-world-replay-20261016/alignment-tests.log) pasa 11 archivos y 55 pruebas enfocadas, con dos aserciones ampliadas para comprobar la pausa exacta. Su [compilación independiente](../../../../output/landing-world-replay-20261016/alignment-final-build.log) pasa con entrada de 853,40 kB (232,43 kB gzip) y CSS de 41,49 kB. Se mantiene el umbral consultivo de 800 kB. La compilación raíz no se repitió para aquel ajuste de alineación.

El [pase anterior de movimiento](../../../../output/landing-world-motion-20261016/focused-final.log) pasa 13 archivos y 68 pruebas enfocadas. Su [compilación raíz](../../../../output/landing-world-motion-20261016/root-build.log) y la [independiente final](../../../../output/landing-world-motion-20261016/preview-build-final.log) están aprobadas; aquella entrada independiente ocupa 857,34 kB (233,51 kB gzip) y el CSS 43,83 kB. Conserva el aviso superior a 800 kB. Su [única ejecución del detector](../../../../output/landing-world-motion-20261016/detector.json) conserva 52 avisos consultivos —39 de tipografía y 13 de color— más el aviso previo de Inter. La tipografía de conexión y el movimiento son extensiones acotadas; no se añadieron supresiones ni se cambiaron umbrales.

La [prueba de navegador del movimiento](../../../../output/landing-world-motion-20261016/browser-evidence.json) registra las peticiones normal y reducida al mismo destino de autenticación, la cancelación móvil antes de la petición y el restablecimiento al volver, sin errores registrados. La URL de autenticación se intercepta y responde con una página local de prueba para verificar el destino y el regreso; no se autenticó una sesión real de Steam. Las capturas finales, los hashes y la revisión visual independiente quedan en [REVIEW.md](REVIEW.md).

La [prueba histórica en Chrome](../../../../output/playwright/landing-world-replay-20261016/browser-playback-evidence.json) registra las antiguas pausas en 1,485868 y 6,185971 segundos, tiempo detenido durante los comentarios, continuación automática, fin a los 10,9 segundos y ningún error registrado. Esas pausas tardías corresponden a la implementación anterior y quedan sustituidas por la búsqueda exacta a 1,3 y 6 segundos; sus medidas de disposición también preceden al ajuste final de altura. La revisión visual, las capturas y la validación actual de hashes se registran aparte en [REVIEW.md](REVIEW.md). Esta documentación procede del código y de las evidencias retenidas; no constituye una comprobación propia del navegador ni afirma un resultado final de alineación. El detector anterior conserva sus 38 avisos consultivos. La [ejecución única de la restauración anterior](../../../../output/landing-world-replay-20261016/player-marker-detector.json) conserva 62 avisos consultivos y el aviso previo de Inter; los colores rojo/amarillo y el redondeo de sus etiquetas recuperan el diseño original solicitado. No se han añadido supresiones, cambiado umbrales ni repetido el detector para sustituir esos resultados.

Para ver la compilación independiente secundaria sin el servidor de desarrollo:

```powershell
npm --prefix frontend run preview -- --config landing-world.vite.config.js --host 127.0.0.1 --port 3315
```

Abrir `http://127.0.0.1:3315/landing-world.html`.

## Límites del contenido

El mapa usa geometría, UV y texturas originales de Mirage extraídas del VPK local de CS2 con Source 2 Viewer 20.1. El modelo anterior no tenía texturas ni UV. `scripts/landing-world/prepare_textured_map.py` prepara el nuevo export sin modificar la fuente; los comandos y hashes quedan en `output/landing-world-assets-20261015/PROVENANCE.json`.

La escena carga `mirage-exploration.glb` (71.413.376 bytes), preparado desde el export nativo sin comprimir retenido. `site_high` conserva los 277.445 triángulos de A y sus atributos de posición, normal, tangente y UV, con caras y orientación verificadas de forma independiente. Meshopt comprime A sin pérdida. El nivel distante de A tiene 97.467 triángulos y el fondo 376.044; comparten las 500 imágenes KTX2 con mipmaps, hasta 1024 px cerca de A y 256 px para imágenes exclusivas del fondo. La geometría completa entra a 68 unidades y se mantiene hasta 78; permanece activa durante la exploración cercana. Solo los atributos distantes reducen precisión.

Las imágenes y máscaras alfa nativas permanecen. Se excluye `COLOR_0` de viento/variación de la vegetación para evitar que el material PBR tiña las hojas de rojo/negro. El render utiliza iluminación propia y no reproduce todos los shaders, mezclas y lightmaps del juego. El decodificador KTX2 conserva la CSP existente: 500 imágenes, 14.163 comparaciones de mip/formato y 359.024.480 bytes decodificados coinciden con el codec original. Meshopt utiliza el decodificador que ya aporta Drei; no se añaden dependencias de aplicación.

La instantánea local actual estima 190,53 MB de texturas de la escena frente a 558,10 MB del recurso base, y 69,12 MB de geometría frente a 97,54 MB. Son estimaciones de formatos y buffers, no VRAM total del controlador ni medidas de FPS. La resolución puede bajar durante movimiento sostenido lento; al detenerse recupera el DPR nítido inicial, limitado a 1,5, y deja de renderizar. El DPR de movimiento medido se reutiliza en la siguiente interacción.

Recuperar la geometría nativa de A y texturas cercanas más detalladas aumenta la descarga de 31,8 a 71,4 MB, dentro del presupuesto de 75 MB fijado durante la etapa de previsualización. Es un intercambio explícito de red por fidelidad. [EXPLORATION.md](EXPLORATION.md) y [PROVENANCE.json](../../../../output/landing-world-exploration-20261015/PROVENANCE.json) documentan el recurso actual, sus hashes, comandos y comprobaciones. [PERFORMANCE.md](PERFORMANCE.md) se conserva como evidencia histórica del pase simplificado anterior. Quedan pruebas en hardware modesto real, formatos de compresión alternativos y redes lentas; no se afirma una carga inicial más rápida ni FPS universales.

No se cargan partidas personales ni se procesa una demo. El vídeo y los comentarios son una demostración guionizada del landing previo; el arco y humo de Mirage son ilustrativos. Coach IA y retos se presentan como propuesta. Los contadores comerciales sin evidencia no se han trasladado al hero. La cámara y los anclajes son exclusivamente de presentación, no transformaciones del contrato factual.

[ATTRIBUTION.md](../../../public/landing-world/ATTRIBUTION.md) registra la procedencia del modelo y el historial de transformaciones. El VPK, los mapas fuente, agentes y medios originales permanecen en solo lectura. Esta entrega sustituye el landing oficial en la raíz de la aplicación sin desplegarlo remotamente; no cambia el plan canónico del AI Coach ni acredita nuevas capacidades.
