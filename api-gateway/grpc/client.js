const path = require("path");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");

const PROTO_PATH = path.join(__dirname, "../proto/urlshortener.proto");

// Load the proto with timestamp support
const packageDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: [
    path.join(__dirname, "../node_modules/google-proto-files"), // supplies google/protobuf/timestamp.proto
  ],
});

const { urlshortener } = grpc.loadPackageDefinition(packageDef);

// Single shared channel to the Go gRPC server — reused across all requests.
const client = new urlshortener.URLShortener(
  process.env.GRPC_TARGET || "localhost:50051",
  grpc.credentials.createInsecure()
);

/**
 * Wraps a gRPC unary call in a Promise so we can use async/await throughout.
 * @param {string} method - Name of the RPC method on the client stub.
 * @param {object} request - Protobuf request message fields.
 * @returns {Promise<object>} Resolved with the response message.
 */
function call(method, request) {
  return new Promise((resolve, reject) => {
    client[method](request, (err, response) => {
      if (err) reject(err);
      else resolve(response);
    });
  });
}

module.exports = {
  /**
   * Creates (or reactivates) a short URL.
   * @param {string}  url
   * @param {number}  [expiryMinutes=1440]
   * @returns {Promise<{ short_code: string, expires_at: object }>}
   */
  shortenURL: (url, expiryMinutes = 0) =>
    call("ShortenURL", { url, expiry_minutes: expiryMinutes }),

  /**
   * Resolves a short code to its original URL.
   * @param {string} code
   * @returns {Promise<{ url: string }>}
   */
  resolveURL: (code) =>
    call("ResolveURL", { code }),

  /**
   * Soft-deletes a short URL.
   * @param {string} code
   * @returns {Promise<{ message: string }>}
   */
  deleteURL: (code) =>
    call("DeleteURL", { code }),
};