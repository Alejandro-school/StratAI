# Analizador de demos

Servicio Go que convierte demos de Counter-Strike 2 en eventos, métricas y datos de repetición consumibles por el resto de StratAI.

## Responsabilidades

- Parsear demos.
- Capturar eventos de ronda, combate, economía, granadas y movimiento.
- Calcular análisis mecánicos y tácticos.
- Generar los modelos de salida y persistir resultados en Redis.
- Exponer endpoints internos de procesamiento y salud.

## Estructura

```text
analyzers/     Cálculos de alto nivel
api/           Endpoints HTTP internos
db/            Acceso a Redis
handlers/      Captura de eventos por dominio
middleware/    Seguridad HTTP transversal
models/        Contratos y estructuras de datos
parser/        Orquestación del parsing y exportación
pkg/           Paquetes reutilizables con dominio explícito
scripts/       Reprocesamiento operativo
main.go        Arranque del servicio HTTP
```

`process_demo.go` es una utilidad local excluida de la compilación normal mediante `//go:build ignore`; no es un segundo servidor.

## Desarrollo

Desde la raíz:

```powershell
Push-Location backend/go-service
go mod download
Pop-Location
npm run go-service
```

El servicio escucha en `http://localhost:8080` y requiere Redis. Sus endpoints de procesamiento están protegidos para comunicación interna.

## Pruebas

```powershell
npm run test:go
```

Las pruebas Go se colocan junto al paquete probado con sufijo `_test.go`.

Última revisión: 26 de julio de 2026.
