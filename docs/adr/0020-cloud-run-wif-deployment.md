# ADR 0020 — Cloud Run split services y GitHub WIF

Status: Accepted

## Context

El MVP ya contiene un gateway Go y un servicio oficial Laya en OCI. Necesita una
demo accesible desde Argentina y despliegues reproducibles tras CI, evitando
service account keys. El gateway conserva `SessionStore` y `EventBus` en memoria;
el clasificador Laya funciona técnicamente pero su calidad no está validada.
La licencia Apache 2.0 del checkpoint multilingual está identificada y el archivo
actual de pesos ocupa 644 MB.

## Decision

- Usar `southamerica-east1` como región default configurable.
- Publicar `sintonizados` en Cloud Run con lecturas públicas, operador autenticado,
  una instancia máxima y memoria local; no afirmar alta disponibilidad.
- Publicar `sintonizados-laya` como servicio separado y privado, con CPU, memoria,
  autoscaling y credencial propios. El gateway valida Cloud Run ID token en
  `X-Serverless-Authorization` y el bearer de Laya se conserva en `Authorization`.
- Poner claves runtime en Secret Manager. Gateway y Laya tienen service accounts
  distintas y sólo reciben los secretos que necesitan.
- Autenticar GitHub Actions con OIDC/WIF restringido a repository ID,
  `refs/heads/main` y GitHub Environment `production`. Sólo el deployer puede
  desplegar; no se crean claves JSON, Owner o Editor.
- Etiquetar y desplegar imágenes por SHA completo en Artifact Registry con tags
  inmutables, después del CI `verify` exitoso;
  `workflow_dispatch` exige también un CI exitoso para ese SHA.
- Mantener descarga del peso al iniciar con min instance 1. Evita añadir 644 MB a
  la imagen durante el MVP; se medirá readiness en Cloud Run. El snapshot HF
  observado no queda fijado por la receta actual y es un límite de reproducibilidad.
- No crear VM inicialmente. Evaluarla sólo ante fallos medidos de Cloud Run.

## Alternatives considered

- Compute Engine desde el inicio: descartado por la carga operativa y ausencia de
  evidencia de que Cloud Run no soporte este servicio CPU.
- GitHub secret con una clave JSON: descartado por credenciales persistentes.
- Gateway y Laya dentro de un único contenedor: descartado; separación facilita
  IAM, recursos y reemplazo del motor.
- Poner pesos en la imagen: diferido hasta medir startup; crece el artefacto y el
  cache del checkpoint no necesita incluirse para el MVP.
- Varias instancias gateway: descartado mientras sesiones/eventos no sean
  distribuidos (#7); sticky sessions no garantizan consistencia.

## Consequences

CD conserva las revisiones Cloud Run y despliega en orden Laya → readiness →
gateway. Min=1 y CPU siempre asignada sostienen el servicio listo pero generan
costo continuo; el runbook incluye scale-to-zero y teardown explícito. Las dos
instancias se limitan a una. Los health checks sólo validan que los procesos y el
modelo están disponibles, no exactitud de Laya ni aceptación bilingüe.

## Future evolution

Después de medir startup, CPU, memoria, latencia y calidad, decidir si se fija y
precarga el snapshot, se cambia el almacenamiento de pesos o se mueve Laya a otra
plataforma. Resolver SessionStore/EventBus distribuido antes de ampliar gateway;
añadir workers/PubSub sólo con un consumidor real.
