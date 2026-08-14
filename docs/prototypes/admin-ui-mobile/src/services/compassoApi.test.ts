import { CompassoAPIClient, CompassoAPIError } from "./compassoApi";

function jsonResponse(payload: unknown, status = 200) {
  return new Response(JSON.stringify(payload), { status, headers: { "Content-Type": "application/json" } });
}

describe("CompassoAPIClient", () => {
  it("enables remote mode when an API base URL is configured", async () => {
    vi.resetModules();
    vi.stubEnv("VITE_COMPASSO_API_BASE_URL", "http://localhost:8181");
    vi.stubEnv("MODE", "development");

    const { remoteMode } = await import("./compassoApi");

    expect(remoteMode).toBe(true);
  });

  it("loads CSRF and sends it with authenticated mutations", async () => {
    const requests: RequestInit[] = [];
    const fetcher = vi.fn(async (_url: string | URL | Request, options?: RequestInit) => {
      requests.push(options ?? {});
      if (requests.length === 1) return jsonResponse({ authenticated: false, csrf_token: "csrf-1", setup_required: false });
      if (requests.length === 2) return jsonResponse({ authenticated: true, login: "admin", csrf_token: "csrf-2", setup_required: false });
      return jsonResponse({ id: "device-1", name: "Teste" }, 201);
    });
    const api = new CompassoAPIClient("", fetcher as typeof fetch);

    await api.login("admin", "segredo");
    await api.createDevice("Teste");

    expect(new Headers(requests[2].headers).get("X-CSRF-Token")).toBe("csrf-2");
    expect(requests[2].credentials).toBe("include");
  });

  it("turns network and API failures into typed errors", async () => {
    const networkAPI = new CompassoAPIClient("", vi.fn().mockRejectedValue(new Error("offline")) as typeof fetch);
    await expect(networkAPI.loadSession()).rejects.toMatchObject({ status: 0 });

    const rejectedAPI = new CompassoAPIClient("", vi.fn().mockResolvedValue(jsonResponse({ error: "authentication required" }, 401)) as typeof fetch);
    await expect(rejectedAPI.loadSession()).rejects.toEqual(new CompassoAPIError("authentication required", 401));
  });
});
