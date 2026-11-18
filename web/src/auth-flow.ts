export function isMfaRequiredError(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  return error.message === "mfa required";
}
