# Runtime

Un servidor Go crea un runtime por sesión con una cola acotada de 32 chunks y
un contexto cancelable. `Transcriber.Run` consume audio ordenado. Publica finals
en el bus; un consumidor por sesión invoca decisión, traducción y publicación.
Las sesiones avanzan independientemente; el orden dentro de una sesión se conserva.

Ingestión acepta chunks de 2..32000 bytes pares, PCM16 LE mono 16 kHz. Las secuencias
empiezan en 1. Un salto o duplicado devuelve 409; cola llena devuelve 429 sin
consumir secuencia. 202 significa aceptado en memoria, no procesado ni durable.
Un fallo de transcripción/traducción lleva a `failed`; no hay traducción ficticia como fallback.

Para OBS local, MediaMTX recibe RTMP por `session_id`; el comando Go `obs-feed`
ejecuta FFmpeg en el mismo contenedor auxiliar, decodifica sólo la pista de audio,
la remuestrea a PCM16 mono 16 kHz y la publica en chunks de 100 ms por el endpoint
HTTP existente. El cliente puede aplicar backpressure ante 429 y cierra la sesión
al detener OBS. La vista `/obs/{session_id}` escucha el SSE existente y se agrega a
OBS como Browser Source transparente; no crea un canal alternativo de subtítulos.

El cierre deja de aceptar audio, drena transcripción y traducción y espera hasta
25 s en HTTP. Es idempotente al terminar. Un timeout HTTP no cancela automáticamente
el procesamiento; consultar estado. SIGTERM cancela proveedores, marca las sesiones
interrumpidas y da hasta 8 s para cerrar HTTP/SSE. No hay recuperación tras reinicio.

El servidor requiere token de operador incluso en demo. Lectura de charlas y
subtítulos es pública; escritura y métricas exigen bearer. No hay usuarios,
multi-tenancy, CORS abierto ni credenciales en el navegador.

Los límites actuales son por proceso: 16 sesiones retenidas por default (incluidas
finalizadas), 200 subtítulos por sesión, 20 transcripts de contexto, 32 chunks en
cola, 32 eventos por suscriptor y 2 MiB por mensaje de proveedor. No hay limpieza
TTL: reiniciar o aumentar MAX_SESSIONS conociendo el costo. Para exposición pública
masiva hacen falta cuotas de lectores/conexiones y controles de admisión adicionales.

### Cierre Gemini (issue #13)

EOF envía `audioStreamEnd` incluso sin audio pendiente. Se drenan finales hasta
`turnComplete`, cierre WebSocket normal o 10 s. La ausencia de un último final
puede ser silencio: se registra `final_wait_timeout` y se conservan los finales;
no se inventa un final a partir de parciales. Errores del proveedor y cancelación
siguen siendo errores distintos. El log `gemini_stream_closed` incluye session_id,
last_audio_timestamp, last_transcript_timestamp, stream_close_timestamp,
final_wait_duration y close_reason (`normal_stream_completion`,
`final_wait_timeout`, `provider_timeout`, `provider_error`, `context_cancellation`).
El deadline total de 9 minutos y la renovación pendiente (#4) son independientes.
