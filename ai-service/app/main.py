from fastapi import FastAPI
from pydantic import BaseModel
import os
import httpx
import base64
import hashlib
import json
import re
from io import BytesIO
from typing import Any

try:
    import pytesseract
except Exception:
    pytesseract = None

try:
    from PIL import Image
    from PIL import ImageFilter, ImageOps
except Exception:
    Image = None
    ImageFilter = None
    ImageOps = None

app = FastAPI(title="Nesio AI Service")

from app.routers.items import router as items_router

app.include_router(items_router)


def gemini_api_key() -> str:
    return os.getenv("GEMINI_API_KEY") or os.getenv("GOOGLE_API_KEY") or ""


def gemini_model_override(name: str, default: str) -> str:
    return os.getenv(name) or os.getenv("GEMINI_MODEL") or default


def gemini_model_for_tier(tier: str) -> str:
    if tier == "quick":
        return gemini_model_override("GEMINI_MODEL_QUICK", "gemini-2.5-flash-lite")
    if tier == "deep":
        return gemini_model_override("GEMINI_MODEL_DEEP", "gemini-2.5-flash")
    return gemini_model_override("GEMINI_MODEL_STANDARD", "gemini-2.5-flash")


def gemini_model_for_card() -> str:
    return gemini_model_override("GEMINI_MODEL_CARD", "gemini-2.5-flash")


def gemini_model_for_vision() -> str:
    return gemini_model_override("GEMINI_MODEL_VISION", "gemini-2.5-flash")


def gemini_model_for_intake() -> str:
    return gemini_model_override("GEMINI_MODEL_INTAKE", "gemini-2.5-flash-lite")

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


class AskRequest(BaseModel):
    question: str
    user_id: str | None = None


class AskResponse(BaseModel):
    type: str
    answer: str
    sources: list[dict[str, Any]] = []


class IntakeParseRequest(BaseModel):
    text: str
    locale: str = "zh"


class IntakeParseResponse(BaseModel):
    intent: str
    title: str
    should_remind: bool
    remind_at: str | None = None
    tags: list[str] = []
    confidence: float = 0.6

@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    return await handle_chat_request(req.message, req.tier, req.mode)


@app.post("/ask", response_model=AskResponse)
async def ask(req: AskRequest):
    answer = await handle_ask(req.question)
    return AskResponse(type="direct", answer=answer, sources=[])


@app.post("/intake/parse", response_model=IntakeParseResponse)
async def parse_intake(req: IntakeParseRequest):
    return await handle_intake_parse(req.text, req.locale)


async def handle_chat_request(message: str, tier: str = "standard", mode: str = "chat") -> ChatResponse:
    # Card mode uses structured card generation
    if mode == "card":
        response = await call_gemini_card(message)
        return response

    # Attempt primary model, fallback to smaller quick model on failure
    try:
        content = await call_gemini_text(
            message=message,
            system_prompt="You are Nesio, a helpful life assistant. Be concise, warm, and practical.",
            model=gemini_model_for_tier(tier),
        )
    except Exception:
        try:
            content = await call_gemini_text(
                message=message,
                system_prompt="You are Nesio, a helpful life assistant. Be concise, warm, and practical.",
                model=gemini_model_for_tier("quick"),
            )
        except Exception:
            content = "AI service temporarily unavailable. Please try again later."

    return ChatResponse(content=content)


async def handle_ask(question: str) -> str:
    q = question.strip()
    if not q:
        return "你可以问我：我的护照在哪、今天最重要的事是什么。"

    # Reuse chat generation path for now; this keeps ask behavior useful even before full retrieval is wired.
    resp = await handle_chat_request(q, tier="standard", mode="chat")
    return resp.content


async def handle_intake_parse(text: str, locale: str = "zh") -> IntakeParseResponse:
    content = text.strip()
    if not content:
        return IntakeParseResponse(intent="memory", title="", should_remind=False, tags=[])

    if gemini_api_key():
        try:
            parsed = await call_gemini_intake_parse(content, locale)
            if parsed.get("title"):
                return IntakeParseResponse(
                    intent=str(parsed.get("intent", "memory")),
                    title=str(parsed.get("title", content[:24])),
                    should_remind=bool(parsed.get("should_remind", False)),
                    remind_at=parsed.get("remind_at"),
                    tags=[str(t) for t in parsed.get("tags", []) if isinstance(t, str)],
                    confidence=float(parsed.get("confidence", 0.75)),
                )
        except Exception:
            pass

    return fallback_intake_parse(content)


