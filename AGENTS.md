# Sintonizados — instrucciones canónicas

Sintonizados ofrece accesibilidad en conferencias mediante audio, transcripción,
traducción inglés → español y subtítulos por charla. Leé primero
`docs/project-context.md`, `README.md` y `docs/planning/mvp.md`.

## Arquitectura y estado

Go orquesta HTTP, sesiones y concurrencia. `internal/domain` contiene tipos y
puertos independientes de proveedores y Smithy; `internal/application` coordina;
`internal/adapters` implementa memoria, Google, demo y HTTP. El frontend mínimo
se sirve embebido desde Go. Contrato público: Smithy 2.0; OpenAPI 3.1 derivado.

Implementado: ingestión PCM, captura desde browser, conector OBS/RTMP con MediaMTX
y FFmpeg OCI, overlay transparente para OBS, dos sesiones concurrentes, SSE con
replay acotado, MemorySessionStore/MemoryEventBus, adapters Google y demo, métricas,
CI OCI.
Las pruebas de Google usan dobles locales; no afirmar validación real sin evidencia.
NO implementado: Laya, Gemma, stores/buses distribuidos, routing multimodelo,
afinidad como solución de consistencia, renovación automática Live, recepción HLS,
SRT/VTT.

## Minimal Change Engineering

1. Leer el flujo relevante, identificar comportamiento actual, esperado y causa.
2. Definir el alcance mínimo y qué comportamiento debe preservarse.
3. Escribir primero una prueba específica del caso faltante o regresión.
4. Ejecutarla antes de cambiar implementación y verificar el fallo esperado;
   si no puede ejecutarse, explicar por qué.
5. Implementar el cambio más pequeño; no refactorizar por oportunidad.
6. Ejecutar la prueba nueva y las existentes; describir cambio y límites.

No instalar toolchains en el host. Baseline: git + Podman + shell.
`./scripts/dev test` ejecuta Go con detector de carreras; `lint` verifica formato
más vet; `smoke` prueba dos fuentes; `contract` compara respuestas reales con el
contrato. `run` inicia la demo, `feed` envía archivos PCM o audio demo.
`obs-start|obs-stop|obs-status` opera MediaMTX y `obs-feed` conecta una publicación
RTMP de OBS con una sesión Go y su Browser Source de subtítulos.
No guardar secretos ni registrar audio/texto completo en logs. No confundir demo
con IA real. No agregar infraestructura distribuida o framework frontend sin motivo.

## Public API workflow

Antes de modificar un endpoint público, revisar y modificar primero el modelo
Smithy correspondiente. No agregar silenciosamente endpoints fuera del modelo.
Una operación modelada no debe documentarse como implementada hasta existir en Go.
Antes de agregar o cambiar una página HTTP pública (por ejemplo `/operator`,
`/talks/{session_id}` o `/obs/{session_id}`), actualizar primero su operación
Smithy y regenerar OpenAPI. Las URLs de transporte entrante de OBS no son endpoints
Go; documentar su protocolo en `docs/operations/local.md`.

1. Modificar `api/smithy/*.smithy`.
2. Ejecutar `./scripts/dev smithy-validate`.
3. Regenerar y validar con `./scripts/dev openapi`.
4. Adaptar pruebas de regresión y comprobar el fallo antes de implementar Go.
5. Adaptar implementación Go y completar tests de contrato.
6. Actualizar documentación y ejecutar CI equivalente local.

No editar OpenAPI manualmente ni agregar un YAML paralelo. Los DTO HTTP quedan
en el adaptador. Las particularidades SSE se documentan en `api/README.md`.
Definition of Done API: Smithy actualizado y válido + OpenAPI regenerado + Go
implementado + tests actualizados + CI pasando (distinguir CI local de remoto).

## Before changing architecture

Revisar `docs/architecture/`, `docs/adr/` y `docs/planning/mvp.md`.
Elegir el menor cambio necesario y justificar decisiones persistentes con ADR.

## After changing architecture

- Actualizar/crear ADR numerado si corresponde; no registrar propuestas como aceptadas.
- Actualizar Smithy y OpenAPI si cambió la API.
- Actualizar tests, documentación y estado del MVP.
- Actualizar el issue correspondiente si existe y el acceso está disponible;
  si falta acceso, dejar el resultado pendiente documentado en el plan MVP.

## Contribuciones

Conservar la licencia. Ejecutar `test`, `lint`, `openapi`, `smoke`, `contract` y
`git diff --check`. Un contrato generado modificado debe acompañar su Smithy.
Respetar cambios ajenos, evitar modificaciones no relacionadas, nunca introducir
secretos. Informar comprobaciones y limitaciones. No declarar despliegues o
integraciones exitosos sin comprobarlos.
