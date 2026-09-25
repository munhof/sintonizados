# ADR 0019 — Laya Multilingual oficial e idioma por fragmento

Estado: aceptado. Sustituye el contrato Before/After de ADR 0010 para el MVP.
Issue: [#12](https://github.com/munhof/sintonizados/issues/12).

## Context

Una charla puede alternar español/inglés e incluir ambos en un final de Gemini.
El idioma declarado de sesión no permite decidir correctamente qué traducir.
El usuario seleccionó https://github.com/NandhaKishorM/laya como implementación oficial.

## Decision

Usar su servidor `laya-serve` sin forks, fijado al commit
`970dc8c5f63d7b886a68409493f37d569424f933` (0.3.20), en OCI CPU independiente.
El endpoint `/v1/systemone` es el protocolo que ofrece el propio upstream;
no incorpora una reimplementación de Jev. Precargar sólo multilingual y forzarlo
explícitamente en cada solicitud. Go consume HTTP detrás de `DecisionEngine.Decide`.

La etapa `classify` recibe TranscriptSegment y contexto; clasifica es/en/mixed/unknown.
`language_confidence` conserva la probabilidad de la opción elegida de Laya, no
su campo `confidence` basado en entropía. No se afirma calibración en conferencias.
`requires_translation` es true para es, en, mixed y unknown: las salidas soportadas
se muestran en inglés y español, conservando el original en su columna;
las preguntas `noul` son opcionales y no se solicitan en este MVP.

Español usa Google con es→en; inglés usa Google con en→es. Mixed se divide
una vez por puntuación (máximo ocho partes) y cada parte se reclasifica; nunca se
adivinan límites lingüísticos internos. Mixed residual/unknown omiten source para
usar autodetección de Google y registran fallback. Si Google informa que un segmento
unknown es español, se conserva en español y se solicita su traducción inglesa;
el metadato refleja `es`. El ejemplo sin puntuación
entre español e inglés puede seguir siendo mixed: no es segmentación semántica.

Errores/timeout de Laya usan DeterministicDecisionEngine (unknown, sin confianza
inventada), con fallback observable. No hay suposición global de inglés.
Session.language se mantiene compatible como hint informativo en/es/auto.
Gemini recibe languageCodes=[] para no sesgar el reconocimiento a inglés.

## Alternatives considered

- Python dentro de Go: descartado; rompe aislamiento operativo.
- Otro repositorio/fork o heurística presentada como Laya: descartado.
- Clasificación única por sesión: incorrecta ante code-switching.
- Segmentación semántica generativa: costo y complejidad fuera del MVP.
- Routing multimodelo/output gate ahora: fuera de alcance (#6).

## Consequences

Servicio y caché de pesos adicionales; descarga inicial y costo CPU. Laya comparte
un worker de inferencia: hay que medir capacidad antes de aumentar sesiones.
Timeout de decisión 5 s por solicitud y fallback; una subdivisión puede requerir
hasta ocho decisiones extra. La autodetección para mixed/unknown puede no resolver
code-switching dentro de un fragmento. Queda explícito, sin prometer cobertura total.
Los pesos los descarga el upstream de su bundle público Hugging Face; ver operación
para reproducibilidad, caché y evidencia. No se guarda texto/audio en logs propios.

## Future evolution

El request incluye Stage, contexto y resultado opcional de traducción; por ahora
sólo classify es soportado. Incorporar etapas de pre-routing y output-gate con
contratos y pruebas nuevos, Gemma fuera del camino crítico, segmentación mejorada,
evaluación de confianza sobre corpus bilingüe y presupuestos/colas por servicio.
