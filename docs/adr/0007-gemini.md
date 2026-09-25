# ADR 0007 — Gemini detrás de Transcriber

Status: Accepted

## Context

Se necesita percepción de audio en vivo sin filtrar protocolo al dominio.

## Decision

Gemini Live con PCM y transcripción dedicada configurable detrás de Transcriber.Run.

## Alternatives considered

Procesar archivos completos; implementar ASR en Go; usar el modelo de traducción para reconocer.

## Consequences

Se necesita clave y acceso al modelo; pruebas locales no prueban disponibilidad real. Conexión limitada a 9 minutos.

## Future evolution

Renovación y nuevos adaptadores de ASR con las mismas pruebas de contrato.
