# ADR 0008 — Google Translation como único Translator

Status: Accepted

## Context

El MVP necesita inglés a español con bajo riesgo y sin routing prematuro.

## Decision

Cloud Translation Basic REST v2; solicitud estructurada con contexto reservado; decisión siempre standard.

## Alternatives considered

Usar LLM generalista en cada frase; implementar varios traductores; Translation Advanced desde el inicio.

## Consequences

API key y API habilitada; Basic no consume glosario ni contexto semántico adicional.

## Future evolution

Advanced o adaptadores de localización cuando haya medición que justifique contexto/glosarios.
