# Desarrollo y demo local

1. `./scripts/dev run` construye y ejecuta con `.env.example`.
2. Abrir `http://localhost:8080`.
3. Ejecutar los dos comandos `feed -demo` del README y visitar ambas charlas.
4. Revisar originales/español, latencia y estado final.
5. Detener con Ctrl-C. IDs y resultados desaparecen con el proceso.

Comprobaciones: `./scripts/dev test`, `lint`, `smoke`, `openapi`, `contract`.
`fmt` aplica gofmt; `tidy` mantiene go.mod/go.sum. `build` genera binarios en build/.
No se requiere Make. Las imágenes/cachés OCI persisten para acelerar posteriores
corridas. El wrapper asume Podman local Linux; `feed` usa network host y no tiene
compatibilidad comprobada con Podman Machine en macOS/Windows.

## Audio real

Copiar `.env.example` a `.env`; cambiar modo y claves. Usar un archivo de voz con
permiso de uso. El emisor no envía encabezados WAV ni interpreta MP3: requiere raw
PCM s16le, un canal, 16000 Hz. Convertir con FFmpeg en un contenedor:

```sh
mkdir -p .local-audio
podman build -f containers/Audio.Containerfile -t localhost/sintonizados-audio:dev .
podman run --rm --userns=keep-id -v "$PWD:/data:Z" localhost/sintonizados-audio:dev \
  -i /data/.local-audio/talk.wav -ac 1 -ar 16000 -f s16le /data/.local-audio/talk.pcm
ENV_FILE=.env ./scripts/dev feed -session talk-1 -title 'Talk 1' \
  -file /data/.local-audio/talk.pcm
```

Ejecutar estos comandos desde la raíz del repositorio. `.local-audio/` está
excluida de Git; el WAV y el PCM permanecen en tu máquina.

Enviar un archivo corto (<8 minutos). El emisor envía 100 ms por chunk y espera
su duración después de cada aceptación; la red puede hacerlo algo más lento que
el original. No hay reconexión automática de proveedor. Si feed falla, consultar
estado y cerrar la sesión; si el error ocurrió tras entrega incierta, no reenviar
el mismo chunk a ciegas. Dos archivos se ejecutan en procesos feed separados.

## Diagnóstico

- 401: token de operador incorrecto (no es la clave Gemini).
- 409: ID repetido, sesión cerrada o secuencia duplicada/faltante.
- 429: cola llena o MAX_SESSIONS agotado (incluye sesiones finalizadas).
- failed: revisar logs estructurados; credenciales/modelo/cuota/red del proveedor.
- SSE: proxy debe permitir streaming sin buffering; reconexión recupera sólo 200.
- Podman bloqueado por sandbox: ejecutar desde terminal autorizada con acceso al
  runtime rootless; no instalar Go/Java como solución alternativa.

OBS, micrófono, RTMP y HLS deben adaptarse al mismo flujo PCM por sesión. Son futuros.
