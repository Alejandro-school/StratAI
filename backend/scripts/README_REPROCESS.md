# 🔄 Scripts de Re-procesamiento de Demos

Este directorio contiene scripts para re-procesar las demos existentes con los datos actualizados.

## 📋 ¿Por qué re-procesar?

Después de actualizar el código del backend, las demos ya procesadas no tendrán:
- ✅ **Fecha real del partido** (anteriormente usaba la fecha de análisis)
- ✅ **Campo Team asignado** (CT/T para cada jugador)

Estos scripts te permiten actualizar todos los datos sin necesidad de volver a descargar las demos desde Steam.

---

## 🚀 Opción 1: Script Go (Recomendado)

**Ventajas:**
- ⚡ Más rápido (procesamiento directo)
- 🎯 Acceso directo a Redis
- 💪 Más eficiente con recursos

### Uso:

1. **Navegar al directorio:**
   ```bash
   cd backend/go-service/scripts
   ```

2. **Ejecutar el script:**
   ```bash
   ./reprocess_demos.exe
   ```
   
   O si no está compilado:
   ```bash
   go run reprocess_all.go
   ```

3. **Seguir las instrucciones:**
   - Ingresar tu Steam ID
   - Confirmar si deseas limpiar datos anteriores
   - Esperar a que procese todas las demos

### Ejemplo de salida:
```
🔄 Script de Re-procesamiento de Demos
============================================================

📊 Se encontraron 23 demos

🗑️  ¿Deseas limpiar los datos anteriores en Redis? (s/n): s
✅ Se eliminaron 23 entradas anteriores

🚀 Iniciando re-procesamiento...

[1/23] 📦 match_33CSnOTOYYfb9NpSx2noAsFiJ.dem
  ✅ Procesada exitosamente
     • Mapa: de_mirage
     • Fecha: 2025-10-10 14:23:45
     • Resultado: victory (16 - 14)
     • Jugadores: 10

...

============================================================
📈 Resumen del Re-procesamiento
  ✅ Exitosas: 23
  ❌ Fallidas: 0
  📊 Total: 23

🎉 ¡Re-procesamiento completado!
```

---

## 🐍 Opción 2: Script Python

**Ventajas:**
- 🔧 Fácil de modificar
- 📝 Más legible para no-Go developers

### Requisitos:
```bash
pip install requests
```

### Uso:

1. **Asegúrate de que el servidor Go esté corriendo:**
   ```bash
   cd backend/go-service
   go run main.go
   ```

2. **En otra terminal, ejecutar el script:**
   ```bash
   cd backend/scripts
   python reprocess_demos.py
   ```

3. **Seguir las instrucciones interactivas**

---

## 📊 Datos Actualizados

Después de ejecutar cualquiera de los scripts, las demos tendrán:

### Fecha Correcta
- **Antes:** `2025-10-15 20:30:00` (fecha de análisis)
- **Ahora:** `2025-10-10 14:23:45` (fecha real del partido, tomada del archivo .dem)

### Campo Team
- **Antes:** Campo vacío `""`
- **Ahora:** `"CT"` o `"T"` para cada jugador

### Impacto en el Frontend
- ✅ **HistoryGames**: Mostrará fechas reales de las partidas
- ✅ **MatchDetails**: Separará correctamente "Mi Equipo" vs "Equipo Enemigo"
- ✅ **Dashboard**: Estadísticas agrupadas correctamente por equipo

---

## ⚠️ Notas Importantes

1. **Redis debe estar corriendo:**
   - Verifica que Redis esté activo antes de ejecutar los scripts
   
2. **Demos en el lugar correcto:**
   - Las demos deben estar en `backend/data/demos/*.dem`
   
3. **Backups (opcional):**
   - Los scripts pueden limpiar Redis antes de procesar
   - Si quieres hacer backup: `redis-cli SAVE`

4. **Recarga el frontend:**
   - Después de re-procesar, recarga el navegador (Ctrl+F5)

---

## 🆘 Solución de Problemas

### Error: "No se encontró el archivo .dem"
- Verifica que las demos estén en `backend/data/demos/`
- El script busca archivos `*.dem`

### Error: "Error al conectar con Redis"
- Asegúrate de que Redis esté corriendo
- Verifica las credenciales en el archivo `.env`

### Las demos no aparecen actualizadas
- Recarga el frontend completamente (Ctrl+F5)
- Verifica que Redis se haya actualizado: `redis-cli KEYS "processed_demos:*"`

### El script es muy lento
- Es normal si tienes muchas demos (cada una tarda ~2-5 segundos)
- El script Go es más rápido que el Python

---

## 📞 Soporte

Si encuentras problemas, verifica:
1. Logs del servidor Go
2. Logs de Redis
3. Que los archivos .dem estén completos y no corruptos

