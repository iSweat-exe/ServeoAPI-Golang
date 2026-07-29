/**
 * ServeoAPI V2 client
 * ---------------------------------------------------------------------------
 * Typed wrapper around every route in the ServeoAPI V2 OpenAPI spec.
 *
 * Corrections baked in vs. informal/earlier client code for this same API
 * (see SKILL.md §5 for the full list):
 *   - stats/logs/system-events/image-pull are SSE, not WebSocket.
 *   - exec is the ONLY WebSocket route, and its first frame must be
 *     {"type":"auth","token":"<JWT>"} — not {"auth":...}.
 *   - There is no /v2/auth/refresh. On 401, re-run login().
 *   - Host metrics are GET /v2/system/ ; Docker-engine metrics are the
 *     separate, smaller GET /v2/docker/system/info.
 *   - User ids and API-key ids are numbers; every other resource id is a
 *     string; volumes are addressed by name.
 *
 * Copy this file in and trim to what you need rather than rewriting the
 * streaming/auth plumbing from scratch — that plumbing is the part most
 * likely to be subtly wrong in a hand-rolled client.
 * ---------------------------------------------------------------------------
 */

// ============================================================================
// Types (mirrors the `definitions` section of the OpenAPI spec exactly)
// ============================================================================

export interface LoginRequest {
  username: string;
  password: string;
}
export interface LoginResponse {
  token: string;
}

export interface UserResponse {
  id: number;
  username: string;
  permissions: string;
  profile_picture: string;
  status: string;
  last_connection: number;
}
export interface CreateUserRequest {
  username: string;
  password: string;
  permissions: string;
  profile_picture: string;
}
export interface UpdateUserRequest {
  permissions?: string;
  profile_picture?: string;
  status?: string;
}
export interface UpdatePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface ApiKeyResponse {
  id: number;
  name: string;
  prefix: string;
  created_at: string;
}
export interface CreateApiKeyRequest {
  name: string;
}
export interface CreateApiKeyResponse {
  id: number;
  name: string;
  /** Only present on creation — cannot be retrieved again afterwards. */
  token: string;
}

export interface ContainerInfo {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
}
export interface CreateContainerRequest {
  name?: string;
  image: string;
  env?: string[];
  /** e.g. {"8080": "80"} */
  ports?: Record<string, string>;
  /** e.g. ["myvol:/data", "/var/serveoapi/data/app:/app"] — restricted server-side */
  volumes?: string[];
  restart_policy?: string;
}
export type ContainerAction = "start" | "stop" | "restart";

export interface ImageInfo {
  id: string;
  repo_tags: string[];
  size: number;
  created: number;
}
export interface PullImageRequest {
  /** e.g. "nginx:latest" */
  image: string;
}

export interface NetworkInfo {
  id: string;
  name: string;
  driver: string;
}
export interface CreateNetworkRequest {
  name: string;
  driver?: string;
  labels?: Record<string, string>;
}

export interface VolumeInfo {
  name: string;
  driver: string;
  mountpoint: string;
}
export interface CreateVolumeRequest {
  name: string;
  driver?: string;
  labels?: Record<string, string>;
}

export interface DockerSystemInfo {
  containers: number;
  containers_running: number;
  containers_stopped: number;
  images: number;
  mem_total: number;
  ncpu: number;
  server_version: string;
}
export interface DeployStackRequest {
  name: string;
  /** raw docker-compose.yml text */
  content: string;
}

export interface FileInfo {
  name: string;
  /** relative path to the file-manager root */
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
}

export interface TemplateVariable {
  /** placeholder name used inside `docker`, e.g. "SERVER_NAME" */
  name: string;
  label: string;
  description: string;
  default: string;
  required: boolean;
}
export interface TemplateInfo {
  id: string;
  name: string;
  description: string;
  category: string;
  logo: string;
  /** exact payload to POST to /v2/docker/containers/create, with placeholders substituted */
  docker: CreateContainerRequest;
  variables: TemplateVariable[];
}

export interface MetadataResponse {
  author: string;
  api_version: string;
  commit_hash: string;
  go_version: string;
  protocol_version: number;
  github_link: string;
  deprecated: boolean;
  deprecation_info: string;
}

export interface HostSystemResponse {
  hostname: string;
  platform: string;
  os: string;
  arch: string;
  kernel_version: string;
  uptime: number;
  cpu_usage: number;
  ram_total: number;
  ram_used: number;
  disk_total: number;
  disk_used: number;
  network_rx: number;
  network_tx: number;
  ping: string;
}

export interface OvhMeResponse {
  name: string;
  firstname: string;
  email: string;
}

