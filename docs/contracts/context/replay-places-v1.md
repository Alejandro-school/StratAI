# Zonas 3D para el replay de desarrollo

Versión 1, 09/10/2026. Instrumento optativo: paquete
`stratai.replay_place_volumes@1`, resultado `stratai.replay_place@1`, proyección
`stratai.team_replay_projection@5` e input `stratai.replay_context_input@3`.
El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene estado y orden.
No amplía bundle 3.8, replay base 5, catálogo 1.7.0 ni los 116 schemas runtime.

## Fuente y transformaciones

Se reutiliza `backend/data/maps/de_mirage/de_mirage_texto.vmap`, DMX keyvalues2 4,
VMAP 40. El extractor enlaza las 23 entidades `env_cs_place` con sus mallas por
ID, recorre los ciclos de half-edges y conserva vértices y caras. No usa el
JSON de puntos como volumen ni presume nombres NAV que el parser no expone.
`position.*` no puede convertirse en zona jugable. El paquete contiene 23
volúmenes con nombre; no hay entidades place internas excluidas en esta fuente.

En la representación auditada las entidades son hijas directas de un mundo
con transformación identidad. Las mallas y su meshData también deben tener
transformación identidad, y se rechazan rotaciones/escalas de entidad no
soportadas. Los orígenes de entidad se conservan como procedencia, sin volver
a sumarlos a vértices que ya están en coordenadas mundiales. Referencia primaria:
[MapExtract de ValveResourceFormat, commit 82e3fd6](https://github.com/ValveResourceFormat/ValveResourceFormat/blob/82e3fd6a37f409427ec70429e7d8cd1f900aa76c/ValveResourceFormat/IO/Extract/MapExtract.cs)
aplica worldTransform al reconstruir el hull, mediante
[HammerMeshBuilder](https://github.com/ValveResourceFormat/ValveResourceFormat/blob/82e3fd6a37f409427ec70429e7d8cd1f900aa76c/ValveResourceFormat/IO/Hammer/HammerMeshBuilder.cs).
Se trata de geometría recuperada del recurso local, no de prueba del fichero
de autoría original ni de coincidencia exacta con el build de la demo.

Se rechazan caras degeneradas/no planas, mallas abiertas/no manifold,
poliedros no convexos, índices/ciclos inválidos y jerarquías no soportadas.
El paquete conserva SHA-256/tamaño de fuente, IDs, origen, vértices, caras,
planos normalizados y límites. `reference_id` sella el JSON canónico del
paquete sin ese campo. El lector valida tanto el hash como planos y topología.

## Significado del resultado

La prueba usa todos los planos del volumen real, incluyendo altura Z.
Las cajas envolventes sólo describen límites, no determinan pertenencia.
La tolerancia numérica es 0,0001 unidades mundiales. Para snapshots propios
con XY serializadas como enteros se añade un intervalo de ±0,5 por eje;
contactos calculados conservan XY flotantes. No se fabrica Z=0 en @5 cuando
la fuente permitida carece de altura.

| Estado | Significado |
|---|---|
| `mapped` | Interior inequívoco de una única zona con nombre en la referencia local |
| `boundary` | El punto o su intervalo de precisión toca una frontera |
| `ambiguous` | Compatible con volúmenes de nombres distintos |
| `unavailable` | No hay volumen, referencia compatible o posición XYZ válida |

Sólo `mapped` emite `zone`; los demás conservan `zone=null`, candidatos y motivo.
Varios hulls con el mismo nombre no constituyen por sí solos un conflicto.
Todos los resultados llevan `build_compatibility=unverified`: la pertenencia
a la referencia local es una estimación, nunca observación nativa de zona.
El visor rotula los casos únicos «mapa de referencia».

Ejemplos comprobados desde caras y vértices:

- (-1500, 750, 0): Apartments, apartamentos B; a Z=100 queda fuera.
- (-400, -400, -100): Catwalk, corta B.
- (-1500, 500, -50): solape Apartments/BombsiteB, sin elegir una zona.
- (-2000, 450, 30): dentro de la caja de Apartments pero fuera de su cara oblicua.
- (-128, 600, -50): frontera de BackAlley, sin zona única.

## Frontera causal y versiones

La proyección calcula zonas después de filtrar por bando. Sólo utiliza la
posición propia o el contacto permitido, con su observation_tick y
availability_tick. No consulta el snapshot omnisciente de un rival para resolver
su zona. Elimina nombres place recibidos en bruto y no asigna una zona viva a
cadáveres. No atribuye zonas a eventos de muerte sin Z ni a posiciones históricas
de C4. Situación, rumbo, ruta, línea de visión e intención siguen siendo conceptos
distintos.

@5 se activa al aportar metadata `map_name=de_mirage`. Los consumidores sin
esa referencia mantienen @4. El adaptador Python @3 exige mapa y referencia,
recalcula cada resultado desde posición/precisión y valida fechas antes de
emitir un hecho `place_context` separado. `mapped` tiene status `estimated`;
frontera/solape/hueco tienen status `unknown`. Los inputs @1 y @2 conservan
semántica y recibos exactos en las diez ventanas históricas.

Go y sus heurísticas generales de callout no cambian en este alcance. El módulo
de puntos `spatial_reference.py` y los consumidores canónicos/support mantienen
su contrato anterior. No se acredita percepción exacta, ruta viable, decisión
táctica, entrenamiento ni elegibilidad por añadir una zona al replay.

## Verificación y uso en comparación

El [cierre](../../../outputs/stratai-zones-20261009/RESULTADO.txt) conserva pruebas,
smoke, geometría regenerada y recomputación independiente de 29.017 posiciones.
La demo aprobada se procesa una sola vez después de pruebas a TEMP nuevo retenido;
el nuevo extractor se ejecuta sobre el VMAP real dentro del mismo cierre.
La referencia regenerada coincide byte por byte con el paquete publicado en código.
Frente a errores del modelo, conservar la respuesta y señalar el desacuerdo
con el hecho; no corregir retroactivamente su texto ni convertirlo en verdad.
