$version: "2"
namespace sintonizados.api
@length(min: 1, max: 64)
@pattern("^[a-zA-Z0-9_-]+$")
string SessionID
@length(min: 1, max: 200)
string Title
@timestampFormat("date-time")
timestamp Instant
@error("client") @httpError(400)
structure BadRequest { @required message: String }
@error("client") @httpError(401)
structure Unauthorized { @required message: String }
@error("client") @httpError(404)
structure NotFound { @required message: String }
@error("client") @httpError(409)
structure Conflict { @required message: String }
@error("client") @httpError(413)
structure TooLarge { @required message: String }
@error("client") @httpError(429)
structure Busy { @required message: String }
@error("server") @httpError(503)
structure Unavailable { @required message: String }
