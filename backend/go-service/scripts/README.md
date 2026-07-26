# Scripts de reprocesamiento de demos

## Script recomendado: `reprocess_parallel.py`

Reprocesa cada archivo `backend/data/demos/match_*.dem` mediante el servicio Go.
Mantiene el mismo `match_id`, conserva `date` y `duration_seconds` de los metadatos existentes y verifica que Go genere los exports fundamentales.

Al finalizar, reconstruye los agregados de los jugadores afectados en `backend/data/users/`, para que el dashboard utilice los datos nuevos. No descarga demos ni modifica la lista `processed_demos` de Redis.

### Requisitos

1. El entorno Python del backend con las dependencias de `backend/requirements.txt`.
2. El servicio Go iniciado desde su propio directorio, para que las rutas de demos se validen correctamente:

   ```powershell
   cd backend/go-service
   go run main.go
   ```

### Uso

Desde `backend`:

```powershell
.\venv\Scripts\python.exe go-service/scripts/reprocess_parallel.py
```

Por defecto utiliza dos workers, un límite seguro porque cada parseo ya usa varios hilos internos. Opciones útiles:

```powershell
# Ver qué demos se procesarían sin cambiar datos
.\venv\Scripts\python.exe go-service/scripts/reprocess_parallel.py --dry-run

# Ajustar concurrencia y timeout por demo
.\venv\Scripts\python.exe go-service/scripts/reprocess_parallel.py --workers 3 --timeout 900

# Solo reprocesar exports; no regenerar agregados de usuario
.\venv\Scripts\python.exe go-service/scripts/reprocess_parallel.py --skip-aggregate-rebuild
```

Si alguna demo falla, las demás continúan. El proceso devuelve código distinto de cero al terminar con errores.
