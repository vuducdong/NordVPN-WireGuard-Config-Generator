# API Documentation

Base URL: `https://[WORKER_DOMAIN]` or `http://localhost:8787` for local development.

## Response Format

- **Success**: Returns standard JSON with a `200 OK` status code.
- **Error**: Returns a JSON object `{"error": "message"}` with an appropriate HTTP status code (400, 401, 404, 429, 500, 503).

## Endpoints

### 1. Get Server List

Retrieves the cached, highly compressed list of available WireGuard-compatible NordVPN servers.

**Endpoint:** `GET /api/servers`

**Headers:**
- `If-None-Match`: (Optional) The ETag from a previous request.

**Response:**
- **200 OK**: Returns the server data.
- **304 Not Modified**: If the provided ETag matches the current data version.
- **503 Service Unavailable**: If the server cache is initializing.

**Response Body Structure:**
- `k`: Array of shared WireGuard public keys.
- `l`: Nested topology array of Country tuples: `[CountryName, CountryLowCode, Array of City tuples]`.
  - Each City tuple follows: `[CityName, Array of Server tuples]`.
  - Each Server tuple follows: `[number, load, ipNumeric, keyIndex, hostnameOverride?, dedupSuffix?]`.

---

### 2. Exchange Token

Exchanges a NordVPN access token for a WireGuard private key. The backend acts as a proxy to NordVPN and does not store the token or the key.

**Endpoint:** `POST /api/key`

**Request Body:**
```json
{
  "token": "string"
}
```

**Validation:**
- `token`: Must be a 64-character hexadecimal string.

**Response:**
```json
{
  "key": "string"
}
```

---

### 3. Manual Sync / Database Refresh

Forces the backend worker to immediately fetch the latest server list from NordVPN and update the Cloudflare KV database.

**Endpoint:** `POST /api/sync`

**Headers:**
- `Authorization`: `Bearer <deploy_token>`

**Response:**
- **200 OK**: `{ "success": true }`
- **401 Unauthorized**: If token is missing, mismatched, or already consumed.

---

## Rate Limiting

Rate limiting is enforced at the edge based on the extracted client IP.
- **Standard Endpoints (`/api/servers`, `/api/key`)**: `100` requests per 1 minute.