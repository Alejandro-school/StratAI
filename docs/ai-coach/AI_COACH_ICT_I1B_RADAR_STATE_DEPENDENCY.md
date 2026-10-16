# ICT-I1.B — vínculo pendiente entre spotting y radar mostrado

**Alcance corregido por el usuario el 22/09:** trabajar desde la replay real.
Se retiran la solicitud de vídeos de jugadores, la captura web y las pruebas
controladas destinadas a generar nuevas grabaciones. Los requisitos factuales
de esta nota limitan las afirmaciones sobre el HUD; no bloquean todo el análisis
de la replay ni habilitan fuentes rechazadas. [Criterio literal](../evidence/AI_COACH_REPLAY_ONLY_SCOPE_CORRECTION_2026-09-22.json).

La sonda del 22/09/2026 demuestra que la demo conocida contiene estado nativo
de spotting. No demuestra qué representación de radar recibió cada jugador.
El [plan](../AI_COACH_IMPLEMENTATION_PLAN.md) fija el estado y la siguiente acción.

## Lo que ya puede comprobarse

En la grabación `demo-a11eb0d600ee702c9edb`, build 10896 y patch 14177,
`CCSPlayerPawn` declara `m_bSpotted` como `bool` y `m_bSpottedByMask` como
`uint32[2]`. Se observaron las rutas directas, no las tres rutas anidadas
consultadas. Sus cambios y bits son comprobables y conservan entidad, serie,
orden de entrega y reloj del SDK. Los slots derivados siguen siendo candidatos
según la fórmula del SDK; no son una lista acreditada de receptores de HUD.

`CSVCMsg_ServerInfo.is_hltv` es verdadero y `player_slot` es 10. No apareció
`ProcessSpottedEntityUpdate` en los dos caminos instrumentados. Esto no prueba
que una grabación de espectador carezca de toda información de radar; delimita
el resultado de esta sonda y versión del decoder.

## Evidencia necesaria para distinguir punto e interrogante

Hace falta un contraste independiente que relacione, para esta versión:

1. El contacto que origina la información y el observador correspondiente.
2. El jugador cuyo radar recibe esa información.
3. La representación de contacto actual, la de contacto perdido y su retirada.
4. El reloj del contraste y los fotogramas de la demo, con su incertidumbre.
5. La ubicación comunicada si se pretende usar una zona táctica. La posición
   global oculta del rival no puede rellenarla.

Una fuente auxiliar admisible puede ser la lógica de cliente de esa versión
con procedencia verificable, vinculada a la replay existente. Se retira la vía
de capturas personales del jugador. La reproducción desde espectador sólo sirve
si antes se acredita que su radar reproduce la perspectiva del jugador elegido.
Un punto o interrogante dibujado por una herramienta nuestra sería circular.

El contraste debe cubrir aparición, pérdida y retirada, además de varios
observadores, reaparición del mismo rival, muerte/cambio de lado y discontinuidad
del reloj. Los ejemplos se eligen antes de mirar el resultado visual. Un caso
concordante no acredita todas las condiciones ni una duración universal.

## Criterios de aceptación y límites

- Coincidencia independiente de identidad, perspectiva y tiempo. Las diferencias
  se registran; no se ajustan offsets o segundos para forzar concordancia.
- No deducir estado visual sólo por el nombre `m_bSpotted`, por cambiar un bit
  o por la advertencia LOS/FOV de la biblioteca.
- Un `false` o una máscara cero no separan por sí solos «interrogante todavía
  visible» y «marcador retirado». Esos estados permanecen desconocidos.
- La explicación experta sigue vigente: el radar deriva del contacto visual.
  «Unos segundos» no fija un TTL. No hay una pregunta nueva de juego que resolver.
- Si sólo se obtiene evidencia de estado nativo, conservarla como diagnóstico.
  No activar E01, recomendaciones, etiquetas automáticas o entrenamiento.

No se requiere otra demo para confirmar de nuevo la misma carencia. Cualquier
nueva reproducción debe estar dentro de la autorización y capacidades
disponibles. Este documento no autoriza descargas, adquisición, publicación,
reproceso del corpus ni modificación de releases.

## Preparación realizada el 22/09

Se han [preparado seis cortes](../../outputs/stratai-radar-hud-binding-20260922/Cortes_para_contrastar_el_radar.md)
sin ejecutar el contraste visual. La instalación local se ha actualizado a
1.41.8.1; no está vinculada a la demo. El layout histórico inspeccionado contiene
EnemyOnMap/EnemyGhost, pero no acredita la lógica que los selecciona ni el
receptor. [Resultado y límites de la consulta](../archive/ai-coach/AI_COACH_RADAR_HUD_CONTRAST_PREPARATION_2026-09-22.md).
La petición de una grabación del jugador queda retirada por la corrección de
alcance. Los seis cortes siguen siendo referencias de la replay; no generan
nuevas muestras de juego ni prueban estados del HUD. No repetir estas
comprobaciones sin material nuevo.
