# ADR 0001 — Go como lenguaje y orquestador

Status: Accepted

## Context

Hay que coordinar sesiones, red y concurrencia con una toolchain pequeña.

## Decision

Usar Go para HTTP, workers, eventos, estado y composición. ML permanece en proveedores.

## Alternatives considered

Python para todo; backend Node; múltiples servicios iniciales.

## Consequences

Binario estático y concurrencia nativa; clientes de proveedores requieren adaptadores explícitos.

## Future evolution

Agregar servicios ML externos sólo cuando una función lo requiera.
