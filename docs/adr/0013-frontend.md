# ADR 0013 — Frontend mínimo servido por Go

Status: Accepted

## Context

El visor sólo necesita lista, detalle y actualizaciones sin recarga.

## Decision

Templates embebidos, CSS y vanilla JS con EventSource.

## Alternatives considered

React/Next y una segunda toolchain de build.

## Consequences

Un artefacto desplegable; SSE y fetch requieren pruebas en navegador; sin portal de documentación pesado.

## Future evolution

Mejoras de accesibilidad y controles de lectura; framework sólo si aparece una necesidad concreta.