// ============================================================================
// Errors
// ============================================================================

/** Thrown for any non-2xx response. `body` is the raw text — most error
 * responses in this API are plain strings, not JSON objects. */
export class ServeoApiError extends Error {
  constructor(
    public status: number,
    public body: string,
    path: string,
  ) {
    super(`ServeoAPI ${status} on ${path}: ${body}`);
    this.name = "ServeoApiError";
  }
}

// ============================================================================
// SSE helper — required because native EventSource can't send Authorization
// headers, and several SSE routes here are POST (image pull), which
// EventSource can't do at all.
// ============================================================================

/**
 * Streams Server-Sent Events from an authenticated fetch response.
 * Yields each event's `data:` payload as a string. Handles multi-line
 * `data:` fields per the SSE spec (joined with \n) and ignores comment/
 * keep-alive lines (starting with `:`).
 */
export async function* streamSSE(
  response: Response,
): AsyncGenerator<string, void, unknown> {
  if (!response.body) {
    throw new Error("Response has no body to stream");
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      const events = buffer.split("\n\n");
      buffer = events.pop() ?? "";

      for (const rawEvent of events) {
        const dataLines = rawEvent
          .split("\n")
          .filter((line) => line.startsWith("data:"))
          .map((line) => line.slice(5).trimStart());
        if (dataLines.length > 0) {
          yield dataLines.join("\n");
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

// ============================================================================
// WebSocket helper for the ONE real-duplex route: containers/{id}/exec
// ============================================================================

export interface ExecSocketHandlers {
  onOutput: (data: ArrayBuffer | Blob) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
}

/**
 * Wraps the exec WebSocket, sending the required
 * {"type":"auth","token":...} handshake as the first frame.
 *
 * NOTE: the exact JSON shape for outbound input/resize messages isn't
 * pinned by the spec beyond "JSON messages for input/resize" — the
 * `sendInput`/`sendResize` shapes below are a reasonable starting guess.
 * Confirm against a captured real frame (browser devtools / server source)
 * before relying on them in production.
 */
export class DockerExecSocket {
  private ws: WebSocket;

  constructor(wsBaseUrl: string, containerId: string, token: string, handlers: ExecSocketHandlers) {
    this.ws = new WebSocket(`${wsBaseUrl}/v2/docker/containers/${containerId}/exec`);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {
      // Required first frame — auth happens inside the socket, not via headers.
      this.ws.send(JSON.stringify({ type: "auth", token }));
    };
    this.ws.onmessage = (event) => handlers.onOutput(event.data);
    this.ws.onclose = (event) => handlers.onClose?.(event);
    this.ws.onerror = (event) => handlers.onError?.(event);
  }

  /** Unconfirmed shape — verify against a real frame before relying on this. */
  sendInput(data: string) {
    this.ws.send(JSON.stringify({ type: "input", data }));
  }

  /** Unconfirmed shape — verify against a real frame before relying on this. */
  sendResize(cols: number, rows: number) {
    this.ws.send(JSON.stringify({ type: "resize", cols, rows }));
  }

  close() {
    this.ws.close();
  }
}

// ============================================================================
// Main client
// ============================================================================

export interface ServeoClientOptions {
  /** e.g. "https://api.isweat.pro" — no trailing slash. */
  baseUrl: string;
  /** Set after login(), or pass an existing token up front. */
  token?: string;
}

export class ServeoClient {
  private baseUrl: string;
  private token?: string;

  constructor(opts: ServeoClientOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/$/, "");
    this.token = opts.token;
  }

  setToken(token: string) {
    this.token = token;
  }

  private get wsBaseUrl(): string {
    return this.baseUrl.replace(/^http/, "ws");
  }

  /** Low-level request helper. Throws ServeoApiError on non-2xx. */
  private async request<T>(
    method: string,
    path: string,
    opts: {
      body?: unknown;
      contentType?: string;
      query?: Record<string, string | boolean | undefined>;
      rawBody?: BodyInit;
      parseAs?: "json" | "text" | "blob" | "none";
    } = {},
  ): Promise<T> {
    const url = new URL(this.baseUrl + path);
    if (opts.query) {
      for (const [k, v] of Object.entries(opts.query)) {
        if (v !== undefined) url.searchParams.set(k, String(v));
      }
    }

    const headers: Record<string, string> = {};
    if (this.token) headers["Authorization"] = `Bearer ${this.token}`;

    let body: BodyInit | undefined;
    if (opts.rawBody !== undefined) {
      body = opts.rawBody;
      if (opts.contentType) headers["Content-Type"] = opts.contentType;
    } else if (opts.body !== undefined) {
      body = JSON.stringify(opts.body);
      headers["Content-Type"] = "application/json";
    }

    const res = await fetch(url.toString(), { method, headers, body });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new ServeoApiError(res.status, text, path);
    }

    const parseAs = opts.parseAs ?? "json";
    if (parseAs === "none" || res.status === 204) return undefined as T;
    if (parseAs === "text") return (await res.text()) as unknown as T;
    if (parseAs === "blob") return (await res.blob()) as unknown as T;
    // Some 2xx success responses (e.g. compose deploy, ovh reboot) are
    // plain strings even though they're typed "json" in the spec's tag —
    // guard against JSON.parse throwing on those.
    const raw = await res.text();
    try {
      return raw.length ? (JSON.parse(raw) as T) : (undefined as T);
    } catch {
      return raw as unknown as T;
    }
  }

  // --- Auth ---------------------------------------------------------------

  async login(username: string, password: string): Promise<LoginResponse> {
    const res = await this.request<LoginResponse>("POST", "/v2/auth/login", {
      body: { username, password } satisfies LoginRequest,
    });
    this.token = res.token;
    return res;
  }

  /** No /v2/auth/refresh exists — call login() again when a token expires. */
  async logout(): Promise<void> {
    await this.request<void>("POST", "/v2/auth/logout", { parseAs: "none" });
  }

  // --- Users ----------------------------------------------------------------

  users = {
    list: (): Promise<UserResponse[]> => this.request("GET", "/v2/users/"),

    create: (body: CreateUserRequest): Promise<UserResponse> =>
      this.request("POST", "/v2/users/", { body }),

    me: (): Promise<UserResponse> => this.request("GET", "/v2/users/me"),

    updateMyPassword: (body: UpdatePasswordRequest): Promise<void> =>
      this.request("PUT", "/v2/users/me/password", { body, parseAs: "none" }),

    /** id is an integer; this is PATCH, not PUT. */
    update: (id: number, body: UpdateUserRequest): Promise<UserResponse> =>
      this.request("PATCH", `/v2/users/${id}`, { body }),

    /** id is an integer. 400 if you try to delete your own account. */
    delete: (id: number): Promise<void> =>
      this.request("DELETE", `/v2/users/${id}`, { parseAs: "none" }),
  };

  // --- API Keys ---------------------------------------------------------

  apiKeys = {
    list: (): Promise<ApiKeyResponse[]> => this.request("GET", "/v2/apikeys/"),

    create: (name: string): Promise<CreateApiKeyResponse> =>
      this.request("POST", "/v2/apikeys/create", {
        body: { name } satisfies CreateApiKeyRequest,
      }),

    /** id is an integer. */
    delete: (id: number): Promise<Record<string, string>> =>
      this.request("DELETE", `/v2/apikeys/${id}`),
  };

  // --- Docker: Containers -------------------------------------------------

  containers = {
    list: (): Promise<ContainerInfo[]> => this.request("GET", "/v2/docker/containers/"),

    create: (body: CreateContainerRequest): Promise<ContainerInfo> =>
      this.request("POST", "/v2/docker/containers/create", { body }),

    /** Response shape is untyped in the spec ({}) — inspect a real payload before binding to a type. */
    inspect: (id: string): Promise<unknown> =>
      this.request("GET", `/v2/docker/containers/${id}`),

    delete: (id: string, force = false): Promise<void> =>
      this.request("DELETE", `/v2/docker/containers/${id}`, {
        query: { force },
        parseAs: "none",
      }),

    action: (id: string, action: ContainerAction): Promise<void> =>
      this.request("POST", `/v2/docker/containers/${id}/${action}`, { parseAs: "none" }),

    start: (id: string) => this.containers.action(id, "start"),
    stop: (id: string) => this.containers.action(id, "stop"),
    restart: (id: string) => this.containers.action(id, "restart"),

    /** WebSocket — the only real-duplex route in the API. See DockerExecSocket. */
    exec: (id: string, handlers: ExecSocketHandlers): DockerExecSocket => {
      if (!this.token) throw new Error("Client has no token set — call login() first");
      return new DockerExecSocket(this.wsBaseUrl, id, this.token, handlers);
    },

    /** SSE — auth requires a header, so this uses fetch, not EventSource. */
    streamLogs: async function* (this: ServeoClient, id: string): AsyncGenerator<string> {
      const res = await fetch(`${this.baseUrlPublic}/v2/docker/containers/${id}/logs`, {
        headers: this.authHeaderPublic(),
      });
      if (!res.ok) throw new ServeoApiError(res.status, await res.text(), `/v2/docker/containers/${id}/logs`);
      yield* streamSSE(res);
    }.bind(this),

    /** SSE — same auth caveat as streamLogs. */
    streamStats: async function* (this: ServeoClient, id: string): AsyncGenerator<string> {
      const res = await fetch(`${this.baseUrlPublic}/v2/docker/containers/${id}/stats`, {
        headers: this.authHeaderPublic(),
      });
      if (!res.ok) throw new ServeoApiError(res.status, await res.text(), `/v2/docker/containers/${id}/stats`);
      yield* streamSSE(res);
    }.bind(this),
  };

  // --- Docker: Images ------------------------------------------------------

  images = {
    list: (): Promise<ImageInfo[]> => this.request("GET", "/v2/docker/images/"),

    /** SSE response to a POST — pull progress. */
    pull: async function* (this: ServeoClient, image: string): AsyncGenerator<string> {
      const res = await fetch(`${this.baseUrlPublic}/v2/docker/images/pull`, {
        method: "POST",
        headers: { ...this.authHeaderPublic(), "Content-Type": "application/json" },
        body: JSON.stringify({ image } satisfies PullImageRequest),
      });
      if (!res.ok) throw new ServeoApiError(res.status, await res.text(), "/v2/docker/images/pull");
      yield* streamSSE(res);
    }.bind(this),

    /** id can be an image ID or a repo:tag string. */
    delete: (id: string, force = false): Promise<void> =>
      this.request("DELETE", `/v2/docker/images/${encodeURIComponent(id)}`, {
        query: { force },
        parseAs: "none",
      }),
  };

  // --- Docker: Networks ----------------------------------------------------

  networks = {
    list: (): Promise<NetworkInfo[]> => this.request("GET", "/v2/docker/networks/"),

    create: (body: CreateNetworkRequest): Promise<NetworkInfo> =>
      this.request("POST", "/v2/docker/networks/", { body }),

    delete: (id: string): Promise<void> =>
      this.request("DELETE", `/v2/docker/networks/${id}`, { parseAs: "none" }),
  };

  // --- Docker: Volumes -----------------------------------------------------

  volumes = {
    list: (): Promise<VolumeInfo[]> => this.request("GET", "/v2/docker/volumes/"),

    create: (body: CreateVolumeRequest): Promise<VolumeInfo> =>
      this.request("POST", "/v2/docker/volumes/", { body }),

    /** Addressed by name, not id — the only delete route in the API that works this way. */
    delete: (name: string, force = false): Promise<void> =>
      this.request("DELETE", `/v2/docker/volumes/${encodeURIComponent(name)}`, {
        query: { force },
        parseAs: "none",
      }),
  };

  // --- Docker: System & Compose --------------------------------------------

  dockerSystem = {
    /** SSE. */
    streamEvents: async function* (this: ServeoClient): AsyncGenerator<string> {
      const res = await fetch(`${this.baseUrlPublic}/v2/docker/system/events`, {
        headers: this.authHeaderPublic(),
      });
      if (!res.ok) throw new ServeoApiError(res.status, await res.text(), "/v2/docker/system/events");
      yield* streamSSE(res);
    }.bind(this),

    /** Docker-engine metrics — NOT host metrics. See client.system.get() for host CPU/RAM/disk. */
    info: (): Promise<DockerSystemInfo> => this.request("GET", "/v2/docker/system/info"),

    prune: (): Promise<void> => this.request("POST", "/v2/docker/system/prune", { parseAs: "none" }),

    deployCompose: (body: DeployStackRequest): Promise<string> =>
      this.request("POST", "/v2/docker/compose/deploy", { body, parseAs: "text" }),
  };

  // --- Files (chrooted per-container) --------------------------------------

  files = {
    list: (server: string, path?: string): Promise<FileInfo[]> =>
      this.request("GET", `/v2/files/${encodeURIComponent(server)}/list`, {
        query: { path },
      }),

    /** Returns raw bytes (application/octet-stream) — decode as text yourself if applicable. */
    read: (server: string, path: string): Promise<Blob> =>
      this.request("GET", `/v2/files/${encodeURIComponent(server)}/read`, {
        query: { path },
        parseAs: "blob",
      }),

    /** Body is raw text/plain — the full new file content, not a JSON envelope. */
    write: (server: string, path: string, content: string): Promise<void> =>
      this.request("POST", `/v2/files/${encodeURIComponent(server)}/write`, {
        query: { path },
        rawBody: content,
        contentType: "text/plain",
        parseAs: "none",
      }),

    /**
     * multipart/form-data upload. The spec doesn't name the file field —
     * "file" is an unconfirmed guess; verify against server code or a
     * working request before depending on it.
     */
    upload: (server: string, path: string, file: File | Blob, fieldName = "file"): Promise<void> => {
      const form = new FormData();
      form.append(fieldName, file);
      return this.request("POST", `/v2/files/${encodeURIComponent(server)}/upload`, {
        query: { path },
        rawBody: form,
        parseAs: "none",
      });
    },

    delete: (server: string, path: string): Promise<void> =>
      this.request("DELETE", `/v2/files/${encodeURIComponent(server)}/delete`, {
        query: { path },
        parseAs: "none",
      }),
  };

  // --- Templates ------------------------------------------------------------

  templates = {
    list: (): Promise<TemplateInfo[]> => this.request("GET", "/v2/templates/"),

    get: (id: string): Promise<TemplateInfo> => this.request("GET", `/v2/templates/${id}`),

    /**
     * Convenience recipe (SKILL.md §6): fetch a template, substitute
     * `variables` into `docker`, and return the ready-to-POST payload.
     * Placeholder syntax isn't pinned by the spec — this assumes
     * `{{VAR_NAME}}`-style tokens inside string fields; confirm against
     * one real template before relying on it.
     */
    async buildDeployPayload(
      this: ServeoClient,
      id: string,
      values: Record<string, string>,
    ): Promise<CreateContainerRequest> {
      const template = await this.templates.get(id);
      for (const v of template.variables) {
        if (v.required && values[v.name] === undefined && v.default === undefined) {
          throw new Error(`Missing required template variable: ${v.name}`);
        }
      }
      const json = JSON.stringify(template.docker);
      const substituted = json.replace(/\{\{(\w+)\}\}/g, (_, name: string) => {
        return values[name] ?? template.variables.find((v) => v.name === name)?.default ?? "";
      });
      return JSON.parse(substituted);
    },
  };

  // --- Metadata / System / OVH ----------------------------------------------

  metadata = {
    get: (): Promise<MetadataResponse> => this.request("GET", "/v2/metadata/"),
  };

  /** Host metrics (CPU/RAM/disk/network of the machine ServeoAPI runs on). */
  system = {
    get: (): Promise<HostSystemResponse> => this.request("GET", "/v2/system/"),
  };

  ovh = {
    me: (): Promise<OvhMeResponse> => this.request("GET", "/v2/ovh/me"),

    listDedicatedServers: (): Promise<string[]> => this.request("GET", "/v2/ovh/dedicated/server"),

    /** Hard reboot — no server-side confirmation step; gate this in your UI. */
    rebootDedicatedServer: (serviceName: string): Promise<string> =>
      this.request("POST", `/v2/ovh/dedicated/server/${encodeURIComponent(serviceName)}/reboot`, {
        parseAs: "text",
      }),
  };

  // --- MCP Server -----------------------------------------------------------

  mcp = {
    /** Establish the SSE connection for MCP JSON-RPC communication. */
    connect: async function* (this: ServeoClient): AsyncGenerator<string> {
      const res = await fetch(`${this.baseUrlPublic}/v2/mcp/`, {
        headers: this.authHeaderPublic(),
      });
      if (!res.ok) throw new ServeoApiError(res.status, await res.text(), "/v2/mcp/");
      yield* streamSSE(res);
    }.bind(this),

    /** Send an MCP JSON-RPC message (e.g. tool call or init) via POST. */
    send: (message: Record<string, unknown>): Promise<unknown> =>
      this.request("POST", "/v2/mcp/", {
        body: message,
      }),
  };

  // --- internal accessors used by the generator-function methods above -----
  // (arrow-function class fields can't easily close over `this.baseUrl`/
  // `this.token` inside a `function*` body, so expose them explicitly)

  private get baseUrlPublic(): string {
    return this.baseUrl;
  }
  private authHeaderPublic(): Record<string, string> {
    return this.token ? { Authorization: `Bearer ${this.token}` } : {};
  }
}

// ============================================================================
// Usage example
// ============================================================================
//
// const client = new ServeoClient({ baseUrl: "https://api.isweat.pro" });
// await client.login("admin", "hunter2");
//
// const containers = await client.containers.list();
// await client.containers.start(containers[0].id);
//
// for await (const line of client.containers.streamLogs(containers[0].id)) {
//   console.log(line);
// }
//
// const socket = client.containers.exec(containers[0].id, {
//   onOutput: (chunk) => term.write(new Uint8Array(chunk as ArrayBuffer)),
// });
// socket.sendInput("ls -la\n");