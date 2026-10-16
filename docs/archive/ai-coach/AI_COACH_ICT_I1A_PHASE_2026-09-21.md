# Cierre ICT-I1.A — 21 de septiembre de 2026

Implementado y verificado el primer registro descriptivo temporal del alcance
acordado. El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) conserva estado,
orden y siguiente acción. [Evidencia completa](../../evidence/AI_COACH_ICT_I1A_SINGLE_DEMO_SMOKE_2026-09-21.json).

## Resultado

El módulo `ai_coach/interpretation/` produce registros por corte y lado analítico,
con identidad, procedencia, posiciones, vida, siete hechos físicos, tenencia C4,
desplazamientos entre muestras válidas, exclusiones y desconocidos. Cada registro
enlaza al anterior sin usar descriptores posteriores C0, respuestas expertas,
títulos de fichas o rivales ocultos. El lector recompone los registros completos.

El [contrato candidato](../../contracts/context/canonical-temporal-description-v1.md) registra el
schema `stratai.temporal_description@1`, versión `1.0.0-candidate.1`.
El verificador reconoce 107 schemas y 19 razones; catálogo 1.7.0 sin cambios.
Otros contextos C0 permanecen separados; no se declara que este adaptador los
proyecte todos. La perspectiva analítica no acredita conocimiento compartido.

El encuadre E01–E04 ya estaba recibido en Excel v6. No hubo nuevo criterio táctico,
por lo que se conserva ese libro sin reescribir respuestas ni crear otra versión.
Los cuatro motivos quedan representados con requisitos pendientes y sin reglas
positivas habilitadas. Se conserva la excepción E02 de todo el equipo en B.

## Verificación

- `npm run check:all`: aprobado.
- Suite del AI Coach: 1.888 pruebas aprobadas. Después de separar los objetos
  emitidos del estado interno, 62 pruebas finales afectadas aprobadas, incluidas
  las 43 pruebas nuevas de este componente. No se repite la suite ajena al ajuste.
- Contratos: 107 schemas/19 razones; análisis estático de los archivos nuevos aprobado.
- Invariancia ante futuro truncado o modificado, rivales ocultos, respuestas
  humanas, descriptores de ejecución y hashes globales que incluyen futuro.
- Identidad, cambio de vida, hueco, instancia/época, ausencias, recursos desconocidos,
  atribución temporal, manipulación de registros y salida fuera de TEMP contrastados.
- Los casos E01–E04 contrastan etiquetas positivas sin fuente, ausencia de prueba
  y precondiciones incumplidas. Demuestran abstención, no detección táctica positiva.

## Una demo real

Demo `demo-a11eb0d600ee702c9edb`, SHA-256
`a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`. Un único match aprobado, nueva captura después
de las pruebas, sin reproceso de corpus. Parser v16, export 3.8.0 y quality 12.
Los comandos exactos, versiones, duraciones y hashes están en la evidencia.

| Comprobación | Resultado |
|---|---:|
| Registros temporales CT/T | 61432 |
| Estados actor/corte | 306740 |
| Posiciones contrastadas directamente | 205324 |
| Desplazamientos recalculados de forma independiente | 205045 |
| Hechos físicos contrastados | 1217088 |
| Tenencias C4 positivas contrastadas | 13770 |
| Ausencias C4 con inventario contrastado | 191554 |
| Hipótesis en abstención | 245728 |
| Recomendaciones/capacidades elegibles nuevas | 0 |

El bundle supera el validador semántico canónico; escritor y lector coinciden.
La generación completa tardó 1237.647 s
y su lector 1212.141 s en este entorno.
Es un flujo de desarrollo offline con coste elevado de validación: cProfile sobre
200 cortes identifica los validadores de schemas como coste principal. No se
retiraron controles para reducir esa duración ni se acredita latencia de producto.
El auditor independiente usa las fuentes originales para recalcular diferencias
y verificar reloj, identidades, munición nativa e inventario/propietario C4.
Se observan tanto diferencias cero como distintas de cero y tenencia C4 positiva
y negativa. Los conteos por estado y razón, incluidas las ausencias, están retenidos.

K1/K2 conservan coordenadas y tenencia comprobadas en un auditor separado; sus
respuestas no son entrada del motor. Siguen siendo desarrollo conocido, sin
evaluación táctica independiente ni una inferencia automática de push B.
Los 1.417 registros del prefijo hasta K2 permanecen idénticos al corromper futuro,
estado/contexto/inventario rival oculto, introducir respuestas humanas y ejecución
C0 y alterar el hash global del manifiesto en copias de memoria. El hash final de
ese prefijo coincide con K2 dentro de la generación de la partida completa.

## Retención y límites

Retenidos 83 archivos / 2068944645 bytes en
`C:\Users\kerch\AppData\Local\Temp\stratai-ict-i1a-6r5w7pdo`, con inventario y hashes. La compilación aislada usa un
overlay que únicamente desactiva el borrado automático de staging; no modifica
la lógica del productor. Los 44 manifests aprobados conservan sus fingerprints.
No hubo limpieza, adquisición, publicación, promoción, entrenamiento ni CURRENT.

Contacto legal, receptores de información, vínculo espacial semántico, función,
intención, viabilidad y superioridad siguen pendientes. E01/E02 de auditoría,
S2–S3, cero elegibles y evaluación independiente conservan sus límites.
La fase cierra el componente descriptivo ICT-I1.A, no el intérprete táctico completo.
