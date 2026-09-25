# ADR 0005 — Afinidad como optimización

Status: Accepted

## Context

Cloud Run ofrece afinidad de mejor esfuerzo y puede reemplazar instancias.

## Decision

No depender de sticky sessions. MVP de un proceso; antes de varias réplicas distribuir coordinación y estado.

## Alternatives considered

Escalar memoria local con cookies de afinidad.

## Consequences

Reinicios pierden sesiones y hay restricciones de despliegue de demo. No se promete HA.

## Future evolution

Afinidad podrá reducir red cuando cualquier réplica pueda recuperar el estado correcto.
