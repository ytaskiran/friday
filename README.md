# Friday

WhatsApp messaging API built with Go. Uses [whatsmeow](https://github.com/tulir/whatsmeow) for the WhatsApp multi-device protocol and SQLite for storage.

## Features

- WhatsApp session management (QR pairing, connect/disconnect)
- Contact lookup and phone number validation
- Message drafts with template placeholders (`{name}`, `{company}`, etc.)
- Contact groups and per-contact custom attributes
- Batch messaging with real-time SSE progress streaming
- Web UI for all operations

## Prerequisites

- Go 1.24+
- GCC (required for `go-sqlite3` CGo bindings)

## Quick Start

```bash
go build -o friday .
./friday
```

The server starts on `:8080`. Open `http://localhost:8080` to connect your WhatsApp session via QR code.

## API

All endpoints are under `/api/`:

| Resource | Endpoints |
|---|---|
| WhatsApp | `/api/whatsapp/status`, `connect`, `disconnect`, `send`, `qr`, `qr.png` |
| Contacts | `/api/contacts`, `search`, `validate` |
| Drafts | `/api/drafts` (CRUD + preview + send) |
| Attributes | `/api/contacts/{jid}/attributes`, `/api/attributes/keys` |
| Groups | `/api/groups` (CRUD + members) |
| Batch Runs | `/api/batch-runs` (CRUD + cancel + SSE stream) |
| Health | `/health` |
