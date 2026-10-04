import { expect, test } from "@playwright/test";
import { grant, newUser } from "./helpers";

test("team admin adds a member who can view but not manage the team", async ({ browser, baseURL }) => {
  const owner = await newUser(browser, baseURL, "Owner");
  const member = await newUser(browser, baseURL, "Member");
  const name = `Team ${Date.now()}`;
  const p = owner.page;

  await p.goto("/teams");
  await p.getByRole("button", { name: "New team" }).click();
  await p.getByLabel("Name", { exact: true }).fill(name);
  await p.getByRole("button", { name: "Create" }).click();
  await p.getByRole("link", { name }).click();
  await expect(p.getByRole("heading", { name })).toBeVisible();

  await grant(p, p.getByRole("main"), member.email, "member");

  const m = member.page;
  await m.goto("/teams");
  await m.getByRole("link", { name }).click();
  await expect(m.getByText("Only team admins can see and change membership.")).toBeVisible();
  await expect(m.getByRole("button", { name: "Edit" })).toBeHidden();
  await expect(m.getByRole("button", { name: "Delete" })).toBeHidden();

  await p.getByRole("button", { name: "Delete" }).click();
  await p.getByRole("alertdialog").getByRole("button", { name: "Delete" }).click();
  await expect(p).toHaveURL(/\/teams$/);

  await m.goto("/teams");
  await expect(m.getByRole("link", { name })).toBeHidden();

  await owner.context.close();
  await member.context.close();
});
