# File Transfer System - Project Structure

## Project Setup

```
file-transfer-system/
├── backend/
│   ├── main.go
│   ├── models/
│   │   └── file.go
│   ├── handlers/
│   │   └── upload.go
│   ├── database/
│   │   └── db.go
│   ├── utils/
│   │   └── checksum.go
│   ├── uploads/
│   ├── go.mod
│   └── .env
└── frontend/
    ├── src/
    │   ├── App.tsx
    │   ├── components/
    │   │   └── FileUpload.tsx
    │   ├── services/
    │   │   └── uploadService.ts
    │   └── main.tsx
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    └── .env
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
SERVER_PORT=8080
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
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
npm install axios
```

2. Create `.env` file in frontend/:
```env
VITE_API_URL=http://localhost:8080
```

3. Run the frontend:
```bash
npm run dev
```

## Features

- ✅ Chunked file upload (1MB chunks)
- ✅ SHA256 checksum verification
- ✅ Progress tracking
- ✅ Error detection and retry logic
- ✅ PostgreSQL storage of file metadata
- ✅ Complete file reconstruction on server
- ✅ Original filename and type preservation
