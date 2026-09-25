# ADR 0018 — Conector OBS mediante RTMP y overlay Browser Source

Status: Accepted

## Context

La demo necesita recibir audio mezclado desde una escena OBS en vivo, enviarlo por
el pipeline PCM existente y mostrar subtítulos dentro de la grabación. OBS ya puede
publicar por RTMP y mostrar páginas web como Browser Source. El dominio Go no debe
depender del protocolo de publicación ni de FFmpeg.

## Decision

En desarrollo local, OBS publica a MediaMTX en un contenedor OCI por RTMP, con la
ruta identificada por `session_id`. `cmd/obsfeed` se ejecuta en un contenedor que
incluye FFmpeg, extrae y remuestrea sólo audio a PCM16 mono 16 kHz y lo envía al
endpoint HTTP de ingestión existente. `/obs/{session_id}` sirve una página
transparente que consume el SSE actual y se agrega como Browser Source a la escena.
MediaMTX queda fijado al tag `1.21.1` y el puerto de publicación se enlaza a loopback.

La API pública modela la página del overlay en Smithy; OpenAPI es derivado. El
ingestión RTMP no añade otro endpoint HTTP al backend ni expone detalles FFmpeg al
dominio.

## Alternatives considered

- Leer dispositivos de audio directamente desde Go: añadiría captura específica
  de plataforma al proceso principal.
- Controlar OBS con OBS WebSocket: permitiría control remoto, pero no entrega por
  sí mismo la mezcla de audio al pipeline de reconocimiento.
- Cargar un plugin de audio propio dentro de OBS: requiere instalación y empaquetado
  adicional para cada sistema operativo.
- Publicar sólo subtítulos por fuera de OBS: no los compone dentro del archivo
  grabado.

## Consequences

- Se conserva el límite HTTP/PCM entre las fuentes y el gateway Go.
- La recepción y conversión corren separadas del servidor y no cambian el dominio.
- La grabación sólo contiene subtítulos cuando la fuente Browser Source está activa
  en la escena que OBS graba.
- El procedimiento requiere el gateway, MediaMTX y `obs-feed`; el modo demo sigue
  mostrando frases programadas y no transcribe la voz capturada.
- Se comprobó el modelo y el código localmente; la aceptación con OBS del usuario y
  credenciales reales de Google queda pendiente.

## Future evolution

Agregar control de acceso/publish authorization a MediaMTX antes de aceptar fuentes
por una red compartida, tolerar reconexiones RTMP dentro de la misma sesión y evaluar
HLS u otros protocolos para escenarios remotos. No habilitar RTMP público en Cloud
Run sin rediseñar ingreso, autenticación, límites y estado de sesión.
