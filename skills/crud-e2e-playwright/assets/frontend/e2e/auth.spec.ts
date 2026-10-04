import { expect, test } from "@playwright/test";
import { NO_SESSION, PASSWORD, signup } from "./helpers";

test.describe("authentication", () => {
  test.use({ storageState: NO_SESSION });

  test("anonymous visitors are sent to the login page", async ({ page }) => {
    await page.goto("/teams");
    await expect(page).toHaveURL(/\/login$/);
  });

  test("member signs up, signs out and signs back in", async ({ page }) => {
    const email = await signup(page, "Member");
    await expect(page.getByRole("link", { name: "Users" })).toBeHidden();

    await page.getByRole("button", { name: "Member" }).click();
    await page.getByRole("menuitem", { name: "Sign out" }).click();
    await expect(page).toHaveURL(/\/login$/);

    await page.getByLabel("Email", { exact: true }).fill(email);
    await page.getByLabel("Password", { exact: true }).fill(PASSWORD);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("heading", { name: /Welcome/ })).toBeVisible();
  });

  test("wrong password is rejected", async ({ page }) => {
    const email = await signup(page, "Forgetful");
    await page.context().clearCookies();
    await page.goto("/login");
    await page.getByLabel("Email", { exact: true }).fill(email);
    await page.getByLabel("Password", { exact: true }).fill("not-the-password");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByText(/invalid email or password/i)).toBeVisible();
    await expect(page).toHaveURL(/\/login$/);
  });
});

test("admin manages users; members cannot", async ({ page, browser, baseURL }) => {
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
  await expect(page.getByRole("cell", { name: /admin-.*@example\.com/ })).toBeVisible();

  const context = await browser.newContext({ baseURL, storageState: NO_SESSION });
  const member = await context.newPage();
  await signup(member, "Curious");
  await member.goto("/admin/users");
  await expect(member).toHaveURL(/\/$/);
  await context.close();
});
