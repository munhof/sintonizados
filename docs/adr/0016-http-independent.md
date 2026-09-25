# ADR 0016 — HTTP Go independiente del modelo

Status: Accepted

## Context

Smithy no proporciona automáticamente el servidor Go que necesita el proyecto.

## Decision

Implementar handlers y DTO en adaptador HTTP; dominio sin detalles de Smithy o proveedor.

## Alternatives considered

Generador experimental obligatorio; estructuras de contrato dentro del dominio.

## Consequences

Tests de rutas y respuestas protegen coherencia; la implementación manual exige disciplina model-first.

## Future evolution

Evaluar generación auxiliar estable sin alterar los puertos del dominio.
