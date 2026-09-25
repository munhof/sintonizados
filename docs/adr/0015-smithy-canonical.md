# ADR 0015 — Smithy canónico y OpenAPI derivado

Status: Accepted

## Context

Dos contratos editados a mano divergirían.

## Decision

Model-first: editar Smithy y generar OpenAPI 3.1.0 con plugin oficial. SSE documentado además en Markdown.

## Alternatives considered

Mantener YAML y Smithy independientes; convertir formatos manualmente.

## Consequences

No editar artefacto generado. CI valida modelo, generación y resultado.

## Future evolution

Conectar herramientas de documentación y clientes al OpenAPI generado.
