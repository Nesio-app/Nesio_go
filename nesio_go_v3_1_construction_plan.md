# Nesio_go 施工计划书 v3.1
## 物品管理系统 + 智能识别 + 聚合页 + 全生命周期追踪

> 给 Code Agent 的详细施工文件
> 版本：v3.1 | 2026-08-02

---

## 目录

1. [数据模型](#一数据模型)
2. [物品聚合页](#二物品聚合页)
3. [智能识别流程](#三智能识别流程)
4. [房间与容器系统](#四房间与容器系统)
5. [有效期与证件到期提醒](#五有效期与证件到期提醒)
6. [重复物品检测](#六重复物品检测)
7. [API 设计](#七api-设计)
8. [前端组件](#八前端组件)
9. [施工时间线](#九施工时间线)
10. [Code Agent 指令清单](#十code-agent-指令清单)

---

## 一、数据模型

### 1.1 物品表（life_nodes 扩展）

```sql
-- 物品专用扩展表（与 life_nodes 1:1）
CREATE TABLE item_details (
    node_id uuid PRIMARY KEY REFERENCES life_nodes(id) ON DELETE CASCADE,

    -- 位置信息
    room_id uuid REFERENCES rooms(id),
    container_id uuid REFERENCES containers(id),
    location_note TEXT, -- 自由文本补充，如"书架第三层"

    -- 有效期
    expiry_date DATE,
    expiry_remind_days INT DEFAULT 30, -- 提前 N 天提醒

    -- 证件专用
    is_document BOOLEAN DEFAULT false,
    document_type VARCHAR(32), -- passport / visa / license / id / warranty
    document_number TEXT,
    issuing_authority TEXT,

    -- 购买信息
    purchase_date DATE,
    purchase_price DECIMAL(10,2),
    purchase_currency VARCHAR(3) DEFAULT 'CNY',
    retailer TEXT,

    -- 视觉信息
    visual_hash TEXT,       -- pHash
    clip_embedding VECTOR(1024),
    primary_image_url TEXT,
    gallery_urls TEXT[],    -- 多角度照片

    -- 数量
    quantity INT DEFAULT 1,
    unit VARCHAR(16),       -- 个 / 瓶 / 盒 / 包

    -- 状态
    condition VARCHAR(16) DEFAULT 'good', -- new / good / fair / poor
    is_lent BOOLEAN DEFAULT false,
    lent_to TEXT,

    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_items_room ON item_details(room_id);
CREATE INDEX idx_items_container ON item_details(container_id);
CREATE INDEX idx_items_expiry ON item_details(expiry_date) WHERE expiry_date IS NOT NULL;
CREATE INDEX idx_items_document ON item_details(is_document, expiry_date) WHERE is_document = true;
```

### 1.2 房间表

```sql
CREATE TABLE rooms (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,           -- "客厅" / "卧室" / "厨房"
    icon TEXT DEFAULT '🏠',       -- emoji 图标
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 默认房间种子数据
INSERT INTO rooms (user_id, name, icon, sort_order) VALUES
($1, '客厅', '🛋️', 1),
($1, '卧室', '🛏️', 2),
($1, '厨房', '🍳', 3),
($1, '书房', '📚', 4),
($1, '卫生间', '🚿', 5),
($1, '玄关', '🚪', 6),
($1, '储藏室', '📦', 7),
($1, '阳台', '🌿', 8);
```

### 1.3 容器表

```sql
CREATE TABLE containers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    room_id uuid REFERENCES rooms(id) ON DELETE SET NULL,
    name TEXT NOT NULL,           -- "冰箱" / "衣柜" / "抽屉" / "药箱"
    icon TEXT DEFAULT '📦',
    parent_container_id uuid REFERENCES containers(id), -- 嵌套容器
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 默认容器种子数据
INSERT INTO containers (user_id, room_id, name, icon, sort_order) VALUES
($1, (SELECT id FROM rooms WHERE name='厨房'), '冰箱', '❄️', 1),
($1, (SELECT id FROM rooms WHERE name='厨房'), '橱柜', '🗄️', 2),
($1, (SELECT id FROM rooms WHERE name='卧室'), '衣柜', '👔', 1),
($1, (SELECT id FROM rooms WHERE name='卧室'), '床头柜', '🛏️', 2),
($1, (SELECT id FROM rooms WHERE name='书房'), '书架', '📚', 1),
($1, (SELECT id FROM rooms WHERE name='玄关'), '鞋柜', '👟', 1),
($1, NULL, '药箱', '💊', 1); -- 可移动容器，不绑定房间
```

### 1.4 物品-标签关联（替代 Graph Engine）

```sql
-- 简单标签关联，不做复杂图谱
CREATE TABLE item_tags (
    item_id uuid REFERENCES life_nodes(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    confidence FLOAT DEFAULT 1.0,
    source VARCHAR(32), -- ai / manual / ocr
    PRIMARY KEY (item_id, tag)
);

CREATE INDEX idx_tags_tag ON item_tags(tag);
```

---

## 二、物品聚合页

### 2.1 页面结构

```
┌─────────────────────────────────────────┐
│  🔍 搜索物品...                    [📷]  │  ← 顶部搜索 + 相机入口
├─────────────────────────────────────────┤
│  🏠 全部  🛋️客厅  🛏️卧室  🍳厨房  ...   │  ← 房间标签栏（横向滚动）
├─────────────────────────────────────────┤
│  📦 全部  ❄️冰箱  🗄️橱柜  📦药箱  ...   │  ← 容器筛选（二级）
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │ [图片]  │ │ [图片]  │ │ [图片]  │  │  ← 物品网格
│  │ 牛奶    │ │ 护照    │ │ 耳机    │  │
│  │ ❄️冰箱  │ │ 📦抽屉  │ │ 🛏️床头  │  │
│  │ ⚠️3天后 │ │ ⚠️30天后│ │         │  │  ← 有效期提醒
│  └─────────┘ └─────────┘ └─────────┘  │
│                                         │
├─────────────────────────────────────────┤
│  [+] 添加物品                           │  ← 底部 FAB
└─────────────────────────────────────────┘
```

### 2.2 排序规则

```sql
-- 聚合页查询
SELECT 
    n.id, n.name, n.type, n.image_url, n.tags,
    i.room_id, i.container_id, i.expiry_date, i.is_document,
    r.name as room_name, r.icon as room_icon,
    c.name as container_name, c.icon as container_icon,
    CASE 
        WHEN i.expiry_date IS NOT NULL THEN 
            (i.expiry_date - CURRENT_DATE)
        ELSE NULL 
    END as days_until_expiry
FROM life_nodes n
JOIN item_details i ON n.id = i.node_id
LEFT JOIN rooms r ON i.room_id = r.id
LEFT JOIN containers c ON i.container_id = c.id
WHERE n.user_id = $1 
  AND n.type = 'thing'
  AND ($2::uuid IS NULL OR i.room_id = $2)
  AND ($3::uuid IS NULL OR i.container_id = $3)
ORDER BY 
    CASE WHEN i.expiry_date IS NOT NULL AND i.expiry_date <= CURRENT_DATE + 7 THEN 0 ELSE 1 END,
    i.expiry_date ASC NULLS LAST,
    n.created_at DESC;
```

### 2.3 前端组件

```tsx
// frontend/src/pages/ItemsPage.tsx
export default function ItemsPage() {
  const [selectedRoom, setSelectedRoom] = useState<string | null>(null);
  const [selectedContainer, setSelectedContainer] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const { data: items } = useSWR(
    `/api/v1/items?room=${selectedRoom}&container=${selectedContainer}&q=${searchQuery}`,
    fetcher
  );

  const { data: rooms } = useSWR('/api/v1/rooms', fetcher);
  const { data: containers } = useSWR(
    selectedRoom ? `/api/v1/containers?room=${selectedRoom}` : '/api/v1/containers',
    fetcher
  );

  return (
    <div className="items-page">
      {/* 搜索栏 */}
      <div className="search-header">
        <input 
          type="text" 
          placeholder="搜索物品..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        <button onClick={() => openCamera()}>📷</button>
      </div>

      {/* 房间标签 */}
      <div className="room-tabs">
        <button 
          className={!selectedRoom ? 'active' : ''}
          onClick={() => { setSelectedRoom(null); setSelectedContainer(null); }}
        >
          🏠 全部
        </button>
        {rooms?.map(r => (
          <button 
            key={r.id}
            className={selectedRoom === r.id ? 'active' : ''}
            onClick={() => { setSelectedRoom(r.id); setSelectedContainer(null); }}
          >
            {r.icon} {r.name}
          </button>
        ))}
      </div>

      {/* 容器筛选 */}
      <div className="container-tabs">
        <button 
          className={!selectedContainer ? 'active' : ''}
          onClick={() => setSelectedContainer(null)}
        >
          📦 全部
        </button>
        {containers?.map(c => (
          <button 
            key={c.id}
            className={selectedContainer === c.id ? 'active' : ''}
            onClick={() => setSelectedContainer(c.id)}
          >
            {c.icon} {c.name}
          </button>
        ))}
      </div>

      {/* 物品网格 */}
      <div className="items-grid">
        {items?.map(item => (
          <ItemCard key={item.id} item={item} />
        ))}
      </div>

      {/* 添加按钮 */}
      <FAB onClick={() => openAddItemSheet()}>+</FAB>
    </div>
  );
}

function ItemCard({ item }: { item: Item }) {
  const expiryWarning = item.days_until_expiry !== null && item.days_until_expiry <= 7;
  const documentWarning = item.is_document && item.days_until_expiry !== null && item.days_until_expiry <= 30;

  return (
    <div className={`item-card ${expiryWarning ? 'warning' : ''} ${documentWarning ? 'document-warning' : ''}`}
         onClick={() => openItemDetail(item.id)}>
      <div className="item-image">
        {item.image_url ? <img src={item.image_url} /> : <span>📦</span>}
      </div>
      <div className="item-info">
        <h4>{item.name}</h4>
        <div className="location">
          {item.room_icon} {item.room_name} · {item.container_icon} {item.container_name}
        </div>
        {item.days_until_expiry !== null && (
          <div className={`expiry ${expiryWarning ? 'urgent' : ''}`}>
            {item.days_until_expiry < 0 ? `已过期 ${-item.days_until_expiry} 天` :
             item.days_until_expiry === 0 ? '今天过期' :
             item.days_until_expiry <= 7 ? `⚠️ ${item.days_until_expiry} 天后过期` :
             `${item.days_until_expiry} 天后过期`}
          </div>
        )}
        {item.quantity > 1 && <span className="quantity">x{item.quantity}</span>}
      </div>
    </div>
  );
}
```

---

## 三、智能识别流程

### 3.1 通用识别管道

```
用户拍照/上传图片/文件
    ↓
POST /api/v1/items/analyze
    ↓
后端并行处理：
    ├── GPT-4V 通用识别
    │   ├── 物品名称
    │   ├── 类别（食品/证件/电子产品/衣物/药品/化妆品/文件）
    │   ├── 品牌（如有）
    │   ├── 颜色
    │   ├── 数量/规格
    │   ├── 有效期（如有，提取日期）
    │   ├── 是否为证件（护照/身份证/驾照/签证/保修卡）
    │   └── 建议存放位置
    ├── CLIP 视觉 embedding
    ├── pHash 感知哈希
    └── OCR 文字提取（端上 Vision 或云端）
    ↓
查重检测（pHash + 向量相似度）
    ↓
如果重复 → 提示"家里已有，放在 [位置]"
    ↓
如果没重复 → 创建 thing 节点 + item_details
    ↓
返回识别结果，用户确认/修改：
    ├── 名称
    ├── 房间（下拉选择）
    ├── 容器（下拉选择）
    ├── 有效期（日期选择器，可空）
    ├── 数量
    └── 标签（AI 生成，用户可编辑）
    ↓
用户确认 → 写入数据库
```

### 3.2 后端识别服务

```python
# ai-service/app/routers/items.py
from fastapi import APIRouter, UploadFile, File, Form
from PIL import Image
import imagehash
import io
import json

router = APIRouter(prefix="/items")

ITEM_EXTRACTION_PROMPT = """You are an item recognition expert. Analyze the image and extract:

1. name: Concise item name (max 20 chars)
2. category: One of [food, document, electronics, clothing, medicine, cosmetic, book, tool, other]
3. brand: Brand name if visible, else null
4. color: Dominant color
5. quantity: Number or amount visible
6. unit: piece / bottle / box / bag / can / pair
7. expiry_date: ISO date if visible (food/medicine/cosmetic), else null
8. is_document: true if passport/ID/license/visa/warranty/receipt
9. document_type: passport / id / license / visa / warranty / receipt / null
10. document_number: if visible and is_document, else null
11. suggested_room: living_room / bedroom / kitchen / bathroom / study / storage / balcony
12. suggested_container: fridge / cabinet / wardrobe / drawer / shelf / medicine_box / null
13. tags: Array of 3-8 relevant keywords for search
14. description: One sentence description

Respond ONLY with valid JSON."""

@router.post("/analyze")
async def analyze_item(
    file: UploadFile = File(...),
    user_id: str = Form(...),
    locale: str = "zh"
):
    content = await file.read()

    # 1. GPT-4V 识别
    img_b64 = base64.b64encode(content).decode()
    resp = await openai.chat.completions.create(
        model="gpt-4o",
        messages=[
            {"role": "system", "content": ITEM_EXTRACTION_PROMPT},
            {"role": "user", "content": [
                {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{img_b64}"}}
            ]}
        ],
        response_format={"type": "json_object"}
    )

    extraction = json.loads(resp.choices[0].message.content)

    # 2. 生成视觉指纹
    img = Image.open(io.BytesIO(content))
    phash = str(imagehash.phash(img))
    clip_embedding = await clip_embed(content)

    # 3. 查重
    duplicates = await find_item_duplicates(user_id, phash, clip_embedding, extraction.get("name", ""))

    # 4. 查找用户房间和容器
    rooms = await pg.fetchall("SELECT id, name, icon FROM rooms WHERE user_id = $1", user_id)
    containers = await pg.fetchall("SELECT id, name, icon, room_id FROM containers WHERE user_id = $1", user_id)

    # 5. 映射建议位置到实际 ID
    suggested_room = find_best_match(extraction.get("suggested_room"), rooms)
    suggested_container = find_best_match(extraction.get("suggested_container"), containers)

    return {
        "extraction": extraction,
        "duplicates": duplicates,
        "visual_hash": phash,
        "clip_embedding": clip_embedding,
        "rooms": rooms,
        "containers": containers,
        "suggested_room_id": suggested_room,
        "suggested_container_id": suggested_container,
        "image_url": await upload_image(content)
    }

async def find_item_duplicates(user_id: str, phash: str, embedding: list, name: str):
    """三重查重策略"""
    results = []

    # 策略1: 视觉相似（pHash 汉明距离 < 12）
    visual_dups = await pg.fetchall("""
        SELECT n.id, n.name, i.primary_image_url, i.room_id, i.container_id,
               hamming_distance(i.visual_hash, $2) as distance
        FROM life_nodes n
        JOIN item_details i ON n.id = i.node_id
        WHERE n.user_id = $1 AND n.type = 'thing'
          AND hamming_distance(i.visual_hash, $2) < 12
        ORDER BY distance
        LIMIT 3
    """, user_id, phash)
    results.extend([{**r, "match_type": "visual"} for r in visual_dups])

    # 策略2: 语义相似（CLIP 向量）
    semantic_dups = await qdrant.search(
        collection="item_visual",
        vector=embedding,
        filter={"must": [
            {"key": "user_id", "match": {"value": user_id}},
            {"key": "type", "match": {"value": "thing"}}
        ]},
        limit=3,
        score_threshold=0.88
    )
    for r in semantic_dups:
        if not any(d["id"] == r.id for d in results):
            results.append({
                "id": r.id, "name": r.payload["name"], 
                "match_type": "semantic", "score": r.score
            })

    # 策略3: 名称相似（同一类物品）
    name_dups = await pg.fetchall("""
        SELECT n.id, n.name, i.primary_image_url
        FROM life_nodes n
        JOIN item_details i ON n.id = i.node_id
        WHERE n.user_id = $1 AND n.type = 'thing'
          AND similarity(n.name, $2) > 0.6
        LIMIT 3
    """, user_id, name)
    for r in name_dups:
        if not any(d["id"] == r["id"] for d in results):
            results.append({**r, "match_type": "name"})

    return results
```

### 3.3 创建物品

```python
@router.post("/create")
async def create_item(req: CreateItemRequest):
    # 1. 创建 life_nodes 记录
    node_id = await pg.execute("""
        INSERT INTO life_nodes (user_id, type, name, tags, image_url, source, confidence)
        VALUES ($1, 'thing', $2, $3, $4, 'camera', 0.9)
        RETURNING id
    """, req.user_id, req.name, req.tags, req.image_url)

    # 2. 创建 item_details
    await pg.execute("""
        INSERT INTO item_details (
            node_id, room_id, container_id, location_note,
            expiry_date, expiry_remind_days,
            is_document, document_type, document_number,
            purchase_date, purchase_price,
            visual_hash, clip_embedding, primary_image_url,
            quantity, unit, condition
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
    """, 
        node_id, req.room_id, req.container_id, req.location_note,
        req.expiry_date, req.expiry_remind_days or 30,
        req.is_document, req.document_type, req.document_number,
        req.purchase_date, req.purchase_price,
        req.visual_hash, req.clip_embedding, req.image_url,
        req.quantity or 1, req.unit or '个', 'good'
    )

    # 3. 存入 Qdrant（用于语义搜索和查重）
    await qdrant.upsert(
        collection="item_visual",
        points=[{
            "id": str(node_id),
            "vector": req.clip_embedding,
            "payload": {
                "user_id": req.user_id,
                "type": "thing",
                "name": req.name,
                "tags": req.tags,
                "room_id": str(req.room_id) if req.room_id else None,
                "container_id": str(req.container_id) if req.container_id else None
            }
        }]
    )

    # 4. 如果有有效期，创建提醒
    if req.expiry_date:
        remind_at = req.expiry_date - timedelta(days=req.expiry_remind_days or 30)
        await create_expiry_reminder(req.user_id, node_id, req.name, req.expiry_date, remind_at)

    # 5. 如果是证件，创建到期提醒
    if req.is_document and req.expiry_date:
        await create_document_reminder(req.user_id, node_id, req.name, req.document_type, req.expiry_date)

    return {"id": node_id, "message": "物品已创建"}
```

---

## 四、房间与容器系统

### 4.1 管理页面

```tsx
// frontend/src/pages/RoomsPage.tsx
export default function RoomsPage() {
  const { data: rooms } = useSWR('/api/v1/rooms', fetcher);
  const [editingRoom, setEditingRoom] = useState(null);

  return (
    <div className="rooms-page">
      <h2>🏠 我的空间</h2>

      {rooms?.map(room => (
        <div key={room.id} className="room-section">
          <div className="room-header">
            <span className="room-icon">{room.icon}</span>
            <h3>{room.name}</h3>
            <span className="item-count">{room.item_count} 件物品</span>
            <button onClick={() => setEditingRoom(room)}>编辑</button>
          </div>

          <div className="containers-list">
            {room.containers?.map(c => (
              <div key={c.id} className="container-card">
                <span>{c.icon}</span>
                <span>{c.name}</span>
                <span>{c.item_count} 件</span>
              </div>
            ))}
            <button onClick={() => addContainer(room.id)}>+ 添加容器</button>
          </div>
        </div>
      ))}

      <button onClick={() => addRoom()}>+ 添加房间</button>
    </div>
  );
}
```

### 4.2 添加物品时选择位置

```tsx
// frontend/src/components/LocationPicker.tsx
export default function LocationPicker({ value, onChange }) {
  const { data: rooms } = useSWR('/api/v1/rooms', fetcher);
  const { data: containers } = useSWR(
    value.room_id ? `/api/v1/containers?room=${value.room_id}` : null,
    fetcher
  );

  return (
    <div className="location-picker">
      <div className="field">
        <label>放在哪个房间？</label>
        <div className="room-grid">
          {rooms?.map(r => (
            <button 
              key={r.id}
              className={value.room_id === r.id ? 'selected' : ''}
              onClick={() => onChange({ ...value, room_id: r.id, container_id: null })}
            >
              <span className="icon">{r.icon}</span>
              <span>{r.name}</span>
            </button>
          ))}
        </div>
      </div>

      {value.room_id && (
        <div className="field">
          <label>放在哪里？</label>
          <div className="container-grid">
            <button 
              className={!value.container_id ? 'selected' : ''}
              onClick={() => onChange({ ...value, container_id: null })}
            >
              直接放在{r.name}
            </button>
            {containers?.map(c => (
              <button 
                key={c.id}
                className={value.container_id === c.id ? 'selected' : ''}
                onClick={() => onChange({ ...value, container_id: c.id })}
              >
                <span>{c.icon}</span>
                <span>{c.name}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="field">
        <label>具体位置备注（可选）</label>
        <input 
          type="text" 
          placeholder="如：书架第三层"
          value={value.location_note || ''}
          onChange={(e) => onChange({ ...value, location_note: e.target.value })}
        />
      </div>
    </div>
  );
}
```

---

## 五、有效期与证件到期提醒

### 5.1 提醒生成逻辑

```go
// backend/internal/worker/expiry_reminder.go
func (w *Worker) CheckExpiringItems(ctx context.Context) error {
    now := time.Now()

    // 1. 查找即将过期的物品（食品、药品、化妆品）
    items, _ := w.db.QueryContext(ctx, `
        SELECT n.id, n.name, i.expiry_date, i.expiry_remind_days,
               i.room_id, i.container_id, r.name as room_name, c.name as container_name
        FROM life_nodes n
        JOIN item_details i ON n.id = i.node_id
        LEFT JOIN rooms r ON i.room_id = r.id
        LEFT JOIN containers c ON i.container_id = c.id
        WHERE i.expiry_date IS NOT NULL
          AND i.expiry_date <= $1
          AND NOT EXISTS (
              SELECT 1 FROM today_cards 
              WHERE node_id = n.id 
                AND local_day = $2
                AND title LIKE '%过期%'
          )
    `, now.AddDate(0, 0, 7), now.Format("2006-01-02"))

    for _, item := range items {
        daysUntil := int(item.ExpiryDate.Sub(now).Hours() / 24)

        var title, body string
        if daysUntil < 0 {
            title = fmt.Sprintf("%s 已过期", item.Name)
            body = fmt.Sprintf("%s 已过期 %d 天，建议检查是否还能使用", item.Name, -daysUntil)
        } else if daysUntil == 0 {
            title = fmt.Sprintf("%s 今天过期", item.Name)
            body = fmt.Sprintf("%s 今天过期，放在 %s %s", item.Name, item.RoomName, item.ContainerName)
        } else {
            title = fmt.Sprintf("%s 还有 %d 天过期", item.Name, daysUntil)
            body = fmt.Sprintf("%s 还有 %d 天过期，放在 %s %s", item.Name, daysUntil, item.RoomName, item.ContainerName)
        }

        w.cardRepo.Create(ctx, &TodayCard{
            UserID:    item.UserID,
            LocalDay:  now.Format("2006-01-02"),
            Slot:      "guidance",
            NodeID:    &item.ID,
            Title:     title,
            Body:      body,
            Severity:  2,
            ActionLabel: "查看",
        })

        // 如果小于3天，发 Push
        if daysUntil <= 3 {
            w.pushService.Send(ctx, item.UserID, title, body)
        }
    }

    // 2. 查找即将到期的证件
    docs, _ := w.db.QueryContext(ctx, `
        SELECT n.id, n.name, i.document_type, i.expiry_date, i.document_number
        FROM life_nodes n
        JOIN item_details i ON n.id = i.node_id
        WHERE i.is_document = true
          AND i.expiry_date IS NOT NULL
          AND i.expiry_date <= $1
    `, now.AddDate(0, 3, 0)) // 提前3个月提醒证件

    for _, doc := range docs {
        daysUntil := int(doc.ExpiryDate.Sub(now).Hours() / 24)

        title := fmt.Sprintf("%s 即将到期", doc.Name)
        body := fmt.Sprintf("您的 %s（%s）还有 %d 天到期，请提前办理续期", 
            doc.Name, doc.DocumentNumber, daysUntil)

        w.cardRepo.Create(ctx, &TodayCard{
            UserID:      doc.UserID,
            LocalDay:    now.Format("2006-01-02"),
            Slot:        "pinned",
            NodeID:      &doc.ID,
            Title:       title,
            Body:        body,
            Severity:    3,
            ActionLabel: "查看证件",
        })

        w.pushService.Send(ctx, doc.UserID, title, body)
    }

    return nil
}
```

### 5.2 证件专用识别

```python
DOCUMENT_PROMPT = """You are a document recognition expert. Analyze the image and determine:

1. is_document: true if this is passport/ID card/driver license/visa/warranty card/insurance card
2. document_type: passport / national_id / driver_license / visa / warranty / insurance / other
3. document_number: The document number if clearly visible
4. name: Full name on document if visible
5. expiry_date: ISO date of expiration if visible
6. issuing_authority: Authority that issued the document
7. country: Country of issue

If not a document, return is_document: false and treat as regular item.

Respond ONLY with valid JSON."""
```

---

## 六、重复物品检测

### 6.1 检测策略

```
三重查重：
1. 视觉相似（pHash 汉明距离 < 12）→ 同一物品不同角度
2. 语义相似（CLIP 向量相似度 > 0.88）→ 同类物品
3. 名称相似（PG similarity > 0.6）→ 同一品牌型号

用户拍照后：
    → 如果检测到重复：
        "这个物品家里已经有了：
         [图片] 牛奶 1L
         放在：🍳厨房 ❄️冰箱
         有效期：2026-08-15

         [更新数量] [这是新的] [取消]"

    → 如果没有重复：
        直接进入创建流程
```

### 6.2 前端重复提示

```tsx
// frontend/src/components/DuplicateAlert.tsx
export default function DuplicateAlert({ duplicates, onUpdate, onCreateNew, onCancel }) {
  return (
    <div className="duplicate-alert">
      <div className="alert-header">
        <span>⚠️</span>
        <h3>这个物品家里已经有了</h3>
      </div>

      {duplicates.map(dup => (
        <div key={dup.id} className="duplicate-item">
          <img src={dup.primary_image_url} className="thumb" />
          <div className="info">
            <h4>{dup.name}</h4>
            <p>放在：{dup.room_icon} {dup.room_name} {dup.container_icon} {dup.container_name}</p>
            {dup.expiry_date && <p>有效期至：{formatDate(dup.expiry_date)}</p>}
            <p>匹配方式：{dup.match_type === 'visual' ? '看起来一样' : 
                         dup.match_type === 'semantic' ? '看起来相似' : '名字相似'}</p>
          </div>
        </div>
      ))}

      <div className="actions">
        <button onClick={() => onUpdate(duplicates[0].id)}>
          📦 更新现有记录（数量+1）
        </button>
        <button onClick={onCreateNew}>
          🆕 这是新的（创建新记录）
        </button>
        <button onClick={onCancel}>
          ❌ 取消
        </button>
      </div>
    </div>
  );
}
```

---

## 七、API 设计

### 7.1 物品相关

```yaml
# 物品识别与创建
POST   /api/v1/items/analyze           # 分析图片/文件
POST   /api/v1/items/create            # 创建物品（用户确认后）
GET    /api/v1/items                    # 物品列表（支持 room/container/search 筛选）
GET    /api/v1/items/{id}               # 物品详情
PATCH  /api/v1/items/{id}               # 更新物品
DELETE /api/v1/items/{id}               # 删除物品
POST   /api/v1/items/{id}/duplicate    # 标记为重复（合并到另一个）

# 房间与容器
GET    /api/v1/rooms                    # 房间列表
POST   /api/v1/rooms                    # 创建房间
PATCH  /api/v1/rooms/{id}               # 更新房间
DELETE /api/v1/rooms/{id}               # 删除房间

GET    /api/v1/containers               # 容器列表（可按 room 筛选）
POST   /api/v1/containers               # 创建容器
PATCH  /api/v1/containers/{id}          # 更新容器
DELETE /api/v1/containers/{id}          # 删除容器

# 位置查询
POST   /api/v1/items/where-is           # "我的护照在哪？"
       body: { query: "护照" }
       response: { 
         items: [{
           name: "护照",
           room: { name: "卧室", icon: "🛏️" },
           container: { name: "抽屉", icon: "🗄️" },
           location_note: "床头柜第二层",
           image_url: "..."
         }]
       }

# 有效期提醒
GET    /api/v1/items/expiring           # 即将过期物品
GET    /api/v1/items/documents          # 证件列表（含到期日）
POST   /api/v1/items/{id}/snooze-expiry # 推迟提醒
```

### 7.2 "东西在哪" API

```go
// backend/internal/api/items.go
func (h *ItemHandler) WhereIs(c echo.Context) error {
    userID := getUserID(c)
    query := c.QueryParam("q")

    // 1. 语义搜索找到物品
    embedding := h.embedder.Embed(query)

    // 2. 多路召回
    var items []ItemLocation

    // 语义召回
    semantic := h.qdrant.Search("item_visual", embedding,
        map[string]interface{}{"user_id": userID.String(), "type": "thing"}, 5)

    // 名称召回
    nameMatches := h.itemRepo.SearchByName(ctx, userID, query, 5)

    // 合并去重
    seen := make(map[string]bool)
    for _, r := range semantic {
        if !seen[r.ID] {
            seen[r.ID] = true
            item := h.itemRepo.GetWithLocation(ctx, uuid.MustParse(r.ID))
            items = append(items, item)
        }
    }
    for _, r := range nameMatches {
        if !seen[r.ID.String()] {
            seen[r.ID.String()] = true
            items = append(items, r)
        }
    }

    // 3. 生成自然语言回答
    if len(items) == 1 {
        item := items[0]
        answer := fmt.Sprintf("%s 放在 %s %s", 
            item.Name, item.RoomName, item.ContainerName)
        if item.LocationNote != "" {
            answer += "，" + item.LocationNote
        }
        return c.JSON(200, map[string]interface{}{
            "type": "found",
            "answer": answer,
            "item": item,
        })
    }

    if len(items) > 1 {
        return c.JSON(200, map[string]interface{}{
            "type": "multiple",
            "answer": fmt.Sprintf("找到了 %d 个相关物品", len(items)),
            "items": items,
        })
    }

    return c.JSON(200, map[string]interface{}{
        "type": "not_found",
        "answer": "没有找到这个物品，要添加吗？",
    })
}
```

---

## 八、前端组件

### 8.1 物品详情页

```tsx
// frontend/src/pages/ItemDetailPage.tsx
export default function ItemDetailPage({ itemId }) {
  const { data: item } = useSWR(`/api/v1/items/${itemId}`, fetcher);
  const [isEditing, setIsEditing] = useState(false);

  if (!item) return <Loading />;

  return (
    <div className="item-detail">
      {/* 图片轮播 */}
      <div className="image-gallery">
        <img src={item.primary_image_url} className="main-image" />
        {item.gallery_urls?.map(url => <img key={url} src={url} className="thumb" />)}
        <button onClick={() => addPhoto()}>+ 添加照片</button>
      </div>

      {/* 基本信息 */}
      <div className="info-section">
        <h1>{item.name}</h1>
        <div className="tags">
          {item.tags?.map(t => <span key={t} className="tag">{t}</span>)}
        </div>

        {/* 位置 */}
        <div className="location-card">
          <h3>📍 放在哪里</h3>
          <div className="location-path">
            <span>{item.room_icon} {item.room_name}</span>
            <span>→</span>
            <span>{item.container_icon} {item.container_name}</span>
          </div>
          {item.location_note && <p className="note">{item.location_note}</p>}
          <button onClick={() => setIsEditing(true)}>修改位置</button>
        </div>

        {/* 有效期 */}
        {item.expiry_date && (
          <div className={`expiry-card ${item.days_until_expiry <= 7 ? 'warning' : ''}`}>
            <h3>⏰ 有效期</h3>
            <p>{formatDate(item.expiry_date)}</p>
            <p>{item.days_until_expiry > 0 ? `还有 ${item.days_until_expiry} 天` : '已过期'}</p>
          </div>
        )}

        {/* 证件信息 */}
        {item.is_document && (
          <div className="document-card">
            <h3>🪪 证件信息</h3>
            <p>类型：{item.document_type}</p>
            <p>号码：{item.document_number}</p>
            <p>到期日：{formatDate(item.expiry_date)}</p>
          </div>
        )}

        {/* 数量 */}
        <div className="quantity-control">
          <button onClick={() => updateQuantity(-1)}>-</button>
          <span>{item.quantity} {item.unit}</span>
          <button onClick={() => updateQuantity(1)}>+</button>
        </div>

        {/* 操作 */}
        <div className="actions">
          <button onClick={() => markAsLent()}>借出</button>
          <button onClick={() => moveItem()}>移动位置</button>
          <button onClick={() => deleteItem()} className="danger">删除</button>
        </div>
      </div>
    </div>
  );
}
```

### 8.2 "东西在哪" 组件

```tsx
// frontend/src/components/WhereIs.tsx
export default function WhereIs() {
  const [query, setQuery] = useState('');
  const [result, setResult] = useState(null);

  async function search() {
    const res = await fetch(`/api/v1/items/where-is?q=${encodeURIComponent(query)}`);
    setResult(await res.json());
  }

  return (
    <div className="where-is">
      <div className="search-box">
        <input 
          type="text" 
          placeholder="我的护照在哪？"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && search()}
        />
        <button onClick={search}>🔍</button>
      </div>

      {result && (
        <div className="result">
          {result.type === 'found' && (
            <div className="found">
              <p className="answer">{result.answer}</p>
              {result.item.image_url && <img src={result.item.image_url} />}
              <button onClick={() => navigate(`/items/${result.item.id}`)}>
                查看详情
              </button>
            </div>
          )}

          {result.type === 'multiple' && (
            <div className="multiple">
              <p>{result.answer}</p>
              {result.items.map(item => (
                <div key={item.id} className="item-row" onClick={() => navigate(`/items/${item.id}`)}>
                  <img src={item.primary_image_url} />
                  <span>{item.name}</span>
                  <span>{item.room_icon} {item.room_name}</span>
                </div>
              ))}
            </div>
          )}

          {result.type === 'not_found' && (
            <div className="not-found">
              <p>{result.answer}</p>
              <button onClick={() => openCamera()}>📷 拍照添加</button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
```

---

## 九、施工时间线（12 周）

| 周 | 模块 | 任务 | 产出 |
|---|---|---|---|
| **1** | 基础设施 | Docker Compose、数据库 10 张表、Go API 骨架、用户认证 | 可运行的 API 服务 |
| **2** | 房间容器 | rooms/containers CRUD、默认种子数据、前端页面 | 空间管理页面 |
| **3** | 物品识别 | GPT-4V 提取、CLIP embedding、pHash、图片上传 | /items/analyze API |
| **4** | 查重系统 | 三重查重（视觉/语义/名称）、重复提示 UI | 重复检测可用 |
| **5** | 物品创建 | 创建流程、位置选择器、标签编辑、Qdrant 存储 | 完整添加物品流程 |
| **6** | 聚合页 | 物品网格、房间/容器筛选、搜索、排序 | ItemsPage 可用 |
| **7** | 详情页 | 物品详情、图片轮播、位置修改、数量控制 | ItemDetailPage 可用 |
| **8** | 有效期 | 过期提醒 Worker、Today Card 生成、Push 通知 | 自动提醒可用 |
| **9** | 证件系统 | 证件识别、到期提醒、证件专用页面 | 证件管理可用 |
| **10** | "在哪" | WhereIs API、自然语言回答、语音输入 | "东西在哪" 可用 |
| **11** | 连接器 | Tesla/Plaid/Granola/Google/Flomo/Apple Health | 6 个连接器 |
| **12** | 日报+上线 | 日报生成、TTS、Capacitor iOS、App Store | 上线 |

---

## 十、Code Agent 指令清单

### 10.1 数据库迁移

```bash
# 创建迁移文件
mkdir -p backend/migrations
cat > backend/migrations/001_items_system.sql << 'EOF'
-- 见上文"数据模型"部分
EOF

# 运行迁移
psql $DATABASE_URL -f backend/migrations/001_items_system.sql
```

### 10.2 后端任务清单

```
[ ] 1. 创建 internal/items 包
    [ ] models.go - Item, Room, Container, ItemDetail 结构体
    [ ] repository.go - CRUD 操作
    [ ] service.go - 业务逻辑（识别、查重、提醒）
    [ ] handler.go - HTTP handler

[ ] 2. 创建 internal/vision 包
    [ ] gpt4v.go - GPT-4V 调用
    [ ] clip.go - CLIP embedding
    [ ] phash.go - 感知哈希
    [ ] duplicate.go - 查重逻辑

[ ] 3. 创建 internal/worker 任务
    [ ] expiry_reminder.go - 过期检查
    [ ] document_reminder.go - 证件到期检查
    [ ] daily_brief.go - 日报生成

[ ] 4. 更新 ai-service
    [ ] routers/items.py - 物品识别 API
    [ ] prompts/item_extraction.py - 提取 prompt
    [ ] prompts/document_extraction.py - 证件提取 prompt

[ ] 5. Qdrant collection
    [ ] 创建 item_visual collection
    [ ] 定义 payload schema
```

### 10.3 前端任务清单

```
[ ] 1. 创建 pages/ItemsPage.tsx
[ ] 2. 创建 pages/ItemDetailPage.tsx
[ ] 3. 创建 pages/RoomsPage.tsx
[ ] 4. 创建 components/CaptureBar.tsx（更新版）
[ ] 5. 创建 components/CameraSheet.tsx（物品专用）
[ ] 6. 创建 components/LocationPicker.tsx
[ ] 7. 创建 components/DuplicateAlert.tsx
[ ] 8. 创建 components/WhereIs.tsx
[ ] 9. 创建 components/ItemCard.tsx
[ ] 10. 更新 BottomNav.tsx（短按相机）
```

### 10.4 iOS 端上任务

```
[ ] 1. VisionPlugin.swift - 通用 OCR（非药品专用）
[ ] 2. LocationPlugin.swift - 地理围栏
[ ] 3. PushNotification.swift - APNs 配置
[ ] 4. HealthKitPlugin.swift - Apple Health 读取
```

### 10.5 测试清单

```
[ ] 1. 拍照识别物品（食品/证件/电子产品）
[ ] 2. 重复检测（同一物品不同角度）
[ ] 3. 位置选择（房间→容器→备注）
[ ] 4. 有效期提醒（7天内过期）
[ ] 5. 证件到期提醒（3个月内）
[ ] 6. "东西在哪"查询
[ ] 7. 聚合页筛选（房间/容器/搜索）
[ ] 8. 连接器数据同步
[ ] 9. 日报生成
[ ] 10. iOS 端到端测试
```

---

## 附录：Prompt 模板

### 通用物品识别

```
You are an item recognition expert. Analyze the image and extract:

1. name: Concise item name (max 20 chars, user's language)
2. category: food / document / electronics / clothing / medicine / cosmetic / book / tool / furniture / other
3. brand: Brand name if visible, else null
4. color: Dominant color
5. quantity: Number visible (default 1)
6. unit: piece / bottle / box / bag / can / pair / set
7. expiry_date: ISO date if visible (food/medicine/cosmetic), else null
8. is_document: true if passport/ID/license/visa/warranty/receipt/insurance
9. document_type: passport / national_id / driver_license / visa / warranty / insurance / receipt / null
10. document_number: if visible and is_document
11. suggested_room: living_room / bedroom / kitchen / bathroom / study / storage / balcony / entrance
12. suggested_container: fridge / cabinet / wardrobe / drawer / shelf / medicine_box / shoe_rack / null
13. tags: Array of 3-8 keywords for search
14. description: One sentence description

Rules:
- Be factual, do not invent information not visible
- Output language: same as user's input language
- For documents, always extract expiry_date if visible
- For food/medicine/cosmetic, always extract expiry_date if visible

Respond ONLY with valid JSON.
```

### 证件专用识别

```
You are a document expert. If the image is a document, extract:
- document_type
- document_number
- full_name (if visible)
- date_of_birth (if visible)
- issue_date (if visible)
- expiry_date (CRITICAL - must extract if visible)
- issuing_country
- issuing_authority

If not a document, return is_document: false.
```

---

*本文件为 Code Agent 施工指南，按周执行，每周结束时对照"产出"验收。*
