# Reparación AUD-O01 — estilo y requisito Go

20 de septiembre de 2026. Cerrado en el alcance acotado de la auditoría.

El contrato espacial utiliza functools.cache en lugar de lru_cache(maxsize=None), con el mismo almacenamiento sin límite. Se ordenan imports de test_spatial_candidate.py y el README indica Go 1.24.0 o posterior, conforme al mínimo declarado por go.mod. La comparación de fuente y AST confirma que no cambian las pruebas ni el resto del módulo; el único cambio del README es ese requisito. Se aplicó la skill crafting-effective-readmes para la edición de la guía de instalación.

Ruff completo de ai_coach y las 70 pruebas afectadas pasan. El mínimo publicado se contrasta con go.mod; no se afirma una ejecución en Go 1.24.0: el entorno verificado de esta entrega usa Go 1.26.2. Los checks públicos y la demo real del cierre C01 siguen siendo la evidencia funcional del cambio NAV. Estas sustituciones de estilo y texto no alteran contratos, datos derivados ni algoritmos, y no requieren otro parsing. La suite AI Coach completa de C01 conserva sus dos fallos iniciales de conteo y la repetición corregida de las pruebas afectadas; no se transforma retrospectivamente en un recibo verde.

Se conservan fuentes, corpus, temporales, libros, HEAD y staging. E01/E02 siguen sin sus originales; este ajuste no recupera esa evidencia. La siguiente acción es preparar el alcance del intérprete descriptivo temporal para revisión experta antes de implementarlo. No se prepara K3 ni se cambian criterios tácticos, fuentes o gates.

[Evidencia](../../evidence/AI_COACH_STYLE_REQUIREMENT_REPAIR_2026-09-20.json). [Comparación](../../../outputs/stratai-audit-followup-20260920/o-checks.json). [Preservación](../../../outputs/stratai-audit-followup-20260920/o-preservation.json).