@app.post("/chat/quick", response_model=ChatResponse)
async def chat_quick(req: ChatRequest):
    return await handle_chat_request(req.message, "quick", req.mode)


@app.post("/chat/standard", response_model=ChatResponse)
async def chat_standard(req: ChatRequest):
    return await handle_chat_request(req.message, "standard", req.mode)


@app.post("/chat/deep", response_model=ChatResponse)
async def chat_deep(req: ChatRequest):
    return await handle_chat_request(req.message, "deep", req.mode)

async def call_gemini_text(message: str, system_prompt: str, model: str) -> str:
    api_key = gemini_api_key()
    if not api_key:
        return "AI not configured. Add GEMINI_API_KEY to environment."

    payload = {
        "systemInstruction": {"parts": [{"text": system_prompt}]},
        "contents": [
            {
                "role": "user",
                "parts": [{"text": message}],
            }
        ],
        "generationConfig": {
            "temperature": 0.7,
            "maxOutputTokens": 500,
        },
    }

    return await gemini_generate(model, payload)


async def extract_item(content: bytes, filename: str, locale: str = "zh") -> dict[str, Any]:
    if gemini_api_key():
        try:
            return await call_gemini_item_extraction(content, locale)
        except Exception:
            pass

    local_extraction = extract_item_offline(content=content, filename=filename, locale=locale)
    if local_extraction is not None:
        return local_extraction

    base_name = os.path.splitext(os.path.basename(filename))[0].strip() or "新物品"
    if is_generic_camera_name(base_name):
        base_name = "新物品"
    lower_name = base_name.lower()
    category = "other"
    suggested_room = "storage"
    suggested_container = "drawer"
    tags = ["待确认", "拍照"]

    if any(key in lower_name for key in ["pill", "medicine", "药", "capsule"]):
        category = "medicine"
        suggested_room = "bedroom"
        suggested_container = "medicine_box"
        tags = ["药品", "提醒", "健康"]
    elif any(key in lower_name for key in ["passport", "id", "证件"]):
        category = "document"
        suggested_room = "study"
        suggested_container = "drawer"
        tags = ["证件", "重要", "到期"]
    elif any(key in lower_name for key in ["milk", "food", "snack", "饮", "食"]):
        category = "food"
        suggested_room = "kitchen"
        suggested_container = "fridge"
        tags = ["食品", "库存", "有效期"]

    return {
        "name": base_name,
        "category": category,
        "brand": None,
        "color": None,
        "quantity": 1,
        "unit": "piece",
        "expiry_date": None,
        "is_document": category == "document",
        "document_type": "passport" if "passport" in lower_name else None,
        "document_number": None,
        "suggested_room": suggested_room,
        "suggested_container": suggested_container,
        "tags": tags,
        "description": "拍照识别结果，建议手动确认后保存。",
        "locale": locale,
    }


