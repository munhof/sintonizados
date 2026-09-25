# MVP y aceptación

## Implementado en este bootstrap

- [x] Go, sesiones aisladas y pipeline por eventos.
- [x] Emisor PCM progresivo y dos fuentes simultáneas.
- [x] Proveedores Gemini Live/Google Translation detrás de puertos.
- [x] UI lista/detalle, originales, español y SSE con replay acotado.
- [x] Timestamps, latencia, logs y métricas por sesión.
- [x] Smithy 2.0 → OpenAPI 3.1.0 versionado; validador independiente.
- [x] Scripts OCI, Containerfile, CI y ADRs.
- [x] Interfaces de razonamiento y decisión sin cargar Gemma/Laya.

## Aceptación real pendiente

- [ ] Autenticar Google, confirmar modelo/cuotas y validar dos sesiones simultáneas
  con audio real, calidad y latencia: [issue #1](https://github.com/munhof/sintonizados/issues/1).
- [ ] Desplegar la demo de Cloud Run y comprobar SSE desde un navegador externo:
  [issue #2](https://github.com/munhof/sintonizados/issues/2).
- [ ] Publicar cambios y comprobar GitHub Actions remoto: [issue #11](https://github.com/munhof/sintonizados/issues/11).
- [x] Prueba de navegador y revisión visual en desktop y móvil.
- [ ] Auditar teclado/lector de pantalla e incorporar feedback de accesibilidad:
  [issue #3](https://github.com/munhof/sintonizados/issues/3).

Los endpoints del contrato están implementados; esto no significa que la aceptación
real con proveedores esté terminada. El modo demo entrega frases programadas.

## Evidencia local

Ver `docs/operations/verification.md` para comandos/resultados observados. No hay
URLs de CI o despliegue verificadas todavía. Las tareas de aceptación y evolución
se siguen en los issues enlazados abajo; no equivalen a funcionalidades implementadas.

## Roadmap

1. Renovación Live, reconexión y control de costos/cuotas: [issue #4](https://github.com/munhof/sintonizados/issues/4).
2. Gemma para entidades y glosarios con enriquecimiento asincrónico: [issue #5](https://github.com/munhof/sintonizados/issues/5).
3. Laya para pre-routing y output gating: [issue #6](https://github.com/munhof/sintonizados/issues/6).
4. Store y bus distribuidos con ownership, leases, idempotencia y publicación durable:
   [issue #7](https://github.com/munhof/sintonizados/issues/7). Afinidad opcional,
   nunca requisito de consistencia.
5. Fuentes en vivo de escenario/micrófono (OBS/RTMP/HLS): [issue #8](https://github.com/munhof/sintonizados/issues/8).
   Idiomas adicionales y exportación SRT/VTT: [issue #9](https://github.com/munhof/sintonizados/issues/9).
6. Retención, privacidad, límites de carga y pruebas operativas: [issue #10](https://github.com/munhof/sintonizados/issues/10).

## Definition of Done API

Smithy actualizado + validación correcta + OpenAPI regenerado + Go implementado +
tests actualizados + CI pasando. Una capacidad sólo modelada sigue pendiente.
