# ADR 0011 — MemorySessionStore y MemoryEventBus

Status: Accepted

## Context

Dos charlas pueden demostrarse sin infraestructura distribuida.

## Decision

Store concurrente con snapshots copiados; bus en memoria y retención acotada.

## Alternatives considered

Redis, Firestore, PubSub desde el inicio.

## Consequences

Datos efímeros y callbacks de actualización locales; reemplazar interfaces no implementa por sí solo distribución.

## Future evolution

Rediseñar update atómico/versionado y añadir leases, outbox y deduplicación al distribuir.
