$version: "2"
namespace sintonizados.api
/// Ordered PCM16 little-endian mono 16 kHz chunks, 2..32000 bytes, even length.
/// Sequence starts at 1. Retries of an accepted sequence return 409; never replay silently.
@http(method: "POST", uri: "/api/sessions/{session_id}/audio", code: 202)
operation IngestAudio { input: AudioInput, output: AudioAccepted, errors: [BadRequest, Unauthorized, NotFound, Conflict, TooLarge, Busy] }
structure AudioInput {
    @required @httpLabel session_id: SessionID
    @required @httpHeader("X-Audio-Sequence") sequence: Sequence
    @required @httpPayload audio: PCM
}
@range(min: 1)
long Sequence
@mediaType("application/octet-stream")
@length(min: 2, max: 32000)
blob PCM
structure AudioAccepted { @required sequence: Sequence }
