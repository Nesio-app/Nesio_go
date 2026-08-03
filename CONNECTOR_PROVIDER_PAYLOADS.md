# Connectors Import Payload Examples

Use these request bodies directly with:

```bash
curl -X POST "$API_BASE/api/v1/connectors/<provider>/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '<payload>'
```

- `sync=true` (default): import then sync immediately.
- `sync=false`: import only, no immediate sync.

## 1) tesla_fleet

Required field:
- `access_token`

```json
{
  "access_token": "TESLA_BEARER_TOKEN",
  "endpoint": "https://fleet-api.prd.na.vn.cloud.tesla.com/api/1/vehicles"
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/tesla_fleet/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "TESLA_BEARER_TOKEN",
    "endpoint": "https://fleet-api.prd.na.vn.cloud.tesla.com/api/1/vehicles"
  }'
```

## 2) plaid

Required fields:
- `client_id`
- `secret`
- `access_token`

```json
{
  "client_id": "PLAID_CLIENT_ID",
  "secret": "PLAID_SECRET",
  "access_token": "PLAID_ACCESS_TOKEN",
  "endpoint": "https://production.plaid.com/accounts/balance/get"
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/plaid/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "PLAID_CLIENT_ID",
    "secret": "PLAID_SECRET",
    "access_token": "PLAID_ACCESS_TOKEN",
    "endpoint": "https://production.plaid.com/accounts/balance/get"
  }'
```

## 3) granola

Required field:
- `endpoint`

Optional:
- `access_token`

```json
{
  "endpoint": "https://your-granola-api.example.com/items",
  "access_token": "OPTIONAL_GRANOLA_BEARER"
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/granola/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://your-granola-api.example.com/items",
    "access_token": "OPTIONAL_GRANOLA_BEARER"
  }'
```

## 4) flomo

Required field:
- `endpoint`

Optional:
- `access_token`

```json
{
  "endpoint": "https://your-flomo-api.example.com/items",
  "access_token": "OPTIONAL_FLOMO_BEARER"
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/flomo/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "https://your-flomo-api.example.com/items",
    "access_token": "OPTIONAL_FLOMO_BEARER"
  }'
```

## 5) google_timeline

Required field:
- `timeline_events` (non-empty array)

```json
{
  "timeline_events": [
    {
      "name": "Morning commute",
      "lat": 31.2304,
      "lng": 121.4737,
      "start_at": "2026-08-02T08:10:00+08:00",
      "end_at": "2026-08-02T08:55:00+08:00"
    },
    {
      "name": "Office",
      "lat": 31.215,
      "lng": 121.52,
      "start_at": "2026-08-02T09:00:00+08:00",
      "end_at": "2026-08-02T18:10:00+08:00"
    }
  ]
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/google_timeline/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "timeline_events": [
      {
        "name": "Morning commute",
        "lat": 31.2304,
        "lng": 121.4737,
        "start_at": "2026-08-02T08:10:00+08:00",
        "end_at": "2026-08-02T08:55:00+08:00"
      }
    ]
  }'
```

## 6) apple_health

Required field:
- `entries` (non-empty array)

```json
{
  "entries": [
    {
      "metric": "steps",
      "value": 10234,
      "unit": "count",
      "recorded_at": "2026-08-02T21:00:00+08:00"
    },
    {
      "metric": "sleep_duration",
      "value": 7.2,
      "unit": "hour",
      "recorded_at": "2026-08-02T07:00:00+08:00"
    }
  ]
}
```

Example:

```bash
curl -X POST "$API_BASE/api/v1/connectors/apple_health/import?sync=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entries": [
      {
        "metric": "steps",
        "value": 10234,
        "unit": "count",
        "recorded_at": "2026-08-02T21:00:00+08:00"
      }
    ]
  }'
```

## Common Errors

- `400 unsupported provider`: provider not in the supported list.
- `400 missing required field: <field>`: payload does not satisfy provider required fields.
- `502 connector imported but sync failed: ...`: imported into DB but immediate sync failed (upstream unavailable, invalid token, endpoint error).
