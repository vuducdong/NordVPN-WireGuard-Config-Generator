# API Documentation

Base URL: `https://[WORKER_DOMAIN]` or `http://localhost:8787` for local development.

## Response Format

- **Success**: Returns standard JSON, plain text, or binary streams with a `200 OK` status code.
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
- `h`: Array of column headers (`["name", "load", "station"]`).
- `l`: Nested object structure: `Country -> City -> Array of Servers`. Each server is an array matching the headers in `h`.

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

### 3. Generate Configuration

Generates a WireGuard configuration based on the selected server.

**Endpoints:**
- `POST /api/config` - Returns configuration text.
- `POST /api/config/download` - Returns configuration as a downloadable `.conf` file.
- `POST /api/config/qr` - Returns configuration as an SVG QR code image (Requires `mode: "server"`).

**Request Body:**
```json
{
  "name": "string",       // Required (Server name, e.g., "us1234")
  "dns": "string",        // Optional (Default: 103.86.96.100)
  "endpoint": "string",   // Optional ("hostname" or "station", default: "hostname")
  "keepalive": number,    // Optional (15-120, default: 25)
  "mode": "string"        // Optional ("server" or "client", default: "server")
}
```

**Validation Rules:**
- `name`: Required. Must map to an existing active server.
- `dns`: Optional. Comma-separated IPv4 addresses.
- `endpoint`: Optional. Resolves WireGuard Endpoint to domain name (`hostname`) or raw IP (`station`).
- `mode`: Controls zero-knowledge behavior.
  - `server`: The backend returns standard configuration output with an empty `PrivateKey=` block.
  - `client`: The backend returns a JSON payload containing the filename and a template string where `PrivateKey=__CLIENT_PK__`. The client locally hydrates the key.

**Response formats for `mode: "client"`:**
```json
{
  "filename": "us1234.conf",
  "template": "[Interface]\nPrivateKey=__CLIENT_PK__\n..."
}
```

---

### 4. Generate Batch Configurations

Generates configurations for all active cached servers matching the provided parameters.

**Endpoint:** `POST /api/config/batch`

**Request Body:**
```json
{
  "country": "string",    // Optional (Empty = global match)
  "city": "string",       // Optional (Empty = all cities in country)
  "dns": "string",        // Optional (Default: 103.86.96.100)
  "endpoint": "string",   // Optional ("hostname" or "station", default: "hostname")
  "keepalive": number,    // Optional (15-120, default: 25)
  "mode": "string"        // Optional ("server" or "client", default: "server")
}
```

**Response formats:**
- **`mode: "server"`**: Returns a standard `application/octet-stream` ZIP file containing all requested configurations with empty `PrivateKey` fields. The directory structure adapts to specificity (Global generates `Country/City/file.conf`, City generates flat files).
- **`mode: "client"`**: Returns a JSON payload mapping archive structure to templates for client-side hydration and compilation:
```json
{
  "archiveName": "NordVPN_us_new_york",
  "templates": [
    {
      "name": "us1234.conf",
      "template": "[Interface]\nPrivateKey=__CLIENT_PK__\n..."
    }
  ]
}
```

---

### 5. Manual Sync / Database Refresh

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
- **Standard Endpoints (`/api/servers`, `/api/key`, `/api/config/*`)**: `100` requests per 1 minute.
- **Batch Endpoint (`/api/config/batch`)**: `5` requests per 1 minute.