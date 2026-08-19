import type { CommunicationResponse, Device, DeviceDetailResponse, DeviceResponse, Routine, Session } from "./types";

const productionAPIBase = import.meta.env.PROD
  ? `${window.location.protocol}//${window.location.hostname}:8181`
  : "";
const runtimeBase = (window.COMPASSO_CONFIG?.apiBaseUrl ?? window.COMPASSO_CONFIG?.apiBaseURL ?? "").trim();
const envBase = (import.meta.env.VITE_COMPASSO_API_BASE_URL ?? "").trim();
export const remoteMode = import.meta.env.VITE_COMPASSO_REMOTE === "true" || Boolean(runtimeBase || envBase || productionAPIBase);

class API {
  private csrf = "";
  private base = import.meta.env.DEV ? "" : runtimeBase || envBase || productionAPIBase;

  private async request<T>(path: string, init: RequestInit = {}, mutate = false): Promise<T> {
    const headers = new Headers(init.headers);
    headers.set("Accept", "application/json");
    if (init.body) headers.set("Content-Type", "application/json");
    if (mutate && this.csrf) headers.set("X-CSRF-Token", this.csrf);
    const response = await fetch(`${this.base}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
    if (response.status === 204) return undefined as T;
    const isJSON = response.headers.get("Content-Type")?.includes("application/json") ?? false;
    const payload = isJSON ? await response.json().catch(() => ({})) : {};
    if (!response.ok) {
      const wrongServer = response.status === 405 && !isJSON;
      throw new Error(wrongServer
        ? "A interface está apontando para o servidor de arquivos, não para a API. Revise runtime-config.js."
        : payload.error ?? "Não foi possível concluir a operação.");
    }
    return payload as T;
  }

  async session() { const value = await this.request<Session>("/api/v1/admin/session"); this.csrf = value.csrf_token; return value; }
  async login(login: string, password: string) {
    if (!this.csrf) await this.session();
    const value = await this.request<Session>("/api/v1/admin/session", { method: "POST", body: JSON.stringify({ login, password, csrf_token: this.csrf }) });
    this.csrf = value.csrf_token; return value;
  }
  async logout() { await this.request<void>("/api/v1/admin/session", { method: "DELETE" }, true); }
  private async device(id: string): Promise<Device> {
    const detail = await this.request<DeviceDetailResponse>(`/api/v1/admin/devices/${id}`);
    return {
      id: detail.device.id, name: detail.device.name,
      online: detail.status.online,
      graphical_session_active: detail.status.graphical_session_active,
      monitoring_paused: detail.control.monitoring_paused, manual_block: detail.control.manual_block,
      counting: detail.status.counting, used_seconds: detail.status.used_seconds,
      remaining_seconds: detail.status.remaining_seconds, bonus_seconds: detail.status.bonus_seconds,
      today_quota_seconds: detail.status.today_quota_seconds, warning_minutes: detail.policy.warning_minutes,
      last_seen_at: detail.device.last_seen_at, weekly_quota_seconds: detail.policy.weekly_quota_seconds,
      routines: detail.policy.routines, password_set: detail.policy.password_set,
    };
  }
  async devices() {
    const list = await this.request<{ devices: DeviceResponse[] }>("/api/v1/admin/devices");
    return Promise.all(list.devices.map((summary) => this.device(summary.id)));
  }
  loadDevice(id: string) { return this.device(id); }
  createDevice(name: string) { return this.request<DeviceResponse>("/api/v1/admin/devices", { method: "POST", body: JSON.stringify({ name }) }, true); }
  deleteDevice(id: string) { return this.request<void>(`/api/v1/admin/devices/${id}`, { method: "DELETE" }, true); }
  rename(id: string, name: string) { return this.request(`/api/v1/admin/devices/${id}`, { method: "PATCH", body: JSON.stringify({ name }) }, true); }
  command(id: string, command: string) { return this.request(`/api/v1/admin/devices/${id}/commands`, { method: "POST", body: JSON.stringify({ command }) }, true); }
  bonus(id: string, minutes: number) { return this.request<{ message: string; operation_id: string }>(`/api/v1/admin/devices/${id}/bonus`, { method: "POST", body: JSON.stringify({ minutes }) }, true); }
  bonusStatus(id: string, operationId: string) { return this.request<{ acknowledged: boolean }>(`/api/v1/admin/devices/${id}/commands/${operationId}`); }
  policy(id: string, weekly: number[], warning: number) { return this.request(`/api/v1/admin/devices/${id}/policy`, { method: "PUT", body: JSON.stringify({ weekly_quota_seconds: weekly, warning_minutes: warning }) }, true); }
  updatePassword(id: string, password: string, confirmation: string) { return this.request(`/api/v1/admin/devices/${id}/password`, { method: "PUT", body: JSON.stringify({ password, password_confirmation: confirmation }) }, true); }
  issueToken(id: string) { return this.request<{ device_id: string; device_token: string }>(`/api/v1/admin/devices/${id}/token`, { method: "POST" }, true); }
  revokeToken(id: string) { return this.request<void>(`/api/v1/admin/devices/${id}/token`, { method: "DELETE" }, true); }
  routine(id: string, routine: Omit<Routine, "id">, routineId?: string) { return this.request(`/api/v1/admin/devices/${id}/routines${routineId ? `/${routineId}` : ""}`, { method: routineId ? "PUT" : "POST", body: JSON.stringify(routine) }, true); }
  deleteRoutine(id: string, routineId: string) { return this.request(`/api/v1/admin/devices/${id}/routines/${routineId}`, { method: "DELETE" }, true); }
  communication(id: string, after = 0) { return this.request<CommunicationResponse>(`/api/v1/admin/devices/${id}/communication?limit=200${after ? `&after=${after}` : ""}`); }
  setCommunicationRetention(id: string, retentionDays: number) { return this.request<{ retention_days: number }>(`/api/v1/admin/devices/${id}/communication/settings`, { method: "PUT", body: JSON.stringify({ retention_days: retentionDays }) }, true); }
  deleteCommunication(id: string) { return this.request<{ deleted: number }>(`/api/v1/admin/devices/${id}/communication`, { method: "DELETE" }, true); }
}

export const api = new API();
