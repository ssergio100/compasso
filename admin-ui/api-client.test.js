"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

function loadAPIClient(mockFetch) {
  const browserWindow = { fetch: mockFetch };
  const context = vm.createContext({ window: browserWindow, Headers });
  const source = fs.readFileSync(path.join(__dirname, "api-client.js"), "utf8");
  vm.runInContext(source, context);
  return new browserWindow.CompassoAPI.CompassoAPIClient("https://api.example.test/");
}

function loadCompassoAPI() {
  const browserWindow = { fetch: async () => {} };
  const context = vm.createContext({ window: browserWindow, Headers });
  const source = fs.readFileSync(path.join(__dirname, "api-client.js"), "utf8");
  vm.runInContext(source, context);
  return browserWindow.CompassoAPI;
}

function jsonResponse(payload, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

test("login obtains CSRF token and uses credentialed JSON requests", async () => {
  const requests = [];
  const responses = [
    jsonResponse({ authenticated: false, csrf_token: "login-csrf" }),
    jsonResponse({ authenticated: true, login: "admin", csrf_token: "session-csrf" }),
  ];
  const client = loadAPIClient(async (url, options) => {
    requests.push({ url, options });
    return responses.shift();
  });

  const session = await client.login("admin", "test-password");

  assert.equal(session.authenticated, true);
  assert.equal(requests[0].url, "https://api.example.test/api/v1/admin/session");
  assert.equal(requests[0].options.credentials, "include");
  assert.deepEqual(JSON.parse(requests[1].options.body), {
    login: "admin", password: "test-password", csrf_token: "login-csrf",
  });
});

test("state-changing calls send the session CSRF header", async () => {
  const requests = [];
  const responses = [
    { authenticated: true, login: "admin", csrf_token: "session-csrf" },
    { message: "bonus queued", operation_id: "operation-123" },
    { acknowledged: false },
  ];
  const client = loadAPIClient(async (url, options) => {
    requests.push({ url, options });
    return jsonResponse(responses.shift());
  });
  await client.loadSession();
  const confirmation = await client.addBonus("device-123", 15);
  const status = await client.loadBonusStatus("device-123", confirmation.operation_id);

  const bonusRequest = requests[1];
  assert.equal(bonusRequest.url, "https://api.example.test/api/v1/admin/devices/device-123/bonus");
  assert.equal(bonusRequest.options.method, "POST");
  assert.equal(bonusRequest.options.headers.get("X-CSRF-Token"), "session-csrf");
  assert.deepEqual(JSON.parse(bonusRequest.options.body), { minutes: 15 });
  assert.equal(confirmation.operation_id, "operation-123");
  assert.equal(status.acknowledged, false);
  assert.equal(requests[2].url, "https://api.example.test/api/v1/admin/devices/device-123/commands/operation-123");
});

test("initial setup creates the administrator after installation", async () => {
  const requests = [];
  const responses = [
    jsonResponse({ authenticated: false, setup_required: true, csrf_token: "setup-csrf" }),
    jsonResponse({ authenticated: true, login: "sergio", csrf_token: "session-csrf" }, 201),
  ];
  const client = loadAPIClient(async (url, options) => {
    requests.push({ url, options });
    return responses.shift();
  });

  const session = await client.completeInitialSetup("sergio", "senha", "senha");

  assert.equal(session.authenticated, true);
  assert.equal(requests[1].url, "https://api.example.test/api/v1/admin/setup");
  assert.deepEqual(JSON.parse(requests[1].options.body), {
    login: "sergio", password: "senha", password_confirmation: "senha", csrf_token: "setup-csrf",
  });
});

test("API errors preserve HTTP status and safe server message", async () => {
  const client = loadAPIClient(async () => jsonResponse({ error: "credenciais inválidas" }, 401));
  await assert.rejects(client.loadSession(), (error) => {
    assert.equal(error.name, "CompassoAPIError");
    assert.equal(error.status, 401);
    assert.equal(error.message, "credenciais inválidas");
    return true;
  });
});

test("an older online poll never returns a consumed second", () => {
  const { mergeLiveStatus } = loadCompassoAPI();
  const currentStatus = {
    local_date: "2026-08-09", online: true, counting: true,
    used_seconds: 1, remaining_seconds: 599,
  };
  const staleServerStatus = {
    local_date: "2026-08-09", online: true, counting: true,
    used_seconds: 0, remaining_seconds: 600,
  };

  const mergedStatus = mergeLiveStatus(currentStatus, staleServerStatus);

  assert.equal(mergedStatus.used_seconds, 1);
  assert.equal(mergedStatus.remaining_seconds, 599);
});

test("a real allowance increase is preserved while merging status", () => {
  const { mergeLiveStatus } = loadCompassoAPI();
  const currentStatus = {
    local_date: "2026-08-09", online: true, counting: true,
    used_seconds: 20, remaining_seconds: 580,
  };
  const statusAfterBonus = {
    local_date: "2026-08-09", online: true, counting: true,
    used_seconds: 18, remaining_seconds: 1182,
  };

  const mergedStatus = mergeLiveStatus(currentStatus, statusAfterBonus);

  assert.equal(mergedStatus.used_seconds, 20);
  assert.equal(mergedStatus.remaining_seconds, 1180);
});

test("a new day accepts the server counters from zero", () => {
  const { mergeLiveStatus } = loadCompassoAPI();
  const previousDay = {
    local_date: "2026-08-09", online: true, counting: true,
    used_seconds: 600, remaining_seconds: 0,
  };
  const newDay = {
    local_date: "2026-08-10", online: true, counting: true,
    used_seconds: 0, remaining_seconds: 3600,
  };

  const mergedStatus = mergeLiveStatus(previousDay, newDay);

  assert.equal(mergedStatus.used_seconds, 0);
  assert.equal(mergedStatus.remaining_seconds, 3600);
});
