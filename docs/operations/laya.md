# Laya Multilingual en Podman

Implementación oficial: https://github.com/NandhaKishorM/laya, commit fijado en
[Containerfile](../../containers/Laya.Containerfile), con dependencias de ejecución fijadas en
[lock](../../containers/laya-requirements.lock). Se usa `laya-serve` del propio
proyecto. No instalar Python ni ML en el host ni en el contenedor Go.

```sh
./scripts/dev laya-start
./scripts/dev laya-status
podman logs --tail 30 sintonizados-laya
```

El primer inicio descarga exclusivamente el checkpoint multilingual del bundle
`convaiinnovations/laya` en Hugging Face. Esperar status=ok y loaded=[multilingual] antes de probar;
no se necesitan claves Google para este servicio. La imagen usa PyTorch CPU,
usuario 10001 y cuatro threads. El volumen `sintonizados-laya-cache` conserva los
pesos fuera de Git. El puerto HTTP está publicado sólo en 127.0.0.1:8090.

En `.env` conservar las credenciales existentes y agregar/cambiar:

```dotenv
PROVIDER_MODE=google
DECISION_ENGINE=laya
LAYA_URL=http://sintonizados-laya:8000
```

Después de finalizar las charlas activas, detener el gateway anterior y arrancarlo:

```sh
podman stop sintonizados
# Compatible con bash y fish:
env ENV_FILE=.env ./scripts/dev run
```

Reiniciar Go pierde sesiones en memoria. `run` conecta a la red Podman
`sintonizados`, donde el nombre `sintonizados-laya` se resuelve por DNS.
En un Go ejecutado fuera de esa red usar `http://127.0.0.1:8090`.
Abrir `/operator`, crear una sesión nueva y alternar frases completas en español
e inglés. Los metadatos aparecen en `/api/sessions/{id}/subtitles`; logs
`segment_decided` muestran idioma, probabilidad, proveedor y fallback sin texto.

`DECISION_ENGINE=deterministic` permite funcionar sin Laya: devuelve unknown y
Google autodetecta el origen. No equivale a clasificación local de idiomas.
Si Laya falla, el gateway hace ese fallback automáticamente. Cancelar la sesión
no dispara traducciones nuevas. No existe retry silencioso de inferencia.

## Operación directa, sin wrapper

```sh
podman network create sintonizados # una vez
podman build -f containers/Laya.Containerfile -t localhost/sintonizados-laya:dev .
podman run -d --name sintonizados-laya --network sintonizados \
  -p 127.0.0.1:8090:8000 \
  -v sintonizados-laya-cache:/home/laya/.cache/huggingface \
  localhost/sintonizados-laya:dev
```

El upstream permite `LAYA_API_KEY`: para autenticar entre servicios, poner el mismo
valor en `LAYA_API_KEY` del `.env` del gateway y en un env-file aparte de Laya,
por ejemplo `.laya.env`, e iniciar con `LAYA_ENV_FILE=.laya.env ./scripts/dev laya-start`.
El wrapper nunca pasa `.env` de Google al servicio Laya. Si cambian ese env-file
con un contenedor existente, eliminar/recrear el contenedor Laya para que tome el valor.
`./scripts/dev laya-stop` detiene sin borrar caché. Para actualizar una imagen,
detener y eliminar sólo el contenedor Laya y volver a ejecutar `laya-start`.
No usar Compose. No desplegar este puerto públicamente sin autenticación.

## Límites

El commit del software está fijado; el servidor upstream descarga los pesos de
la revisión disponible de Hugging Face durante el primer arranque. En el despliegue
verificado se descargó snapshot `55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851`;
la receta fija software/dependencias, pero actualmente no impone ese hash en un
volumen HF nuevo. La caché conserva
snapshots, pero un volumen nuevo puede recibir pesos actualizados. Para releases
reproducibles hay que fijar/exportar también ese snapshot; no afirmar un lock de
pesos que el entrypoint upstream todavía no aplica.

La probabilidad no demuestra exactitud. Evaluar frases cortas, nombres técnicos,
cambios dentro de una cláusula y dos charlas concurrentes. Un mixed residual usa
Google autodetect con limitaciones documentadas en ADR 0019. Routing adaptativo,
output gate y Gemma continúan pendientes.

## Calidad medida en este equipo

`./scripts/dev laya-smoke` ejecutó el checkpoint real multilingual en CPU: para
“Hola, ¿cómo estamos? ¿Todo bien?”, “We are going to talk about neural networks
and scientific research.” y una frase mixta, las tres respuestas fueron `unknown`
(con probabilidad elegida 0.46, 0.56 y 0.60; 0.15–0.28 s por decisión). Pruebas
adicionales con instrucciones en inglés/español y formatos de estado produjeron
clasificaciones equivocadas con frecuencia. El protocolo HTTP, checkpoint y adapter
funcionan; **la clasificación no está validada para traducir en vivo**.

Por eso la implementación conserva la respuesta/probabilidad del modelo, entrega
`unknown` a Cloud Translation sin fuente impuesta y usa `detectedSourceLanguage`
para preservar literalmente los casos que Google detecta como español. No hay
automatismo por sesión. Repetir la prueba después de cambiar el checkpoint o
plantilla; mejorar calidad requiere corpus etiquetado y medir por clase (confusión,
recall y latencia), no ajustar frases de demostración. No se validó aún Google real
con esta nueva ruta ni captura en vivo tras el cambio.

## Cloud Run

En Google Cloud Laya se ejecuta como `sintonizados-laya` privado, separado del
gateway Go. Configuración inicial: CPU, 2 vCPU, 4 GiB, concurrency 2, `min=1`,
`max=1`, CPU siempre asignada, `LAYA_THREADS=2`, multilingual precargado y bearer
`LAYA_API_KEY` desde Secret Manager. No recibe claves Gemini ni Translation.

La llamada del gateway adjunta un ID token de Cloud Run, con audiencia igual a la
URL Laya, mediante `X-Serverless-Authorization`; también adjunta el bearer Laya en
`Authorization`. El token ID proviene del metadata server y se cachea hasta un
minuto antes de expirar. En local `LAYA_CLOUD_AUDIENCE` queda vacío y se conserva
el comportamiento Podman documentado arriba.

El model card oficial identifica licencia Apache 2.0; el archivo multilingual
`model.safetensors` tiene 644 MB. Se decidió mantener la descarga a startup y
guardar los pesos en la capa writable efímera de Cloud Run, sin añadirlos a la
imagen OCI. Cloud Run mantiene una instancia mínima activa; el CD registra el
tiempo desde deploy hasta health con el modelo cargado. La revisión de la
aplicación Laya está fijada pero la descarga no fuerza el snapshot de pesos; el
SHA observado `55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851` no es un lock aplicado a
un cache vacío. La respuesta `/health` valida carga, no exactitud lingüística.
