# File Transfer System - Project Structure

## Project Setup

```
file-transfer-system/
├── backend/
│   ├── main.go
│   ├── config/
│   │   └── config.go
│   ├── types/
│   │   └── interfaces.go
│   ├── models/
│   │   └── file.go
│   ├── handlers/
│   │   └── upload.go
│   ├── repository/
│   │   └── file_repository.go
│   ├── services/
│   │   └── file_service.go
│   ├── database/
│   │   └── db.go
│   ├── uploads/
│   ├── go.mod
│   ├── .env
│   └── .env.example
└── frontend/
    ├── src/
    │   ├── App.tsx
    │   ├── main.tsx
    │   ├── index.css
    │   ├── components/
    │   │   └── FileUpload.tsx
    │   └── services/
    │       └── uploadService.ts
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── tailwind.config.js
    ├── postcss.config.js
    ├── .env
    └── .env.example
```

## Setup Instructions

### Backend Setup

1. Initialize Go module:
```bash
cd backend
go mod init file-transfer-backend
go get github.com/gofiber/fiber/v2
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/joho/godotenv
```

2. Create `.env` file in backend/:
```env
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=filetransfer
DB_PORT=5432
SERVER_PORT=8081
UPLOAD_DIR=./uploads
```

3. Create PostgreSQL database:
```sql
CREATE DATABASE filetransfer;
```

4. Run the backend:
```bash
go run main.go
```

### Frontend Setup

1. Create React + TypeScript project:
```bash
npm create vite@latest 
cd frontend
npm install
npm install axios
npm install -D tailwindcss@3 postcss autoprefixer
npx tailwindcss init -p
```

2. Create `.env` file in frontend/:
```env
VITE_API_URL=http://localhost:8081
```

3. Run the frontend:
```bash
npm run dev
```

## Features

- ✅ Chunked file upload (configurable chunk size via .env)
- ✅ SHA256 checksum verification
- ✅ **WebSocket real-time progress updates**
- ✅ **Interface-based architecture (IDatabase, IUploadHandler, IFileRepository, etc.)**
- ✅ **Configuration via structs loaded from .env**
- ✅ **No hardcoded values - all configurable**
- ✅ Progress tracking with live updates
- ✅ Error detection and retry logic
- ✅ PostgreSQL storage of file metadata
- ✅ Complete file reconstruction on server
- ✅ Original filename and type preservation
- ✅ Repository pattern for database operations
- ✅ Service layer for business logic
