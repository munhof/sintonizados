# Runtime

Un servidor Go crea un runtime por sesión con una cola acotada de 32 chunks y
un contexto cancelable. `Transcriber.Run` consume audio ordenado. Publica finals
en el bus; un consumidor por sesión invoca decisión, traducción y publicación.
Las sesiones avanzan independientemente; el orden dentro de una sesión se conserva.

Ingestión acepta chunks de 2..32000 bytes pares, PCM16 LE mono 16 kHz. Las secuencias
empiezan en 1. Un salto o duplicado devuelve 409; cola llena devuelve 429 sin
consumir secuencia. 202 significa aceptado en memoria, no procesado ni durable.
Un fallo de proveedor lleva a `failed`; no hay traducción ficticia como fallback.

El cierre deja de aceptar audio, drena transcripción y traducción y espera hasta
25 s en HTTP. Es idempotente al terminar. Un timeout HTTP no cancela automáticamente
el procesamiento; consultar estado. SIGTERM cancela proveedores, marca las sesiones
interrumpidas y da hasta 8 s para cerrar HTTP/SSE. No hay recuperación tras reinicio.

El servidor requiere token de operador incluso en demo. Lectura de charlas y
subtítulos es pública; escritura y métricas exigen bearer. No hay usuarios,
multi-tenancy, CORS abierto ni credenciales en el navegador.

Los límites actuales son por proceso: 16 sesiones retenidas por default (incluidas
finalizadas), 200 subtítulos por sesión, 20 transcripts de contexto, 32 chunks en
cola, 32 eventos por suscriptor y 2 MiB por mensaje de proveedor. No hay limpieza
TTL: reiniciar o aumentar MAX_SESSIONS conociendo el costo. Para exposición pública
masiva hacen falta cuotas de lectores/conexiones y controles de admisión adicionales.
