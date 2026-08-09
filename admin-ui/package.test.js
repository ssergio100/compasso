"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const read = (name) => fs.readFileSync(path.join(__dirname, name), "utf8");

test("frontend is configured at runtime and calls only the versioned API", () => {
  const client = read("api-client.js");
  const runtimeTemplate = read("runtime-config.template.js");
  assert.match(runtimeTemplate, /COMPASSO_API_BASE_URL/);
  assert.match(runtimeTemplate, /configuredAPIBaseURL === "auto"/);
  assert.doesNotMatch(client, /server\/storage|\.db\b|device_token\s*=/);
  assert.match(client, /\/api\/v1\/admin\/session/);
  assert.match(client, /\/api\/v1\/admin\/devices/);
});

test("frontend image contains no backend source or binary", () => {
  const dockerfile = read("Dockerfile");
  assert.match(dockerfile, /^USER nginx:nginx$/m);
  assert.doesNotMatch(dockerfile, /COPY .*server|tempo-server|go build/);
  assert.match(dockerfile, /COMPASSO_API_BASE_URL|compasso-admin-ui/);
});
