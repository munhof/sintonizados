$version: "2"
namespace sintonizados.api
@readonly @auth([])
@http(method: "GET", uri: "/api/sessions", code: 200)
operation ListSessions { output: SessionsOutput }
structure SessionsOutput { @required sessions: Sessions }
list Sessions { member: Session }
@readonly @auth([])
@http(method: "GET", uri: "/api/sessions/{session_id}", code: 200)
operation GetSession { input: SessionInput, output: Session, errors: [NotFound] }
structure SessionInput { @required @httpLabel session_id: SessionID }
@http(method: "POST", uri: "/api/sessions", code: 201)
operation CreateSession { input: CreateInput, output: Session, errors: [BadRequest, Unauthorized, Conflict, Busy, Unavailable] }
structure CreateInput {
    @required session_id: SessionID
    @required title: Title
    @required language: Language
}
/// Session language is only a display hint; classification is per transcript segment.
enum Language {
    AUTO = "auto"
    ES = "es"
    EN = "en"
}
enum SessionStatus {
    ACTIVE = "active"
    ENDED = "ended"
    FAILED = "failed"
}
enum TranscriptionStatus {
    WAITING = "waiting"
    STREAMING = "streaming"
    COMPLETE = "complete"
    ERROR = "error"
}
structure Session {
    @required session_id: SessionID
    @required title: Title
    @required language: Language
    @required status: SessionStatus
    @required transcription_status: TranscriptionStatus
    @required mode: String
    @required created_at: Instant
    @required subtitle_count: Long
    error: String
}
@http(method: "POST", uri: "/api/sessions/{session_id}/end", code: 200)
operation EndSession { input: SessionInput, output: Session, errors: [Unauthorized, NotFound, Conflict, Unavailable] }
