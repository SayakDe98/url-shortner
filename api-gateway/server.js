require("dotenv").config();
const express         = require("express");
const morgan          = require("morgan");
const urlRoutes       = require("./routes/urls");
const analyticsRoutes = require("./routes/analytics");

const app  = express();
const PORT = process.env.PORT || 3000;

// ── Middleware ────────────────────────────────────────────────────────────────
app.use(express.json());
app.use(morgan("dev"));

// ── Routes ───────────────────────────────────────────────────────────────────
app.use("/", urlRoutes);        // POST /shorten  GET /:code  DELETE /urls/:code
app.use("/", analyticsRoutes);  // POST /track    GET /stats/:code
                                // POST /ai/detect GET /ai/flagged

// ── Global error handler ─────────────────────────────────────────────────────
app.use((err, _req, res, _next) => {
  console.error("[Unhandled]", err);
  res.status(500).json({ error: "Internal server error" });
});

// ── Start ─────────────────────────────────────────────────────────────────────
app.listen(PORT, () => {
  console.log(`API Gateway        :${PORT}`);
  console.log(`URL service  gRPC  → ${process.env.GRPC_TARGET            ?? "localhost:50051"}`);
  console.log(`Analytics    gRPC  → ${process.env.ANALYTICS_GRPC_TARGET  ?? "localhost:50052"}`);
});