# ADR 0006 — Eventos con bus intercambiable

Status: Accepted

## Context

Transcripción y traducción deben evolucionar independientemente.

## Decision

Puerto EventBus; TranscriptFinal se consume en otro worker. Eventos de observación para ingreso, publicación y latencia.

## Alternatives considered

Invocar traductor directamente desde el adaptador Gemini; instalar Kafka de entrada.

## Consequences

Bus local fiable para el consumidor crítico y de mejor esfuerzo para observadores; no hay durabilidad.

## Future evolution

Mensajes versionados, particiones, ACK, idempotencia y outbox al adoptar broker.
