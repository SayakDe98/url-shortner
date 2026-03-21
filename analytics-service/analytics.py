import re
import random
from concurrent import futures
from typing import Dict

import grpc

# Generated from: python -m grpc_tools.protoc (see generate_proto.sh)
from generated import analytics_pb2, analytics_pb2_grpc

# ---------------------------------------------------------------------------
# In-memory stores
# Replace with Redis / a real DB in production.
# ---------------------------------------------------------------------------
clicks: Dict[str, int] = {}
flagged_urls: Dict[str, float] = {}  # url → fraud_score


# ---------------------------------------------------------------------------
# Fraud detection model
# ---------------------------------------------------------------------------

SUSPICIOUS_KEYWORDS = [
    "login",
    "verify",
    "update-password",
    "free-money",
    "crypto",
    "bank",
]

def simple_malicious_model(url: str, ip: str) -> float:
    """
    Simulated ML model — returns a fraud score in [0, 1].
    Replace with real inference (e.g. scikit-learn / ONNX / TF Serving) later.

    Features used:
      - Presence of suspicious keywords  (+0.5)
      - URL length > 100 chars           (+0.2)
      - Long digit sequences (5+)        (+0.2)
      - Small random noise               (+0.0–0.1)
    """
    score = 0.0

    if any(kw in url.lower() for kw in SUSPICIOUS_KEYWORDS):
        score += 0.5

    if len(url) > 100:
        score += 0.2

    if re.search(r"\d{5,}", url):
        score += 0.2

    score += random.uniform(0, 0.1)
    return min(score, 1.0)


# ---------------------------------------------------------------------------
# gRPC service implementation
# ---------------------------------------------------------------------------

class AnalyticsServicer(analytics_pb2_grpc.AnalyticsServicer):

    def TrackClick(self, request, context):
        """
        Records a click for the given short code and runs fraud detection.
        Replaces: POST /track
        TODO: publish to Redis queue / Kafka topic instead of processing inline.
        """
        clicks[request.code] = clicks.get(request.code, 0) + 1

        fraud_score = simple_malicious_model(request.url, request.ip)
        if fraud_score > 0.7:
            flagged_urls[request.url] = fraud_score

        return analytics_pb2.TrackClickResponse(
            message="tracked",
            fraud_score=fraud_score,
        )

    def GetStats(self, request, context):
        """
        Returns click count for a short code.
        Replaces: GET /stats/{code}
        TODO: read from Redis instead of in-memory dict.
        """
        return analytics_pb2.GetStatsResponse(
            clicks=clicks.get(request.code, 0),
        )

    def DetectMalicious(self, request, context):
        """
        Runs fraud detection on a URL + IP pair on demand.
        Replaces: POST /ai/detect
        """
        fraud_score = simple_malicious_model(request.url, request.ip)
        is_malicious = fraud_score > 0.7

        if is_malicious:
            flagged_urls[request.url] = fraud_score

        return analytics_pb2.DetectResponse(
            url=request.url,
            fraud_score=fraud_score,
            is_malicious=is_malicious,
        )

    def GetFlaggedURLs(self, request, context):
        """
        Returns all URLs that exceeded the fraud threshold.
        Replaces: GET /ai/flagged
        """
        entries = [
            analytics_pb2.FlaggedURLEntry(url=url, fraud_score=score)
            for url, score in flagged_urls.items()
        ]
        return analytics_pb2.FlaggedURLsResponse(flagged=entries)


# ---------------------------------------------------------------------------
# Server bootstrap
# ---------------------------------------------------------------------------

def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ("grpc.max_receive_message_length", 4 * 1024 * 1024),  # 4 MB
        ],
    )
    analytics_pb2_grpc.add_AnalyticsServicer_to_server(AnalyticsServicer(), server)

    port = "[::]:50052"  # separate port from the Go gRPC service (50051)
    server.add_insecure_port(port)
    server.start()
    print(f"Analytics gRPC server listening on {port}")
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
