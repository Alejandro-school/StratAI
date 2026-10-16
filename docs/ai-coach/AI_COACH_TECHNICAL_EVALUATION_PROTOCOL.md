# Recorrido de rendimiento, evaluación y aprendizaje

Diseño técnico del 22 de septiembre de 2026, solicitado por el usuario mediante
«Bien. Pues aplica el recorrido». El [plan canónico](../AI_COACH_IMPLEMENTATION_PLAN.md)
mantiene el estado y el orden; este documento fija el procedimiento y sus límites.

## 1. Rendimiento con equivalencia

Medir generación completa, carga/validación, parsing y recomputación por separado.
Comparar el registro completo con la salida de referencia, además de comprobar
datos válidos e inválidos con los dos validadores. No cachear aprobaciones de datos
mutables ni retirar verificaciones semánticas, de identidad, procedencia o futuro.
Cerrar el cambio con tests y exactamente una demo aprobada en un TEMP nuevo.

La aceleración se limita a cinco schemas de support/registro temporal y usa
`jsonschema-rs==0.57.1`, Draft 2020-12. La compilación se cachea por contenido del
schema. El validador Python conserva la comprobación del schema, la semántica de
valores Python no JSON y los errores. Los otros schemas conservan su motor.
Referencia de implementación: [guía del mantenedor](https://github.com/Stranger6667/jsonschema/blob/master/crates/jsonschema-py/MIGRATION_FROM_JSONSCHEMA.md).

## 2. Primer lote técnico de cinco partidas

La [selección congelada](../../outputs/stratai-performance-20260922/sample-plan.json)
incluye la partida habitual de desarrollo y cuatro partidas distintas: mínimo,
tercio, dos tercios y máximo tamaño de archivo entre las otras 43 aprobadas, con
SHA como desempate. Se selecciona antes de ejecutar los nuevos productores.
La selección cubre tamaños; no representa niveles, estilos ni toda la población.

Tras cerrar la optimización se procesan las cuatro partidas adicionales, con el
mismo binario y código sellados, fuentes de sólo lectura y TEMP exclusivo por
partida. Como máximo dos procesos de partida simultáneos. No migrar ni promover
los 44 bundles. Conservar fallos, outputs, comandos y hashes.

Por partida:

- Verificar checksum, captura de las cuatro fuentes, schemas y uniones exactas.
- Generar ambos lados y recomponer todos los registros con el lector.
- Recalcular independientemente posiciones, diferencias entre muestras,
  identidades, munición nativa, C4 positiva/negativa y referencias.
- Contrastar registros espaciados uniformemente con el validador Python original.
- Registrar todos los denominadores, estados y motivos de abstención, además
  del tiempo de parsing, generación, lectura y auditoría.

Una excepción o diferencia bloquea esa partida. No sustituirla por otra para
obtener un lote aparentemente correcto. Corregir con pruebas y repetir la misma
entrada en otro TEMP retenido. Nunca convertir el número de filas en partidas
independientes ni las hipótesis indisponibles en falsos negativos tácticos.

## 3. Primera capacidad y revisión experta

Continuar el alcance ya acordado ICT-I1, con **revisión de una hipótesis colectiva
ante una señal nueva atribuible**. E01–E04, Q1/Q18/Q26 y K1/K2 conservan las
respuestas recibidas; no volver a pedir su aprobación general. La primera regla
se elegirá por fuente positiva y cobertura verificable, no por la facilidad de
producir una etiqueta.

Para concretarla se requiere un paquete por oportunidad: prefijo propio/aliado,
señal nueva, actor y receptor acreditados, instante disponible, vínculo espacial,
lectura anterior, alternativas viables y desconocidos. Un evento global de baja,
la posición real rival o una etiqueta humana no certifican lo que podía saber
el jugador. Los datos actuales no completan ese paquete para E01–E04.

Ejemplo de E02 que debe guiar la revisión: un aliado consigue una baja en medio;
podría abrirse una oportunidad de cambiar la ruta. La baja no prueba que la ruta
esté libre, sea transitable a tiempo o resulte superior. Se conserva la excepción
ya acordada de todo el equipo en B. Si falta receptor o vínculo espacial, la
salida sigue siendo desconocida. Con todos los requisitos presentes, revisar
con el experto únicamente la nueva condición para abrir/mantener la alternativa.

La muestra humana debe incluir mantener la lectura, modificarla, varias opciones
equivalentes e información insuficiente. Ocultar desenlace posterior, posiciones
rivales no acreditadas y respuesta del sistema antes de obtener el juicio.
No presentar un formulario de etiquetado como sustituto de fuentes ausentes.

## 4. Separación del dataset antes de ajustar modelos

Las cinco partidas de este lote son **desarrollo conocido**. Las otras 39 quedan
como reserva de selección, no como un test independiente ya certificado: hay
auditorías históricas y hace falta registrar exposición previa, fechas y cruces
de jugadores/equipos. No asignar filas aleatorias de una partida a varios grupos.

Antes del primer ajuste, congelar manifests de desarrollo/entrenamiento,
calibración y test por partida, con aislamiento temporal e informe de jugadores
compartidos. Ajustar prompts, reglas, normalización, referencias y umbrales sólo
en el grupo autorizado. Las partidas de test usadas para corregir la versión
pasan a desarrollo y se renueva el test antes de evaluar la siguiente versión.

El [diseño del piloto](AI_COACH_INITIAL_PILOT_DESIGN.md) conserva sus familias,
muestra dirigida y control aleatorio. Este lote técnico no lo reemplaza. Los
mínimos y condiciones de evaluación del plan siguen vigentes; 44 demos
disponibles no demuestran que se hayan alcanzado por acción/contexto.

## 5. Aprendizaje condicionado a evidencia

Comparar primero reglas y frecuencias, después regresión regularizada y, si
aporta una mejora medible, modelos de preferencias o temporales pequeños. Guardar
versiones de datos, etiquetas, código y modelo. Evaluar precisión de las
afirmaciones, consejos incorrectos, cobertura, abstención, calibración, coste y
dependencia por partida/jugador; usar los criterios de promoción del plan.

El entrenamiento sigue dependiendo de fuentes suficientes, etiquetas revisadas,
separación sin fuga y evaluación independiente. Analizar más demos no actualiza
pesos automáticamente. Los 61.432 registros de una partida tampoco son 61.432
decisiones tácticas supervisadas. No se habilita aprendizaje online.
