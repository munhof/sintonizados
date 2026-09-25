# Eventos

Todo evento interno contiene ID, session ID, timestamp, correlation ID y payload.
El dominio define SessionStarted, AudioChunkReceived, TranscriptPartial,
TranscriptFinal, TranslationRequested, TranslationProduced, SubtitlePublished,
LatencyObserved, SessionEnded y TranscriptionComplete (reservado, no emitido hoy).

El desacoplamiento ejecutable principal es Transcriber → TranscriptFinal → consumidor
de traducción. El bus entrega ese evento con backpressure. La cola de audio es
local y directa; su evento notifica recepción sin duplicar bytes PCM en el bus.
Los eventos de traducción/publicación permiten observadores; no hay un worker
separado por cada nombre de evento.

MemoryEventBus permite suscripciones fiables (cola 32, bloquean con contexto cuando
se llenan) y observadores de mejor esfuerzo (descartan cuando se llenan). Suscribirse
antes de publicar es obligatorio; no hay replay, persistencia, ACK ni entrega entre
procesos. Cancelar suscripción elimina el registro; el consumidor no debe esperar
que el canal se cierre. No suscribir observadores fiables que no drenan eventos.

El visor SSE recupera snapshots del store, no depende de entrega de observadores;
así un lector lento no bloquea al pipeline. Cada subtitle tiene ID creciente por
sesión y su history está acotado a 200. Detalles de protocolo: [API](../../api/README.md).

Para sustituir por NATS/PubSub/Redis Streams: serializar payloads tipados/versionados,
particionar por sesión, ACK y retries, idempotencia por event_id, outbox para estado
+ evento y DLQ. Los timestamps locales no sustituyen el orden lógico. Ninguno de
esos componentes distribuidos es requisito del MVP.
