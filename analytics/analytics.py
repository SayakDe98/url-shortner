from fastapi import FastAPI
from pydantic import BaseModel
from typing import Dict
import re
import random

app = FastAPI()

clicks: Dict[str, int] = {}
flagged_urls: Dict[str, float] = {}  # stores fraud score


# -----------------------------
# Models
# -----------------------------

class ClickEvent(BaseModel):
    code: str
    url: str
    ip: str


class DetectionRequest(BaseModel):
    url: str
    ip: str


# -----------------------------
# Basic AI Model
# -----------------------------

def simple_malicious_model(url: str, ip: str) -> float:
    """
    Simulated ML model.
    Returns fraud score between 0 and 1.
    Replace with real ML inference later.
    """

    suspicious_keywords = [
        "login",
        "verify",
        "update-password",
        "free-money",
        "crypto",
        "bank"
    ]

    score = 0.0

    # Heuristic signals (mock ML features)
    if any(keyword in url.lower() for keyword in suspicious_keywords):
        score += 0.5

    if len(url) > 100:
        score += 0.2

    if re.search(r"\d{5,}", url):  # long digit sequences
        score += 0.2

    # add small random noise to simulate probabilistic output
    score += random.uniform(0, 0.1)

    return min(score, 1.0)


# -----------------------------
# Track Click
# -----------------------------

@app.post("/track")
def track_click(event: ClickEvent):
    clicks[event.code] = clicks.get(event.code, 0) + 1

    # Call AI model during tracking (async recommended in real system)
    fraud_score = simple_malicious_model(event.url, event.ip)

    if fraud_score > 0.7:
        flagged_urls[event.url] = fraud_score

    return {
        "message": "tracked",
        "fraud_score": fraud_score
    }


# -----------------------------
# Get Stats
# -----------------------------

@app.get("/stats/{code}")
def get_stats(code: str):
    return {
        "clicks": clicks.get(code, 0)
    }


# -----------------------------
# AI Detection Endpoint
# -----------------------------

@app.post("/ai/detect")
def detect_malicious(request: DetectionRequest):
    fraud_score = simple_malicious_model(request.url, request.ip)

    is_malicious = fraud_score > 0.7

    if is_malicious:
        flagged_urls[request.url] = fraud_score

    return {
        "url": request.url,
        "fraud_score": fraud_score,
        "is_malicious": is_malicious
    }


# -----------------------------
# List Flagged URLs
# -----------------------------

@app.get("/ai/flagged")
def get_flagged_urls():
    return flagged_urls