# Pipeline de IA

| Función | Puerto | MVP | Evolución |
| --- | --- | --- | --- |
| Percepción | Transcriber | Gemini Live | Otros ASR |
| Decisión | DecisionEngine | Determinista | Laya pre-router y output gate |
| Transformación | Translator | Google Translation Basic | Glosarios/localización |
| Razonamiento | ReasoningEngine | Sólo interfaz | Gemma asincrónico |

Una conexión Gemini por sesión envía PCM en tiempo real y recibe hipótesis y
finals. Sólo finals solicitan traducción. El modelo configurable por defecto es
`gemini-3.5-transcribe-live`; el adaptador usa su protocolo de transcripción
especializada. Cambiarlo por un modelo conversacional no garantiza compatibilidad.
Se revisó la [documentación oficial de transcripción](https://ai.google.dev/gemini-api/docs/live-api/live-transcribe).
El adaptador limita su conexión a 9 minutos, no implementa renovación y trata
`goAway`, desconexión, 45 s sin respuestas o ausencia de final tras 10 s de cierre
como fallo. Debe validarse acceso y comportamiento real con la cuenta de la demo.

`TranslationRequest` lleva texto, idiomas y `SessionKnowledge`. La implementación
Basic envía texto plano inglés → español y decodifica entidades HTML de la respuesta;
los campos semánticos quedan disponibles para adaptadores futuros, sin afirmar que
Basic los use. Contrato externo del proveedor: [Cloud Translation v2](https://docs.cloud.google.com/translate/docs/reference/rest/v2/translate).
La clave se envía en [X-Goog-Api-Key](https://docs.cloud.google.com/docs/authentication/api-keys-use).

`DecisionEngine.Before` escoge el único traductor; `After` autoriza publicación.
La interfaz admite retry/escalate, pero el MVP falla explícitamente si un motor
solicita una estrategia todavía no soportada. Laya no tiene adaptador instalado.
Gemma tampoco se invoca: enriquecerá snapshots versionados fuera del flujo crítico.

No se reintentan silenciosamente solicitudes con entrega incierta. El feeder sólo
reintenta 429, que indica que no se aceptó el chunk. Una caída de proveedor termina
la sesión y exige iniciar una nueva; reanudación y deduplicación durable son futuras.

## Medición

Se registra `audio_ingress_at`, `transcript_at`, `translation_at`, `published_at`
y milisegundos de transcripción, llamada de traducción y extremo a extremo.
Los relojes son del mismo proceso; la latencia de captura/red previa al ingreso y
la presentación efectiva en el navegador quedan fuera.

Gemini no entrega una correlación exacta chunk→utterance. Se usa el primer chunk
pendiente desde el último final; por eso la latencia de transcripción es una
**estimación por ventana**, que puede incluir silencios y buffers. No es latencia
por palabra. La traducción mide la llamada, mientras E2E incluye cola y publicación.
`/metrics` expone contador de subtítulos y gauges del último valor por sesión;
no representa percentiles ni histogramas. Los logs JSON conservan cada observación
sin registrar el texto ni el audio.
