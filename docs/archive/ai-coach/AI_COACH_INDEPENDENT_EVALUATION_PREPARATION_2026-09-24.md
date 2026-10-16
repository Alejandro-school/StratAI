# Preparación de TP10: censo y protocolo de evaluación

24 de septiembre de 2026. Cierre de la auditoría y preparación documental;
la evaluación táctica sigue pendiente. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md)
contiene el estado vigente y la respuesta de alcance que falta.

Se amplía a las 44 partidas aprobadas la auditoría anterior de siete, sin
ejecutar el detector sobre nuevas jugadas. Hay 311 participantes distintos;
37 partidas no pertenecen al desarrollo conocido y 13 de ellas no comparten
participantes con él. Entre estas 13 existen nueve parejas con cruces. Una
selección determinista por SHA permite proponer tres partidas cuyos 30
participantes son distintos entre sí y de los de desarrollo.

Ninguna de las 44 partidas es posterior a la última de desarrollo:
25/08/2026 09:01:42 UTC. Todas aparecen en la auditoría agregada del 27/08;
la exposición táctica privada no registrada continúa desconocida. Estos hechos
impiden certificar separación temporal o ausencia absoluta de exposición.

El [protocolo v0.1](../../ai-coach/AI_COACH_INDEPENDENT_EVALUATION_PROTOCOL.md) concreta
alcance, selección condicional, prefijos temporales, revisión ciega, ejemplos,
dependencias, métricas por partida y sensibilidad secundaria. Se conserva un
[formulario vacío](../../../outputs/stratai-independent-evaluation-20260924/Ficha_de_revision_ciega.md).
No se han creado fichas de episodios reales ni respuestas humanas nuevas.

Se preguntó al usuario si evaluar otros jugadores del mismo periodo o exigir
también partidas posteriores. La respuesta está pendiente; el protocolo no
transforma esa consulta en aceptación. AGENTS.md exige encuadre experto antes
de la implementación dependiente de una nueva fase. IC19, IC24 y las lecturas
R1/R2/K1/K2 se conservan sin nuevas consultas. Excel v14 permanece intacto,
pues no hay otro criterio recibido que registrar.

## Verificación realizada

- Comando: `backend/venv/Scripts/python.exe -B outputs/stratai-independent-evaluation-20260924/audit_cohort.py`.
- Contraste de 44 manifiestos externos y 44 canónicos, con identidad de partida,
  SHA de demo declarado y estado aprobado. Participantes y fechas se validan
  contra ambas entradas de artefacto: 176 entradas contrastadas por hash, bytes
  y schema. Las fechas son metadatos originales observados; no se consultan otra
  vez al proveedor.
- Comprobación de diez IDs válidos y únicos por partida. Se retienen aliases
  locales P001–P311, sin nombres ni correspondencia de Steam IDs.
- 946 pares recalculados por pertenencia de participantes, separadamente del
  cálculo por intersección. Auditoría retenida con 179 fuentes verificadas
  antes/después y 159 archivos protegidos conservados, incluyendo el sello de
  código temporal, contratos, catálogo y libro v14.
- Presencia de las 44 demos originales comprobada. No se recalculó el hash de
  sus bytes ni se revalidaron sus datos de juego; esa comprobación corresponde
  a la posterior ejecución seleccionada. No confundir procedencia de metadatos
  con nueva validación de toda la replay.
- La [verificación de entrega](../../../outputs/stratai-independent-evaluation-20260924/delivery-verification.json)
  contrasta hashes, totales, pares, selección condicional y enlaces locales.
  `git diff --check` se aplica a los documentos modificados. La
  [evidencia](../../evidence/AI_COACH_INDEPENDENT_EVALUATION_PREPARATION_2026-09-24.json)
  enlaza los artefactos retenidos.

No se modifica código runtime, schema, productor ni hecho derivado. Por ello
no corresponde otro smoke ni repetir las suites funcionales del cierre
temporal. Cero parsing, predicciones, etiquetas, recomendaciones o entrenamiento.
El script de auditoría rechaza sobrescribir su resultado. Se conservan fuentes,
corpus, libros, temporales y cambios preexistentes; no hay adquisición, limpieza,
promoción, publicación ni mutación de CURRENT.
