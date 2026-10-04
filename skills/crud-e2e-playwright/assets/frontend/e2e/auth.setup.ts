import { expect, test as setup } from "@playwright/test";
import { ADMIN_STATE, signup } from "./helpers";

setup("first account becomes admin", async ({ page }) => {
  await signup(page, "Admin");
  await expect(
    page.getByRole("link", { name: "Users" }),
    "first signup must be admin; is the app_e2e database fresh? run `task e2e`",
  ).toBeVisible();
  await page.context().storageState({ path: ADMIN_STATE });
});
