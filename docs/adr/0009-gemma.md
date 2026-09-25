# ADR 0009 — Gemma fuera del camino crítico

Status: Accepted

## Context

Enriquecer contexto puede aportar significado pero añade costo, latencia y operación.

## Decision

Definir ReasoningEngine sin implementar Gemma ni invocarlo en cada subtítulo.

## Alternatives considered

Gemma obligatorio entre transcripción y traducción.

## Consequences

El MVP funciona sin pesos locales ni GPU; enriquecimiento sigue pendiente.

## Future evolution

Consumidor asincrónico de transcripts con snapshots versionados de SessionKnowledge.
