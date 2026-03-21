const grpc = require("@grpc/grpc-js");

/**
 * Maps a gRPC error to the appropriate HTTP status code and a clean message.
 * Used by every route handler so error semantics are consistent.
 *
 * gRPC code  → HTTP status
 * ─────────────────────────
 * INVALID_ARGUMENT  → 400
 * NOT_FOUND         → 404
 * ALREADY_EXISTS    → 409
 * INTERNAL          → 500
 * UNAVAILABLE       → 503
 * (everything else) → 500
 */
function grpcErrorToHttp(err) {
  const map = {
    [grpc.status.INVALID_ARGUMENT]: { status: 400, message: err.details },
    [grpc.status.NOT_FOUND]:        { status: 404, message: err.details },
    [grpc.status.ALREADY_EXISTS]:   { status: 409, message: err.details },
    [grpc.status.INTERNAL]:         { status: 500, message: "Internal service error" },
    [grpc.status.UNAVAILABLE]:      { status: 503, message: "Service unavailable" },
  };

  return map[err.code] ?? { status: 500, message: "Unexpected error" };
}

module.exports = { grpcErrorToHttp };