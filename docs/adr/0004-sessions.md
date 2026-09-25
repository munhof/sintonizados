# ADR 0004 — Sesiones como unidad de ejecución

Status: Accepted

## Context

Charlas concurrentes no pueden mezclar transcripts, contexto o subtítulos.

## Decision

ID estable y worker/cola por sesión; orden local y errores por sesión.

## Alternatives considered

Pipeline global; compartir un contexto de modelo entre charlas.

## Consequences

Aislamiento y concurrencia simple; se imponen límites de sesiones retenidas y colas.

## Future evolution

Ownership distribuido y particiones por session_id.
