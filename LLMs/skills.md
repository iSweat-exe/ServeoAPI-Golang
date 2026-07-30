---
name: serveo-api-v2
description: >-
  Reference and client patterns for the ServeoAPI V2 backend — a JWT-secured REST API (with SSE and WebSocket streams) covering Docker containers/images/networks/volumes/compose, a chrooted per-server file manager, 1-click app templates, host system metrics, and OVHcloud dedicated-server control. Use this skill whenever writing, reviewing, or debugging code that calls routes under /v2/..., or building a feature talking to "ServeoAPI" or a matching panel backend (e.g. Pyrodactyl-style server management panels): Bearer/JWT auth, the exec WebSocket terminal, SSE streams (stats/logs/events/image pull), container start/stop/restart, file list/read/write/upload/delete, template-based deploys, or OVH reboot calls. Also trigger on symptom descriptions without the API being named, e.g. "the stats websocket never opens", "EventSource won't send my auth header", "upload wants multipart but list wants query params", "403 even though I'm logged in", "how do I pull an image and show progress", "how do I connect to the container terminal". Make sure to use this skill any time ServeoAPI, api.isweat.pro, or a docker-management panel backend comes up, even if the person doesn't name the API explicitly.
---

# ServeoAPI V2 client reference

ServeoAPI V2 is a backend exposing Docker (containers, images, networks, volumes, compose), a chrooted file manager, OVHcloud dedicated-server control, 1-click templates, and system metadata, over JSON REST plus SSE and WebSocket streams. This skill is built directly from the project's own OpenAPI 2.0 spec (served at `/swagger/index.html` on the API host), so treat it as the source of truth over any informal notes lying around the codebase or a README — several details below **correct** things an earlier informal doc for this same API got wrong (see §5).

Two companion files do the heavy lifting — read them before writing client code by hand:
- **`references/endpoints.md`** — every route with exact request/response schemas, organized by resource. Read the relevant section before guessing a field name.
- **`assets/serveo-client.ts`** — a typed TS client implementing every endpoint below, the corrected WebSocket handshake, and an SSE helper that works with Bearer auth. Copy it in and trim rather than rewriting the wrapper from scratch.

## 1. Authentication

