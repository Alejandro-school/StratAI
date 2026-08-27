# Corpus golden de contratos

Estado: `stratai.golden_corpus@1`, versión 1.

El corpus golden de Block 7 valida invariantes del parser y del export. No es un
dataset de entrenamiento, no contiene labels para promover un modelo y todos
sus casos declaran `training_allowed=false`.

## Ubicación y manifest

```text
backend/go-service/testdata/golden-corpus/v1/
├── manifest.json
└── cases.jsonl
```

El manifest fija:

- `corpus_id=golden-corpus-v1`;
- 21 casos;
- SHA-256 de `cases.jsonl`:
  `c1189afa1db872577b2c3c075f6f6f3b1df15590e9c2e482a4f3e2128aaaeb6a`;
- `training_allowed=false`;
- casos omitidos y su motivo.

Cambiar cualquier caso exige una nueva versión o actualizar de forma explícita
el manifest y su hash. Un hash, número de casos, schema o permiso de training
inconsistente bloquea la publicación.

## Cobertura v1

| Grupo | Casos |
| --- | --- |
| Referencias reales | `nuke_normal`, `cache_normal` |
| Rondas y objetivo | `side_switch`, `plant_explode`, `defuse` |
| Utility | `round_without_utility`, `flash_enemy`, `flash_team`, `flash_self`, `he_damage`, `fire_damage` |
| Combate y engagements | `combat_miss`, `trade`, `multi_target`, `clutch` |
| Filtrado y lifecycle | `warmup`, `truncated_match`, `callbacks_incomplete` |
| Identidad y disponibilidad | `disconnected_player`, `bot_player`, `unavailable_actor_position` |

Nuke y Cache enlazan los dos exports piloto reales. Los demás son fixtures de
contrato mínimos: verifican hechos, valores derivados esperados, warnings y
hard failures sin presentarse como partidas observadas.

### Overtime

La versión 1 no afirma cobertura de overtime porque ninguna de las 20 demos
locales disponibles contiene un caso confirmado. El manifest lo registra en
`omitted_cases` con su razón. No se inventó una partida ni una etiqueta factual.
Cuando exista una demo verificable, se añadirá como nueva versión del corpus.

## Semántica de un caso

Cada línea de `cases.jsonl` contiene:

- `case_id` estable;
- `source`, distinguiendo demo real de fixture sintético de contrato;
- `tags` para cobertura;
- `input` mínimo;
- `expected_facts`;
- `expected_derived_values`;
- `expected_warnings`;
- `expected_hard_failures`;
- `training_allowed=false`.

Un warning esperado representa una limitación observada. No permite convertir
`unavailable` o `inferred` en un hecho.

## Ejecutar el gate

```powershell
backend\venv\Scripts\python.exe backend\go-service\scripts\golden_corpus_validator.py `
  backend\go-service\testdata\golden-corpus\v1
```

Salida de éxito:

```json
{"errors": [], "status": "passed"}
```

`publication_validator.py` ejecuta este gate además de validar el bundle. Por
tanto, un corpus ausente, modificado sin actualizar su hash o inválido impide
el commit de cualquier export.

Pruebas:

```powershell
backend\venv\Scripts\python.exe -m pytest `
  backend\go-service\scripts\test_canonical_export_validator.py `
  backend\go-service\scripts\test_block7_publication.py
```

Este corpus permanece limitado a validación durante Block 7. Ampliación de
corpus, FACEIT, ventanas ML, feature registry y entrenamiento pertenecen a
trabajo posterior y no se han iniciado.
