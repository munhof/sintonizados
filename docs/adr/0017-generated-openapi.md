# ADR 0017 — Generación y versionado de OpenAPI

Status: Accepted

## Context

Jurados y usuarios deben poder inspeccionar el contrato en GitHub sin herramientas.

## Decision

Versionar api/generated/openapi/sintonizados.openapi.json, derivado únicamente de smithy build; CI rechaza diferencias.

## Alternatives considered

Generar sólo en releases; mantener un OpenAPI manual.

## Consequences

Diffs generados pueden ser grandes; se revisan junto con Smithy. Smithy sigue siendo fuente de verdad.

## Future evolution

Publicar artefactos de release o portal estático cuando sea necesario.
