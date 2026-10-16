# Cierre técnico del intérprete contextual — 23 de septiembre de 2026

ICT-I2 implementa el [contrato contextual](../../contracts/context/canonical-contextual-interpreter-v1.md)
y la regla provisional aceptada por el usuario. [Evidencia](../../evidence/AI_COACH_CONTEXTUAL_INTERPRETER_SMOKE_2026-09-23.json).
El [plan canónico](../../AI_COACH_IMPLEMENTATION_PLAN.md) mantiene el orden y los gates.

La entrada conserva las 65 muestras de cada uno de los dos intervalos de desarrollo,
con identidad, reloj, recursos físicos, C4 y referencias. Los orígenes `env_cs_place`
del VMAP local aportan referencias trazables, con zonas y compatibilidad de build
sin acreditar. La primera interpretación propone B en K1 y mantiene B en K2.
Conserva al portador como hecho y a T5 como posible preparador de utilidad.
K1/K2 se usaron para desarrollar la regla: la concordancia no es precisión independiente.

El primer ensayo sólo con reducción de distancia admitía A y B. Se corrigió en
desarrollo contrastando la dirección relativa hacia los sites y conservando empates.
El detector no lee las respuestas humanas. El sufijo estacionario con humo equipado
tiene 17 muestras en K1 y 45 en K2; no se convierte en lanzamiento ni cobertura.

## Verificación

- `npm run check:all`: correcto (100.22 s).
- AI Coach: 1.938 pruebas correctas en la ejecución completa y una comprobación
  antigua del número de schemas corregida de 107 a 108 y repetida con éxito.
  La ejecución afectada final completa añade 264 comprobaciones correctas.
- Registro público de contratos y Ruff correctos. Catálogo y razones intactos.
- Exactamente una demo aprobada, `demo-a11eb0d600ee702c9edb`, SHA-256 `a11eb0d600ee702c9edb59e8f3ab54b113a1e552bd65a40ce53a2a0f54df1262`.
- Parser 98.844 s; generación contextual 21.8 s;
  lector 23.745 s; validación canónica 324.421 s.
- Recomputación independiente: 650 posiciones, 650 deltas, 4.550 hechos físicos,
  130 tenencias C4 positivas, 520 negativas y 10 señales geométricas. Son apariciones
  en dos ventanas solapadas de una sola partida, no ejemplos independientes.
- Invariancia real a futuro truncado, posiciones rivales ocultas, nombre del caso,
  respuesta humana y ganador. Pruebas adversariales de ausencia, identidad, lado,
  mapa, ambigüedad y alteración de archivos conservadas.

Se retienen 86 archivos / 1485277595 bytes en TEMP, con
inventario y hashes. Los 44 manifests aprobados, el catálogo, el libro v8 y el modo
temporal de interfaz conservan sus hashes. No hubo borrado, descarga, promoción,
migración ni entrenamiento. Las dos salidas de desarrollo anteriores también se
conservan; la primera contiene la ambigüedad detectada, sin reescritura.

## Alcance pendiente

Quedan pendientes el contorno semántico y compatibilidad del mapa, la evaluación
táctica independiente y las alternativas viables. `coaching.assess_readiness`
devuelve abstención con requisitos concretos; no es un recomendador, un perfil
personal ni un plan de práctica implementados. No se cierra ninguna familia
prescriptiva ni se modifica la elegibilidad por este prototipo.

La evaluación técnica posterior utilizará cinco capturas de desarrollo y dos
partidas de reserva elegidas antes de sus predicciones, con código congelado.
Ese lote no aporta por sí solo respuestas humanas ni acreditación de independencia.