def extract_item_offline(content: bytes, filename: str, locale: str = "zh") -> dict[str, Any] | None:
    if Image is None:
        return None

    try:
        image = Image.open(BytesIO(content)).convert("RGB")
    except Exception:
        return None

    text_lines: list[str] = []
    if pytesseract is not None:
        text_lines = run_ocr_pass(image)
        if not text_lines and ImageOps is not None and ImageFilter is not None:
            enhanced = ImageOps.autocontrast(image.convert("L")).resize(
                (max(1, image.width * 2), max(1, image.height * 2))
            ).filter(ImageFilter.SHARPEN)
            text_lines = run_ocr_pass(enhanced)

    normalized_text = " ".join(text_lines)
    normalized_lower = normalized_text.lower()

    name = infer_name_from_text(text_lines)
    if not name:
        base_name = os.path.splitext(os.path.basename(filename))[0].strip()
        if base_name and not is_generic_camera_name(base_name):
            name = base_name
        else:
            name = "新物品"

    category = "other"
    suggested_room = "storage"
    suggested_container = "drawer"
    tags = ["待确认", "拍照"]
    is_document = False
    document_type = None
    document_number = None

    if any(k in normalized_lower for k in ["passport", "护照", "id card", "身份证", "driver", "驾照", "security alert"]):
        category = "document"
        suggested_room = "study"
        suggested_container = "document_box"
        tags = ["证件", "重要", "待核验"]
        is_document = True
        if "passport" in normalized_lower or "护照" in normalized_lower:
            document_type = "passport"
        elif "id card" in normalized_lower or "身份证" in normalized_lower:
            document_type = "id_card"
        elif "driver" in normalized_lower or "驾照" in normalized_lower:
            document_type = "driver_license"

        match = re.search(r"\b([A-Z0-9]{6,18})\b", normalized_text)
        if match:
            document_number = match.group(1)
    elif any(k in normalized_lower for k in ["medicine", "tablet", "capsule", "药", "片", "mg", "ml"]):
        category = "medicine"
        suggested_room = "bedroom"
        suggested_container = "medicine_box"
        tags = ["药品", "健康", "待确认"]
    elif any(k in normalized_lower for k in ["milk", "drink", "snack", "food", "食品", "饮料", "零食"]):
        category = "food"
        suggested_room = "kitchen"
        suggested_container = "fridge"
        tags = ["食品", "库存", "待确认"]
    elif any(k in normalized_lower for k in ["invoice", "bill", "receipt", "账单", "发票", "订单"]):
        category = "bill"
        suggested_room = "study"
        suggested_container = "folder"
        tags = ["财务", "待处理", "待确认"]

    expiry_date = extract_expiry_date(normalized_text)

    description = "拍照识别完成，请确认关键字段。"
    if normalized_text:
        description = f"识别到文本: {normalized_text[:120]}"

    return {
        "name": name,
        "category": category,
        "brand": None,
        "color": None,
        "quantity": 1,
        "unit": "piece",
        "expiry_date": expiry_date,
        "is_document": is_document,
        "document_type": document_type,
        "document_number": document_number,
        "suggested_room": suggested_room,
        "suggested_container": suggested_container,
        "tags": tags,
        "description": description,
        "locale": locale,
    }


def run_ocr_pass(image) -> list[str]:
    if pytesseract is None:
        return []
    try:
        ocr_text = pytesseract.image_to_string(image, lang="chi_sim+eng")
        return [line.strip() for line in ocr_text.splitlines() if line and line.strip()]
    except Exception:
        return []


def infer_name_from_text(lines: list[str]) -> str:
    noise = {
        "re", "fw", "fwd", "mail", "email", "from", "to", "cc", "subject", "all", "全部来源", "全部标签"
    }
    for line in lines[:8]:
        clean = re.sub(r"\s+", " ", line).strip(" -_:;,.，。")
        if len(clean) < 2:
            continue
        lower = clean.lower()
        if lower in noise:
            continue
        if any(k in lower for k in ["task prioritization", "security alert", "passport", "护照", "身份证", "驾照"]):
            return clean[:48]
    return ""


def is_generic_camera_name(name: str) -> bool:
    lower = name.strip().lower()
    if lower in {"", "image", "photo", "img", "scan", "camera", "screenshot", "new", "untitled"}:
        return True
    return bool(re.fullmatch(r"(img|dsc|pxl|mvimg|wx_camera|camera)_?\d+", lower))


def extract_expiry_date(text: str) -> str | None:
    if not text:
        return None
    match = re.search(r"(20\d{2})[-/.年](\d{1,2})[-/.月](\d{1,2})", text)
    if not match:
        return None
    year = int(match.group(1))
    month = int(match.group(2))
    day = int(match.group(3))
    if not (1 <= month <= 12 and 1 <= day <= 31):
        return None
    return f"{year:04d}-{month:02d}-{day:02d}"


async def call_gemini_item_extraction(content: bytes, locale: str) -> dict[str, Any]:
    prompt = (
        "You are an item recognition expert. Return JSON only with keys: "
        "name, category, brand, color, quantity, unit, expiry_date, is_document, "
        "document_type, document_number, suggested_room, suggested_container, tags, description."
    )
    img_b64 = base64.b64encode(content).decode("utf-8")
    payload = {
        "systemInstruction": {"parts": [{"text": prompt}]},
        "contents": [
            {
                "role": "user",
                "parts": [
                    {"text": f"locale={locale}"},
                    {
                        "inlineData": {
                            "mimeType": "image/jpeg",
                            "data": img_b64,
                        }
                    },
                ],
            }
        ],
        "generationConfig": {
            "temperature": 0.2,
            "maxOutputTokens": 400,
            "responseMimeType": "application/json",
        },
    }
    text = await gemini_generate(gemini_model_for_vision(), payload)
    return parse_json_payload(text)


