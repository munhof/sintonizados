# ADR 0008 — Google Translation como único Translator

Status: Accepted

## Context

El MVP debe completar subtítulos en inglés y español para charlas que pueden
alternar ambos idiomas. La detección ocurre por fragmento y el traductor no debe
depender de un idioma global de sesión.

## Decision

Usar Cloud Translation Basic REST v2 mediante el único puerto `Translator`.
Enviar source/target explícitos según cada fragmento (`en→es` o `es→en`). Para
fragmentos mixed/unknown, pedir autodetección a español; si se detecta español,
conservarlo y solicitar además su traducción al inglés. `SessionKnowledge` sigue
disponible en el contrato Go, sin afirmar que Basic consume ese contexto.

## Alternatives considered

Usar LLM generalista en cada frase; implementar varios traductores; Translation Advanced desde el inicio.

## Consequences

API key y API habilitada; Basic no consume glosario ni contexto semántico adicional.
Las etiquetas de idioma dependen del clasificador por fragmento o de la detección
del proveedor en fallback; no implican calidad lingüística validada.

## Future evolution

Advanced o adaptadores de localización cuando haya medición que justifique contexto/glosarios.
