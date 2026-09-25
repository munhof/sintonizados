$version: "2"
namespace sintonizados.api
use aws.protocols#restJson1
use smithy.api#httpBearerAuth

/// Sintonizados conference accessibility gateway. See api/README.md for SSE and PCM semantics.
@title("Sintonizados")
@restJson1
@httpBearerAuth
service Sintonizados {
    version: "0.1.0"
    operations: [Health, ListSessions, GetSession, CreateSession, EndSession, IngestAudio, GetSubtitles, StreamSubtitles, Metrics, Home, TalkPage]
}

@readonly
@auth([])
@http(method: "GET", uri: "/health", code: 200)
operation Health { output: HealthOutput }
structure HealthOutput {
    @required
    status: String
    @required
    mode: String
}

@readonly
@auth([])
@http(method: "GET", uri: "/", code: 200)
operation Home { output: HTMLResponse }
@readonly
@auth([])
@http(method: "GET", uri: "/talks/{session_id}", code: 200)
operation TalkPage { input: SessionInput, output: HTMLResponse, errors: [NotFound] }
@mediaType("text/html")
string HTML
structure HTMLResponse { @required @httpPayload body: HTML }

@readonly
@http(method: "GET", uri: "/metrics", code: 200)
operation Metrics { output: MetricsOutput, errors: [Unauthorized] }
@mediaType("text/plain")
string MetricsText
structure MetricsOutput { @required @httpPayload body: MetricsText }
