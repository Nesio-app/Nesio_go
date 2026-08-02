from fastapi import FastAPI
from pydantic import BaseModel
import os
import httpx

app = FastAPI(title="Nesio AI Service")

class ChatRequest(BaseModel):
    message: str
    tier: str = "standard"

class ChatResponse(BaseModel):
    content: str
    sources: list[str] = []

@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    # Tier-based routing
    if req.tier == "quick":
        model = "gpt-4o-mini"
    elif req.tier == "deep":
        model = "claude-3-5-sonnet"
    else:
        model = "gpt-4o"

    # Try primary model, fallback on error
    try:
        content = await call_openai(req.message, model)
    except Exception:
        try:
            content = await call_openai(req.message, "gpt-4o-mini")
        except Exception:
            content = "AI service temporarily unavailable. Please try again later."

    return ChatResponse(content=content)

async def call_openai(message: str, model: str) -> str:
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        return "AI not configured. Add OPENAI_API_KEY to environment."

    async with httpx.AsyncClient() as client:
        resp = await client.post(
            "https://api.openai.com/v1/chat/completions",
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
            json={
                "model": model,
                "messages": [
                    {"role": "system", "content": "You are Nesio, a helpful life assistant. Be concise, warm, and practical."},
                    {"role": "user", "content": message}
                ],
                "temperature": 0.7,
                "max_tokens": 500
            },
            timeout=30.0
        )
        resp.raise_for_status()
        data = resp.json()
        return data["choices"][0]["message"]["content"]

@app.get("/health")
async def health():
    return {"status": "ok"}
