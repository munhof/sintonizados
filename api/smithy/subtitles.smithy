$version: "2"
namespace sintonizados.api
@readonly @auth([])
@http(method: "GET", uri: "/api/sessions/{session_id}/subtitles", code: 200)
operation GetSubtitles { input: SessionInput, output: SubtitlesOutput, errors: [NotFound] }
structure SubtitlesOutput { @required subtitles: Subtitles }
list Subtitles { member: Subtitle }
enum SegmentLanguage {
 ES = "es"
 EN = "en"
 MIXED = "mixed"
 UNKNOWN = "unknown"
}
@range(min: 0, max: 1)
double LanguageConfidence
structure Subtitle {
    @required segment_id: String
    parent_segment_id: String
    @required language: SegmentLanguage
    /// Selected choice probability, not proof of language accuracy; 0 when unavailable.
    @required language_confidence: LanguageConfidence
    @required requires_translation: Boolean
    @required decision_provider: String
    /// Explicit fallback strategy when classification or subdivision is inconclusive.
    decision_fallback: String
    @required id: Long
    @required session_id: SessionID
    @required correlation_id: String
    @required original: String
    @required spanish: String
    @required final: Boolean
    @required audio_ingress_at: Instant
    @required transcript_at: Instant
    @required translation_at: Instant
    @required published_at: Instant
    @required transcription_ms: Double
    @required translation_ms: Double
    @required end_to_end_ms: Double
}
/// SSE event 'subtitle', JSON Subtitle payload, id is session-local sequence.
/// Replays retained last 200 subtitles; Last-Event-ID resumes. Heartbeats every 15 s.
/// See api/README.md for reset events, retention, reconnect and partial transcript semantics.
@readonly @auth([])
@http(method: "GET", uri: "/api/sessions/{session_id}/events", code: 200)
operation StreamSubtitles { input: StreamInput, output: StreamOutput, errors: [BadRequest, NotFound] }
structure StreamInput {
    @required @httpLabel session_id: SessionID
    @httpHeader("Last-Event-ID") last_event_id: String
}
@streaming @mediaType("text/event-stream")
blob EventStream
structure StreamOutput { @required @httpPayload body: EventStream }
