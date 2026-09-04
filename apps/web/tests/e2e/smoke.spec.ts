import { expect, test } from "@playwright/test";

async function signIn(page: import("@playwright/test").Page) {
  await page.goto("/login");
  await page.fill("#email", "admin@example.com");
  await page.fill("#password", "admin123");
  await page.click("#login-submit");
  await page.waitForURL("**/dashboard");
}

test("unauthenticated users are redirected to login", async ({ page }) => {
  await page.goto("/dashboard");
  await expect(page).toHaveURL(/\/login/);
});

test("sign in lands on dashboard and shows user identity", async ({ page }) => {
  await signIn(page);
  await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
  await expect(page.locator("#user-name")).toHaveText("Admin User");
});

test("leads table renders and supports create + delete with confirmation", async ({
  page,
}) => {
  await signIn(page);
  await page.goto("/leads");

  const rows = page.locator("tbody tr");
  await expect(rows.first()).toBeVisible();

  await page.getByRole("button", { name: "New Lead" }).click();
  await page.fill("#firstName", "E2E");
  await page.fill("#lastName", "Tester");
  await page.fill("#email", "e2e.tester@example.com");
  await page.getByRole("button", { name: "Create lead" }).click();

  const newRow = page.locator("tbody tr", {
    hasText: "e2e.tester@example.com",
  });
  await expect(newRow).toBeVisible();

  await newRow.getByRole("button", { name: "Delete" }).click();
  await expect(page.getByRole("alertdialog")).toBeVisible();
  await page.getByRole("button", { name: "Delete lead" }).click();
  await expect(
    page.locator("tbody tr", { hasText: "e2e.tester@example.com" }),
  ).toHaveCount(0);
});

test("command palette opens with Ctrl+K and navigates", async ({ page }) => {
  await signIn(page);
  await page.waitForLoadState("networkidle");
  await page.keyboard.press("Control+k");
  const input = page.getByPlaceholder("Type a command or search...");
  await expect(input).toBeVisible();
  await input.fill("payments");
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/payments$/);
});

test("sign out returns to login", async ({ page }) => {
  await signIn(page);
  await page.click("#logout-btn");
  await expect(page).toHaveURL(/\/login/);
});
