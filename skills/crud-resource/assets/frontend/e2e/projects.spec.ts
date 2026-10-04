import { expect, test } from "@playwright/test";
import { newUser, share } from "./helpers";

// Generated per resource by add-resource.sh. When you add required fields to the form, fill them below.
test("owner creates a project, shares it read-only, edits and deletes it", async ({ browser, baseURL }) => {
  const owner = await newUser(browser, baseURL, "Owner");
  const other = await newUser(browser, baseURL, "Other");
  const name = `Project ${Date.now()}`;
  const p = owner.page;

  await p.goto("/projects");
  await p.getByRole("button", { name: "New project" }).click();
  await p.getByLabel("Name", { exact: true }).fill(name);
  await p.getByRole("button", { name: "Create" }).click();
  await expect(p.getByText("Project created")).toBeVisible();
  await p.getByRole("link", { name }).click();
  await expect(p).toHaveURL(/\/projects\/\d+$/);
  const url = p.url();

  // Not shared yet: invisible in lists and forbidden by URL.
  const o = other.page;
  await o.goto("/projects");
  await expect(o.getByRole("link", { name })).toBeHidden();
  await o.goto(url);
  await expect(o.getByText(/do not have view access/)).toBeVisible();

  // Shared as viewer: visible, but no edit/delete/share.
  await share(p, other.email, "viewer");
  await o.goto("/projects");
  await o.getByRole("link", { name }).click();
  await expect(o.getByRole("heading", { name })).toBeVisible();
  await expect(o.getByRole("button", { name: "Edit" })).toBeHidden();
  await expect(o.getByRole("button", { name: "Delete" })).toBeHidden();
  await expect(o.getByRole("button", { name: "Share" })).toBeHidden();

  // Owner edits, then deletes; the viewer loses access.
  await p.getByRole("button", { name: "Edit" }).click();
  await p.getByLabel("Name", { exact: true }).fill(`${name} v2`);
  await p.getByRole("button", { name: "Save" }).click();
  await expect(p.getByRole("heading", { name: `${name} v2` })).toBeVisible();

  await p.getByRole("button", { name: "Delete" }).click();
  await p.getByRole("alertdialog").getByRole("button", { name: "Delete" }).click();
  await expect(p).toHaveURL(/\/projects$/);
  await o.goto(url);
  await expect(o.getByText(/do not have view access/)).toBeVisible();

  await owner.context.close();
  await other.context.close();
});
