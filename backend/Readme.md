# DriveClone — Backend

Go REST + WebSocket API. Handles auth, file/folder management, chunked uploads with integrity verification, and optional async storage in AWS S3.

---

## Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.22+ |
| HTTP framework | [Fiber v2](https://github.com/gofiber/fiber) |
| Database | PostgreSQL via [GORM](https://gorm.io) |
| Auth | JWT HS256 — `golang-jwt/jwt` |
| WebSocket | `gofiber/websocket` |
| Password hashing | `bcrypt` |
| Cloud storage | AWS S3 — `aws-sdk-go-v2` |
| Config | `.env` + `godotenv` |
| Logging | Custom `slog` handler — colored terminal + JSON file |

---

## Project structure

```
file-transfer-backend/
├── config/
│   └── config.go           Reads .env into typed structs (Server, DB, JWT, S3, Upload)
├── database/
│   └── database.go         GORM connection + AutoMigrate
├── handlers/
│   ├── auth.go             Register · Login · Me · UpdateProfile · ChangePassword
│   ├── file.go             WS upload handler + REST file actions + async S3 goroutine
│   └── folder.go           Folder CRUD · trash · restore · delete (cascades)
├── logger/
│   └── logger.go           slog handler: colored terminal + append-only JSON file
├── middleware/
│   └── middleware.go       JWTMiddleware · WSJWTMiddleware · UserIDFromToken · WSUserID
├── models/
│   └── models.go           User · Folder · FileUpload · FileChunk
├── repository/
│   ├── file_repository.go
│   ├── folder_repository.go
│   └── user_repository.go
├── services/
│   ├── auth_service.go     Register/Login business logic, token generation
│   ├── checksum_service.go SHA-256 helpers
│   └── s3_service.go       Upload · Delete · PresignDownload · BuildKey
├── types/
│   └── types.go            Interface definitions for all layers
├── utils/
│   └── utils.go            BindAndValidate · AppError · Respond
├── logs/
│   └── app.json            Auto-created on first run (gitignore this)
├── main.go                 Wire all layers, register routes, start Fiber
├── test_s3.go              Standalone S3 connectivity test (go run test_s3.go)
├── .env.example
└── go.mod
```

---

## Quick start

```bash
# 1. Enter the directory
cd file-transfer-backend

# 2. Copy env and fill in values
cp .env.example .env

# 3. Download dependencies
go mod download

# 4. Start (auto-migrates on first run)
go run main.go
```

The server listens on `SERVER_PORT` (default `8081`).

---

## Environment variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8081` | Listen port |
| `MAX_BODY_SIZE` | `104857600` | Max HTTP body in bytes (100 MB) |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `driveclone` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |

### Auth

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | `change-me-in-production` | HMAC signing secret — **always override** |
| `JWT_EXPIRY_HOURS` | `72` | Token lifetime in hours |

### Upload

| Variable | Default | Description |
|----------|---------|-------------|
| `UPLOAD_DIR` | `./uploads` | Root directory for local file storage |
| `CHUNK_SIZE` | `1048576` | Chunk size hint in bytes (1 MB) |

### AWS S3 (optional)

Leave blank to use local disk storage. S3 is enabled automatically when both `AWS_ACCESS_KEY_ID` and `AWS_S3_BUCKET` are set.

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | — | e.g. `eu-central-1` |
| `AWS_ACCESS_KEY_ID` | — | IAM access key |
| `AWS_SECRET_ACCESS_KEY` | — | IAM secret key |
| `AWS_S3_BUCKET` | — | Bucket name |

### Other

| Variable | Default | Description |
|----------|---------|-------------|
| `DROP_TABLES` | — | Set `true` to wipe and re-migrate the schema on next start |

---

## REST API

All routes except `/health` and `/api/auth/*` require `Authorization: Bearer <token>`.

### Auth

| Method | Path | Body | Description |
|--------|------|------|-------------|
| `POST` | `/api/auth/register` | `{name, email, password}` | Create account → `{token, user}` |
| `POST` | `/api/auth/login` | `{email, password}` | Sign in → `{token, user}` |
| `GET` | `/api/me` | — | Current user |
| `PATCH` | `/api/me` | `{name}` | Update display name |
| `POST` | `/api/me/password` | `{current_password, new_password}` | Change password |

### Files

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/files` | List files (optional `?folder_id=N`) |
| `GET` | `/api/files/recent` | 20 most-recently-updated files |
| `GET` | `/api/files/starred` | Starred files |
| `GET` | `/api/files/trash` | Trashed files |
| `GET` | `/api/files/:id/download` | Get download URL (JSON `{url}` for S3, stream for local) |
| `PATCH` | `/api/files/:id/move` | Move to folder — `{folder_id: N\|null}` |
| `PATCH` | `/api/files/:id/star` | Toggle star |
| `PATCH` | `/api/files/:id/trash` | Soft-delete |
| `PATCH` | `/api/files/:id/restore` | Restore from trash |
| `DELETE` | `/api/files/:id` | Permanently delete (removes from S3 / disk too) |

### Folders

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/folders` | List folders (optional `?parent_id=N`) |
| `POST` | `/api/folders` | Create — `{name, parent_id?}` |
| `GET` | `/api/folders/trash` | Trashed folders |
| `PATCH` | `/api/folders/:id/trash` | Soft-delete |
| `PATCH` | `/api/folders/:id/restore` | Restore from trash |
| `DELETE` | `/api/folders/:id` | Permanently delete (cascades to files inside) |

### Health

```
GET /health  →  {"status":"ok","time":<unix>}
```

---

## WebSocket upload protocol

```
GET /ws/upload?token=<jwt>
```

One WebSocket connection per file. Three-phase protocol:

```
Client                                    Server
  │                                          │
  │── {type:"init", data:{                   │
  │     file_name, file_type, file_size,     │
  │     checksum, rel_path, folder_id}}  ───►│  creates FileUpload row (status=pending)
  │◄── {type:"init_ack",                     │
  │     file_upload_id, file_name} ──────────│
  │                                          │
  │  (for each chunk i = 0..N-1)            │
  │── {type:"chunk", data:{                  │
  │     file_upload_id, chunk_index,         │
  │     total_chunks, checksum, data}} ─────►│  verifies chunk SHA-256
  │◄── {type:"progress",                     │
  │     progress_percent, status} ───────────│
  │                                          │
  │── {type:"complete",                      │
  │     data:{file_upload_id}} ─────────────►│  assembles + verifies whole-file SHA-256
  │◄── {type:"done", file:{...}} ────────────│  status=processing (S3) or completed (local)
  │                                          │
  │  (S3 only — user continues working)      │── go PutObject ──► AWS S3
  │                                          │   status → completed on success
  │                                          │   status → failed on error
```

All `data` field values are base64-encoded. Chunk checksums are SHA-256 hex strings. An `{type:"error", message:"..."}` can be sent at any point.

---

## Async S3 upload

When S3 is configured, the server responds `done` to the WebSocket **before** the file reaches S3. The PutObject call runs in a goroutine. The file record transitions through:

```
pending → uploading → processing → completed
                                 → failed (if S3 errors)
```

The frontend shows `processing` files with a pulsing cloud icon and disables download until `completed`.

---

## Logging

Two outputs simultaneously:

**Terminal** — colored by level, file IDs highlighted in bright cyan:
```
2026/03/23 14:05:01  INFO   ⬆  upload started   file_id=42  file_name=report.pdf  file_size=2097152
2026/03/23 14:05:02  INFO   ⏳ queued for S3    file_id=42  file_name=report.pdf
2026/03/23 14:05:04  INFO   ✓  upload complete   file_id=42  elapsed=2.1s
2026/03/23 14:05:10  WARN   jwt auth failed      path=/api/folders  ip=127.0.0.1
2026/03/23 14:05:11  ERROR  S3 upload failed     file_id=43  err=NoSuchBucket
```

| Level | Color |
|-------|-------|
| DEBUG | Cyan |
| INFO | Green |
| WARN | Yellow |
| ERROR | Red |

**`logs/app.json`** — newline-delimited JSON, one record per line, never truncated:
```json
{"ts":"2026-03-23T14:05:04Z","level":"INFO","msg":"✓  upload complete","file_id":42,"elapsed":"2.1s"}
```

---

## Database schema

```
users
  id, name, email, password (bcrypt), avatar_url, created_at, updated_at, deleted_at

folders
  id, user_id, parent_id (nullable), name, trashed, created_at, updated_at, deleted_at

file_uploads
  id, user_id, folder_id (nullable), file_name, file_type, file_size
  total_chunks, checksum (SHA-256 hex), status, file_path (local path or S3 key)
  rel_path (folder upload relative path), starred, trashed
  created_at, updated_at, deleted_at

file_chunks
  id, file_upload_id, chunk_index, chunk_size, checksum, status
  (kept for /upload/verify/:id endpoint — not used by the WS upload path)
```

GORM runs `AutoMigrate` on every startup. To reset the schema:

```bash
DROP_TABLES=true go run main.go
```

---

## Testing S3

A standalone test script verifies your AWS credentials without starting the full server:

```bash
go run test_s3.go
```

It uploads a small text file, generates a presigned download URL, downloads and verifies the content, then deletes the file. All four steps are reported with ✅ / ❌ and timing.

---

## Development

```bash
# Live reload (requires github.com/air-verse/air)
air

# Run all tests
go test ./...

# Build binary
go build -o driveclone-server .
```