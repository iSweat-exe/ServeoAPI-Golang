# ServeoAPI V2 — Endpoint Reference

Generated from the project's own Swagger 2.0 spec (`host: localhost:8080` in the raw doc, but the deployed host is the API's actual domain — e.g. `https://api.isweat.pro`; `basePath: /`). All paths below are relative to that host.

Every route except `POST /v2/auth/login` requires `Authorization: Bearer <token>` (declared in the spec as the `ApiKeyAuth` security scheme — an apiKey-in-header scheme, not a separate API-key system; it's the JWT from login). Where the spec response schema is empty (`"schema": {}`), that's noted explicitly below rather than guessed at.

## Table of contents

- [Auth](#auth)
- [Users](#users)
- [API Keys](#api-keys)
- [Docker — Containers](#docker--containers)
- [Docker — Images](#docker--images)
- [Docker — Networks](#docker--networks)
- [Docker — Volumes](#docker--volumes)
- [Docker — System & Compose](#docker--system--compose)
- [Files](#files)
- [Templates](#templates)
- [Metadata](#metadata)
- [System (host metrics)](#system-host-metrics)
- [OVHcloud](#ovhcloud)
- [MCP Server (unconfirmed)](#mcp-server-unconfirmed)

---

## Auth

### `POST /v2/auth/login`
No auth required — this is how you get a token.

**Body** (`auth.LoginRequest`):
```ts
{ username: string, password: string }
```
**200** (`auth.LoginResponse`): `{ token: string }`
**401**: plain-string error body ("Invalid credentials")

### `POST /v2/auth/logout`
Auth required. Marks the user offline server-side (does not blacklist the JWT itself — treat the token as still cryptographically valid until it expires).
**204**: No Content.

---

## Users

### `GET /v2/users/`
List all users. No pagination params documented — assume full collection.
**200**: array of `users.UserResponse`:
```ts
{
  id: number,
  username: string,
  permissions: string,       // format not pinned by the spec — inspect a real value (looks like a delimited permission-string list, e.g. "docker.read,files.write")
  profile_picture: string,
  status: string,
  last_connection: number    // looks like a unix timestamp given the type + name, but the spec only says integer — confirm before formatting as a date
}
```

### `POST /v2/users/`
Create a user.
**Body** (`users.CreateUserRequest`):
```ts
{ username: string, password: string, permissions: string, profile_picture: string }
```
**201**: `users.UserResponse` (see above)
**400**: plain-string error body

### `GET /v2/users/me`
Current authenticated user's profile.
**200**: `users.UserResponse`

### `PUT /v2/users/me/password`
Change your own password.
**Body** (`users.UpdatePasswordRequest`):
```ts
{ old_password: string, new_password: string }
```
**204**: No Content
**400** / **401** / **500**: plain-string error body

### `PATCH /v2/users/{id}`
`id` is an **integer** path param. Modify another user's permissions/profile — **not** a `PUT`, despite the "update" naming pattern elsewhere.
**Body** (`users.UpdateUserRequest`):
```ts
{ permissions: string, profile_picture: string, status: string }  // status: used to force-ban or force-offline a user
```
**200**: `users.UserResponse`

### `DELETE /v2/users/{id}`
`id` is an **integer** path param.
**204**: No Content
**400**: plain-string error body — returned specifically when trying to delete your own account ("Cannot delete yourself")

---

## API Keys

Personal Access Tokens, primarily meant for AI/agent use per the spec description.

### `GET /v2/apikeys/`
**200**: array of `apikeys.ApiKeyResponse`:
```ts
{ id: number, name: string, prefix: string, created_at: string }
```
Note: no `token` field here — the full token is only ever returned once, at creation.

### `POST /v2/apikeys/create`
**Body** (`apikeys.CreateApiKeyRequest`): `{ name: string }` (required)
**201**: `apikeys.CreateApiKeyResponse`:
```ts
{ id: number, name: string, token: string }  // token: only shown once — store it immediately, it can't be retrieved again
```

### `DELETE /v2/apikeys/{id}`
`id` is an **integer** path param.
**200**: `Record<string, string>` (a generic string-map object; the spec doesn't name the key, e.g. could be `{"message": "..."}` — inspect a real response before parsing a specific field)

---

## Docker — Containers

### `GET /v2/docker/containers/`
List all containers.
**200**: array of `docker.ContainerInfo`:
```ts
{ id: string, names: string[], image: string, state: string, status: string }
```

### `POST /v2/docker/containers/create`
Creates **and starts** the container. Bind mounts are restricted for security — see §3/§7 in SKILL.md.
**Body** (`docker.CreateContainerRequest`):
```ts
{
  name: string,
  image: string,
  env: string[],
  ports: Record<string, string>,      // e.g. {"8080": "80"} — host:container, exact direction not spelled out beyond the example, confirm with one real call
  volumes: string[],                  // e.g. ["myvol:/data", "/var/serveoapi/data/app:/app"] — restricted host prefix, don't expose free-form host paths in a UI
  restart_policy: string
}
```
**201**: `docker.ContainerInfo` (see above)
**400** / **500**: plain-string error body

### `GET /v2/docker/containers/{id}`
`id` is a string (container ID). Inspect a container.
**200**: schema is empty in the spec (`{}`) — the handler returns *some* JSON object with container detail, but its shape isn't typed in the OpenAPI doc. Don't assume it matches `ContainerInfo`; log/inspect a real response before binding to a type.

### `DELETE /v2/docker/containers/{id}`
Query param: `force` (boolean, optional) — force-remove a running container.
**204**: No Content

### `POST /v2/docker/containers/{id}/{action}`
`action` path param must be one of `start`, `stop`, `restart` — this is genuinely one route in the spec, not three.
**204**: No Content

### `GET /v2/docker/containers/{id}/exec` — **WebSocket**
No `security` block on this route in the spec (unlike everything else) — the HTTP upgrade itself isn't gated by the `Authorization` header; auth happens over the socket instead.
- Upgrades to WebSocket.
- **First message you must send, immediately after `onopen`:** `{"type": "auth", "token": "<JWT>"}`.
- After that, server→client frames are **raw binary** (feed directly into `xterm.js`'s `write`).
- Client→server (keystrokes, resize) are JSON, but the spec doesn't pin the exact shape beyond "subsequent JSON messages for input/resize" — capture one real outbound frame from working client code (or the Go handler source) before hardcoding a shape.
- Response codes documented (for the initial HTTP part, before upgrade): `101` Switching Protocols, `400`, `401`, `403`, `404`, `500` — all plain-string bodies for the error cases.

### `GET /v2/docker/containers/{id}/logs` — **SSE**
`text/event-stream`. Requires the normal Bearer header — use `fetch`, not native `EventSource`.
**200**: Event Stream (untyped string payload per event in the spec — treat each SSE `data:` line as raw log text unless you observe otherwise).

### `GET /v2/docker/containers/{id}/stats` — **SSE**
Same auth caveat as `logs`. Streams CPU/RAM stats.
**200**: Event Stream — the spec types this as a bare string too, meaning the *event framing* is typed but the JSON-ness of each `data:` payload isn't guaranteed by the OpenAPI doc. In practice expect JSON per event (CPU %, memory) — verify against one real event before assuming exact field names.

---

## Docker — Images

### `GET /v2/docker/images/`
**200**: array of `docker.ImageInfo`:
```ts
{ id: string, repo_tags: string[], size: number, created: number }
```

### `POST /v2/docker/images/pull` — **SSE**
A POST whose *response* is a stream — unusual, don't assume SSE routes are always GET.
**Body** (`docker.PullImageRequest`): `{ image: string }` (e.g. `"nginx:latest"`)
**200**: Event Stream of pull progress (untyped per-event payload in the spec — same caveat as container stats/logs).

### `DELETE /v2/docker/images/{id}`
`id` is the image ID **or** tag (string). Query param: `force` (boolean, optional).
**204**: No Content

---

## Docker — Networks

### `GET /v2/docker/networks/`
**200**: array of `docker.NetworkInfo`:
```ts
{ id: string, name: string, driver: string }
```

### `POST /v2/docker/networks/`
**Body** (`docker.CreateNetworkRequest`):
```ts
{ name: string, driver: string, labels: Record<string, string> }
```
**201**: `docker.NetworkInfo`
**400** / **500**: plain-string error body

### `DELETE /v2/docker/networks/{id}`
`id` is a string.
**204**: No Content

---

## Docker — Volumes

### `GET /v2/docker/volumes/`
**200**: array of `docker.VolumeInfo`:
```ts
{ name: string, driver: string, mountpoint: string }
```

### `POST /v2/docker/volumes/`
**Body** (`docker.CreateVolumeRequest`):
```ts
{ name: string, driver: string, labels: Record<string, string> }
```
**201**: `docker.VolumeInfo`
**400** / **500**: plain-string error body

### `DELETE /v2/docker/volumes/{name}`
Addressed by **`name`**, not an `id` — the only delete route in the API that works this way. Query param: `force` (boolean, optional).
**204**: No Content

---

## Docker — System & Compose

### `GET /v2/docker/system/events` — **SSE**
Same auth-header caveat as other SSE routes.
**200**: Event Stream, untyped payload per the spec.

### `GET /v2/docker/system/info`
**Docker-engine** info — *not* host metrics (see `/v2/system/` below for that).
**200**: `docker.SystemInfo`:
```ts
{
  containers: number,
  containers_running: number,
  containers_stopped: number,
  images: number,
  mem_total: number,
  ncpu: number,
  server_version: string
}
```

### `POST /v2/docker/system/prune`
Removes unused Docker data (dangling images, stopped containers, etc. — the spec doesn't itemize exactly what's pruned).
**204**: No Content

### `POST /v2/docker/compose/deploy`
Deploys a stack from raw compose YAML. Same volume-security restriction as container create.
**Body** (`docker.DeployStackRequest`):
```ts
{ name: string, content: string }  // content: raw docker-compose.yml text
```
**201**: plain string (success message — not a structured object)
**400** / **500**: plain-string error body

---

## Files

Path param `{server}` is a **container name**, not a numeric/UUID id — this is a chrooted-per-container file manager, so every file route is scoped to one container's filesystem.

### `GET /v2/files/{server}/list`
Query param: `path` (string, optional — relative path; omit for root).
**200**: array of `files.FileInfo`:
```ts
{ name: string, path: string, is_dir: boolean, size: number, mod_time: string }
```

### `GET /v2/files/{server}/read`
Query param: `path` (string, **required**).
**Response**: `application/octet-stream` — raw bytes, not JSON. The spec's response schema is empty (`{}`); handle it as a binary/blob download, decoding to text yourself if you know it's a text file.

### `POST /v2/files/{server}/write`
Query param: `path` (string, **required**).
**Body**: raw `text/plain` — the file's new full content, not a JSON-wrapped field. Don't `JSON.stringify` the body or set `Content-Type: application/json` here.
**Response**: schema empty in the spec — confirm status code/body shape against a live call; treat any 2xx as success.

### `POST /v2/files/{server}/upload`
Query param: `path` (string, **required**) — the destination directory.
**Body**: `multipart/form-data`. The spec does not name the form field for the file part — inspect the handler or a working client call before hardcoding a field name (a common convention like `file` is a reasonable first guess, not a confirmed one).
**Response**: schema empty in the spec.

### `DELETE /v2/files/{server}/delete`
Query param: `path` (string, **required**) — file or directory; the spec doesn't document a recursive-delete flag, so assume directory delete removes contents and confirm before relying on that in destructive tooling.
**Response**: schema empty in the spec.

---

## Templates

1-click app/game templates. Each template embeds the exact Docker payload needed to deploy it — see SKILL.md §6 for the full recipe.

### `GET /v2/templates/`
**200**: array of `templates.TemplateInfo` (see below)

### `GET /v2/templates/{id}`
`id` is a string.
**200**: `templates.TemplateInfo`:
```ts
{
  id: string,
  name: string,
  description: string,
  category: string,   // e.g. "game", "database", "app", "lang"
  logo: string,
  docker: docker.CreateContainerRequest,   // the exact payload to POST to /v2/docker/containers/create
  variables: templates.TemplateVariable[]
}
```
```ts
// templates.TemplateVariable
{
  name: string,          // placeholder name used inside `docker` (e.g. "SERVER_NAME")
  label: string,         // UI label (e.g. "Server Name")
  description: string,   // UI helper text
  default: string,
  required: boolean
}
```
**404**: plain-string error body ("Template not found")

---

## Metadata

### `GET /v2/metadata/`
**200**: `metadata.MetadataResponse`:
```ts
{
  author: string,
  api_version: string,
  commit_hash: string,
  go_version: string,
  protocol_version: number,
  github_link: string,
  deprecated: boolean,
  deprecation_info: string
}
```
Field names are **snake_case** — a prior informal doc for this API used PascalCase (`AppVersion`, `CommitHash`); that was wrong (see SKILL.md §5.6).

---

## System (host metrics)

### `GET /v2/system/`
Host-level metrics — CPU/RAM/disk/network for the machine ServeoAPI runs on. **Not** the same as `/v2/docker/system/info` (which is Docker-engine-scoped container/image counts). No `/info` suffix on this route.
**200**: `system.SystemResponse`:
```ts
{
  hostname: string,
  platform: string,
  os: string,
  arch: string,
  kernel_version: string,
  uptime: number,
  cpu_usage: number,     // likely a percentage given the name/type, but the spec doesn't state the unit explicitly — confirm against a real value
  ram_total: number,
  ram_used: number,
  disk_total: number,
  disk_used: number,
  network_rx: number,
  network_tx: number,
  ping: string
}
```

---

## OVHcloud

### `GET /v2/ovh/me`
Connectivity/sanity check — fetches OVH account info.
**200**: `ovh.OvhMeResponse`:
```ts
{ name: string, firstname: string, email: string }
```

### `GET /v2/ovh/dedicated/server`
List dedicated (bare-metal) servers on the linked OVH account.
**200**: `string[]` — an array of service names (e.g. `"ns300000.ip-1-2-3.eu"`), not objects.

### `POST /v2/ovh/dedicated/server/{serviceName}/reboot`
`serviceName` is a string path param (the same service-name strings returned by the list route above).
Triggers a **hard** reboot — there's no confirmation step server-side, so gate this behind a confirmation in any UI.
**200**: plain string ("Reboot initiated" or similar — the spec types it as string, not a structured object).

---

## MCP Server

This endpoint exposes a Model Context Protocol (MCP) server over SSE transport, providing AI tools to manage the panel directly (e.g., Docker, Files, OVH). It uses the `mcp-go` library's `StreamableHTTPServer`.

### `GET /v2/mcp/` — **SSE**
Requires `Authorization: Bearer <token>` and the `mcp.use` permission on the user.
Establishes the Server-Sent Events (SSE) connection for MCP JSON-RPC communication.
**200**: Event Stream of MCP JSON-RPC messages.

### `POST /v2/mcp/`
Sends JSON-RPC messages to the MCP server.
**Body**: JSON-RPC MCP message (e.g. `{"jsonrpc": "2.0", "method": "...", ...}`).
**Response**: Standard MCP JSON-RPC response.