const { Router } = require("express");
const grpcClient = require("../grpc/client");
const { grpcErrorToHttp } = require("../grpc/errors");

const router = Router();
const BASE_URL = process.env.BASE_URL || "http://localhost:3000";

/**
 * POST /shorten
 * Body: { url: string, expiry_minutes?: number }
 *
 * Creates a short URL via the Go gRPC service and returns the full short URL
 * along with its expiry timestamp.
 */
router.post("/shorten", async (req, res) => {
  const { url, expiry_minutes } = req.body;

  if (!url) {
    return res.status(400).json({ error: "url is required" });
  }

  try {
    const response = await grpcClient.shortenURL(url, expiry_minutes);

    return res.status(201).json({
      short_url:  `${BASE_URL}/${response.short_code}`,
      short_code: response.short_code,
      expires_at: response.expires_at, // google.protobuf.Timestamp { seconds, nanos }
    });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

/**
 * GET /:code
 * Resolves a short code and issues a 302 redirect to the original URL.
 * Returns 404 / 410 with a JSON body for API clients that don't follow redirects.
 */
router.get("/:code", async (req, res) => {
  const { code } = req.params;

  try {
    const response = await grpcClient.resolveURL(code);
    return res.redirect(302, response.url);
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

/**
 * DELETE /urls/:code
 * Soft-deletes a short URL in the Go service and evicts it from Redis.
 */
router.delete("/urls/:code", async (req, res) => {
  const { code } = req.params;

  try {
    const response = await grpcClient.deleteURL(code);
    return res.status(200).json({ message: response.message });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

module.exports = router;