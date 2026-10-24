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
