# Vivos de ambos equipos y resumen factual — 25/09/2026

La ficha E02 v3 muestra **«Bomba plantada. 2 aliados vivos · 1 rival vivo · Bomba: 0:39.27»**
en el corte final. El resumen cambia con el control temporal y presenta plantado,
vivos de ambos equipos y reloj disponible. Los datos sin confirmar permanecen
visibles como incertidumbre. [Ficha](../../../outputs/stratai-team-status-20260925/review/E02.html).

## Problema y cambio

El usuario señala que existen datos que la IA no muestra. La respuesta y el
[literal IC29](../../../outputs/stratai-team-status-20260925/request.json) quedan
registrados: teníamos el dato de rivales en la captura, pero sólo lo habíamos
auditado a petición del experto. La ficha no lo mostraba. El detector táctico
también es limitado: aplica una regla provisional de entrada y aún no interpreta
defensa posplantado. Una abstención sobre el plan no debe ocultar hechos disponibles.

Se incorpora un módulo factual por prefijo y una ficha versionada v3. En ámbito
5v5 exige cinco registros actuales, únicos y válidos por lado para mostrar un
total exacto. Las ausencias, desconexiones, duplicados y conflictos producen
incertidumbre; no se rellenan con el último estado. Reinicia el censo al cambiar
ronda, instancia o epoch. Usa identidad/lado/vida, sin posiciones ni recursos
ocultos del rival. [Contrato](../../contracts/context/canonical-team-status-review-v1.md).

Los campos de v2 permanecen iguales. Los originales v1/v2, la lectura de E02 y
las predicciones congeladas se conservan. El resumen es factual, no presenta la
predicción al revisor ni convierte «plantada» en «bien defendida». La nueva
información se incorpora a la ficha de revisión; no se introduce silenciosamente
en el detector congelado. Interpretación y recomendaciones mantienen sus dependencias.

## Verificación

- 115 pruebas afectadas, incluidas 18 nuevas, y 43 pruebas del backend pasadas.
  Lint público y específico y contrato de 115 schemas/19 razones pasados.
- Tras las pruebas se procesó exactamente una demo aprobada:
  `demo-0beb9110ec408f80c39b`, SHA-256
  `0beb9110ec408f80c39ba19f90729773705108e21d88d0df20b60267a82f91c7`.
  Un pase, TEMP nuevo vacío y sin promoción. Go permanece intacto; se reutiliza
  su binario verificado del cierre anterior, con el overlay que retiene staging.
  Parser v16, canonical 3.8.0, quality 12. El productor cambiado de fichas v3 se
  ejecuta con la captura nueva y su lector recalcula JSON/HTML y manifiesto.
- La auditoría independiente recalcula **1.170 muestras y 2.340 recuentos por lado**
  tanto desde flags nativos como desde las bajas públicas anteriores al corte.
  Los cinco aliados y cinco rivales están observados en todos esos cortes.
  Las ramas parcial/conflicto se verifican con fixtures; no se atribuyen a esta demo.
- E01 conserva 65 muestras y contexto automático desconocido, junto a su exclusión
  experta previa. E02 conserva 1.105 muestras, objetivo C4 y sus datos de inventario,
  posiciones aliadas, plantado y reloj. E02 empieza 5 contra 5 y termina 2 contra 1.
- Revisión visual de E02 en escritorio y móvil, control temporal inicial/final,
  reproducción/pausa y ausencia de desbordamiento horizontal. El Excel v20 actualiza
  F33/G33 de IC28 y añade IC29 en A34:G34, conservando 14 hojas y los literales previos.

[Comandos, tiempos, hashes y resultados](../../evidence/AI_COACH_TEAM_STATUS_REVIEW_2026-09-25.json).
[Inventario TEMP retenido](../../../outputs/stratai-team-status-20260925/retained-inventory.json).
Los 44 manifests y el SHA de la demo original se verificaron intactos. No hubo
descargas, limpieza, entrenamiento, CURRENT, promoción ni migración 44/44.

## Evaluación y siguiente acción

El cambio de instrumento ocurre después de la lectura de E02. El recibo v3
mantiene sus IDs originales y no reetiqueta ni solicita nuevamente esa lectura.
Se conservan 36 casos históricos, E01 excluida, E02 contrastada descriptivamente,
34 pendientes y cero puntuaciones de precisión. TP10 v0.6 declara la nueva vista.

Continuar con E03 sin regenerar predicciones y declarando las limitaciones de su
ficha v1. La defensa posplantado automática necesitará su encuadre experto propio;
el sistema aún no acredita esa interpretación ni la calidad de la colocación.