async def call_gemini_intake_parse(text: str, locale: str) -> dict[str, Any]:
    prompt = (
        "You parse user quick notes for personal OS intake. "
        "Return JSON only with keys: intent, title, should_remind, remind_at, tags, confidence. "
        "intent must be one of memory/task/reminder/query/item. "
        "remind_at must be RFC3339 string or null."
    )
    payload = {
        "systemInstruction": {"parts": [{"text": prompt}]},
        "contents": [
            {
                "role": "user",
                "parts": [{"text": f"locale={locale}\ntext={text}"}],
            }
        ],
        "generationConfig": {
            "temperature": 0.2,
            "maxOutputTokens": 220,
            "responseMimeType": "application/json",
        },
    }
    text_response = await gemini_generate(gemini_model_for_intake(), payload)
    return parse_json_payload(text_response)


def fallback_intake_parse(text: str) -> IntakeParseResponse:
    from datetime import timedelta

    lower = text.lower()
    should_remind = any(token in text for token in ["提醒", "到期", "明天", "后天", "今天", "点", "分钟后"]) or "remind" in lower
    intent = "memory"
    tags: list[str] = ["输入框"]
    if "买" in text or "采购" in text:
        intent = "task"
        tags.append("待办")
    if "在哪" in text or "哪里" in text:
        intent = "query"
        tags.append("检索")
    if should_remind:
        intent = "reminder"
        tags.append("提醒")

    remind_at = None
    if should_remind:
        now = datetime_now_local()
        if "明天" in text:
            remind_at = (now + timedelta(days=1)).replace(hour=9, minute=0, second=0, microsecond=0).isoformat()
        elif "后天" in text:
            remind_at = (now + timedelta(days=2)).replace(hour=9, minute=0, second=0, microsecond=0).isoformat()
        else:
            remind_at = (now + timedelta(hours=2)).replace(second=0, microsecond=0).isoformat()

    title = text[:24]
    return IntakeParseResponse(
        intent=intent,
        title=title,
        should_remind=should_remind,
        remind_at=remind_at,
        tags=tags,
        confidence=0.62,
    )


def datetime_now_local():
    from datetime import datetime

    return datetime.now().astimezone()

async def call_gemini_card(message: str) -> ChatResponse:
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

    text = await gemini_generate(
        gemini_model_for_card(),
        {
            "systemInstruction": {"parts": [{"text": system_prompt}]},
            "contents": [
                {
                    "role": "user",
                    "parts": [{"text": user_prompt}],
                }
            ],
            "generationConfig": {
                "temperature": 0.3,
                "maxOutputTokens": 250,
                "responseMimeType": "application/json",
            },
        },
    )

    card = None
    try:
        card = CardPayload.model_validate_json(text)
    except Exception:
        try:
            # attempt to extract JSON substring
            start = text.find("{")
            end = text.rfind("}")
            if start != -1 and end != -1:
                card = CardPayload.model_validate_json(text[start : end+1])
        except Exception:
            card = None

    if not card:
        return ChatResponse(content="AI card generation failed. Please try again.")

    if card.body and len(card.body) < 40:
        card.body = card.body

    return ChatResponse(content=text, card=card)


async def gemini_generate(model: str, payload: dict[str, Any]) -> str:
    api_key = gemini_api_key()
    if not api_key:
        raise RuntimeError("GEMINI_API_KEY missing")

    async with httpx.AsyncClient() as client:
        resp = await client.post(
            f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={api_key}",
            headers={"Content-Type": "application/json"},
            json=payload,
            timeout=45.0,
        )
        resp.raise_for_status()
        data = resp.json()

    candidates = data.get("candidates") or []
    if not candidates:
        raise RuntimeError("Gemini returned no candidates")

    content = candidates[0].get("content") or {}
    parts = content.get("parts") or []
    texts: list[str] = []
    for part in parts:
        text = part.get("text")
        if isinstance(text, str) and text.strip():
            texts.append(text)

    if not texts:
        raise RuntimeError("Gemini returned empty content")

    return "\n".join(texts)


def parse_json_payload(text: str) -> dict[str, Any]:
    try:
        return json.loads(text)
    except Exception:
        start = text.find("{")
        end = text.rfind("}")
        if start != -1 and end != -1:
            return json.loads(text[start : end + 1])
        raise

@app.get("/health")
async def health():
    return {"status": "ok"}
