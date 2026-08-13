export interface SessionResponse {
  authenticated: boolean;
  login?: string;
  csrf_token: string;
  setup_required: boolean;
}

export interface DeviceResponse {
  id: string;
  name: string;
  last_seen_at: string | null;
  policy_revision: number;
  applied_policy_revision: number;
  graphical_session_active: boolean;
  online: boolean;
}

export interface RoutineResponse {
  id: string;
  name: string;
  days: [boolean, boolean, boolean, boolean, boolean, boolean, boolean];
  start_second: number;
  end_second: number;
  enabled: boolean;
}

export interface PolicyResponse {
  revision: number;
  monitoring_paused: boolean;
  manual_block: boolean;
  warning_minutes: number;
  password_set: boolean;
  weekly_quota_seconds: [number, number, number, number, number, number, number];
  routines: RoutineResponse[];
}

export interface ControlResponse {
	revision: number;
	monitoring_paused: boolean;
	manual_block: boolean;
}

export interface DeviceStatusResponse {
  local_date: string;
  today_quota_seconds: number;
  bonus_seconds: number;
  used_seconds: number;
  remaining_seconds: number;
  counting: boolean;
  online: boolean;
  graphical_session_active: boolean;
  next_block: string;
  policy_revision: number;
  applied_policy_revision: number;
	control_revision: number;
	applied_control_revision: number;
	desired_state: "policy" | "paused" | "blocked";
	actual_state: "offline" | "unblocked" | "blocked";
	control_status: "active" | "pause_requested" | "paused" | "block_requested" | "blocked" | "offline";
}

export interface AuditEventResponse {
  id: string;
  kind: string;
  origin: string;
  details: string;
  created_at: string;
}

export interface DeviceDetailResponse {
  device: DeviceResponse;
  policy: PolicyResponse;
	control: ControlResponse;
  status: DeviceStatusResponse;
  events: AuditEventResponse[];
}

export class CompassoAPIError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = "CompassoAPIError";
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  mutatesState?: boolean;
}

export class CompassoAPIClient {
  private csrfToken = "";

  constructor(
    private readonly apiBaseURL = "",
    private readonly fetcher: typeof fetch = window.fetch.bind(window),
  ) {}

