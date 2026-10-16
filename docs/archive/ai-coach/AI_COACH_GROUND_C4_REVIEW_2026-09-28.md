# C4 caído en el mapa y contexto en Estado y eventos — 28/09/2026

El usuario pide marcar el C4 caído y mover el contexto anterior a la columna de
estadísticas. Se implementan ambos cambios en las 34 fichas existentes, dentro
del alcance factual y de presentación autorizado. «Ver contexto anterior» queda
inmediatamente debajo de Estado y eventos; conserva reproducción, deslizador,
retorno y corte fijo de respuesta. El mapa muestra un icono y etiqueta C4.

`objective_events.position` en bomb_drop es la posición del jugador. No basta
para situar la bomba después del lanzamiento. El productor nuevo
`review_bomb_context.py` lee los snapshots nativos de `stratai.replay_round@5`
desde artefactos ligados por bytes/SHA al manifiesto canónico. Proyecta sólo
posición x/y de C4, alias local y procedencia, mediante unión por tick exacto.
Exige lado T, ronda elegible, dropped/preplant, posición observada nativa y ausencia
de portador/planta compatibles con la ficha. Al faltar datos, recoger o plantar
se elimina el marcador. No interpola ni arrastra posiciones anteriores, no usa
coordenadas del dropper, ni ofrece ubicación global de la bomba a una ficha CT.
La implementación del SDK fijado v5.2.0 vincula CC4.OnPositionUpdate a
LastOnGroundPosition; Bomb.Position la usa cuando no hay portador.

E04 termina en tick 44635: bomba en x=1100.125, y=-971.40625. El jugador que
la soltó estaba en x=1186.3310546875, y=-552.0681762695312 en tick 43269.
Las muestras posteriores verifican el desplazamiento del objeto. El sexto bloque
`bomb-data`, contrato `stratai.review_bomb_context@1`, contiene 25.025 cortes en
34 casos y 7.064 posiciones dibujables. Los cinco bloques anteriores no cambian.
Se reutilizaron las tres capturas retenidas; no se reprocesó el corpus.

Validación: 34 pruebas Python de contexto, 43 de la interfaz pública Python,
Ruff/lint y 26 pruebas del sitio; build correcto. Una sola demo aprobada tras las
pruebas, alias `demo-0beb9110ec408f80c39b`, SHA-256
`0beb9110ec408f80c39ba19f90729773705108e21d88d0df20b60267a82f91c7`.
Parser v16, contrato canónico 3.8.0, quality 12, build
`working-tree-round-context-20260925`. Binario y fuentes Go cotejados con el build
retenido. Destino TEMP nuevo, vacío y aislado, sin sobrescribir ni promocionar.
Extracción 86.780 s, proyección 4.865 s, consumidor 0.295 s, auditoría 8.127 s.
Comandos exactos, hashes y todo el inventario temporal se conservan en evidencia.

La auditoría independiente reconstruye 8.255 cortes de diez casos: 2.445 con
posición, 5.737 sin C4 caído y 73 excluidos por elegibilidad. Comprueba 148 puntos
distintos, estados carried/dropped/planting/planted, correspondencia exacta,
procedencia, identidad de ronda/época, schema y consumidor. Los 44 manifiestos
del corpus, 106 entradas de revisión y las fuentes protegidas previas se conservan.

Playwright verifica 2131×1083, 1366×768 y 390×844, sin desbordamiento horizontal
ni errores de consola. En portátil el contexto está visible a y=233 px. Retroceder
al corte inicial retira el icono; volver al final lo restaura. Capturas revisadas.
Excel v24 incorpora únicamente IC33 en A38:G38 de Contexto colectivo, preservando
las 14 hojas, valores, fórmulas, estilos y controles anteriores.

Instrumento @9, publicación 12 del mismo sitio. Los 34 HTML alojados coinciden
por hash con el build y se conserva cada una de las 36 respuestas anteriores.
No se guardan respuestas técnicas ni expertas durante la verificación alojada.
Pestañas @8 piden recarga y conservan borrador. D1, invitación y sesiones mantienen
su contrato y claves; sólo cambia CONTENT_KEY junto al artefacto cifrado.

No cambia el detector ni se autoriza entrenamiento. Continúa la recogida de
interpretaciones del instante marcado, conservando instrumento/hash y diferencias
de información. [Evidencia](../../evidence/AI_COACH_GROUND_C4_REVIEW_2026-09-28.json).
