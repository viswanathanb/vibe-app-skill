import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api, toQuery } from "./api";

describe("toQuery", () => {
  it("skips empty values", () => {
    expect(toQuery({ limit: 20, offset: 0, q: "", sort: undefined })).toBe("?limit=20&offset=0");
    expect(toQuery({})).toBe("");
  });
});

describe("api", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends the CSRF header and parses JSON", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 1 }), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(api.post("/things", { name: "x" })).resolves.toEqual({ id: 1 });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/things");
    expect(init.headers["X-Requested-With"]).toBe("XMLHttpRequest");
  });

  it("throws ApiError with server details", async () => {
    const body = { error: { code: "bad_request", message: "validation failed", details: { name: "required" } } };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(body), { status: 400 })));

    const err = await api.post("/things", {}).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 400, code: "bad_request", details: { name: "required" } });
  });
});
