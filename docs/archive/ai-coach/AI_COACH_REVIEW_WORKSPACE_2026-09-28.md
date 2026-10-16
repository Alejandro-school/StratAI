# Revisión junto al mapa y contexto de combate/C4 — 28 de septiembre de 2026

Se ejecuta la petición de poner el formulario a la altura del mapa, mostrar las
bajas causadas por T y los CT en intercambios, acortar las etiquetas y registrar
quién suelta el C4 y cuándo. Es una extensión factual del instrumento de revisión,
sin regla de juego, consejo o entrenamiento nuevos. IC32 del Excel v23 conserva
la petición literal y el alcance. La aclaración opcional sobre intercambios no
recibió respuesta durante el trabajo: se comunicó y adoptó daño positivo entre
equipos, sin atribuirle al usuario una confirmación inexistente.

## Resultado publicado

- @6, mismo sitio, invitaciones y sesiones. En escritorio, mapa y formulario
  comienzan juntos y el formulario queda accesible al desplazarse. En móvil,
  mapa, formulario y recursos se apilan.
- La feed muestra ambas direcciones de baja. Las 55 muertes CT del lote pueden
  mostrar su posición del Kill como cruz. Una muerte elimina la marca de contacto
  vivo; daño póstumo no la reactiva.
- Etiquetas de mapa limitadas al alias. Arma, antigüedad y baja se consultan fuera
  del mapa por clic, toque o teclado. No se dibujan frases largas sobre el radar.
- Quinto suplemento: 448 eventos `player_hurt` con daño positivo, 32 drops y
  27 recogidas de C4. Se excluyen cuatro callbacks sin daño positivo. Seis drops
  coinciden con muerte en el tick. «C4 soltado por Tx a mm:ss» no atribuye intención.
  La falta de portador no crea un evento. Los disparos sin daño no revelan CT.

## Evidencia

23 pruebas web, 22 pruebas Python de productores/contrato y 43 del backend,
Ruff y build correctos. Se mantienen los tres campos y el vínculo persistente
caso/episodio/contexto/hash. Verificación por HTTP de los 34 HTML servidos contra
el build, disposición estructural, etiquetas, datos, acceso y guardado. Las
32 versiones previas mantienen su hash; sólo se crean pruebas técnicas 33 y 34.
No se realizó inspección visual del sitio en navegador.

Tras las pruebas se procesa exactamente una demo aprobada:
`demo-0beb9110ec408f80c39b`, SHA-256
`0beb9110ec408f80c39ba19f90729773705108e21d88d0df20b60267a82f91c7`.
Fuente y 44 bundles aprobados de sólo lectura, salida temporal aislada. Productor
101,876 s, materializador 2,461 s, consumidor 0,565 s y auditoría independiente
18,819 s. Se concilian 155 eventos de daño (69 de T y 86 de CT), 10 drops, cinco
recogidas, una caída en el tick de muerte, 20 bajas CT y 8.255 cortes.
No hubo segunda partida ni repetición del parser. Los controles rechazan destinos
no temporales, no vacíos o sobrescrituras. La salida y el inventario se retienen.

Los 160 archivos protegidos y 44 manifests originales no cambian. La comprobación
reconstruye por separado identidad/vida, posiciones, arma, daño, eventos del C4,
disponibilidad, relojes, antigüedad y muerte del rival. Alias opacos en evidencia.
Excel v23 conserva las 14 hojas y sólo añade A37:G37 de «Contexto colectivo».

El parser Go, el corpus, el detector y sus predicciones no cambian. El detector
no recibe estos suplementos; al contrastar lecturas hay que separar información
adicional de errores con entradas equivalentes. Sigue pendiente la evaluación
humana, sin entrenamiento automático.

[Evidencia y comandos](../../evidence/AI_COACH_REVIEW_WORKSPACE_2026-09-28.json).
