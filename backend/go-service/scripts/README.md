# Scripts de Reprocesamiento de Demos

## 🚀 Script Principal: `reprocess_parallel.py`

Reprocesa TODAS las demos con concurrencia (6 workers paralelos).
**Tiempo estimado para 34 demos: ~5-6 minutos** (vs 34 min secuencial).

### Requisitos

1. **Servicio Go en ejecución**

   ```bash
   cd backend/go-service
   go run main.go
   ```

2. **Instalar dependencias Python**
   ```bash
   pip install requests colorama
   ```

### Uso

Desde la carpeta `backend`:

```bash
.\venv\Scripts\python.exe go-service/scripts/reprocess_parallel.py
```

### ⚠️ Importante

- **Los matchIDs se mantienen** - Las demos NO pierden asociación con el usuario
- El script lee la metadata existente y preserva la fecha original
- Puedes interrumpir con `Ctrl+C` en cualquier momento

---

## Scripts Disponibles

| Script                   | Descripción                                |
| ------------------------ | ------------------------------------------ |
| `reprocess_parallel.py`  | ✅ **USAR ESTE** - Reprocesa con 6 workers |
| `reprocess_all_demos.py` | Versión secuencial (más lenta)             |

---

## Flujo de Datos

```
demos/*.dem → Go Service → exports/match_XXX/*.json
                              ↓
                         Redis (processed_demos:{steamID})
                              ↓
                         Frontend (lista de demos del usuario)
```

El script mantiene los `match_id` originales para que Redis siga asociando
las demos al usuario correcto.
