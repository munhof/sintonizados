# ADR 0014 — Smithy como lenguaje de modelado

Status: Accepted

## Context

La interfaz pública necesita diseño explícito independiente de Go.

## Decision

Smithy IDL 2.0 como lenguaje del contrato; CLI y plugins 1.73.0 dentro de OCI.

## Alternatives considered

OpenAPI editado a mano; anotaciones Go code-first.

## Consequences

Se introduce tooling Java sólo en contenedores. El dominio no importa tipos Smithy.

## Future evolution

Generación de clientes si aporta valor; no es condición del MVP.
