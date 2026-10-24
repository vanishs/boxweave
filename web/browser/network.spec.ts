import { test, expect } from "@playwright/test";
test.beforeEach(async ({ page }) => {
  await page.goto("/");
  await page
    .getByLabel("Administrator password")
    .fill("browser test password only");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(
    page.getByRole("heading", { name: "Overview", exact: true }),
  ).toBeVisible();
});
async function tenantIds(page: any) {
  const d = await (await page.request.get("/api/tenants")).json();
  return {
    allinone: d.tenants.find((t: any) => t.providerKey === "allinone").id,
    testing: d.tenants.find((t: any) => t.providerKey === "testing").id,
  };
}
test("A: three Boxes use one native Tenant and separate verification stages", async ({
  page,
}) => {
  const ids = await tenantIds(page);
  await page.getByRole("button", { name: /Boxes$/ }).click();
  await page
    .getByLabel("Tenant scope", { exact: true })
    .selectOption(ids.allinone);
  await page
    .getByRole("button", { name: "+ Create boxes", exact: true })
    .click();
  await expect(page.getByLabel("Create Tenant", { exact: true })).toHaveValue(
    ids.allinone,
  );
  await page.getByLabel("Box names").fill("box1\nbox2\nbox3");
  await expect(
    page.getByRole("button", { name: /Review creation plan/ }),
  ).toBeEnabled();
  await page.getByRole("button", { name: /Review creation plan/ }).click();
  await expect(page.getByRole("dialog")).toContainText(
    '"networkTenant": "allinone"',
  );
  await page
    .getByRole("button", { name: "Confirm create", exact: true })
    .click();
  await expect(page.locator(".receipts summary").first()).toContainText(
    "completed",
  );
  await page.locator(".receipts summary").first().click();
  await page
    .getByRole("button", { name: "Query actual creation state" })
    .click();
  await expect(page.getByText("guard-verified", { exact: true })).toHaveCount(
    3,
  );
  const boxes = await (
    await page.request.get("/api/boxes?tenant=" + ids.allinone)
  ).json();
  expect(
    boxes.containers
      .filter((b: any) => ["box1", "box2", "box3"].includes(b.username))
      .every((b: any) => b.networkTenant === "allinone" && b.tenant.confirmed),
  ).toBe(true);
  await expect(page.getByText(/Network connectivity: not tested/)).toHaveCount(
    3,
  );
});
test("new Tenant can be created and retained in both configuration modes", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Boxes$/ }).click();
  await page
    .getByRole("button", { name: "+ Create boxes", exact: true })
    .click();
  await page
    .getByLabel("Create Tenant", { exact: true })
    .selectOption("__new__");
  await page.getByLabel("New Tenant name").fill("Created from unified");
  await page.getByRole("button", { name: "Create Tenant and select" }).click();
  await expect(
    page.getByLabel("Create Tenant", { exact: true }).locator("option:checked"),
  ).toHaveText("Created from unified");
  await page.getByLabel("Box names").fill("unified-future");
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await page
    .getByLabel("Tenant row 1", { exact: true })
    .selectOption("__new__");
  await page.getByLabel("New Tenant name").fill("Created from row");
  await page.getByRole("button", { name: "Create Tenant and select" }).click();
  await expect(
    page.getByLabel("Tenant row 1", { exact: true }).locator("option:checked"),
  ).toHaveText("Created from row");
  await page
    .getByRole("button", { name: "Unified configuration", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await expect(
    page.getByLabel("Tenant row 1", { exact: true }).locator("option:checked"),
  ).toHaveText("Created from row");
});
test("B: review single/both directions and revoke preserves all other policy", async ({
  page,
}) => {
  const ids = await tenantIds(page);
  await page.getByRole("button", { name: /Network$/ }).click();
  await page
    .getByLabel("Source Tenant", { exact: true })
    .selectOption(ids.allinone);
  await page
    .getByLabel("Destination Tenant", { exact: true })
    .selectOption(ids.testing);
  const initial = await (await page.request.get("/api/tenant-access")).json();
  const preserved = initial.policies.find((p: any) => p.tenant === "testing");
  await page
    .getByRole("button", { name: "Review policy changes", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Review Tenant policy" });
  await expect(dialog).toContainText("Destination policy: testing");
  expect(
    (await (await page.request.get("/api/tenant-access")).json()).relations,
  ).toHaveLength(0);
  await dialog
    .getByRole("button", { name: "Confirm Tenant policy grant" })
    .click();
  await expect(page.getByRole("status")).toContainText("configured");
  let current = await (await page.request.get("/api/tenant-access")).json();
  expect(current.relations).toHaveLength(1);
  expect(current.relations[0].source.providerKey).toBe("allinone");
  expect(current.relations[0].destination.providerKey).toBe("testing");
  await page.getByLabel("Access direction").selectOption("both");
  await page
    .getByRole("button", { name: "Review policy changes", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "Confirm Tenant policy grant" })
    .click();
  await expect
    .poll(
      async () =>
        (await (await page.request.get("/api/tenant-access")).json()).relations
          .length,
    )
    .toBe(2);
  await page.getByLabel("Access operation").selectOption("revoke");
  await page
    .getByRole("button", { name: "Review policy changes", exact: true })
    .click();
  await expect(dialog).toContainText("After: none");
  await dialog
    .getByRole("button", { name: "Confirm Tenant policy revoke" })
    .click();
  await expect
    .poll(
      async () =>
        (await (await page.request.get("/api/tenant-access")).json()).relations
          .length,
    )
    .toBe(0);
  current = await (await page.request.get("/api/tenant-access")).json();
  const after = current.policies.find((p: any) => p.tenant === "testing");
  delete after.allowFromTenants;
  expect(after).toEqual(preserved);
  await page.screenshot({
    path: "test-results/tenant-connectivity.png",
    fullPage: true,
  });
});
test("access Prompt Card preserves source/destination without changing permissions", async ({
  page,
}) => {
  const ids = await tenantIds(page);
  await page.getByRole("button", { name: /Network$/ }).click();
  await page
    .getByLabel("Source Tenant", { exact: true })
    .selectOption(ids.allinone);
  await page
    .getByLabel("Destination Tenant", { exact: true })
    .selectOption(ids.testing);
  await page
    .getByRole("button", { name: "Review policy changes", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm Tenant policy grant" })
    .click();
  await expect(
    page.getByRole("button", { name: "+ Access prompt card", exact: true }),
  ).toBeVisible();
  const before = await (await page.request.get("/api/tenant-access")).json();
  await page
    .getByRole("button", { name: "+ Access prompt card", exact: true })
    .click();
  await page
    .getByLabel("Comment", { exact: true })
    .fill("Preserve this explicit direction");
  await page.getByRole("button", { name: "Add card & save" }).click();
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await expect(page.locator(".prompt-card")).toHaveCount(1);
  await page.evaluate(() =>
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: async (text: string) => {
          (window as any).__copied = text;
        },
      },
    }),
  );
  await page
    .getByRole("button", { name: "Copy Markdown", exact: true })
    .click();
  await expect
    .poll(() => page.evaluate(() => (window as any).__copied))
    .toContain("allinone -> testing");
  const md = await page.evaluate(() => (window as any).__copied);
  expect(md).toContain("TCP not tested");
  expect(md).toContain("Preserve this explicit direction");
  expect(
    (await (await page.request.get("/api/tenant-access")).json()).policies,
  ).toEqual(before.policies);
});
