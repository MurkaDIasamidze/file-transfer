# DriveClone — Backend

Go REST + WebSocket API powering the DriveClone file-storage service.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Framework | [Fiber v2](https://github.com/gofiber/fiber) |
| Database | PostgreSQL via [GORM](https://gorm.io) |
| Auth | JWT (HS256) via `golang-jwt/jwt` |
| Real-time upload | WebSocket (`gofiber/websocket`) |
| Password hashing | bcrypt |
| Config | `.env` + `godotenv` |

## Project structure

```
.
├── config/         Environment config loader
├── database/       GORM connection + AutoMigrate
├── handlers/
│   ├── auth.go     Register, Login, Me, UpdateProfile, ChangePassword
│   ├── file.go     File list/move/star/trash/restore/delete + WS upload handler
│   └── folder.go   Folder CRUD + trash/restore/delete
├── middleware/
│   ├── jwt.go      REST JWT middleware + WebSocket JWT middleware
│   └── logger.go   Structured request logger
├── models/         GORM models (User, Folder, FileUpload, FileChunk)
├── repository/     Data-access layer (UserRepository, FileRepository, FolderRepository)
├── services/       Business logic (AuthService, ChecksumService, FileService)
├── types/          Interface definitions (handlers, repositories, services)
├── utils/          Validation, error helpers, checksum utilities
├── main.go         Wire-up + Fiber app bootstrap
├── .env.example    Environment variable reference
└── Makefile        Common dev tasks
```

## Prerequisites

- Go 1.22 or later
- PostgreSQL 14+
- (Optional) `make`

## Quick start

```bash
# 1. Clone and enter the directory
git clone <repo-url>
cd file-transfer-backend

# 2. Copy and edit the environment file
cp .env.example .env
# Fill in DB_PASSWORD, JWT_SECRET, etc.

# 3. Install dependencies
go mod download

# 4. Run (auto-migrates the database on first start)
go run main.go
```

The server starts on the port defined by `SERVER_PORT` (default `8080`).

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP/WS listen port |
| `MAX_BODY_SIZE` | `104857600` | Max request body (bytes) |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `driveclone` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `UPLOAD_DIR` | `./uploads` | Directory where uploaded files are stored |
| `CHUNK_SIZE` | `1048576` | Server-side chunk size hint (bytes) |
| `JWT_SECRET` | `change-me-in-production` | HMAC secret — **change this** |
| `JWT_EXPIRY_HOURS` | `72` | JWT lifetime in hours |
| `DROP_TABLES` | _(unset)_ | Set to `true` to wipe and recreate the schema |

## REST API

All protected routes require `Authorization: Bearer <token>`.

### Auth

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/register` | Create account |
| `POST` | `/api/auth/login` | Obtain JWT |
| `GET` | `/api/me` | Current user |
| `PATCH` | `/api/me` | Update display name |
| `POST` | `/api/me/password` | Change password |

### Files

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/files` | List files (optional `?folder_id=`) |
| `GET` | `/api/files/recent` | 20 most-recently-updated files |
| `GET` | `/api/files/starred` | Starred files |
| `GET` | `/api/files/trash` | Trashed files |
| `PATCH` | `/api/files/:id/move` | Move to folder (`{folder_id: N\|null}`) |
| `PATCH` | `/api/files/:id/star` | Toggle star |
| `PATCH` | `/api/files/:id/trash` | Move to trash |
| `PATCH` | `/api/files/:id/restore` | Restore from trash |
| `DELETE` | `/api/files/:id` | Permanently delete |

### Folders

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/folders` | Create folder |
| `GET` | `/api/folders` | List folders (optional `?parent_id=`) |
| `GET` | `/api/folders/trash` | Trashed folders |
| `PATCH` | `/api/folders/:id/trash` | Move to trash |
| `PATCH` | `/api/folders/:id/restore` | Restore from trash |
| `DELETE` | `/api/folders/:id` | Permanently delete (cascades to files) |

### WebSocket upload

```
GET /ws/upload?token=<jwt>
```

The upload uses a three-step protocol per file over a single WebSocket connection:

```
Client                          Server
  │  {type:"init", data:{...}}    │
  │ ──────────────────────────>   │
  │  {type:"init_ack", file_upload_id, file_name}
  │ <──────────────────────────   │
  │                               │
  │  {type:"chunk", data:{...}}   │  (repeat for each chunk)
  │ ──────────────────────────>   │
  │  {type:"progress", ...}       │
  │ <──────────────────────────   │
  │                               │
  │  {type:"complete", data:{file_upload_id}}
  │ ──────────────────────────>   │
  │  {type:"done", file:{...}}    │
  │ <──────────────────────────   │
```

Each chunk is base64-encoded and includes a SHA-256 checksum. The server verifies per-chunk checksums and a final whole-file checksum before marking the upload `completed`.

### Health check

```
GET /health  →  {"status":"ok","time":<unix>}
```

## Database schema

```
users          id, name, email, password, avatar_url, timestamps
folders        id, user_id, parent_id, name, trashed, timestamps
file_uploads   id, user_id, folder_id, file_name, file_type, file_size,
               total_chunks, checksum, status, file_path, rel_path,
               starred, trashed, timestamps
file_chunks    id, file_upload_id, chunk_index, chunk_size, checksum, status, timestamps
```

`file_chunks` is kept for the `/upload/verify/:id` endpoint but is not used by the WebSocket upload path (chunks are held in memory during the session).

## Development

```bash
# Run with live reload (requires github.com/air-verse/air)
air

# Reset the database (drops all tables and re-migrates)
DROP_TABLES=true go run main.go

# Run tests
go test ./...
```