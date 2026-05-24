# API Documentation

Base URL: `http://localhost:3000`

## Response Format

- **Success**: Returns standard JSON with a `200 OK` status code.
- **Error**: Returns a JSON object `{"error": "message"}` with an appropriate HTTP status code (400, 401, 404, 429, 500, 503).

## Endpoints

### 1. Get Server List

Retrieves the cached list of available WireGuard-compatible NordVPN servers.

**Endpoint:** `GET /api/servers`

**Headers:**
- `If-None-Match`: (Optional) The ETag from a previous request.

**Response:**
- **200 OK**: Returns the server data.
- **304 Not Modified**: If the provided ETag matches the current data version.
- **503 Service Unavailable**: If the server cache is initializing.

**Response Body Structure:**
The data is optimized for network payload size.
- `h`: Array of column headers (e.g., `["name", "load", "station"]`).
- `l`: Nested object structure: `Country -> City -> Array of Servers`.
  - Each server is an array matching the headers in `h`.

### 2. Exchange Token

Exchanges a NordVPN access token for a WireGuard private key.

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
  "key": "string" // The NordLynx private key
}
```

### 3. Generate Configuration

Generates a WireGuard configuration based on the selected server and user credentials.

**Endpoints:**
- `POST /api/config` - Returns configuration as plain text.
- `POST /api/config/download` - Returns configuration as a downloadable `.conf` file.
- `POST /api/config/qr` - Returns configuration as an SVG QR code image.

**Request Body:**
```json
{
  "country": "string",    // Required
  "city": "string",       // Required
  "name": "string",       // Required (Server name, e.g., "us1234")
  "privateKey": "string", // Optional (If omitted, generated config will be invalid/empty)
  "dns": "string",        // Optional (Default: 103.86.96.100)
  "endpoint": "string",   // Optional ("hostname" or "station", default: hostname)
  "keepalive": number     // Optional (15-120, default: 25)
}
```

**Validation Rules:**
- `country`, `city`, `name`: Required fields. Must not be empty.
- `privateKey`: Optional. If provided, must be a valid 44-character Base64 WireGuard key ending in `=`. If omitted, configuration is generated with an empty `PrivateKey` field.
- `dns`: Optional. If provided, must be a list of valid comma-separated IPv4 addresses. Defaults to `103.86.96.100`.
- `endpoint`: Optional. Must be either `"hostname"` or `"station"`. Defaults to `hostname` representation.
- `keepalive`: Optional. If provided, must be an integer between `15` and `120`. Defaults to `25`.

**Response Headers:**
- **Text**: `Content-Type: text/plain`
- **File**: `Content-Type: application/x-wireguard-config`, `Content-Disposition: attachment; filename="[srv_code][srv_num].conf"`
- **QR**: `Content-Type: image/svg+xml`

### 4. Generate Batch Configurations

Generates a compressed `.nord` file (ZIP format) containing configuration files for matching active cached servers.

**Endpoint:** `POST /api/config/batch`

**Request Body:**
```json
{
  "country": "string",    // Optional (If omitted, matches all servers globally)
  "city": "string",       // Optional (If omitted, matches all servers in country)
  "privateKey": "string", // Optional (If omitted, generated configs will be invalid/empty)
  "dns": "string",        // Optional (Default: 103.86.96.100)
  "endpoint": "string",   // Optional ("hostname" or "station", default: hostname)
  "keepalive": number     // Optional (15-120, default: 25)
}
```

**Validation Rules:**
- `country`: Optional. If omitted or empty, compiles configurations for all active servers globally.
- `city`: Optional. If omitted or empty, compiles configurations for all active servers in the specified country.
- `privateKey`: Optional. If provided, must be a valid 44-character Base64 WireGuard key ending in `=`. If omitted, configurations are generated with an empty `PrivateKey` field.
- `dns`: Optional. If provided, must be a list of valid comma-separated IPv4 addresses. Defaults to `103.86.96.100`.
- `endpoint`: Optional. Must be either `"hostname"` or `"station"`. Defaults to `hostname` representation.
- `keepalive`: Optional. If provided, must be an integer between `15` and `120`. Defaults to `25`.

**Response Archive Structure:**
The folder structure inside the returned archive dynamically adjusts to parameter specificity to prevent filename collisions:
- **Global Batch** (no country provided): Generates nested directories formatted as `[Country]/[City]/[filename].conf`. File name is `NordVPN_All.nord`.
- **Country Batch** (country provided, no city): Generates nested directories formatted as `[City]/[filename].conf`. File name is `NordVPN_[Country].nord`.
- **City Batch** (both provided): Generates a flat archive directory containing configurations straight in the archive root formatted as `[filename].conf`. File name is `NordVPN_[Country]_[City].nord`.

**Response Headers:**
- **File**: `Content-Type: application/octet-stream`
- **Disposition**: `Content-Disposition: attachment; filename="NordVPN_[Name].nord"`

## Rate Limiting

- **Standard Endpoints**: `100` requests per 1 minute per IP address.
- **Batch Generation Endpoint (`/api/config/batch`)**: `5` requests per 1 minute per IP address.