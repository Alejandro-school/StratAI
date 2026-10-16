# Preparación de F01-K1 v0.2

17 de septiembre de 2026. Preparación documental completada; interpretación experta pendiente. Se ejecuta el siguiente paso autorizado por el usuario: reconstruir sólo la primera ficha con datos suficientes para presentar una situación concreta. El [plan v2.46](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado y continuación.

## Entrega

- [Ficha individual F01-K1 v0.2](../../ai-coach/AI_COACH_F01_K1_REVIEW_v0.2.md), con cinco actores, recursos y una pregunta de interpretación.
- [Minimapa](../../../outputs/stratai-f01-k1-review-20260917-0a53280/F01-K1_v0.2_mapa.png), orientación horizontal y trayectorias exclusivamente de los cuatro segundos anteriores. SVG reproducible junto al PNG.
- [Evidencia](../../evidence/AI_COACH_F01_K1_REVIEW_PREPARATION_2026-09-17.json), fuentes, hashes, corte, verificaciones y límites.

No se edita el Excel: no hay respuesta humana nueva que anotar y la revisión se presenta directamente en la conversación. Se conserva su SHA-256 recibido `9DA2162C83FF0A43A33925F51135117D8926C7DBD2AD1DAD6466797BE401560C`, su estructura de nueve columnas y las 18 respuestas literales. También permanece intacto el cuaderno ficticio v0.1.

## Selección y sustitución de premisas

Se usa únicamente material retenido de la partida aprobada `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. La primera ronda evita suponer un historial rival inexistente. El instante permite localizar a los cinco T, sus recursos y movimiento antes de eventos publicados en esa ronda.

Corte: ronda 1, tick 8524, frame 8542, secuencia 1389, instancia 2, época 0. Se consulta replay y sidecars de 3E.21 ya conservados. Los contextos de 3E.15 y 3E.16 corroboran tenencia y reloj en el mismo corte. No se ejecuta ningún productor, materializador runtime ni nuevo parsing.

Las premisas reales son 0–0, 1:43, pistolas y C4 en T4. Sustituyen explícitamente 4–4, 1:30, rifles y C4 en T3 del caso ficticio. Los alias se asignan localmente por posición x al corte; no son roles y no heredan las identidades ficticias. No se trasladan las respuestas anteriores como aprobación de esta versión.

La preparación abrió archivos completos de ronda para localizar datos y examinó muestras propias posteriores al corte. No se considera selección ni revisión ciega. La selección repara el formato, no mide exactitud táctica; la ficha entregada excluye esas muestras, desenlaces e hipótesis propuestas. La respuesta humana v0.2 y su exposición efectiva siguen pendientes de registro.

## Comprobaciones reales

- Cinco actores coincidentes con replay, estados de precisión completa y observaciones support; x/y del gráfico redondeadas como en el replay. Salud, armadura, arma, inventario y dinero contrastados en el corte exacto.
- 193 frames propios entre ticks 7756 y 8524, paso 4 ticks; el gráfico utiliza sólo los últimos 65, de 8268 a 8524. Ninguna muestra futura en el artefacto de revisión.
- Ningún evento del replay hasta el corte. No se transforma esa ausencia en falta de contactos visuales o comunicación.
- Reloj independiente: `19757 × 0.015625 = 308.703125`; inicio nativo `296.703125`, duración `115`; transcurridos `12`, restantes `103`. Contexto retenido 3E.16, línea 1389: cinco actores con reloj alineado, en marcha y 103 segundos.
- C4: contexto retenido 3E.15, línea 1389, confirma T4 portador y los otros cuatro sin C4. La posición física de la bomba permanece indisponible; no se confunde con la posición del portador.
- T5 mantiene x/y durante las 17 muestras del último segundo; pitch actual 75,27° hacia abajo. Su flecha representa yaw, no una línea de visión.
- Radar derivado de posiciones y configuración de replay (`pos_x=-3230`, `pos_y=1713`, `scale=5`), con inspección visual del PNG. Un segundo agente revisa claridad, causalidad y actores; otro corrobora datos con las fuentes. Son controles técnicos, no juicios humanos de juego.
- Verificación documental de enlaces, espacios, integridad de archivos protegidos y un único prompt. Resultados exactos y scripts retenidos en la evidencia.

## Límites conservados

El radar local no acredita la compatibilidad exacta demo–assets, planta, colisiones o contacto. La ficha incluye alturas observadas y pide señalar ambigüedades de planta/acceso antes de atribuir función. Las etiquetas automáticas `Underpass`/`position.alleyac` no se usan como verdad espacial.

La fuente física distingue `raw_clip=20` y hecho exportado `ammo_in_magazine=19` para los Glock. No se diagnostica aquí un fallo del productor ni se resuelve la semántica; la ficha no usa munición exacta para valorar disparo/apoyo. Sin juicio de disparo, guardar o compra, ese dato no se sustituye por un valor supuesto.

Comunicación, receptores y contacto legal completo permanecen desconocidos. Tampoco se acredita intención mental, rol, superioridad de una acción, apoyo viable ni corrección táctica. No hay motor nuevo, entrenamiento, ampliación de schemas, promoción ni cierre S2–S3. Las 18 valoraciones de hipótesis siguen pendientes.

## Continuación y retención

Preparación cerrada; siguiente acción: recoger la primera lectura del usuario de esta ficha y su valoración de suficiencia. No preparar otro corte ni implementar antes de esa aportación, conforme al flujo acordado en AGENTS.md y el plan. Se conserva un único prompt para ese paso.

TEMP nuevo retenido: `C:/Users/kerch/AppData/Local/Temp/stratai-f01-k1-preparation-0a53280f9a29431295bb4296019d4745`. Contiene copias documentales previas, prefijo anonimizado, scripts, corroboraciones e inventario. Fuentes, corpus, bundles, temporales anteriores, Excel y CURRENT permanecen sin escritura. Las suites funcionales y un smoke nuevo no aplican: es preparación documental por lectura de salidas existentes, sin cambio de productores, contratos o runtime.