- `POST /v2/auth/login` — body `{username, password}` → `{token}`. The only route that doesn't require auth.
- Every other route: `Authorization: Bearer <token>` header (declared in the spec as `ApiKeyAuth`, an apiKey-in-header scheme — it's a JWT despite the scheme name).
- `POST /v2/auth/logout` → 204, marks the user offline.
- **There is no refresh endpoint.** Don't build token-refresh logic against `/v2/auth/refresh` — it isn't in the spec. When a token expires, send the user back through login.

## 2. Streaming: know which routes are SSE and which are WebSocket

This is the area most likely to trip up client code, because the two protocols need different handling and the route names alone don't tell you which is which.

**SSE (`text/event-stream`, plain HTTP GET/POST, one-way server→client):**
`/v2/docker/containers/{id}/stats`, `/v2/docker/containers/{id}/logs`, `/v2/docker/system/events`, `/v2/docker/images/pull` (a POST whose *response* streams).

Browsers' native `EventSource` can't send an `Authorization` header, so it can't hit these directly. Use `fetch` with a `ReadableStream` reader (see `streamSSE` in the client asset) or a library that supports custom headers — don't reach for plain `EventSource` here and then wonder why auth fails.

**WebSocket (real duplex connection):**
Only `/v2/docker/containers/{id}/exec` (the interactive terminal, pairs with `xterm.js`). This route carries **no `security` block in the spec** — the handshake itself is unauthenticated at the HTTP-upgrade level; auth happens *inside* the socket. Send `{"type": "auth", "token": "<JWT>"}` as the very first message after `onopen`, before anything else. Server output after that is **raw binary frames** (write straight into `xterm.js`), while client→server input/resize messages are JSON with a shape the spec doesn't fully pin down — capture a real frame before hardcoding one.

## 3. Global conventions

- Content types vary and matter: `application/json` (most bodies), `text/plain` (file write), `multipart/form-data` (file upload), `application/octet-stream` (file read), `text/event-stream` (SSE routes).
- Response format: Use `serveoapi/internal/core/response`. Success responses use `response.SendJSON(w, http.StatusOK, data)` and errors use `response.SendError(w, http.StatusNotFound, "message")`. **Do not** use `json.NewEncoder(w)` or `http.Error(w)` directly.
- Context data: Fetch the authenticated user ID using `contextkeys.GetUserID(r.Context())`.
- Error bodies are typed as plain strings on most documented error responses (400/401/403/404/500) — don't assume a structured `{error: "..."}` JSON shape; parse defensively.
- Dependency Injection: All endpoints are methods on a module-level struct: `type Handler struct { DB *gorm.DB }`. Do not use global database variables.
- RBAC: permission strings (e.g. `docker.read`, `files.write`, `mcp.use`) live on the user; a `403` means the token is valid but the permission is missing — it is not an auth problem, and re-login won't fix it.
- No list endpoint (users, containers, images, networks, volumes, templates, apikeys) documents pagination — treat them as returning the full collection.
- `POST /v2/docker/containers/create` and `POST /v2/docker/compose/deploy` both explicitly restrict bind mounts for security — don't build a UI that lets someone mount an arbitrary host path; confirm the allowed prefix against the server first (the sample volume string in the spec, `/var/serveoapi/data/app:/app`, hints at the allowed host prefix, but confirm rather than assume).
- Path params: most IDs are strings (container/image/network ids, template ids) **except** user ids and API-key ids, which are integers. Volumes are addressed by `name`, not `id`.

## 4. Endpoint map

Full parameter/schema detail for every row is in `references/endpoints.md`.

| Resource | Routes |
|---|---|
| Auth | `POST /v2/auth/login`, `POST /v2/auth/logout` |
| Users | `GET/POST /v2/users/`, `GET /v2/users/me`, `PUT /v2/users/me/password`, `PATCH /v2/users/{id}` (int id), `DELETE /v2/users/{id}` (int id) |
| API Keys | `GET /v2/apikeys/`, `POST /v2/apikeys/create`, `DELETE /v2/apikeys/{id}` (int id) |
| Docker containers | `GET /v2/docker/containers/`, `POST /v2/docker/containers/create`, `GET/DELETE /v2/docker/containers/{id}`, `POST /v2/docker/containers/{id}/{action}` (start\|stop\|restart, one route), `GET /v2/docker/containers/{id}/exec` (WS), `GET /v2/docker/containers/{id}/logs` (SSE), `GET /v2/docker/containers/{id}/stats` (SSE) |
| Docker images | `GET /v2/docker/images/`, `POST /v2/docker/images/pull` (SSE), `DELETE /v2/docker/images/{id}` |
| Docker networks | `GET/POST /v2/docker/networks/`, `DELETE /v2/docker/networks/{id}` |
| Docker volumes | `GET/POST /v2/docker/volumes/`, `DELETE /v2/docker/volumes/{name}` (name, not id) |
| Docker system/compose | `GET /v2/docker/system/events` (SSE), `GET /v2/docker/system/info`, `POST /v2/docker/system/prune`, `POST /v2/docker/compose/deploy` |
| Files (`{server}` = container name) | `GET .../list`, `GET .../read`, `POST .../write` (raw text), `POST .../upload` (multipart), `DELETE .../delete` |
| Templates | `GET /v2/templates/`, `GET /v2/templates/{id}` |
| Metadata | `GET /v2/metadata/` |
| Health & Prometheus | `GET /health` (open), `GET /prometheus` (requires JWT) |
| System (host metrics) | `GET /v2/system/` |
| OVHcloud | `GET /v2/ovh/me`, `GET /v2/ovh/dedicated/server`, `POST /v2/ovh/dedicated/server/{serviceName}/reboot` |
| MCP Server | `GET/POST /v2/mcp/` (Streamable HTTP, requires `mcp.use` permission). Uses standard MCP over SSE transport. |

## 5. Corrections vs. earlier informal notes

If any older doc, comment, or memory of this API in the project disagrees with the points below, trust these — they come straight from the OpenAPI spec:

1. Start/stop/restart is **one** route, `POST /v2/docker/containers/{id}/{action}`, not three separate ones.
2. `stats` and `logs` are **SSE**, not WebSocket.
3. The WebSocket terminal route is called **`exec`**, not `terminal`, and its auth handshake is `{"type": "auth", "token": ...}`, not `{"auth": ...}`.
4. There is **no `/v2/auth/refresh`** endpoint.
5. Host metrics live at **`/v2/system/`** (no `/info` suffix) — `/v2/system/info` doesn't exist. Docker-engine metrics are a separate route, `/v2/docker/system/info`, with a different, smaller shape (container/image counts, not host CPU/RAM).
6. Metadata fields are snake_case (`api_version`, `commit_hash`, ...), not `AppVersion`/`CommitHash`.
7. User IDs in `PATCH`/`DELETE /v2/users/{id}` are integers, and the update route is `PATCH`, not `PUT`.

## 6. Deploying from a template (recipe)

Templates aren't just a lookup list — they carry the exact Docker payload to submit:
1. `GET /v2/templates/{id}` → get `docker` (a `CreateContainerRequest` skeleton) and `variables`.
2. Prompt the user for each variable using its `label`/`description`/`default`, enforcing `required`.
3. Substitute values into `docker` wherever each variable's `name` appears as a placeholder (exact placeholder syntax isn't specified — check one real template first).
4. `POST` the filled-in payload to `/v2/docker/containers/create`.

Full field-level detail: `references/endpoints.md#templates`.

## 7. Common pitfalls checklist

Before shipping a feature against this API, check it against these — they're the mistakes most likely to show up in a code review of this codebase:

- [ ] Using `EventSource` for stats/logs/events/pull instead of an auth-header-capable `fetch` stream.
- [ ] Treating `exec` as SSE, or the four SSE routes as WebSocket.
- [ ] Sending the exec auth message as anything other than `{"type": "auth", "token": "<JWT>"}` as the *first* frame.
- [ ] Building `/v2/auth/refresh` client logic — it doesn't exist; re-login on 401 instead.
- [ ] Confusing `/v2/system/` (host CPU/RAM/disk) with `/v2/docker/system/info` (Docker engine container/image counts) — they look similar and return very different shapes.
- [ ] Passing a string ID to `PATCH`/`DELETE /v2/users/{id}` or `DELETE /v2/apikeys/{id}` — these are integers.
- [ ] Addressing a volume by anything other than its `name` on delete.
- [ ] Assuming a `403` means the token is bad — it means the permission is missing; don't trigger a re-login flow on it.
- [ ] Letting a user submit an arbitrary host path into `volumes` on container create or compose deploy — this is deliberately restricted server-side.
- [ ] Assuming `/v2/files/{server}/upload`'s multipart field name — the spec doesn't name it; confirm against server code or a live request before hardcoding.