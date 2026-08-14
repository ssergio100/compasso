(() => {
  "use strict";

  class CompassoAPIError extends Error {
    constructor(message, status) {
      super(message);
      this.name = "CompassoAPIError";
      this.status = status;
    }
  }

  class CompassoAPIClient {
    constructor(apiBaseURL) {
      this.apiBaseURL = String(apiBaseURL || "").replace(/\/$/, "");
      this.csrfToken = "";
    }

    async request(path, options = {}) {
      const headers = new Headers(options.headers || {});
      headers.set("Accept", "application/json");
      if (options.body !== undefined) headers.set("Content-Type", "application/json");
      if (options.mutatesState && this.csrfToken) headers.set("X-CSRF-Token", this.csrfToken);

      const response = await window.fetch(`${this.apiBaseURL}${path}`, {
        method: options.method || "GET",
        credentials: "include",
        cache: "no-store",
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
      if (response.status === 204) return null;

      const contentType = response.headers.get("Content-Type") || "";
      const payload = contentType.includes("application/json") ? await response.json() : {};
      if (!response.ok) {
        throw new CompassoAPIError(payload.error || "Não foi possível concluir a operação.", response.status);
      }
      return payload;
    }

    async loadSession() {
      const session = await this.request("/api/v1/admin/session");
      this.csrfToken = session.csrf_token || "";
      return session;
    }

    async login(login, password) {
      if (!this.csrfToken) await this.loadSession();
      const session = await this.request("/api/v1/admin/session", {
        method: "POST",
        body: { login, password, csrf_token: this.csrfToken },
      });
      this.csrfToken = session.csrf_token || "";
      return session;
    }

    async completeInitialSetup(login, password, passwordConfirmation) {
      if (!this.csrfToken) await this.loadSession();
      const session = await this.request("/api/v1/admin/setup", {
        method: "POST",
        body: {
          login, password, password_confirmation: passwordConfirmation, csrf_token: this.csrfToken,
        },
      });
      this.csrfToken = session.csrf_token || "";
      return session;
    }

    async logout() {
      await this.request("/api/v1/admin/session", { method: "DELETE", mutatesState: true });
      this.csrfToken = "";
    }

    listDevices() { return this.request("/api/v1/admin/devices"); }
    loadDevice(deviceID) { return this.request(`/api/v1/admin/devices/${deviceID}`); }
    loadStatus(deviceID) { return this.request(`/api/v1/admin/devices/${deviceID}/status`); }
    createDevice(name) {
      return this.request("/api/v1/admin/devices", { method: "POST", body: { name }, mutatesState: true });
    }
    renameDevice(deviceID, name) {
      return this.request(`/api/v1/admin/devices/${deviceID}`, { method: "PATCH", body: { name }, mutatesState: true });
    }
    deleteDevice(deviceID) {
      return this.request(`/api/v1/admin/devices/${deviceID}`, { method: "DELETE", mutatesState: true });
    }
    updatePolicy(deviceID, weeklyQuotaSeconds, warningMinutes) {
      return this.request(`/api/v1/admin/devices/${deviceID}/policy`, {
        method: "PUT",
        body: { weekly_quota_seconds: weeklyQuotaSeconds, warning_minutes: warningMinutes },
        mutatesState: true,
      });
    }
    saveRoutine(deviceID, routineID, routine) {
      const suffix = routineID ? `/${routineID}` : "";
      return this.request(`/api/v1/admin/devices/${deviceID}/routines${suffix}`, {
        method: routineID ? "PUT" : "POST", body: routine, mutatesState: true,
      });
    }
    deleteRoutine(deviceID, routineID) {
      return this.request(`/api/v1/admin/devices/${deviceID}/routines/${routineID}`, {
        method: "DELETE", mutatesState: true,
      });
    }
    updatePassword(deviceID, password, passwordConfirmation) {
      return this.request(`/api/v1/admin/devices/${deviceID}/password`, {
        method: "PUT",
        body: { password, password_confirmation: passwordConfirmation },
        mutatesState: true,
      });
    }
    issueToken(deviceID) {
      return this.request(`/api/v1/admin/devices/${deviceID}/token`, { method: "POST", mutatesState: true });
    }
    revokeToken(deviceID) {
      return this.request(`/api/v1/admin/devices/${deviceID}/token`, { method: "DELETE", mutatesState: true });
    }
    addBonus(deviceID, minutes) {
      return this.request(`/api/v1/admin/devices/${deviceID}/bonus`, {
        method: "POST", body: { minutes }, mutatesState: true,
      });
    }
    queueCommand(deviceID, command) {
      return this.request(`/api/v1/admin/devices/${deviceID}/commands`, {
        method: "POST", body: { command }, mutatesState: true,
      });
    }
  }

  const mergeLiveStatus = (currentStatus, incomingStatus) => {
    if (!currentStatus || !incomingStatus.local_date
        || currentStatus.local_date !== incomingStatus.local_date
        || !currentStatus.online || !incomingStatus.online) {
      return { ...incomingStatus };
    }

    // The browser advances the counter between heartbeats. A status poll can
    // therefore contain an older usage sample and must never return seconds to
    // the same online day. Deriving the allowance from the incoming snapshot
    // still lets real quota or bonus changes increase the remaining time.
    const incomingUsedSeconds = Math.max(0, Number(incomingStatus.used_seconds) || 0);
    const currentUsedSeconds = Math.max(0, Number(currentStatus.used_seconds) || 0);
    const effectiveUsedSeconds = Math.max(incomingUsedSeconds, currentUsedSeconds);
    const incomingAllowanceSeconds = incomingUsedSeconds
      + Math.max(0, Number(incomingStatus.remaining_seconds) || 0);

    return {
      ...incomingStatus,
      used_seconds: effectiveUsedSeconds,
      remaining_seconds: Math.max(0, incomingAllowanceSeconds - effectiveUsedSeconds),
    };
  };

  window.CompassoAPI = Object.freeze({ CompassoAPIClient, CompassoAPIError, mergeLiveStatus });
})();
