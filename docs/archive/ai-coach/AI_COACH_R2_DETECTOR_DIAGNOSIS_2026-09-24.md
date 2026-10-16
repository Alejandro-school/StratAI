# Por qué el detector se abstuvo en R2 — 24 de septiembre de 2026

La causa inmediata está en el último movimiento del portador de la C4, T1.
Su distancia a la referencia de A disminuye ampliamente durante los cuatro
segundos, pero aumenta un poco al final. La regla exige que el último incremento
no nulo de distancia sea negativo y excluye a T1. Después exige que el portador
esté entre quienes avanzan: esa segunda condición descarta la hipótesis A.

Se ha reproducido exactamente la predicción congelada, incluido su hash y su
enlace a la lectura anterior. El análisis usa los contextos y referencias
retenidos. No se modifican el detector, los umbrales ni las predicciones.

## Cadena de condiciones comprobada

1. La perspectiva T, las referencias de sitios, la identidad del portador y
   la continuidad están disponibles. T1 lleva C4 en las 65 muestras.
2. Los cuatro vivos conservan trayectorias completas de 65 muestras. T3 queda
   excluido como muerto desde la muestra 6373, sin reiniciar las de los demás.
3. Todos los vivos terminan más cerca de la referencia A que al comienzo; sus
   vectores globales también están más alineados con A que con B.
4. `progression()` exige a la vez aproximación neta y último desplazamiento
   resuelto hacia el sitio. T1 y T4 fallan la segunda parte. T2 y T5 la cumplen.
5. En A hay dos actores aceptados como avanzando, T2/T5. El número mínimo se
   satisface, pero el portador no pertenece al conjunto. La rama se interrumpe
   antes del control de divergencia de otros aliados y de asignar funciones.
6. El resultado sólo conserva `no_joint_continuous_approach_with_c4`, que agrupa
   varias posibles causas. En R2 no significa que falten trayectorias o C4.

Código relevante: [progresión](../../../ai_coach/interpretation/detector.py#L35),
[exigencia sobre el portador](../../../ai_coach/interpretation/detector.py#L145) y
[trayectorias por actor](../../../ai_coach/interpretation/context.py#L146).

| Actor | Aproximación neta a A | Último cambio no nulo de distancia | Aceptado como avanzando |
|---|---:|---:|---|
| T1, C4 | 445,706 | +0,941, alejamiento | No |
| T2 | 38,424 | −7,096, acercamiento | Sí |
| T4 | 325,444 | +2,921, alejamiento | No |
| T5 | 385,335 | −9,973, acercamiento | Sí |

Son unidades de coordenadas hacia un punto de referencia local de desarrollo,
no distancia de ruta física ni prueba de contención en una zona acreditada.
T1 acumula sólo 5,693 unidades de aumento de distancia en sus tres últimos
movimientos y después queda quieto. La pausa final no restablece la aproximación:
el código conserva el último cambio **no nulo**, que todavía es alejamiento.
Esto observa un desplazamiento pequeño; no demuestra ruido de captura, retirada
intencionada ni que deba ignorarse cualquier retroceso.

## Comprobaciones causales y de sensibilidad

- Se recalculan independientemente distancias, diferencias y condiciones por
  actor, y se contrastan con las funciones originales.
- Se recorre cada uno de los 65 prefijos causales del mismo intervalo. Al
  terminar 0,25 segundos antes del corte R2, la misma versión propone A. En el
  siguiente corte, 0,1875 segundos antes de R2, pierde A y termina absteniéndose.
  Son recomputaciones diagnósticas, no predicciones históricas nuevas del lote.
- También se suministra cada lectura anterior al siguiente prefijo. `previous`
  sólo describe el cambio de estado: no aporta una regla de persistencia que
  mantenga la hipótesis frente a ese veto.
- En una prueba aislada se desactiva **sólo** la exigencia del último signo para
  el portador, conservando aproximación neta y todas las demás condiciones. La
  salida pasa a compatible con A, con T1/T2/T5 avanzando. Restaurar la función
  devuelve exactamente la abstención congelada. Esta intervención diagnóstica
  en memoria no es una corrección aprobada ni una nueva versión del detector.
- Las 65 ventanas más cortas que terminan en el mismo T0 se abstienen. Añadir
  historial anterior por sí solo tampoco puede cambiar el signo del último
  movimiento observado: con esta condición intacta, no resolvería el bloqueo.

T5 no provoca la abstención: el detector lo acepta avanzando hacia A. La muerte
de T3 no rompe las trayectorias de los vivos. Tampoco la producen información
rival ausente o cantidades de granadas: esos datos no intervienen en la condición
que descarta A. Continúan siendo relevantes para justificar una decisión o un
consejo, y este diagnóstico no los completa.

## Alcance y continuación

Se identifica sensibilidad de la regla a movimientos finales pequeños y una
explicación de abstención poco específica. La lectura anterior y el avance
global no compensan el veto instantáneo del portador. No se ha demostrado que
la solución correcta sea un umbral numérico concreto, ignorar toda retirada,
prolongar indefinidamente la hipótesis o cambiar las referencias del mapa.

El siguiente encuadre debe distinguir ajuste breve, pausa, reagrupamiento y
retirada sostenida; exigir ejemplos y excepciones, y revisar cuándo mantener,
debilitar o retirar una lectura. La evaluación debe conservar versión congelada,
casos de control y separación independiente. Usar R2 para desarrollar esa
corrección lo mantiene como desarrollo/exploración, no test final.

[Evidencia del diagnóstico](../../evidence/AI_COACH_R2_DETECTOR_DIAGNOSIS_2026-09-24.json)
y [cálculo reproducible](../../../outputs/stratai-r2-detector-diagnosis-20260924/diagnose.py).
Se verifican 77 archivos protegidos sin cambios. Cero parsing nuevo, cambios
runtime o entrenamiento; no corresponde repetir el smoke anterior por esta
inspección y documentación. El grafo MCP no estaba disponible (`Transport closed`)
y se utilizó la lectura directa del código como alternativa.
