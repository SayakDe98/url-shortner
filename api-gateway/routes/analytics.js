const { Router } = require("express");
const analyticsClient = require("../grpc/analyticsClient");
const { grpcErrorToHttp } = require("../grpc/errors");

const router = Router();

/**
 * POST /track
 * Body: { code: string, url: string, ip?: string }
 *
 * Records a click and runs inline fraud detection.
 * The IP falls back to the request's remote address if not supplied in the body.
 */
router.post("/track", async (req, res) => {
  const { code, url } = req.body;
  const ip = req.body.ip || req.ip;

  if (!code || !url) {
    return res.status(400).json({ error: "code and url are required" });
  }

  try {
    const response = await analyticsClient.trackClick(code, url, ip);
    return res.status(200).json({
      message:     response.message,
      fraud_score: response.fraud_score,
    });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

/**
 * GET /stats/:code
 * Returns total click count for a short code.
 */
router.get("/stats/:code", async (req, res) => {
  try {
    const response = await analyticsClient.getStats(req.params.code);
    return res.status(200).json({ clicks: response.clicks });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

/**
 * POST /ai/detect
 * Body: { url: string, ip?: string }
 * Runs the fraud detection model on demand.
 */
router.post("/ai/detect", async (req, res) => {
  const { url } = req.body;
  const ip = req.body.ip || req.ip;

  if (!url) {
    return res.status(400).json({ error: "url is required" });
  }

  try {
    const response = await analyticsClient.detectMalicious(url, ip);
    return res.status(200).json({
      url:          response.url,
      fraud_score:  response.fraud_score,
      is_malicious: response.is_malicious,
    });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

/**
 * GET /ai/flagged
 * Returns all URLs that exceeded the fraud threshold.
 */
router.get("/ai/flagged", async (req, res) => {
  try {
    const response = await analyticsClient.getFlaggedURLs();
    return res.status(200).json({ flagged: response.flagged });
  } catch (err) {
    const { status, message } = grpcErrorToHttp(err);
    return res.status(status).json({ error: message });
  }
});

module.exports = router;