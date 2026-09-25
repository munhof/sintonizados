# Despliegue

La unidad desplegable es una imagen OCI con Go, HTML embebido y CAs. No requiere
volúmenes de datos ni shell. Escucha PORT, logs a stdout, `/health` indica liveness
y modo (no conectividad con proveedores). Ejecutar como usuario 65532.

Local: `podman build` + `podman run`, o `./scripts/dev run`. Sin Compose.
Google Cloud: Artifact Registry + Cloud Run, secretos por Secret Manager. Ver
[runbook](../operations/google-cloud.md). La autenticación y la prueba real quedan
pendientes; no hay una URL desplegada verificada.

La configuración de demo pide una instancia, concurrencia 80, CPU siempre asignada
y timeout HTTP 3600 s. Cada lector SSE consume concurrencia. Con memoria local,
reemplazos, reinicios o despliegues pueden perder/dividir estado incluso con máximo
uno. No hacer rollout durante una demo activa; no prometer alta disponibilidad.

Cloud Run documenta afinidad de mejor esfuerzo, por lo que no habilitamos sticky
sessions como parche de consistencia. Fuente:
[session affinity](https://docs.cloud.google.com/run/docs/configuring/session-affinity).
El paso a múltiples instancias exige el trabajo descrito en [sesiones](sessions.md).
