# Google Cloud — integración preparada

No se ejecutó un despliegue remoto ni se probaron claves reales. El trabajo local
no requiere cuentas. Para completar la aceptación con Google falta acceso humano
a un proyecto con billing y a Gemini Developer API/Cloud Translation.

## Credenciales: qué autentica cada una

| Credencial | Uso |
| --- | --- |
| Login gcloud de operador | Crear recursos, subir imagen y desplegar |
| GEMINI_API_KEY | Gemini Developer API, WebSocket de transcripción |
| GOOGLE_TRANSLATION_API_KEY | Cloud Translation Basic REST; restringir a esa API |
| OPERATOR_TOKEN | Escrituras y métricas de Sintonizados; generar valor propio |
| Service account runtime | Leer los tres secretos en Cloud Run, sin clave JSON |

La implementación inicial de Google usa API keys, no ADC. Autenticarse con gcloud
no reemplaza las dos claves de proveedores. No crear claves descargables de service
accounts para este flujo. Las cuentas/cuotas/modelos deben estar habilitados en el
proyecto elegido.

## Autenticar y preparar

```sh
./scripts/dev gcloud auth login --no-launch-browser
export GOOGLE_CLOUD_PROJECT=tu-proyecto
export GOOGLE_CLOUD_REGION=us-central1
./scripts/cloud init
```

El CLI vive dentro de OCI; su login se guarda en el volumen local
`sintonizados-gcloud`, nunca en Git. El operador necesita permisos para habilitar
servicios, Artifact Registry, service accounts, IAM, Secret Manager y Cloud Run.
Billing se habilita desde la cuenta del usuario. `init` crea repositorio OCI y
service account del runtime; no despliega.

Obtener las claves en Google AI Studio/Google Cloud y crear estos secretos mediante
la consola de Secret Manager (pegar valores allí, no en comandos versionados):

- `sintonizados-operator-token`: token propio aleatorio.
- `sintonizados-gemini-key`: clave Gemini.
- `sintonizados-translation-key`: clave Cloud Translation Basic.

También se puede usar `./scripts/dev gcloud secrets create NAME --data-file=/workspace/credentials/NAME`
con un archivo local ignorado; no subir ese directorio. Si el secreto existe,
usar `secrets versions add` en lugar de `create`.

## Imagen y servicio

```sh
export IMAGE_TAG=mvp
./scripts/cloud push
./scripts/cloud deploy
./scripts/cloud status
```

`push` obtiene un token efímero y lo pasa por stdin a Podman, con authfile temporal
que se elimina. `deploy` otorga acceso a cada secreto al runtime y configura Cloud
Run. No se invoca desde CI. Usa una instancia como objetivo, CPU siempre disponible
para workers, 512 MiB, concurrencia 80 y timeout de 3600 s para SSE.
La lectura es pública; escritura/métricas siguen requiriendo bearer de operador.
El mínimo de una instancia y CPU permanente generan costo aun sin audiencia.

El límite de instancias no garantiza continuidad de memoria: reemplazos y revisiones
pueden coexistir transitoriamente. No hacer rollout durante una charla. Si eso no es
aceptable, completar la arquitectura distribuida antes de usar Cloud Run en producción.
Referencias: [deploy](https://docs.cloud.google.com/sdk/gcloud/reference/run/deploy),
[secretos](https://docs.cloud.google.com/run/docs/configuring/services/secrets) y
[afinidad](https://docs.cloud.google.com/run/docs/configuring/session-affinity).

## Aceptación luego de autenticar

Configurar `.env` local con OPERATOR_TOKEN igual al secreto y dos archivos hablados.

```sh
ENV_FILE=.env ./scripts/dev feed -url https://URL-DEL-SERVICIO -session real-a -title 'Charla A' -file /data/a.pcm &
pid_a=$!
ENV_FILE=.env ./scripts/dev feed -url https://URL-DEL-SERVICIO -session real-b -title 'Charla B' -file /data/b.pcm &
pid_b=$!
wait "$pid_a"
wait "$pid_b"
```

Verificar ambas páginas, originales, traducción, errores y timestamps; guardar
resultados sin secretos en `docs/planning/mvp.md`. No sustituir esta prueba por el
modo demo ni por los mocks. Modelo Live con ventana de 9 minutos en el adaptador;
reconexión/renovación automática queda pendiente.

## GitHub CI

`.github/workflows/ci.yml` no necesita secretos de Google y corre en push/PR. El
bootstrap deja el workflow listo; su ejecución remota requiere publicar los cambios
con una identidad GitHub autorizada. No se ha comprobado una corrida remota.
