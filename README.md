# DriveClone

A self-hosted Google Drive clone built with Go (backend) and React (frontend).  
Upload, organise, and manage files through a clean web interface with real-time progress, folder support, drag & drop, starring, and trash.

---

## Repository layout

```
driveClone/
├── file-transfer-backend/   Go REST + WebSocket API
└── file-transfer-frontend/  React + TypeScript SPA
```

Each sub-directory has its own detailed README:

- [`file-transfer-backend/README.md`](./file-transfer-backend/README.md)
- [`file-transfer-frontend/README.md`](./file-transfer-frontend/README.md)

---

## Architecture overview

```
┌─────────────────────────────────┐        ┌──────────────────────────────────┐
│         React SPA               │        │           Go API                 │
│  (Vite · React 18 · TypeScript) │        │   (Fiber v2 · GORM · JWT)        │
│                                 │        │                                  │
│  ┌──────────┐  ┌─────────────┐  │  HTTP  │  ┌──────────┐  ┌─────────────┐  │
│  │  Zustand │  │    Axios    │◄─┼────────┼─►│ Handlers │  │  Services   │  │
│  │  store   │  │   (REST)    │  │        │  └──────────┘  └─────────────┘  │
│  └──────────┘  └─────────────┘  │        │       │               │         │
│                                 │        │  ┌────▼───────────────▼──────┐  │
│  ┌─────────────────────────┐    │  WS    │  │      Repository layer     │  │
│  │   uploadService.ts      │◄───┼────────┼─►│  (User · File · Folder)  │  │
│  │  (WS chunked upload)    │    │        │  └──────────────┬───────────┘  │
│  └─────────────────────────┘    │        │                 │              │
└─────────────────────────────────┘        │          PostgreSQL            │
                                           │          Local disk            │
                                           └──────────────────────────────┘
```

### Upload flow

Files are uploaded over WebSocket using a chunked protocol with SHA-256 integrity verification at both the chunk and whole-file level.

```
Browser                              Go server
  │  WS /ws/upload?token=<jwt>         │
  │─────────────────────────────────►  │
  │                                    │
  │  {type:"init", data:{name,size,…}} │
  │─────────────────────────────────►  │  creates FileUpload row (status=pending)
  │  {type:"init_ack", file_upload_id} │
  │◄─────────────────────────────────  │
  │                                    │
  │  {type:"chunk", data:{idx,b64,cs}} │  (one message per 256 KB chunk)
  │─────────────────────────────────►  │  verifies per-chunk SHA-256
  │  {type:"progress", …}              │
  │◄─────────────────────────────────  │
  │  … repeat for every chunk …        │
  │                                    │
  │  {type:"complete", file_upload_id} │
  │─────────────────────────────────►  │  reconstructs file, verifies whole-file SHA-256
  │  {type:"done", file:{…}}           │  updates status=completed
  │◄─────────────────────────────────  │
  │  WS close                          │
```

One WebSocket connection is used per file. Sequential uploads keep server resource usage predictable and the client protocol dead-simple.

---

## Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.22+ |
| Node.js | 18+ |
| PostgreSQL | 14+ |

---

## Getting started

### 1. Clone the repository

```bash
git clone <repo-url>
cd driveClone
```

### 2. Start PostgreSQL

```bash
# Docker (quickest)
docker run -d \
  --name driveclone-pg \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=driveclone \
  -p 5432:5432 \
  postgres:16
```

### 3. Start the backend

```bash
cd file-transfer-backend
cp .env.example .env       # edit JWT_SECRET and DB_PASSWORD at minimum
go mod download
go run main.go
# → listening on :8080
```

The database schema is created automatically on first run via GORM AutoMigrate.

### 4. Start the frontend

```bash
cd file-transfer-frontend
cp .env.example .env       # verify VITE_API_URL and VITE_WS_URL point to :8080
npm install
npm run dev
# → http://localhost:5173
```

Open `http://localhost:5173`, register an account, and start uploading.

---

## Docker Compose (optional)

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: driveclone
    volumes:
      - pgdata:/var/lib/postgresql/data

  backend:
    build: ./file-transfer-backend
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PASSWORD: postgres
      JWT_SECRET: change-me-in-production
      UPLOAD_DIR: /data/uploads
    volumes:
      - uploads:/data/uploads
    depends_on:
      - postgres

  frontend:
    build: ./file-transfer-frontend
    ports:
      - "80:80"
    environment:
      VITE_API_URL: http://localhost:8080
      VITE_WS_URL: ws://localhost:8080

volumes:
  pgdata:
  uploads:
```

```bash
docker compose up --build
```

---

## Feature summary

| Feature | Details |
|---------|---------|
| Auth | Register, login, JWT (72 h default), change password |
| File upload | WebSocket chunked, SHA-256 verified, progress per file |
| Folder upload | Full recursive tree via `<input webkitdirectory>` or drag & drop |
| Drag & drop | Files and folders (recursive via FileSystem API) |
| Folders | Create, nest, navigate via breadcrumbs |
| Drag to move | Drag a file card onto a folder card to move it |
| Star | Toggle star on any file |
| Trash | Soft-delete files and folders; restore or delete forever |
| Recent | Last 20 modified files |
| Search | Client-side filter across current view |
| Account | Edit display name, change password |

---

## Configuration reference

See the individual READMEs for the full list of environment variables:

- **Backend** — `SERVER_PORT`, `DB_*`, `JWT_SECRET`, `UPLOAD_DIR`, `CHUNK_SIZE`, …
- **Frontend** — `VITE_API_URL`, `VITE_WS_URL`, `VITE_CHUNK_SIZE`

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes (`git commit -m 'feat: add my feature'`)
4. Push the branch (`git push origin feat/my-feature`)
5. Open a pull request

Please run `go test ./...` (backend) and `npm run typecheck && npm run lint` (frontend) before submitting.

---