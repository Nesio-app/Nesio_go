# Testing

## Frontend E2E

```bash
cd frontend
npm install
npm run test:e2e
```

## Backend Contract Tests

```bash
cd backend
go test ./...
```

## Performance Smoke

```bash
cd backend
k6 run tests/perf/today.js
```

## Security Audit

```bash
cd backend
bash tests/security/audit.sh
```
