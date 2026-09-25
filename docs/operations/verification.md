# Verificación del bootstrap

Ejecución local sobre Linux amd64 con Podman rootless. No se instalaron toolchains
en el host. Resultados observados el 24 de septiembre de 2026 (hora Argentina).

| Comprobación | Resultado |
| --- | --- |
| Pruebas antes de implementación | Fallaron por paquetes aún ausentes; adaptadores Google por tipos ausentes |
| Regresión WebSocket | Reprodujo pérdida del último transcript al competir cierre con mensajes pendientes; corregida con cola ordenada de resultados |
| `./scripts/dev test` | PASS con `go test -race -count=1 ./...` |
| `./scripts/dev lint` | PASS gofmt + go vet |
| `./scripts/dev build` | PASS servidor y feeder compilados dentro de OCI |
| `./scripts/dev smithy-validate` | PASS, 465 shapes incluyendo prelude/dependencias |
| `./scripts/dev openapi` | PASS generación OpenAPI 3.1.0 y openapi-spec-validator 0.7.2 |
| `./scripts/dev contract` | PASS 11 operaciones contra servidor OCI, respuestas JSON Schema y dos fuentes simultáneas |
| `./scripts/dev smoke` | PASS dos sesiones HTTP, 20 subtítulos simulados y latencia |
| CLI feed real en OCI | PASS dos procesos simultáneos, 30 chunks cada uno, 60 subtítulos y ambas sesiones ended |
| `sh -n` wrappers | PASS dev, cloud y contract-check |
| `git diff --check` | PASS |
| Chromium en OCI | PASS navegación, SSE tras 6 s de pausa, dos charlas, cierre y sin errores JS |
| Capturas desktop/móvil | Revisadas a 1280x800 y 390x844; columnas adaptadas sin desborde visible |
| Enlaces locales de documentación | PASS |

Las pruebas Go cubren aislamiento, secuencias duplicadas/faltantes, audio después
de cierre, un observador que no consume más de 32 eventos, autorización, entradas
inválidas, replay SSE, coherencia de rutas y mocks WebSocket/REST de Google.

La comprobación de CLI inició un servidor temporal y dos ejecuciones de
`./scripts/dev feed -demo`; el servidor se eliminó al terminar. No se dejó un
servicio de prueba corriendo. Los tiempos de demo no representan latencia de Google.

## Fuera de esta evidencia

No se utilizaron credenciales reales de IA. No se desplegó Cloud Run, ni se publicó
una corrida de GitHub Actions. Los scripts de despliegue tienen validación sintáctica;
los permisos, APIs, cuotas y disponibilidad del modelo requieren el proyecto real.

La prueba exploratoria de navegador usó Playwright 1.51.1 en OCI sin agregar
Node como dependencia del proyecto. La automatización se ajustó para usar
selectores (su waitForFunction chocaba con CSP); no se relajó la política de la
aplicación. Esta comprobación no equivale a una auditoría de accesibilidad.
