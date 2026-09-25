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
- Si `./scripts/dev run` indica que el nombre `sintonizados` ya existe, hay otro
  gateway ejecutándose. Para cargar la versión recién construida, primero detenelo
  con `podman stop sintonizados`; esto termina las sesiones en memoria de ese proceso.

## OBS por RTMP

El conector recibe la mezcla de audio que OBS publica, no requiere plugin de OBS ni
controla OBS remotamente. MediaMTX recibe RTMP en loopback y el contenedor `obs-feed`
usa FFmpeg para convertir la pista de audio a PCM16 mono 16 kHz y llamar al endpoint
de ingestión ya existente. El gateway sigue siendo Go; el receptor y el conversor
son procesos OCI reemplazables.

1. Arrancá el gateway con `ENV_FILE=.env ./scripts/dev run` y el receptor con
   `./scripts/dev obs-start`.
2. En OBS, **Settings → Stream → Service: Custom...**. Usá Server
   `rtmp://127.0.0.1:1935/live` y como Stream Key el `session_id` (sólo letras,
   números, guion o guion bajo). Iniciá **Start Streaming**.
3. En otra terminal, iniciá el feeder con ese ID y el mismo archivo de entorno:

   ```sh
   ENV_FILE=.env ./scripts/dev obs-feed -session charla-a -title 'Charla A'
   ```

4. En OBS, agregá una **Browser Source** con
   `http://127.0.0.1:8080/obs/charla-a`. Usá el tamaño del canvas (por ejemplo
   1920 × 1080). La fuente tiene fondo transparente y presenta el original, la
   traducción y la hipótesis parcial. Al grabar esa escena, OBS compone los
   subtítulos junto con cámara/presentación. El Browser Source debe estar activo
   en la escena durante la grabación.
5. Para terminar, detené **Start Streaming** o usá Ctrl-C en `obs-feed`; el feeder
   cierra la sesión y espera el drenado del pipeline. Para detener el receptor:
   `./scripts/dev obs-stop`.

`obs-feed` crea la sesión cuando se inicia, así que usá un ID que no exista desde
el último reinicio del gateway. OBS debe publicar antes de arrancar `obs-feed`; si
FFmpeg no encuentra el stream, revisá primero `./scripts/dev obs-status`, el ID de
sesión y la pista de audio enviada por OBS. Demo mode permite comprobar transporte,
pero produce texto programado; usá `PROVIDER_MODE=google` y credenciales locales
para verificar voz reconocida y traducción real. No subir `.env`.

`obs-start` publica sólo `127.0.0.1:1935`; el contenedor usa la imagen oficial
MediaMTX `bluenviron/mediamtx:1.21.1`. `obs-feed` comparte la red del host Linux
para alcanzar OBS y el gateway en loopback. Esta ruta local no está preparada para
recibir RTMP desde Internet ni está desplegada en Cloud Run. Para otra sesión,
usá otro ID/stream key y ejecutá otro `obs-feed`; MediaMTX acepta rutas RTMP por
sesión.

HLS, RTMP remoto seguro y fuentes de escenario alojadas siguen siendo evoluciones.
