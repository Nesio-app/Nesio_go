from fastapi import APIRouter, UploadFile, File, Form
import base64
import hashlib
import httpx
import json
import os
import re
from io import BytesIO
from typing import Any

from app.prompts.item_extraction import ITEM_EXTRACTION_PROMPT
from app.prompts.document_extraction import DOCUMENT_EXTRACTION_PROMPT

try:
    import pytesseract
except Exception:
    pytesseract = None

try:
    from PIL import Image
except Exception:
    Image = None

router = APIRouter(tags=["items"])


@router.post("/items/analyze")
async def analyze_item(
    file: UploadFile = File(...),
    user_id: str = Form(...),
    locale: str = Form("zh"),
):
    content = await file.read()

    extraction = await extract_item(content=content, filename=file.filename or "", locale=locale)
    visual_hash = hashlib.sha256(content).hexdigest()[:16]

    return {
        "extraction": extraction,
        "duplicates": [],
        "visual_hash": visual_hash,
        "suggested_room_id": None,
        "suggested_container_id": None,
        "image_url": "",
        "user_id": user_id,
    }


@router.post("/vision/analyze")
async def analyze_vision_alias(
    file: UploadFile = File(...),
    user_id: str = Form(...),
    locale: str = Form("zh"),
):
    return await analyze_item(file=file, user_id=user_id, locale=locale)


def gemini_api_key() -> str:
    return os.getenv("GEMINI_API_KEY") or os.getenv("GOOGLE_API_KEY") or ""


def gemini_model_for_vision() -> str:
    return os.getenv("GEMINI_MODEL_VISION") or os.getenv("GEMINI_MODEL") or "gemini-2.5-flash"


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

    return {
        "name": base_name,
        "category": "other",
        "brand": None,
        "color": None,
        "quantity": 1,
        "unit": "piece",
        "expiry_date": None,
        "is_document": False,
        "document_type": None,
        "document_number": None,
        "suggested_room": "storage",
        "suggested_container": "drawer",
        "tags": ["待确认", "拍照"],
        "description": "拍照识别结果，建议手动确认后保存。",
        "locale": locale,
    }


async def call_gemini_item_extraction(content: bytes, locale: str) -> dict[str, Any]:
    img_b64 = base64.b64encode(content).decode("utf-8")
    payload = {
        "systemInstruction": {
            "parts": [{"text": f"{ITEM_EXTRACTION_PROMPT}\n\nDocument extraction fallback:\n{DOCUMENT_EXTRACTION_PROMPT}"}]
        },
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
            "maxOutputTokens": 450,
            "responseMimeType": "application/json",
        },
    }
    text = await gemini_generate(gemini_model_for_vision(), payload)
    return parse_json_payload(text)


def extract_item_offline(content: bytes, filename: str, locale: str = "zh") -> dict[str, Any] | None:
    if Image is None:
        return None

    try:
        image = Image.open(BytesIO(content)).convert("RGB")
    except Exception:
        return None

    text_lines: list[str] = []
    if pytesseract is not None:
        try:
            ocr_text = pytesseract.image_to_string(image, lang="chi_sim+eng")
            text_lines = [line.strip() for line in ocr_text.splitlines() if line and line.strip()]
        except Exception:
            text_lines = []

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
