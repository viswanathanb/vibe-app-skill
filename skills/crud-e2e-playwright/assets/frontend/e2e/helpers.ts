import { expect, type Browser, type Locator, type Page } from "@playwright/test";

export const ADMIN_STATE = "e2e/.auth/admin.json";
export const PASSWORD = "e2e-password-123";
export const NO_SESSION = { cookies: [], origins: [] };

let counter = 0;

/** Unique per run, so tests never collide with data from earlier tests. */
export function uniqueEmail(prefix: string): string {
  counter += 1;
  return `${prefix.toLowerCase()}-${Date.now()}-${counter}@example.com`;
}

export async function signup(page: Page, name: string, email = uniqueEmail(name)): Promise<string> {
  await page.goto("/signup");
  await page.getByLabel("Name", { exact: true }).fill(name);
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("heading", { name: /Welcome/ })).toBeVisible();
  return email;
}

/** Signs up a new member in its own browser context (separate cookies = separate user). */
export async function newUser(browser: Browser, baseURL: string | undefined, name: string) {
  const context = await browser.newContext({ baseURL, storageState: NO_SESSION });
  const page = await context.newPage();
  const email = await signup(page, name);
  return { context, page, email };
}

/** Grants `relation` on the open detail page's object to `who` (email or subject) via the Share dialog. */
export async function share(page: Page, who: string, relation: string) {
  await page.getByRole("button", { name: "Share" }).click();
  const dialog = page.getByRole("dialog");
  await grant(page, dialog, who, relation);
  await page.keyboard.press("Escape");
}

/** Fills an AccessPanel (inside `scope`) and waits for the new entry. */
export async function grant(page: Page, scope: Locator, who: string, relation: string) {
  await scope.getByPlaceholder(/email@example\.com/).fill(who);
  await scope.getByRole("combobox").click();
  await page.getByRole("option", { name: relation, exact: true }).click();
  await scope.getByRole("button", { name: "Add" }).click();
  await expect(scope.getByText(who)).toBeVisible();
}
