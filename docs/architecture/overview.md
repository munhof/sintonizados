# Arquitectura

```mermaid
flowchart TB
  Feed[Fuente PCM / cmd/feed] --> HTTP[Go HTTP adapter]
  OBS[OBS Studio / RTMP] --> MTX[MediaMTX OCI]
  MTX --> FF[FFmpeg + obs-feed OCI]
  FF --> HTTP
  HTTP --> App[Application / Processing Gateway]
  App --> Store[SessionStore / memoria]
  App --> T[Transcriber / Gemini Live]
  T --> Bus[EventBus / memoria]
  Bus --> Worker[Consumidor de TranscriptFinal]
  Worker --> Pre[DecisionEngine.Decide / idioma por fragmento]
  Pre --> Translator[Translator / Google Translation]
  Translator --> History[Historial / subtítulos / latencia]
  Pre -->|es: conservar| History
  History --> SSE[SSE / audiencia]
  SSE --> Overlay[Browser Source transparente / OBS]
  Model[Smithy 2.0] --> OpenAPI[OpenAPI 3.1 generado]
  Model -. contrato externo .-> HTTP
  Gemma[Gemma futuro] -. enriquecimiento .-> Store
  Pre -->|HTTP| Laya[Laya Multilingual oficial / OCI]
```

El Processing Gateway es el servicio de aplicación dentro del proceso Go, sin
una segunda API o microservicio prematuro. Los adaptadores implementan puertos
del dominio. No se genera servidor Go desde Smithy.

| Directorio | Responsabilidad |
| --- | --- |
| `internal/domain` | Sesiones, conocimiento, eventos, puertos de IA y estado |
| `internal/application` | Ciclo de vida y pipeline por sesión |
| `internal/adapters/memory` | Store con snapshots y bus con backpressure |
| `internal/adapters/laya` | Decisiones HTTP del servidor Laya oficial |
| `internal/adapters/google` | WebSocket Gemini y REST Translation |
| `internal/adapters/http` | DTO, validación de entrada, auth y vistas embebidas |
| `api/smithy` | Contrato público; no importado por el dominio |

Memoria y bus son deliberadamente locales. Reemplazar sólo el store no basta
para múltiples réplicas: hay que resolver ownership del stream, secuencias,
entrega durable y fanout. Ver [sesiones](sessions.md) y [eventos](events.md).
