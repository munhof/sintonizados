# Contrato público

Fuente de verdad: `smithy/*.smithy` (Smithy IDL 2.0).
`smithy-build.json` configura el plugin oficial smithy-openapi 1.73.0, restJson1 y
versión OpenAPI 3.1.0. `smithy build` genera `generated/openapi/sintonizados.openapi.json`
mediante el wrapper. El archivo derivado se versiona para inspección en GitHub.
No hay un YAML paralelo ni servidor generado. restJson1 proporciona bindings de
serialización; no introduce infraestructura AWS.

```sh
./scripts/dev smithy-validate
./scripts/dev smithy-build
./scripts/dev openapi
./scripts/dev contract
```

Toda operación del modelo tiene implementación HTTP: health, lista, crear, detalle
(incluye estado), cerrar, audio, history, SSE, métricas y páginas HTML de inicio,
operación, charla y overlay OBS.
Los errores son JSON `{ "message": "..." }`, con X-Amzn-Errortype para el protocolo.
Las rutas de lectura son públicas. Crear/audio/cierre/métricas requieren
`Authorization: Bearer <OPERATOR_TOKEN>`. Una clave Google no autentica esta API.

## Audio HTTP

POST `/api/sessions/{session_id}/audio`: `Content-Type: application/octet-stream`,
`X-Audio-Sequence` entero desde 1. Body PCM16 little endian mono 16 kHz, longitud
par 2..32000. Un chunk representa como máximo un segundo. 202 no garantiza
persistencia. 429 permite reintentar la misma secuencia; 409 rechaza duplicados y
saltos. No reenviar después de timeout sin inspeccionar la sesión. No hay checksum,
idempotency-key ni recuperación durable. Un emisor activo por sesión.

## SSE (semántica adicional)

GET `/api/sessions/{session_id}/events`, `text/event-stream`:

- `subtitle`: JSON del shape Subtitle. `id` entero creciente local a la sesión.
  Sólo finals traducidos se guardan. Dedupe cliente por ID.
- `partial`: `{ "text": "hipótesis original" }`, sin ID, reemplaza la hipótesis
  visible. No se traduce, no se persiste como subtítulo ni se garantiza replay.
- `reset`: `{ "reason": "..." }`, ID 0. Vaciar vista: cursor fuera de retención
  o superior al contador actual. A continuación se reproduce el historial disponible.
- `session`: JSON Session terminal. Cerrar EventSource en ended/failed para evitar
  reconexiones inútiles.
- Comentario heartbeat cada 15 s; el cliente nativo reconecta si la conexión cae.

Last-Event-ID recupera entradas estrictamente posteriores al cursor. Sin cursor
se emiten los últimos 200 subtítulos (con reset si hubo truncamiento). No hay
historial completo, exactly-once ni garantía tras reinicio. Snapshots cada 200 ms;
los observadores lentos no frenan transcripción. Se limita cada escritura a 5 s.
OpenAPI representa el stream como body opaco; estos eventos no son operaciones HTTP
independientes. WebSocket sólo se usa hacia Gemini, no como endpoint público.

## OBS Browser Source

GET `/obs/{session_id}` devuelve una página HTML transparente que se puede agregar
como Browser Source a una escena OBS. Consume los eventos `partial`, `subtitle` y
`session` del SSE de esa sesión. La publicación RTMP hacia MediaMTX y la extracción
FFmpeg viven en el feeder `cmd/obsfeed`, no son operaciones de la API Go.

## Validación y documentación

CI valida Smithy, ejecuta generación y valida OpenAPI independientemente. Falla si
el artefacto versionado difiere. Tests comparan rutas Go con el spec en ambas
direcciones; contract-check valida respuestas reales del contenedor. Esto no prueba
todos los límites posibles; agregar regresiones cuando cambie el contrato.

`./scripts/dev docs` valida y muestra las fuentes de documentación. Markdown y
OpenAPI generado permiten agregar Scalar/Swagger UI después; no hay portal extra.
Referencia: [Smithy → OpenAPI](https://smithy.io/2.0/guides/model-translations/converting-to-openapi.html).

## Idioma por fragmento

`Session.language` es un hint informativo (`auto`, `en`, `es`) compatible con
clientes anteriores; no se usa como source global. Cada Subtitle lleva segment_id,
parent_segment_id opcional (subdivisión), language (`es/en/mixed/unknown`),
language_confidence (probabilidad elegida, 0 si no disponible), requires_translation,
decision_provider y decision_fallback opcional. SSE y consulta de historial
comparten el mismo schema. Los subtítulos contienen `original`, `english` y
`spanish`: para es/en el texto se conserva en su columna y Google Translation
completa la otra. Mixed residual/unknown usa source autodetect; si Google detecta
español, se solicita también inglés. No se garantiza resolver code-switching
dentro de un fragmento.
El servicio privado Laya usa su API upstream; no añade endpoints al backend público.