  private async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const headers = new Headers({ Accept: "application/json" });
    if (options.body !== undefined) headers.set("Content-Type", "application/json");
    if (options.mutatesState && this.csrfToken) headers.set("X-CSRF-Token", this.csrfToken);
    let response: Response;
    try {
      response = await this.fetcher(`${this.apiBaseURL.replace(/\/$/, "")}${path}`, {
        method: options.method ?? "GET",
        credentials: "include",
        cache: "no-store",
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
    } catch {
      throw new CompassoAPIError("Não foi possível alcançar o servidor Compasso.", 0);
    }
    if (response.status === 204) return undefined as T;
    const payload = response.headers.get("Content-Type")?.includes("application/json")
      ? await response.json() as { error?: string }
      : {};
    if (!response.ok) throw new CompassoAPIError(payload.error ?? "Não foi possível concluir a operação.", response.status);
    return payload as T;
  }

  async loadSession() {
    const session = await this.request<SessionResponse>("/api/v1/admin/session");
    this.csrfToken = session.csrf_token ?? "";
    return session;
  }

  async login(login: string, password: string) {
    if (!this.csrfToken) await this.loadSession();
    const session = await this.request<SessionResponse>("/api/v1/admin/session", {
      method: "POST",
      body: { login, password, csrf_token: this.csrfToken },
    });
    this.csrfToken = session.csrf_token ?? "";
    return session;
  }

  async setup(login: string, password: string, passwordConfirmation: string) {
    if (!this.csrfToken) await this.loadSession();
    const session = await this.request<SessionResponse>("/api/v1/admin/setup", {
      method: "POST",
      body: { login, password, password_confirmation: passwordConfirmation, csrf_token: this.csrfToken },
    });
    this.csrfToken = session.csrf_token ?? "";
    return session;
  }

  async logout() {
    await this.request<void>("/api/v1/admin/session", { method: "DELETE", mutatesState: true });
    this.csrfToken = "";
  }

  listDevices() { return this.request<{ devices: DeviceResponse[] }>("/api/v1/admin/devices"); }
  loadDevice(deviceId: string) { return this.request<DeviceDetailResponse>(`/api/v1/admin/devices/${deviceId}`); }
  loadStatus(deviceId: string) { return this.request<DeviceStatusResponse>(`/api/v1/admin/devices/${deviceId}/status`); }
  loadEvents(deviceId: string, limit = 30) { return this.request<{ events: AuditEventResponse[] }>(`/api/v1/admin/devices/${deviceId}/events?limit=${limit}`); }
  createDevice(name: string) { return this.request<DeviceResponse>("/api/v1/admin/devices", { method: "POST", body: { name }, mutatesState: true }); }
  renameDevice(deviceId: string, name: string) { return this.request(`/api/v1/admin/devices/${deviceId}`, { method: "PATCH", body: { name }, mutatesState: true }); }
  deleteDevice(deviceId: string) { return this.request<void>(`/api/v1/admin/devices/${deviceId}`, { method: "DELETE", mutatesState: true }); }
  updatePolicy(deviceId: string, weeklyQuotaSeconds: number[], warningMinutes: number) {
    return this.request(`/api/v1/admin/devices/${deviceId}/policy`, { method: "PUT", body: { weekly_quota_seconds: weeklyQuotaSeconds, warning_minutes: warningMinutes }, mutatesState: true });
  }
  saveRoutine(deviceId: string, routineId: string | undefined, routine: Omit<RoutineResponse, "id">) {
    return this.request<{ id: string }>(`/api/v1/admin/devices/${deviceId}/routines${routineId ? `/${routineId}` : ""}`, { method: routineId ? "PUT" : "POST", body: routine, mutatesState: true });
  }
  deleteRoutine(deviceId: string, routineId: string) { return this.request<void>(`/api/v1/admin/devices/${deviceId}/routines/${routineId}`, { method: "DELETE", mutatesState: true }); }
  updatePassword(deviceId: string, password: string, confirmation: string) { return this.request(`/api/v1/admin/devices/${deviceId}/password`, { method: "PUT", body: { password, password_confirmation: confirmation }, mutatesState: true }); }
  issueToken(deviceId: string) { return this.request<{ device_id: string; device_token: string }>(`/api/v1/admin/devices/${deviceId}/token`, { method: "POST", mutatesState: true }); }
  revokeToken(deviceId: string) { return this.request<void>(`/api/v1/admin/devices/${deviceId}/token`, { method: "DELETE", mutatesState: true }); }
  addBonus(deviceId: string, minutes: number) { return this.request(`/api/v1/admin/devices/${deviceId}/bonus`, { method: "POST", body: { minutes }, mutatesState: true }); }
  queueCommand(deviceId: string, command: "pause_monitoring" | "resume_monitoring" | "block_now" | "clear_manual_block") {
    return this.request(`/api/v1/admin/devices/${deviceId}/commands`, { method: "POST", body: { command }, mutatesState: true });
  }
}

const runtimeApiBaseUrl = window.COMPASSO_CONFIG?.apiBaseUrl?.trim() ?? "";
const configuredApiBaseUrl = runtimeApiBaseUrl || (import.meta.env.VITE_COMPASSO_API_BASE_URL ?? "").trim();
const isTestEnvironment = import.meta.env.MODE === "test";
const effectiveApiBaseUrl = import.meta.env.DEV ? "" : configuredApiBaseUrl;
export const remoteMode = !isTestEnvironment && (
  import.meta.env.MODE === "integration"
  || import.meta.env.VITE_COMPASSO_REMOTE === "true"
  || Boolean(configuredApiBaseUrl)
);
export const compassoAPI = new CompassoAPIClient(effectiveApiBaseUrl);
