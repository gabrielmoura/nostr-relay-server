# API Specification

## Overview

The Nostr Relay Server exposes two HTTP servers:

1. **External (Port)**: Public relay interface for Nostr clients
2. **Internal (Port+1)**: Admin and metrics endpoints

## WebSocket Protocol (NIP-01)

All Nostr communication happens over WebSocket using JSON messages.

### Message Types

#### Client → Relay

```json
["REQ", "<subscription_id>", <filter>]
["REQ", "<subscription_id>", <filter1>, <filter2>]
["EVENT", <event>]
["CLOSE", "<subscription_id>"]
["AUTH", <event>]
["COUNT", "<subscription_id>", <filter>]
```

#### Relay → Client

```json
["EVENT", "<subscription_id>", <event>]
["EOSE", "<subscription_id>"]
["OK", "<event_id>", <success>, <message>]
["NOTICE", "<message>"]
["CLOSED", "<subscription_id>", <reason>]
["AUTH", "<challenge>"]
```

### Filters

Filters define which events to subscribe to:

```json
{
  "ids": ["<event_id>"],
  "authors": ["<pubkey>", "<pubkey>"],
  "kinds": [0, 1, 2],
  "#e": ["<event_id>"],
  "#p": ["<pubkey>"],
  "since": 1690000000,
  "until": 1690100000,
  "limit": 100
}
```

## External Server Routes (Port)

### WebSocket Root `/`

**Upgrade:** WebSocket connection for Nostr protocol

**NIP-11:** Returns relay information when `Accept: application/nostr+json`

#### Request
```http
GET / HTTP/1.1
Upgrade: websocket
Connection: Upgrade
Accept: application/nostr+json
```

#### Response (NIP-11)
```json
{
  "name": "Nostr Relay Server",
  "description": "A Nostr Relay Server",
  "pub_key": "7ef721e77149c737...",
  "supported_nips": [1, 2, 4, 9, 11, 17, 25, 42, 45],
  "software": "https://github.com/gabrielmoura/nostr-relay-server",
  "version": "0.1.0",
  "limitation": {
    "max_message_length": 1048576,
    "max_subscriptions": 20,
    "max_filters": 100,
    "max_limit": 5000,
    "auth_required": false
  }
}
```

### Static Files

| Path | Description |
|------|-------------|
| `/nostr.png` | Relay icon |

### Well-Known Routes

#### `/.well-known/nostr/nip96.json`

Blossom server configuration (NIP-96).

**Response:**
```json
{
  "api_url": "http://localhost:9090/upload",
  "download_url": "http://localhost:9090/blob",
  "supported_nips": [1, 4, 5, 78, 94, 96, 98],
  "content_types": ["image/jpeg", "image/png", "video/mp4"],
  "tos_url": "http://localhost:9090/terms-of-service"
}
```

#### `/.well-known/nostr.json`

NIP-05 and media configuration.

**Query Parameters:**
- `?name=<username>` - Lookup NIP-05 user

**Response:**
```json
{
  "names": {
    "user": "npub1..."
  },
  "media": {
    "api_path": "http://localhost:9090/upload",
    "media_path": "http://localhost:9090/blob",
    "accepted_mimetypes": ["image/jpeg", "image/png"],
    "content_policy": {
      "allow_adult_content": false,
      "allow_violent_content": false
    }
  }
}
```

### Blossom Upload/Download (NIP-96)

#### `POST /upload`

Upload a file to Blossom storage.

**Headers:**
```
Authorization: Nostr <base64_event>
Content-Type: <mime_type>
```

**Body:** Binary file content

**Response (200):**
```json
{
  "hash": "sha256_hash_of_file",
  "url": "http://localhost:9090/blob/sha256_hash",
  "mime_type": "image/png",
  "created_at": 1690000000
}
```

**Errors:**
- `400` - Invalid file type or empty body
- `401` - Authentication failed or hash mismatch
- `500` - Server error

#### `GET /blob/:id`

Download a file by hash.

**Response:** Binary file with `Content-Type` header

#### `HEAD /blob/:id`

Check if file exists.

**Response Headers:**
- `200 OK` if exists
- `404 Not Found` if not

#### `GET /list/:id`

Get file metadata.

**Response:**
```json
{
  "hash": "sha256_hash",
  "link": "http://localhost:9090/blob/sha256_hash",
  "mime_type": "image/png",
  "created_at": 1690000000
}
```

### Redirect Routes

| Path | Redirects To |
|------|--------------|
| `/terms-of-service` | `{api_path}/terms-of-service` |

## Internal Server Routes (Port+1)

### `GET /metrics`

Prometheus metrics endpoint.

**Response:** Prometheus text format

### `GET /admin`

Admin interface placeholder.

**Response:** `"Admin Interface"`

## Negentropy Protocol (NIP-47)

Efficient relay synchronization.

### Messages

```
["NEG-OPEN", "<subscription_id>", <filter>, <scheme>]
["NEG-MSG", "<subscription_id>", <message>]
["NEG-HAVE", "<subscription_id>", <id>, <timestamp>, <size>]
["NEG-NEED", "<subscription_id>", <id>]
["NEG-ERROR", "<subscription_id>", <error>]
["NEG-CLOSE", "<subscription_id>"]
```

## Supported NIPs

| NIP | Name | Status |
|-----|------|--------|
| 01 | Basic Protocol | ✅ |
| 02 | Follow List | ✅ |
| 04 | Encrypted Direct Messages | ✅ |
| 09 | Event Deletion | ✅ |
| 11 | Relay Information | ✅ |
| 17 | Relay List Metadata | ✅ |
| 18 | Public Chat | ✅ |
| 25 | Reactions | ✅ |
| 40 | Expiration Timestamp | ✅ |
| 42 | Authentication | ✅ |
| 45 | Event Counts | ✅ |
| 50 | Search | ✅ |
| 62 | Request to Vanish | ✅ |
| 77 | Kind 30078 | ✅ |
| 96 | Blossom Storage | ✅ |
| 98 | HTTP Auth | ✅ |

## Error Responses

### WebSocket Notice
```json
["NOTICE", "error message here"]
```

### Event Rejection (OK Envelope)
```json
["OK", "<event_id>", false, "reason: event blocked"]
```

### Subscription Closed
```json
["CLOSED", "<subscription_id>", "auth-required: REQ filters are not accepted"]
```

## Rate Limits

Configured in `conf.yaml`:
```yaml
ws:
  rate_limit: 1    # requests per second
  burst: 5         # max burst size
```
