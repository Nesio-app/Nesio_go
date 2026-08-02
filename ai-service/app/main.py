from fastapi import FastAPI
from pydantic import BaseModel
import os
import httpx

app = FastAPI(title="Nesio AI Service")

class CardPayload(BaseModel):
    title: str
    body: str | None = None
    severity: int

class ChatRequest(BaseModel):
    message: str
    tier: str = "standard"
    mode: str = "chat"

class ChatResponse(BaseModel):
    content: str
    sources: list[str] = []
    card: CardPayload | None = None

@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    if req.mode == "card":
        response = await call_openai_card(req.message, req.tier)
        return response

    if req.tier == "quick":
        model = "gpt-4o-mini"
    elif req.tier == "deep":
        model = "claude-3-5-sonnet"
    else:
        model = "gpt-4o"

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

async def call_openai_card(message: str, tier: str) -> ChatResponse:
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        return ChatResponse(content="AI not configured. Add OPENAI_API_KEY to environment.")

    system_prompt = (
        "You are Nesio, a life assistant that extracts structured reminder cards and short memory summaries from unstructured content. "
        "Generate JSON only with keys title, body, and severity. Title should be short and actionable, body should be a concise summary, "
        "severity must be 1, 2, or 3 based on urgency."
    )

    user_prompt = (
        f"Input: {message}\n\n"
        "Output example:\n"
        "{\"title\": \"Pay rent\", \"body\": \"Rent is due tomorrow for your apartment.\", \"severity\": 3}"
    )

    async with httpx.AsyncClient() as client:
        resp = await client.post(
            "https://api.openai.com/v1/chat/completions",
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
            json={
                "model": "gpt-4o",
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt}
                ],
                "temperature": 0.3,
                "max_tokens": 250
            },
            timeout=30.0
        )
        resp.raise_for_status()
        data = resp.json()
        text = data["choices"][0]["message"]["content"]

    card = None
    try:
        card = CardPayload.parse_raw(text)
    except Exception:
        try:
            # attempt to extract JSON substring
            start = text.find("{")
            end = text.rfind("}")
            if start != -1 and end != -1:
                card = CardPayload.parse_raw(text[start : end+1])
        except Exception:
            card = None

    if not card:
        return ChatResponse(content="AI card generation failed. Please try again.")

    if card.body and len(card.body) < 40:
        card.body = card.body

    return ChatResponse(content=text, card=card)

@app.get("/health")
async def health():
    return {"status": "ok"}
