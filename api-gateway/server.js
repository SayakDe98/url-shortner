const express = require("express");
const axios = require("axios");

const app = express();
app.use(express.json());

const GO_SERVICE = "http://localhost:8080";

// Create short URL
app.post("/shorten", async (req, res) => {
  try {
    const response = await axios.post(`${GO_SERVICE}/shorten`, {
      url: req.body.url,
    });

    res.json({
      short_url: `http://localhost:3000/${response.data.short_code}`,
    });
  } catch (err) {
    res.status(500).json({ error: "Service error" });
  }
});

// Redirect
app.get("/:code", async (req, res) => {
  try {
    const response = await axios.get(
      `${GO_SERVICE}/resolve/${req.params.code}`
    );

    res.redirect(response.data.url);
  } catch (err) {
    res.status(404).send("Not found");
  }
});

app.listen(3000, () => console.log("Node API running on 3000"));