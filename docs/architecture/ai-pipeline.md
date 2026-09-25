# Pipeline de IA

| Función | Puerto | MVP | Evolución |
| --- | --- | --- | --- |
| Percepción | Transcriber | Gemini Live | Otros ASR |
| Decisión | DecisionEngine | Laya Multilingual HTTP / fallback determinista | Laya pre-router y output gate |
| Transformación | Translator | Google Translation Basic | Glosarios/localización |
| Razonamiento | ReasoningEngine | Sólo interfaz | Gemma asincrónico |

Una conexión Gemini por sesión envía PCM en tiempo real y recibe hipótesis y
finals. Sólo finals solicitan traducción. El modelo configurable por defecto es
`gemini-3.5-transcribe-live`; el adaptador usa su protocolo de transcripción
especializada. Cambiarlo por un modelo conversacional no garantiza compatibilidad.
Se revisó la [documentación oficial de transcripción](https://ai.google.dev/gemini-api/docs/live-api/live-transcribe).
El adaptador limita su conexión a 9 minutos, no implementa renovación y trata
`goAway`, desconexión anormal y 45 s sin respuestas como fallo. Tras EOF envía
audioStreamEnd y drena hasta 10 s; timeout final se registra sin fallar la sesión.
Ver motivos y timestamps de cierre en [runtime](runtime.md). Debe validarse acceso y comportamiento real con la cuenta de la demo.

`TranslationRequest` lleva texto, idiomas y `SessionKnowledge`. La implementación
Basic envía texto plano en el sentido requerido por fragmento: español → inglés,
inglés → español, o source omitido a español para autodetección en fallback. Si
Google detecta español en ese fallback, se conserva el original y se solicita
también su traducción al inglés. Decodifica entidades HTML de la respuesta;
los campos semánticos quedan disponibles para adaptadores futuros, sin afirmar que
Basic los use. Contrato externo del proveedor: [Cloud Translation v2](https://docs.cloud.google.com/translate/docs/reference/rest/v2/translate).
La clave se envía en [X-Goog-Api-Key](https://docs.cloud.google.com/docs/authentication/api-keys-use).

`DecisionEngine.Decide` clasifica cada final mediante Laya oficial, modelo
multilingual forzado, con límite de 5 s. Español e inglés se conservan en su
columna y se traducen al otro idioma;
mixed se divide por puntuación una vez (hasta ocho partes), se reclasifica y se
mantiene el orden. Mixed residual/unknown usa autodetección de Google; si detecta
español se conserva el original y se solicita la traducción inglesa. Si Laya falla,
la decisión determinista devuelve unknown; logs y subtítulos registran el fallback.
No hay clasificación por sesión. Gemini usa languageCodes=[] para detección automática.
Probabilidad y proveedor pertenecen a cada TranscriptSegment, persistidos en Subtitle.
Los requests incluyen etapa/contexto para evolucionar a routing/gating, pero sólo
classify está implementado. Ver [ADR 0019](../adr/0019-laya-segment-language.md).

Gemma tampoco se invoca: enriquecerá snapshots versionados fuera del flujo crítico.

No se reintentan silenciosamente solicitudes con entrega incierta. El feeder sólo
reintenta 429, que indica que no se aceptó el chunk. Una caída de transcripción/traducción termina
la sesión y exige iniciar una nueva; reanudación y deduplicación durable son futuras.

## Medición

Se registra `audio_ingress_at`, `transcript_at`, `translation_at`, `published_at`
y milisegundos de transcripción, llamada de traducción y extremo a extremo.
Los relojes son del mismo proceso; la latencia de captura/red previa al ingreso y
la presentación efectiva en el navegador quedan fuera.

Gemini no entrega una correlación exacta chunk→utterance. Se usa el primer chunk
pendiente desde el último final; por eso la latencia de transcripción es una
**estimación por ventana**, que puede incluir silencios y buffers. No es latencia
por palabra. La traducción mide la llamada, mientras E2E incluye cola, decisión Laya y publicación.
`/metrics` expone contador de subtítulos y gauges del último valor por sesión;
no representa percentiles ni histogramas. Los logs JSON conservan cada observación
sin registrar el texto ni el audio.
