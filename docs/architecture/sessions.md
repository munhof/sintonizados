# Sesiones y conocimiento

`session_id` estable, elegido por el operador, identifica una charla. Se restringe
a 1..64 caracteres alfanuméricos, guion o guion bajo. El título admite 1..200
caracteres. El campo language acepta `en`, `es` o `auto` como hint informativo
compatible; no determina el idioma de los subtítulos. Español es el destino fijo.
Cada TranscriptSegment recibe su idioma y confianza de DecisionEngine; la sesión
puede alternar idiomas. Subdivisiones mixed conservan parent_segment_id.

`SessionStore` proporciona Create/Get/List/Update. MemorySessionStore usa exclusión
mutua y devuelve copias de snapshots, incluyendo mapas y listas de conocimiento.
Cada sesión retiene idioma, estado, error público genérico, conteo, últimos 200
subtítulos y un contexto de hasta 20 transcripts recientes. `SessionKnowledge`
incluye slots para tópico, terminología, entidades, glosario y metadatos.

Estados: `active` → `ended` o `failed`. Transcripción: `waiting` → `streaming` →
`complete` o `error`. Crear devuelve antes de que el handshake del proveedor
termine; consultar detalles permite observar fallos de setup.

## Escalar

Dentro de un proceso: aumentar sesiones sujeto a cuotas del proveedor, conexiones,
memoria y latencia; medir dos, cuatro y N fuentes antes de ajustar MAX_SESSIONS.
Los lectores SSE consultan snapshots cada 200 ms; esa estrategia necesita medición
y posterior fanout para audiencias grandes.

Entre procesos: RedisSessionStore/FirestoreSessionStore requieren una nueva operación
atómica/versionada que sustituya el callback Update local, leases de ownership de
transcripción, secuencia y deduplicación compartidas, bus durable por sesión y
replay compartido. Los modelos no se copian entre réplicas: se enruta cada sesión
al worker propietario mediante coordinación explícita.

Sticky sessions puede mejorar localidad, pero nunca será la garantía de corrección.
El MVP no admite varias réplicas y pierde estado al reiniciar. La interfaz actual
marca el límite; no equivale a un store distribuido implementado.
