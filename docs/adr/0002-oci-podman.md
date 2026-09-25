# ADR 0002 — OCI y Podman como baseline operativo

Status: Accepted

## Context

El host garantiza Podman, git y shell, no compiladores.

## Decision

Ejecutar Go, Smithy y validadores dentro de OCI mediante scripts/dev.

## Alternatives considered

Instalar SDKs en el host; Make obligatorio.

## Consequences

Primer uso necesita red y almacenamiento para imágenes; no exige Java/Go/Python locales.

## Future evolution

Fijar digests y locks transitivos para releases más reproducibles.
