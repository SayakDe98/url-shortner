const path = require("path");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");

const PROTO_PATH = path.join(__dirname, "../proto/analytics.proto");

const packageDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const { analytics } = grpc.loadPackageDefinition(packageDef);

// Connects to the Python gRPC server on port 50052
const client = new analytics.Analytics(
  process.env.ANALYTICS_GRPC_TARGET || "localhost:50052",
  grpc.credentials.createInsecure()
);

/**
 * Generic promisifier for unary gRPC calls.
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
   * Record a click event and get back a fraud score.
   * @param {string} code  - short code
   * @param {string} url   - original long URL
   * @param {string} ip    - caller IP
   */
  trackClick: (code, url, ip) =>
    call("TrackClick", { code, url, ip }),

  /**
   * Get click count for a short code.
   * @param {string} code
   */
  getStats: (code) =>
    call("GetStats", { code }),

  /**
   * Run fraud detection on a URL + IP pair.
   * @param {string} url
   * @param {string} ip
   */
  detectMalicious: (url, ip) =>
    call("DetectMalicious", { url, ip }),

  /**
   * Fetch all flagged (high fraud score) URLs.
   */
  getFlaggedURLs: () =>
    call("GetFlaggedURLs", {}),
};