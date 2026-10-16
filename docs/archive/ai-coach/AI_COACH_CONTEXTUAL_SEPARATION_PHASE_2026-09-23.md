# Separación de las partidas de evaluación — 23 de septiembre de 2026

La auditoría de las siete partidas está cerrada en su alcance técnico. Las dos
partidas de reserva comparten participantes con desarrollo y son anteriores a la
partida más reciente de desarrollo. Ambas estuvieron incluidas en la auditoría
agregada del corpus del 27 de agosto. **Este lote no acredita generalización a
jugadores nuevos ni a partidas posteriores.** Puede conservarse para contraste
táctico exploratorio cuando se reciban las lecturas humanas.

[Evidencia](../../evidence/AI_COACH_CONTEXTUAL_SEPARATION_2026-09-23.json) y
[script reproducible](../../../outputs/stratai-contextual-separation-20260923/audit_separation.py).
No se cambió el detector ni se ejecutaron nuevas demos.

| Caso | Partida opaca | Fecha observada, UTC | Jugadores compartidos con desarrollo |
|---|---|---|---:|
| R1 | `demo-045df2521c790a7d8a69` | 2026-08-24 20:51:04 | 2 de 10 |
| R2 | `demo-066944f5acb806ba5f81` | 2026-08-25 04:47:09 | 3 de 10 |

La última partida de desarrollo empieza el 25 de agosto a las 09:01:42 UTC.
R1 comparte un participante con `demo-6e4a3660d53931747fa8` y otro con
`demo-a11eb0d600ee702c9edb`. R2 comparte tres con esta última. Las siete partidas
contienen 70 participaciones y 63 personas distintas según las identidades de las
fuentes. El informe usa alias locales; no conserva nombres ni Steam IDs.

## Fuentes y límites

Las capturas usadas por el intérprete tienen `played_at=null`. El metadato
original del corpus sí contiene fecha observada con fuente FACEIT. Se une por
SHA-256 de la demo y se verifica el hash del artefacto contra su manifest
protegido; no se usa la fecha del archivo ni la de procesamiento. Estas fechas
son las registradas en el corpus, no una consulta nueva al proveedor. No se
modifican ni completan retroactivamente las capturas.

Los diez participantes de cada bundle coinciden con los de la captura evaluada.
Las 21 intersecciones por parejas se recomputan mediante una segunda tabla de
pertenencia de cada participante. Compartir jugadores se registra como dependencia
entre partidas; no equivale a demostrar que el detector use su identidad.

La exposición histórica positiva se acredita por pertenencia al inventario de
44 bundles de la auditoría del 27 de agosto. Esto prueba exposición en una
auditoría agregada, no que alguien haya visto estas jugadas ni sus etiquetas.
La exposición privada de desarrolladores/revisores y las lecturas tácticas
anteriores siguen desconocidas. No es necesario repetir búsquedas de inventarios
para las conclusiones técnicas ya comprobadas.

## Verificación y conservación

Se verifican 59 fuentes antes y después, los 47 hashes protegidos del cierre
anterior y las diez fuentes del detector congelado. Los siete templates conservan
los IDs preseleccionados y sus 84 respuestas vacías. La primera ejecución del
script detectó una suposición incorrecta del auditor sobre esos templates: son
12 filas con campos nulos, no listas vacías. Se corrigió esa comprobación y se
repitió la auditoría; no llegó a escribirse un resultado fallido ni se modificaron
los templates. El resultado final satisface todas las comprobaciones.

Sólo se leen manifests y metadatos, sin parsing, rematerialización, adquisición,
promoción, entrenamiento ni cambios runtime. Por este alcance documental no
corresponde repetir el smoke ni las suites funcionales. Se preservan el Excel v9
y sus criterios: no se ha recibido un criterio táctico nuevo que añadir.

## Próxima entrada necesaria

Recibir las lecturas reales de [R1](../../../outputs/stratai-contextual-20260923/R1.md)
y [R2](../../../outputs/stratai-contextual-20260923/R2.md), con planes compatibles,
funciones posibles, señales, incertidumbre y exposición previa del revisor, antes
de mostrar las predicciones. La instrucción «hazlo» autoriza continuar el trabajo;
no contiene una lectura de las jugadas. Las respuestas se registrarán separadas
del detector. El lote permanece con cero revisiones humanas y precisión no medida.

Estas dos lecturas iniciarían el contraste; no sustituirían el diseño y los
revisores independientes exigidos para una evaluación formal. Consejo, perfil,
práctica y seguimiento conservan sus dependencias factuales y de revisión.
