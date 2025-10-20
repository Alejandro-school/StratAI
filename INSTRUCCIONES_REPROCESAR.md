# 🔄 Cómo Reprocesar las Demos para Armas

## ⚠️ Problema
Las demos actuales no tienen los `event_logs` necesarios para detectar las armas.

**Log de consola muestra:**
```
Demo 0 tiene: EventLogs: false, event_logs: false, events: false
Demo 0: Processing 0 events
```

---

## ✅ Solución: Reprocesar las Demos

### **Opción 1: Script Automático (Recomendado)**

1. **Abrir PowerShell en el proyecto**
2. **Ejecutar:**
   ```powershell
   cd backend\go-service\scripts
   .\run_reprocess.bat
   ```

3. **Seguir las instrucciones:**
   - Te pedirá tu Steam ID: `76561198116485358`
   - Preguntará si quieres limpiar datos antiguos: `y`
   - Procesará las 19 demos automáticamente

4. **Esperar** a que termine (mostrará progreso)

5. **Recargar el navegador** (Ctrl+R)

---

### **Opción 2: Script Python**

Si el script Go no funciona:

```powershell
cd backend\scripts
python reprocess_demos.py
```

---

### **Opción 3: Reiniciar el servicio Go**

A veces solo necesitas reiniciar:

```powershell
cd backend\go-service
.\restart.bat
```

---

## 🔍 Verificar que Funcionó

Después de reprocesar, recarga el navegador y verifica en la consola:

**ANTES (Mal):**
```
Demo 0: Processing 0 events ❌
```

**DESPUÉS (Bien):**
```
Demo 0: Processing 1234 events ✅
Demo 0: Found 45 kills for user ✅
Top weapons: [{weapon: 'AK-47', kills: 15}, ...] ✅
```

---

## 📁 Archivos Afectados

El reprocesamiento actualizará:
- Redis (caché de demos procesadas)
- Los datos incluirán `event_logs` con todos los eventos de kill

---

## ❓ Si Sigue Sin Funcionar

1. **Verifica que el servicio Go esté corriendo:**
   ```powershell
   # Debería estar en puerto 8080
   netstat -ano | findstr :8080
   ```

2. **Revisa los logs del servicio Go:**
   - Se mostrarán en la ventana donde ejecutaste `restart.bat`

3. **Verifica Redis:**
   ```powershell
   # Redis debe estar corriendo
   redis-cli ping
   # Debería responder: PONG
   ```

---

## 🎯 Resultado Esperado

Después de reprocesar correctamente, verás en el Dashboard:

```
Armas Preferidas
━━━━━━━━━━━━━━━
🥇 AK-47         45 kills ████████████████░░░
🥈 AWP           32 kills ████████████░░░░░░░
🥉 M4A1-S        28 kills ██████████░░░░░░░░░
4️⃣ Deagle        21 kills ███████░░░░░░░░░░░░
```

---

## 💡 Nota

Este reprocesamiento solo necesitas hacerlo **UNA VEZ**. Las nuevas demos que analices ya incluirán los `event_logs` automáticamente.

