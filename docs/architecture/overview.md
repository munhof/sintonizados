# Arquitectura

```mermaid
flowchart TB
  Feed[Fuente PCM / cmd/feed] --> HTTP[Go HTTP adapter]
  HTTP --> App[Application / Processing Gateway]
  App --> Store[SessionStore / memoria]
  App --> T[Transcriber / Gemini Live]
  T --> Bus[EventBus / memoria]
  Bus --> Worker[Consumidor de TranscriptFinal]
  Worker --> Pre[DecisionEngine.Before]
  Pre --> Translator[Translator / Google Translation]
  Translator --> Post[DecisionEngine.After]
  Post --> History[Historial / subtítulos / latencia]
  History --> SSE[SSE / audiencia]
  Model[Smithy 2.0] --> OpenAPI[OpenAPI 3.1 generado]
  Model -. contrato externo .-> HTTP
  Gemma[Gemma futuro] -. enriquecimiento .-> Store
  Laya[Laya futuro] -. decisiones .-> Pre
  Laya -. gate .-> Post
```

El Processing Gateway es el servicio de aplicación dentro del proceso Go, sin
una segunda API o microservicio prematuro. Los adaptadores implementan puertos
del dominio. No se genera servidor Go desde Smithy.

| Directorio | Responsabilidad |
| --- | --- |
| `internal/domain` | Sesiones, conocimiento, eventos, puertos de IA y estado |
| `internal/application` | Ciclo de vida y pipeline por sesión |
| `internal/adapters/memory` | Store con snapshots y bus con backpressure |
| `internal/adapters/google` | WebSocket Gemini y REST Translation |
| `internal/adapters/http` | DTO, validación de entrada, auth y vistas embebidas |
| `api/smithy` | Contrato público; no importado por el dominio |

Memoria y bus son deliberadamente locales. Reemplazar sólo el store no basta
para múltiples réplicas: hay que resolver ownership del stream, secuencias,
entrega durable y fanout. Ver [sesiones](sessions.md) y [eventos](events.md).
