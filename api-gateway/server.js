require("dotenv").config();
const express = require("express");
const morgan  = require("morgan");
const urlRoutes = require("./routes/urls");

const app  = express();
const PORT = process.env.PORT || 3000;

// ── Middleware ────────────────────────────────────────────────────────────────
app.use(express.json());
app.use(morgan("dev")); // structured request logging

// ── Routes ───────────────────────────────────────────────────────────────────
app.use("/", urlRoutes);

// ── Global error handler ─────────────────────────────────────────────────────
// Catches anything that slips past individual route try/catch blocks.
app.use((err, _req, res, _next) => {
  console.error("[Unhandled]", err);
  res.status(500).json({ error: "Internal server error" });
});

// ── Start ─────────────────────────────────────────────────────────────────────
app.listen(PORT, () =>
  console.log(`API Gateway listening on :${PORT}  →  gRPC @ ${process.env.GRPC_TARGET ?? "localhost:50051"}`)
);