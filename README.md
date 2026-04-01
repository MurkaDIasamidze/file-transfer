# DriveClone

A self-hosted Google Drive clone. Upload, organise, star, trash and download files through a clean web UI. Files can be stored locally or in AWS S3 — switch with a single `.env` change, no code modifications needed.

---

## Repository layout

```
driveClone/
├── file-transfer-backend/    Go REST + WebSocket API
└── file-transfer-frontend/   React + TypeScript SPA
```

Detailed docs for each half:

- [`file-transfer-backend/README.md`](./file-transfer-backend/README.md)
- [`file-transfer-frontend/README.md`](./file-transfer-frontend/README.md)

---

## Architecture

```
┌──────────────────────────────────┐        ┌──────────────────────────────────┐
│           React SPA              │        │            Go API                │
│  Vite · React 18 · TypeScript    │        │   Fiber v2 · GORM · JWT          │
│                                  │        │                                  │
│  ┌───────────┐  ┌─────────────┐  │  HTTP  │  ┌──────────┐  ┌─────────────┐  │
│  │  Zustand  │  │    Axios    │◄─┼────────┼─►│ Handlers │  │  Services   │  │
│  │  store    │  │  (REST)     │  │        │  └──────────┘  └─────────────┘  │
│  └───────────┘  └─────────────┘  │        │        │              │         │
│                                  │        │  ┌─────▼──────────────▼──────┐  │
│  ┌──────────────────────────┐    │  WS    │  │     Repository layer      │  │
│  │    uploadService.ts      │◄───┼────────┼─►│  User · File · Folder     │  │
│  │  (chunked WS upload)     │    │        │  └─────────────┬─────────────┘  │
│  └──────────────────────────┘    │        │                │                │
└──────────────────────────────────┘        │         PostgreSQL              │
                                            │                                 │
                                            │  ┌──────────┐  ┌────────────┐  │
                                            │  │  Local   │  │  AWS S3    │  │
                                            │  │  disk    │  │  (async)   │  │
                                            │  └──────────┘  └────────────┘  │
                                            └─────────────────────────────────┘
```

---

## Upload flow

Files travel over WebSocket in base64-encoded chunks. Each chunk and the complete assembled file are verified with SHA-256 before being written to storage.

```
Browser                               Go server                     AWS S3
  │                                       │                            │
  │── WS /ws/upload?token=<jwt> ─────────►│                            │
  │                                       │                            │
  │── {type:"init", data:{...}} ─────────►│                            │
  │◄── {type:"init_ack", file_upload_id} ─│                            │
  │                                       │                            │
  │── {type:"chunk", data:{...}} ────────►│  (repeated N times)        │
  │◄── {type:"progress", percent:N%} ─────│                            │
  │                                       │                            │
  │── {type:"complete", ...} ────────────►│                            │
  │◄── {type:"done", file:{...}} ─────────│  status="processing"       │
  │                                       │                            │
  │  (user continues working)             │── PutObject ──────────────►│
  │                                       │◄── OK ─────────────────────│
  │                                       │  status="completed"        │
```

When S3 is enabled, the server replies `done` to the client immediately after checksum verification. The actual S3 upload runs in a background goroutine — the user never waits for it.

---

## Download flow

```
Browser                           Go server                       AWS S3
  │                                   │                              │
  │── GET /api/files/:id/download ───►│                              │
  │   Authorization: Bearer <jwt>     │── PresignGetObject(15 min) ──►│
  │                                   │◄── presigned URL ────────────│
  │◄── {url: "https://s3..."} ────────│                              │
  │                                   │                              │
  │── GET https://s3.amazonaws.com/...────────────────────────────────►│
  │◄── file bytes ─────────────────────────────────────────────────────│
  │  (saved as blob, filename preserved)
```

For local storage, the server streams the file directly via `SendFile`. The frontend handles both cases identically — always produces a proper download with the correct filename.

---

## Quick start (Docker Compose)

```yaml
# docker-compose.yml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_DB:       driveclone
      POSTGRES_USER:     postgres
      POSTGRES_PASSWORD: secret
    ports: ["5432:5432"]

  backend:
    build: ./file-transfer-backend
    depends_on: [db]
    env_file: ./file-transfer-backend/.env
    ports: ["8081:8081"]

  frontend:
    build: ./file-transfer-frontend
    ports: ["5173:80"]
    environment:
      VITE_API_URL: http://localhost:8081
      VITE_WS_URL:  ws://localhost:8081
```

```bash
docker compose up --build
```

---

## Feature summary

| Feature | Details |
|---------|---------|
| Auth | JWT, bcrypt passwords, localStorage persistence |
| File upload | WebSocket, chunked, SHA-256 verified, progress bars |
| Folder upload | Full directory tree via FileSystem API, relative paths preserved |
| Drag & drop | Files and folders, recursive walk |
| Storage | Local disk or AWS S3 (async background upload) |
| Download | Presigned S3 URL or direct stream; correct filename always |
| Multi-select | Checkbox selection, bulk star / trash / restore / delete |
| Trash | Soft-delete with restore; permanent delete |
| Starred | Quick access to important files |
| Recent | 20 most recently modified files |
| Account | Edit name, change password, sign out |
| Logging | Colored terminal output + JSON file (`logs/app.json`) |