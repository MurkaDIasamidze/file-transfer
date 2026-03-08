# DriveClone — Frontend

React + TypeScript SPA providing a Google Drive–style interface for the DriveClone file-storage service.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | TypeScript 5 |
| Framework | React 18 |
| Build tool | Vite |
| Styling | Tailwind CSS |
| Routing | React Router v6 |
| State | Zustand |
| HTTP client | Axios |
| Real-time upload | WebSocket (native browser API) |

## Project structure

```
src/
├── components/
│   ├── auth/
│   │   ├── LoginPage.tsx        Email + password login form
│   │   └── RegisterPage.tsx     Registration form with password strength meter
│   └── drive/
│       ├── DrivePage.tsx        Main shell — layout, data loading, action handlers
│       ├── Header.tsx           Search bar + user menu
│       ├── Sidebar.tsx          Navigation (My Drive / Recent / Starred / Trash)
│       ├── FileGrid.tsx         Grid renderer for folders + files
│       ├── FolderCard.tsx       Folder tile with drag-drop target + context menu
│       ├── FileCard.tsx         File tile with star badge + context menu
│       ├── UploadModal.tsx      Upload dialog (file picker, folder picker, drag-drop)
│       ├── CreateFolderModal.tsx New folder dialog
│       └── AccountPage.tsx      Profile + password settings modal
├── services/
│   ├── api.ts                   Axios instance + typed endpoint wrappers
│   └── uploadService.ts         WebSocket chunked-upload engine + FileSystem API helpers
├── store/
│   └── authStore.ts             Zustand store — user, token, setAuth, logout
├── types/
│   └── index.ts                 Shared TypeScript interfaces (User, Folder, FileItem, …)
├── App.tsx                      Router + PrivateRoute guard
└── main.tsx                     Vite entry point
```

## Quick start

```bash
# 1. Install dependencies
npm install

# 2. Copy and configure environment
cp .env.example .env

# 3. Start the dev server
npm run dev
```

The app is available at `http://localhost:5173` by default.

## Environment variables

Create a `.env` file at the project root:

```env
# Base URL of the Go backend REST API
VITE_API_URL=http://localhost:8080

# WebSocket base URL (same host, ws:// scheme)
VITE_WS_URL=ws://localhost:8080

# Chunk size in bytes sent per WebSocket message (default 256 KB)
VITE_CHUNK_SIZE=262144
```

## Features

### Authentication
- Register / login with email + password
- JWT stored in `localStorage`, attached to every API request via Axios interceptor
- Automatic redirect to `/login` on 401

### File management
- **My Drive** — browse files and folders, navigate into sub-folders via breadcrumbs
- **Recent** — 20 most-recently-modified files
- **Starred** — files marked with a star
- **Trash** — soft-deleted files and folders; restore or permanently delete

### Upload
- **File picker** — select one or many files via the native dialog
- **Folder picker** — select a local folder via `<input webkitdirectory>`; the full directory tree is preserved on the server as relative paths
- **Drag & drop** — drop plain files or entire folders onto the drop zone; folders are walked recursively using the browser FileSystem API (`webkitGetAsEntry`) so sub-directories and their contents are all included
- Per-file progress bars while uploading; auto-closes the modal 1.5 s after all files complete
- Files upload sequentially over individual WebSocket connections (one WS per file) using a three-step `init → chunks → complete` protocol with SHA-256 integrity checks at both the chunk and whole-file level

### Folder management
- Create nested folders
- Drag a file card onto a folder card to move it
- Context menu on every item: Open / Move to Trash / Delete Forever / Restore

### Account settings
- Change display name
- Change password (requires current password)
- Sign out

## Upload architecture

Each file uses its own WebSocket connection and a strict request/response flow:

```
openWS()
  sendAndWait({type:"init", ...})          → await init_ack
  for each chunk:
    sendAndWait({type:"chunk", ...})       → await progress
  sendAndWait({type:"complete", ...})      → await done
ws.close()
```

**Why one WS per file?** Multiplexing multiple in-flight files over a shared connection requires routing incoming messages to the correct pending handler — a source of subtle bugs. Using a dedicated connection per file makes the protocol trivially sequential and eliminates all routing complexity.

**Drag-drop folders:** `DataTransfer.files` only contains top-level files and does not set `webkitRelativePath`. The app uses `DataTransferItem.webkitGetAsEntry()` to obtain `FileSystemEntry` objects, which are snapshotted *synchronously* during the drop event (the `DataTransfer` becomes invalid once the handler returns), then walked asynchronously to collect every nested file with its correct relative path.

## Available scripts

```bash
npm run dev        # Start Vite dev server with HMR
npm run build      # Production build → dist/
npm run preview    # Serve the production build locally
npm run lint       # ESLint
npm run typecheck  # tsc --noEmit
```

## Browser support

All modern browsers (Chrome 86+, Firefox 111+, Safari 15.4+, Edge 86+).
The `webkitGetAsEntry` / FileSystem API for folder drag-drop is available in all of the above.