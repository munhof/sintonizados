# ADR 0003 — Container-first sin Compose como fundamento

Status: Accepted

## Context

La demo debe correr con podman build y podman run y migrar a Cloud Run.

## Decision

Una imagen principal con configuración por environment; sin Compose ni Kubernetes.

## Alternatives considered

Compose obligatorio; cluster Kubernetes para el MVP.

## Consequences

Operación local simple; conexiones externas configuradas explícitamente.

## Future evolution

Compose opcional si aparecen componentes locales, conservando comandos OCI independientes.
