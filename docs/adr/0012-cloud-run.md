# ADR 0012 — Estrategia Google Cloud y Cloud Run

Status: Accepted

## Context

Se pide un camino de despliegue OCI con proveedores Google.

## Decision

Preparar Artifact Registry, Secret Manager y Cloud Run de instancia única para demo; CPU sin throttling.

## Alternatives considered

VM administrada a mano; GKE; despliegue multiinstancia con memoria local.

## Consequences

Requiere autenticación, proyecto con billing y roles; no hay despliegue comprobado. Reemplazos pueden perder estado.

## Future evolution

Distribuir sesiones/eventos antes de habilitar escalado horizontal y despliegues sin interrupción.
