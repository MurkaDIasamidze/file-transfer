# DriveClone — Frontend

React + TypeScript SPA. Google Drive–style file manager that talks to the DriveClone Go API over REST and WebSocket.

---

## Stack

| Layer | Technology |
|-------|------------|
| Language | TypeScript 5 |
| Framework | React 18 |
| Build tool | Vite |
| Styling | Tailwind CSS |
| Routing | React Router v6 |
| State | Zustand |
| HTTP | Axios |
| Upload | WebSocket (native browser API) |

---

## Project structure

```
src/
├── components/
│   ├── auth/
│   │   ├── LoginPage.tsx          Email + password sign-in form
│   │   └── RegisterPage.tsx       Registration form
│   └── drive/
│       ├── DrivePage.tsx          Main shell — layout, data loading, all action handlers
│       ├── Sidebar.tsx            Navigation: My Drive · Recent · Starred · Trash
│       ├── Header.tsx             Search bar, breadcrumbs, user menu
│       ├── FileGrid.tsx           Grid renderer — folders + files, bulk action bar, select-all
│       ├── FolderCard.tsx         Folder tile — drag-drop target, context menu
│       ├── FileCard.tsx           File tile — star badge, processing state, context menu, download
│       ├── UploadModal.tsx        Upload dialog — file/folder picker, drag-drop, progress bars
│       ├── CreateFolderModal.tsx  New folder dialog
│       └── AccountPage.tsx        Profile + security settings modal (tabs)
├── services/
│   ├── api.ts                     Axios instance + typed endpoint wrappers (filesApi, foldersApi, authApi)
│   └── uploadService.ts           WebSocket chunked-upload engine + FileSystem API folder walker
├── store/
│   └── authStore.ts               Zustand store — token, user, setAuth, logout, isAuthenticated
├── types/
│   └── index.ts                   Shared interfaces: User · Folder · FileItem · BreadcrumbItem
├── App.tsx                        Router, PrivateRoute, PublicRoute guards
└── main.tsx                       Vite entry point
```

---

## Quick start

```bash
# 1. Install dependencies
npm install

# 2. Configure environment
cp .env.example .env

# 3. Start dev server
npm run dev
```

Open `http://localhost:5173`.

---

## Environment variables

```env
# Base URL of the Go backend (no trailing slash)
VITE_API_URL=http://localhost:8081

# WebSocket base URL (same host, different scheme)
VITE_WS_URL=ws://localhost:8081

# Upload chunk size in bytes (default 256 KB)
VITE_CHUNK_SIZE=262144
```

---

## Features

### Authentication

- Register and sign in with email + password
- JWT stored in `localStorage`, attached to every request via Axios interceptor
- Automatic redirect to `/login` on 401 from any API call
- Back-button proof — `/login` and `/register` redirect to `/` when already authenticated (`PublicRoute` guard)

### File management

| View | Contents |
|------|----------|
| My Drive | Files and folders in the current directory, navigable via breadcrumbs |
| Recent | 20 most recently modified files |
| Starred | Files marked with a star |
| Trash | Soft-deleted files and folders — restore or permanently delete |

### Uploading

- **File picker** — one or many files via the native `<input type="file">` dialog
- **Folder picker** — entire folder via `<input webkitdirectory>`; full directory tree is preserved as relative paths
- **Drag & drop** — drop files or folders onto the upload zone; folders are walked recursively so every nested file is included with its correct relative path
- Per-file progress bars with percentage
- Modal auto-closes 1.5 s after all files complete
- Files with S3 async upload show a pulsing ⏳ icon while `status === "processing"` and become downloadable once `status === "completed"`

### Downloading

Double-click a file card, or use the context menu Download option. Works for both S3-backed and locally-stored files. The correct filename is always preserved.

For S3 files the server returns a presigned URL; the frontend fetches it as a blob and saves it via a temporary `blob://` URL — this is necessary because the `<a download>` attribute is ignored for cross-origin URLs (S3's domain ≠ your app's domain).

### Selection and bulk actions

- Click the **Select all** row at the top of the grid, or hover any item to reveal its checkbox
- Select any combination of files and folders
- Bulk action bar appears when anything is selected
- **Normal view:** bulk Star and bulk Trash
- **Trash view:** bulk Restore and bulk Delete forever
- Press `Esc` to clear selection

### Folder management

- Create nested folders
- Drag a file card onto a folder card to move it (disabled during selection mode)
- Context menu on every item

### Account settings

Two-tab modal (Profile / Security):

- Edit display name — saved immediately
- Change password — requires current password
- Sign out

---

## Upload architecture

Each file gets its own WebSocket connection. The protocol is strictly sequential inside that connection — no multiplexing, no message routing needed:

```
openWS(token)
  sendAndWait({ type:"init",     data:{...} })  → await "init_ack"
  for each chunk:
    sendAndWait({ type:"chunk",  data:{...} })  → await "progress"
  sendAndWait({ type:"complete", data:{...} })  → await "done"
ws.close()
```

`sendAndWait` attaches a one-shot `message` listener, sends the payload, and resolves/rejects based on the next matching server message. After the connection is established, all I/O is sequential — the next message is only sent after the previous one is acknowledged.

Files in a batch upload one at a time (sequential, not concurrent). This keeps total WebSocket connections low and avoids flooding the server when uploading folders with many files.

### Drag-drop folders

`DataTransfer.files` is always flat (no sub-directories, no `webkitRelativePath` for drops). The app uses the FileSystem API instead:

```ts
// SYNC — must happen before the drop handler returns
const entries = collectEntriesSync(e.dataTransfer);

// ASYNC — safe to do after the handler returns
const tasks = await entriesToTasks(entries);
```

`DataTransferItem.webkitGetAsEntry()` must be called synchronously — the `DataTransfer` object becomes invalid once the event handler returns. Entries are collected synchronously then walked asynchronously. `readAllEntries()` loops `readEntries()` until it gets an empty batch (the API yields at most 100 entries per call).

---

## Auth flow

```
localStorage
  └── "token"  (JWT string)
  └── "user"   (JSON-serialised AuthUser)

Zustand authStore
  └── reads from localStorage on module load (synchronous)
  └── setAuth(token, user)  → writes both to localStorage + state
  └── logout()              → clears both
  └── isAuthenticated()     → Boolean(token)

Axios interceptor
  └── reads localStorage.getItem('token') before every request
  └── sets Authorization: Bearer <token>
  └── on 401 → clears storage + redirects to /login

App.tsx route guards
  └── PrivateRoute  → redirects to /login  when !isAuthenticated()
  └── PublicRoute   → redirects to /       when  isAuthenticated()
```

---

## Available scripts

```bash
npm run dev        # Vite dev server with HMR at localhost:5173
npm run build      # Production build → dist/
npm run preview    # Serve dist/ locally
npm run lint       # ESLint
npm run typecheck  # tsc --noEmit (no output files)
```

---

## Browser support

Chrome 86+, Firefox 111+, Safari 15.4+, Edge 86+.

The FileSystem API (`webkitGetAsEntry`, `FileSystemDirectoryReader`) required for folder drag-drop is available in all of the above. `crypto.subtle.digest` (used for SHA-256 chunk checksums) requires a secure context (`https://` or `localhost`).