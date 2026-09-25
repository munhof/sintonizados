# Contexto del producto

Sintonizados nace para la Vibeathon de Nerdearla 2026. Quiere que una audiencia
pueda seguir charlas en inglés con originales y subtítulos en español. Varias
charlas ocurren simultáneamente y cada una tiene contexto propio.

El nombre une radio, conexión con una charla y Sintonía. El proyecto es abierto,
reproducible y orientado a sesiones. La demo mínima necesita dos fuentes de audio,
transcripción, traducción y un visor sin recargas, con latencia observable.

El jurado puede ejecutar el circuito sin cuentas usando modo demo. Ese modo usa
frases fijas y no prueba reconocimiento, traducción ni precisión. La aceptación
real requiere dos grabaciones habladas, acceso a Gemini y Cloud Translation y
una medición de calidad/latencia. El repositorio deja ambas rutas preparadas.

Las entradas en vivo incluyen micrófono de navegador y OBS local. OBS publica RTMP
a MediaMTX OCI; `obs-feed` usa FFmpeg para convertir la pista de audio a PCM y
enviarla al gateway. `/obs/{session_id}` es una Browser Source transparente para
componer originales y traducción en una escena/grabación OBS. La ruta está
implementada en el repositorio; la aceptación en la máquina del usuario con OBS y
proveedores Google debe comprobarse antes de afirmar que la demo real funciona.

La evolución apunta a transferencia de significado y conocimiento contextual,
no sólo sustitución de cadenas. Hoy se retienen 20 transcripts recientes por sesión.
Los contratos reservan glosarios, entidades y metadatos; Google Translation Basic
no aplica esos campos. Gemma podrá enriquecerlos asincrónicamente y Laya podrá
elegir estrategia cuando haya varios modelos. Laya ya clasifica fragmentos en un
servicio OCI Multilingual oficial; la evaluación real inicial no pasó y limita su
uso en vivo. Ver [resultados y límites](operations/laya.md#calidad-medida-en-este-equipo).
El routing multimodelo y output gate siguen futuros.

## Restricciones

- Go como núcleo, sin implementar ML dentro de Go.
- Podman + git + shell como baseline; herramientas extra dentro de OCI.
- Sesiones y eventos con puertos intercambiables; un proceso en el MVP.
- Smithy model-first; OpenAPI derivado y Go independiente del mecanismo de contrato.
- Sin Compose obligatorio, Kubernetes, brokers ni frameworks frontend pesados.
- Proveedores especializados antes que modelos generativos costosos.

Estado y criterios verificables: [planning/mvp.md](planning/mvp.md).
