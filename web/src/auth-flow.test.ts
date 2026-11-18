import { describe, expect, it } from "vitest";
import { isMfaRequiredError } from "./auth-flow";

describe("auth-flow", () => {
  it("detects the MFA required response from the backend", () => {
    expect(isMfaRequiredError(new Error("mfa required"))).toBe(true);
    expect(isMfaRequiredError(new Error("invalid credentials"))).toBe(false);
    expect(isMfaRequiredError(new Error("invalid mfa code"))).toBe(false);
  });
});
