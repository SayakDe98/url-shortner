from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

clicks = {}

class ClickEvent(BaseModel):
    code: str

@app.post("/track")
def track_click(event: ClickEvent):
    clicks[event.code] = clicks.get(event.code, 0) + 1
    return {"message": "tracked"}

@app.get("/stats/{code}")
def get_stats(code: str):
    return {"clicks": clicks.get(code, 0)}