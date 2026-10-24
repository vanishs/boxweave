import { test, expect } from "@playwright/test";
test.beforeEach(async ({ page }) => {
  await page.goto("/");
  await page
    .getByLabel("Administrator password")
    .fill("browser test password only");
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.getByRole("button", { name: /Boxes$/, exact: false }).click();
  await expect(
    page.getByRole("heading", { name: "Boxes", exact: true }),
  ).toBeVisible();
});
test("resource search, details and selected-field batch operations", async ({
  page,
}) => {
  await page.getByLabel("Search resources").fill("alpha");
  await expect(
    page.getByRole("button", { name: "alpha", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "beta", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "alpha", exact: true }).click();
  await expect(page.getByRole("dialog")).toContainText("10.0.0.2");
  await page.getByLabel("Close dialog").click();
  await page.getByLabel("Select alpha", { exact: true }).check();
  await page.getByRole("button", { name: "stop", exact: true }).click();
  await expect(page.getByRole("dialog")).toContainText("alpha");
  await page.getByRole("button", { name: "Confirm stop", exact: true }).click();
  await expect(page.getByText("Operation receipts")).toBeVisible();
});
test("shared and individual create modes retain rows and confirm exact plan", async ({
  page,
}) => {
  await page.getByRole("button", { name: "+ Create boxes" }).click();
  await page.getByLabel("Box names").fill("browser-one\nbrowser-two");
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await page.getByLabel("cpu row 2", { exact: true }).fill("4");
  await page
    .getByRole("button", { name: "Unified configuration", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await expect(page.getByLabel("cpu row 2", { exact: true })).toHaveValue("4");
  await page.getByRole("button", { name: "Review creation plan" }).click();
  await expect(page.getByRole("dialog")).toContainText("browser-two");
  await expect(page.getByRole("dialog")).toContainText('"cpu": "4"');
  await page
    .getByRole("button", { name: "Confirm create", exact: true })
    .click();
  await expect(page.getByText("Operation receipts")).toBeVisible();
});
test("resource cards can repeat, auto-save and restore after reload", async ({
  page,
}) => {
  await page.getByLabel("Search resources").fill("alpha");
  await page.getByRole("button", { name: "+ Prompt card" }).click();
  await page
    .getByLabel("Comment", { exact: true })
    .fill("first independent comment");
  await page.getByRole("button", { name: "Add card & save" }).click();
  await page.getByRole("button", { name: "+ Prompt card" }).click();
  await page
    .getByLabel("Comment", { exact: true })
    .fill("second independent comment");
  await page.getByRole("button", { name: "Add card & save" }).click();
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await expect(page.locator(".prompt-card")).toHaveCount(2);
  await page.getByLabel("Graph name").fill("Browser graph");
  await page
    .getByPlaceholder("What would you like your agent to do?")
    .fill("Check HTTP between both cards");
  await expect(page.getByLabel("Graph name")).toHaveValue("Browser graph");
  await expect(
    page.getByText("Unsaved · 2 cards", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /Browser graph.*2 cards/ }),
  ).toBeVisible();
  await page.reload();
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await expect(page.locator(".prompt-card")).toHaveCount(2);
  await expect(page.getByLabel("Graph name")).toHaveValue("Browser graph");
  await page.screenshot({
    path: "test-results/prompt-graph.png",
    fullPage: true,
  });
});
test("free cards and failed copy keep the current graph until manual completion", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await page.getByRole("button", { name: "+ New", exact: true }).click();
  await page.getByLabel("Graph name").fill("Copy history");
  await page.getByRole("button", { name: "+ Text card" }).click();
  await page.getByLabel("Title", { exact: true }).fill("A standalone note");
  await page
    .getByLabel("Comment", { exact: true })
    .fill("Preserve this free text");
  await page
    .getByPlaceholder("What would you like your agent to do?")
    .fill("Inspect only");
  await expect(page.getByText("Saved · 1 cards")).toBeVisible();
  await page.evaluate(() =>
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async () => {
          throw Error("test denial");
        },
      },
    }),
  );
  await page.getByRole("button", { name: "Copy & new graph" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect(page.locator(".markdown")).toContainText(
    "Preserve this free text",
  );
  await expect(page.locator(".prompt-card")).toHaveCount(1);
  await page.getByRole("button", { name: "I have copied it" }).click();
  await expect(page.locator(".prompt-card")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: /Copy history/ }),
  ).toBeVisible();
});
test("unsupported network operations are absent and mobile layout remains usable", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Network$/ }).click();
  await expect(
    page.getByRole("button", { name: /enable|disable|edit route/i }),
  ).toHaveCount(0);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "+ Create route" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.screenshot({
    path: "test-results/mobile-network.png",
    fullPage: true,
  });
});
test("undirected connections preserve title and comment in Markdown", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await page.getByRole("button", { name: "+ New", exact: true }).click();
  await page.getByRole("button", { name: "+ Text card" }).click();
  await page.getByLabel("Title", { exact: true }).fill("Left card");
  await page.getByLabel("Comment", { exact: true }).fill("Left context");
  await page.getByRole("button", { name: "Duplicate", exact: true }).click();
  const second = page.locator(".react-flow__node").nth(1);
  const box = await second.boundingBox();
  if (!box) throw Error("Missing second card");
  await page.mouse.move(box.x + 80, box.y + 25);
  await page.mouse.down();
  await page.mouse.move(box.x + 280, box.y + 160, { steps: 12 });
  await page.mouse.up();
  await page.getByRole("button", { name: "Fit View", exact: true }).click();
  const source = page.locator(".react-flow__handle-right").first();
  const target = page.locator(".react-flow__handle-left").nth(1);
  await source.dragTo(target);
  await expect(
    page.getByRole("heading", { name: "Edit connection" }),
  ).toBeVisible();
  await page.getByLabel("Title", { exact: true }).fill("Communicate");
  await page
    .getByLabel("Comment", { exact: true })
    .fill("Preserve the undirected relation");
  await page.evaluate(() =>
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async () => {
          throw Error("test denial");
        },
      },
    }),
  );
  await page
    .getByRole("button", { name: "Copy Markdown", exact: true })
    .click();
  await expect(page.locator(".markdown")).toContainText(
    "Preserve the undirected relation",
  );
  await expect(page.locator(".markdown")).toContainText(" -- ");
});
test("a competing saved revision surfaces a recoverable conflict", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await page.getByRole("button", { name: "+ New", exact: true }).click();
  await page.getByLabel("Graph name").fill("Conflict fixture");
  await expect(
    page.getByRole("button", { name: /Conflict fixture.*revision 1/ }),
  ).toBeVisible();
  const status = await (await page.request.get("/api/status")).json();
  const list = await (await page.request.get("/api/graphs")).json();
  const current = list.graphs.find((g: any) => g.name === "Conflict fixture");
  const { id, version, name, prompt, nodes, edges } = current;
  await page.request.put("/api/graphs/" + id, {
    headers: { Origin: "http://127.0.0.1:18555", "X-CSRF-Token": status.csrf },
    data: { version, name, prompt: "Other device", nodes, edges },
  });
  await page
    .getByPlaceholder("What would you like your agent to do?")
    .fill("Keep local edits");
  await expect(page.getByText("Conflict · local edits retained")).toBeVisible();
  await expect(
    page.getByPlaceholder("What would you like your agent to do?"),
  ).toHaveValue("Keep local edits");
  await page
    .getByRole("button", { name: "Save local edits as a new graph" })
    .click();
  await expect(
    page.getByRole("button", {
      name: /Conflict fixture \(recovered\).*revision 1/,
    }),
  ).toBeVisible();
});
test("successful clipboard copy switches to an empty draft and keeps the old graph", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await page.getByRole("button", { name: "+ New", exact: true }).click();
  await page.getByLabel("Graph name").fill("Successful copy history");
  await page.getByRole("button", { name: "+ Text card" }).click();
  await page
    .getByLabel("Comment", { exact: true })
    .fill("Copied content remains saved");
  await page.evaluate(() =>
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async (text: string) => {
          (window as any).__copied = text;
        },
      },
    }),
  );
  await page.getByRole("button", { name: "Copy & new graph" }).click();
  await expect(page.locator(".prompt-card")).toHaveCount(0);
  await expect(page.getByLabel("Graph name")).toHaveValue("");
  expect(await page.evaluate(() => (window as any).__copied)).toContain(
    "Copied content remains saved",
  );
  await expect(
    page.getByRole("button", { name: /Successful copy history.*1 cards/ }),
  ).toBeVisible();
});
