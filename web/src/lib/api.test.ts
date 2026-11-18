import { describe, expect, it, vi, afterEach } from "vitest";
import { api } from "./api";

afterEach(() => vi.restoreAllMocks());

describe("api client", () => {
  it("sends login data and returns the session", async () => {
    const session = { access_token: "access", refresh_token: "refresh", expires_at: "tomorrow" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(session), { status: 200 })));

    await expect(api.login("ana@example.com", "password", "123456")).resolves.toEqual(session);
    expect(fetch).toHaveBeenCalledWith("/api/auth/login", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ email: "ana@example.com", password: "password", mfa_code: "123456" }),
    }));
  });

  it("translates API errors into an exception", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: "invalid credentials" }), { status: 401 })));

    await expect(api.login("ana@example.com", "wrong")).rejects.toThrow("invalid credentials");
  });
});
