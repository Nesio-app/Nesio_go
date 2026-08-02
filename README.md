# Nesio_go

Nesio is a personal operating system prototype with:

- `backend/`: Go API, worker, Redis, Postgres, Qdrant integration
- `ai-service/`: FastAPI-based AI routing and card generation
- `frontend/`: React + Vite mobile-style UI with offline queue and Capacitor iOS shell

## Local Startup

### 1. Start the full local stack

```bash
docker compose up -d --build postgres redis qdrant ai-service api worker
docker compose ps
```

### 2. Start the frontend locally

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0 --port 5173
```

### 3. Optional production-like preview

```bash
cd frontend
npm run build
npm run preview -- --host 0.0.0.0 --port 4173
```

## Local Acceptance Flow

### 1. Register and get a token

```bash
EMAIL="tester_$(date +%s)@nesio.local"
curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/register \
	-H 'Content-Type: application/json' \
	-d "{\"email\":\"$EMAIL\",\"password\":\"secret123\"}"
```

Save the returned `token` and reuse it below:

```bash
TOKEN="paste-token-here"
```

### 2. Verify profile, memories, and today cards

```bash
curl -sS http://127.0.0.1:8080/api/v1/me \
	-H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://127.0.0.1:8080/api/v1/memories \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"title":"联调记录","body":"验证 memory -> today -> UI 链路","tags":["qa","milestone"]}'

curl -sS http://127.0.0.1:8080/api/v1/memories \
	-H "Authorization: Bearer $TOKEN"
```

### 3. Verify signal -> today -> SSE

Terminal A:

```bash
curl -N http://127.0.0.1:8080/api/v1/events \
	-H "Authorization: Bearer $TOKEN"
```

Terminal B:

```bash
FUTURE=$(date -u -d '+30 minutes' +%Y-%m-%dT%H:%M:%SZ)
curl -sS -X POST http://127.0.0.1:8080/api/v1/signals \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d "{\"source\":\"calendar\",\"anchor_id\":\"acceptance-sse\",\"fields\":{\"title\":\"SSE 验证会议\",\"start_time\":\"$FUTURE\",\"location\":\"线上\"},\"raw_data\":\"30分钟后有会议\",\"timestamp\":\"$FUTURE\"}"

curl -sS http://127.0.0.1:8080/api/v1/today \
	-H "Authorization: Bearer $TOKEN"
```

Expected:

- `today_cards` contains the new item
- `events` stream receives `event: today_card`

### 4. Verify domain CRUD

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/domains/日程/tasks \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"title":"给日程领域新增任务"}'

curl -sS -X POST http://127.0.0.1:8080/api/v1/domains/日程/memories \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"title":"一封重要邮件","body":"需要今晚回复"}'

curl -sS http://127.0.0.1:8080/api/v1/domains/日程/detail \
	-H "Authorization: Bearer $TOKEN"
```

### 5. Verify Gmail receive/send

First connect a Gmail connector by writing a valid Gmail OAuth access token:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/connectors/gmail/auth \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"credentials":{"access_token":"YOUR_GMAIL_ACCESS_TOKEN","account":"you@gmail.com"}}'
```

Then:

```bash
curl -sS http://127.0.0.1:8080/api/v1/connectors/gmail/inbox \
	-H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://127.0.0.1:8080/api/v1/connectors/gmail/send \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"to":"friend@example.com","subject":"来自 Nesio 的测试邮件","body":"这是一封联调测试邮件。"}'
```

## Validation Commands

### Backend

```bash
cd backend
go build ./cmd/api ./cmd/worker
go test ./...
```

### Frontend

```bash
cd frontend
npm install
npm run build
npm run test:e2e -- --reporter=line
```

### Security and performance

```bash
cd backend
bash tests/security/audit.sh
k6 run tests/perf/today.js
```

## iOS shell

```bash
cd frontend
npm install
npm run build
npx cap sync ios
npx cap open ios
```

Notes:

- `CocoaPods` and `xcodebuild` are required on macOS to complete native iOS builds.
- `govulncheck`, `gosec`, and `k6` are referenced in scripts and CI; install them locally if you want to run them outside GitHub Actions.
- I could not find the three uploaded `日程` screenshots in the workspace. If you want the UI to match those pixel-for-pixel, upload them into the repo or point me to their paths and I can align the page more tightly.