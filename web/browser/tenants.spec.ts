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
test("metadata CRUD, stable identity, empty status and persistence", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Tenants$/ }).click();
  await page
    .getByRole("button", { name: "+ Create Tenant", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Tenant metadata" });
  await dialog.getByLabel("Name", { exact: true }).fill("Browser empty Tenant");
  await dialog.getByLabel("Description").fill("Display metadata, no network");
  await dialog.getByRole("button", { name: "Save Tenant" }).click();
  const row = page.getByRole("row").filter({
    has: page.getByRole("button", {
      name: "Browser empty Tenant",
      exact: true,
    }),
  });
  await expect(row).toContainText("empty");
  const id = await row.locator(".stable-id").textContent();
  await row
    .getByRole("button", { name: "Browser empty Tenant", exact: true })
    .click();
  await page.getByRole("button", { name: "Edit display metadata" }).click();
  await dialog
    .getByLabel("Name", { exact: true })
    .fill("Browser renamed Tenant");
  await dialog.getByRole("button", { name: "Save Tenant" }).click();
  await page.reload();
  await page.getByRole("button", { name: /Tenants$/ }).click();
  const renamed = page.getByRole("row").filter({
    has: page.getByRole("button", {
      name: "Browser renamed Tenant",
      exact: true,
    }),
  });
  await expect(renamed.locator(".stable-id")).toHaveText(id!);
  await renamed
    .getByRole("button", { name: "Browser renamed Tenant", exact: true })
    .click();
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "Delete empty metadata" }).click();
  await expect(
    page.getByRole("button", { name: "Browser renamed Tenant", exact: true }),
  ).toHaveCount(0);
});
test("scopes filter server resources and creation never fakes a shared Tenant", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Boxes$/ }).click();
  const select = page.getByLabel("Tenant scope", { exact: true });
  const alpha = await select
    .locator("option")
    .filter({ hasText: /^alpha$/ })
    .getAttribute("value");
  await select.selectOption(alpha!);
  await expect(
    page.getByRole("button", { name: "alpha", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "beta", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "+ Create boxes", exact: true })
    .click();
  await page.getByLabel("Box names").fill("tenant-gate-one\ntenant-gate-two");
  await expect(
    page.getByRole("button", { name: /Review creation plan/ }),
  ).toBeDisabled();
  await expect(page.getByRole("dialog")).toContainText(
    "Shared network Tenant creation is unavailable",
  );
  await page.getByLabel("Create Tenant", { exact: true }).selectOption("");
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await page.getByLabel("Tenant row 2", { exact: true }).selectOption(alpha!);
  await page
    .getByRole("button", { name: "Unified configuration", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Individual configuration", exact: true })
    .click();
  await expect(page.getByLabel("Tenant row 2", { exact: true })).toHaveValue(
    alpha!,
  );
  await expect(
    page.getByRole("button", { name: /Review creation plan/ }),
  ).toBeDisabled();
  await page.getByLabel("Tenant row 2", { exact: true }).selectOption("");
  await page.getByRole("button", { name: /Review creation plan/ }).click();
  await expect(page.getByRole("dialog")).toContainText(
    "expected_name_fallback",
  );
  await expect(page.getByRole("dialog")).toContainText("pending");
});
test("Tenant detail context and Box cards preserve mixed Tenant Markdown", async ({
  page,
}) => {
  await page.getByRole("button", { name: /Tenants$/ }).click();
  await page.getByRole("button", { name: "alpha", exact: true }).click();
  await page
    .getByRole("button", { name: "Prompt Context", exact: true })
    .click();
  await page
    .getByRole("button", { name: "+ Tenant prompt card", exact: true })
    .click();
  await page
    .getByLabel("Comment", { exact: true })
    .fill("Tenant context retained");
  await page.getByRole("button", { name: "Add card & save" }).click();
  await page.getByRole("button", { name: /Boxes$/ }).click();
  await page.getByLabel("Tenant scope", { exact: true }).selectOption("");
  await page.getByLabel("Search resources").fill("beta");
  await page
    .getByRole("button", { name: "+ Prompt card", exact: true })
    .click();
  await page.getByRole("button", { name: "Add card & save" }).click();
  await page.getByRole("button", { name: /Prompt Graph$/ }).click();
  await expect(
    page.locator(".prompt-card").filter({ hasText: "Tenant: alpha" }),
  ).toBeVisible();
  await expect(
    page.locator(".prompt-card").filter({ hasText: "Tenant: beta" }),
  ).toBeVisible();
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
    .toBeTruthy();
  const md = await page.evaluate(() => (window as any).__copied);
  expect(md).toContain("[Tenant: alpha]");
  expect(md).toContain("[Tenant: beta]");
  expect(md).toContain("Tenant context retained");
  await page.getByRole("button", { name: /Network$/ }).click();
  await expect(
    page.getByRole("heading", { name: "Provider Guard evidence" }),
  ).toBeVisible();
  await expect(
    page.getByText(/Bidirectional TCP connectivity has not been tested/),
  ).toBeVisible();
  await page.getByRole("button", { name: /Tenants$/ }).click();
  await page.screenshot({
    path: "test-results/tenant-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: /Tenants$/ }).click();
  await page.screenshot({
    path: "test-results/mobile-tenants.png",
    fullPage: true,
  });
});
