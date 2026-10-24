import test from "node:test";
import assert from "node:assert/strict";
import { blank, createItems, copyAndNext } from "./.test-build/model.js";
test("unified configuration produces independent rows", () => {
  const rs = createItems(
    "alpha\nbeta",
    { cpu: "2", memory: "2GB", disk: "10GB" },
    "images:ubuntu/24.04",
  );
  rs[0].resources.cpu = "4";
  assert.equal(rs[1].resources.cpu, "2");
});
test("clipboard failure retains the current graph", async () => {
  let switched = false;
  await assert.rejects(() =>
    copyAndNext(
      async () => {},
      async () => "markdown",
      async () => {
        throw Error("blocked");
      },
      () => {
        switched = true;
      },
    ),
  );
  assert.equal(switched, false);
});
test("save and export precede copy, switching follows copy", async () => {
  const calls = [];
  await copyAndNext(
    async () => {
      calls.push("save");
    },
    async () => {
      calls.push("export");
      return "text";
    },
    async () => {
      calls.push("copy");
    },
    () => calls.push("next"),
  );
  assert.deepEqual(calls, ["save", "export", "copy", "next"]);
});
test("blank draft does not delete history", () => {
  assert.equal(blank().id, "");
  assert.deepEqual(blank().nodes, []);
});
test("unified Tenant selection is retained independently on every row", () => {
  const rows = createItems(
    "frontend backend",
    { cpu: "2", memory: "2GB", disk: "10GB" },
    "images:ubuntu/24.04",
    "tenant-example",
  );
  assert.deepEqual(
    rows.map((r) => r.tenantId),
    ["tenant-example", "tenant-example"],
  );
  rows[0].tenantId = "different";
  assert.equal(rows[1].tenantId, "tenant-example");
});
test("open-tab evidence expires without a successful network refresh", async () => {
  const { expireEvidence } = await import("./.test-build/model.js");
  const now = Date.now();
  const stamp = new Date(now - 181000).toISOString();
  const data = {
    guard: { tenant: { fresh: true, status: "ready", lastPass: stamp } },
    tenants: [{ id: "dev", guardStatus: "verified" }],
    containers: [
      {
        tenant: {
          confirmed: true,
          guard: { status: "verified", lastPass: stamp },
        },
      },
    ],
  };
  const expired = expireEvidence(data, now);
  assert.equal(expired.guard.tenant.status, "unknown");
  assert.equal(expired.tenants[0].guardStatus, "unknown");
  assert.equal(expired.containers[0].tenant.guard.status, "unknown");
  assert.equal(data.tenants[0].guardStatus, "verified");
});
