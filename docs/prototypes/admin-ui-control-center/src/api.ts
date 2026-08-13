type Options = { method?: string; body?: unknown };

export class CompassoApi {
  private csrf = "";
  constructor(private baseUrl = "") {}

  private async request<T>(path: string, options: Options = {}): Promise<T> {
    const headers = new Headers({ Accept: "application/json" });
    if (options.body !== undefined) headers.set("Content-Type", "application/json");
    if (options.method && options.method !== "GET" && this.csrf) headers.set("X-CSRF-Token", this.csrf);
    const response = await fetch(`${this.baseUrl}${path}`, {
      method: options.method ?? "GET", credentials: "include", headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    });
    if (response.status === 204) return undefined as T;
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error ?? "Não foi possível concluir a operação.");
    return payload as T;
  }

  async session() {
    const value = await this.request<{ authenticated: boolean; csrf_token: string; setup_required: boolean }>("/api/v1/admin/session");
    this.csrf = value.csrf_token;
    return value;
  }
  login(login: string, password: string) { return this.request("/api/v1/admin/session", { method: "POST", body: { login, password, csrf_token: this.csrf } }); }
  listDevices() { return this.request<{ devices: unknown[] }>("/api/v1/admin/devices"); }
  addBonus(id: string, minutes: number) { return this.request(`/api/v1/admin/devices/${id}/bonus`, { method: "POST", body: { minutes } }); }
  command(id: string, command: string) { return this.request(`/api/v1/admin/devices/${id}/commands`, { method: "POST", body: { command } }); }
  updatePolicy(id: string, limits: number[], warningMinutes: number) { return this.request(`/api/v1/admin/devices/${id}/policy`, { method: "PUT", body: { weekly_quota_seconds: limits.map((v) => v * 60), warning_minutes: warningMinutes } }); }
}

export const api = new CompassoApi();
