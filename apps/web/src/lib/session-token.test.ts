import { createHmac } from "node:crypto";
import { describe, expect, it } from "vitest";
import { verifySessionToken } from "./session-token";

const SECRET = "test-secret-at-least-32-characters-long";

function b64url(input: string | Buffer): string {
  return Buffer.from(input)
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

function sign(
  payload: Record<string, unknown>,
  secret = SECRET,
  header: Record<string, unknown> = { alg: "HS256", typ: "JWT" },
): string {
  const data = `${b64url(JSON.stringify(header))}.${b64url(JSON.stringify(payload))}`;
  const sig = createHmac("sha256", secret).update(data).digest();
  return `${data}.${b64url(sig)}`;
}

const future = () => Math.floor(Date.now() / 1000) + 3600;
const past = () => Math.floor(Date.now() / 1000) - 3600;

const staffClaims = {
  sub: "1",
  role: "SUPER_ADMIN",
  iss: "odysight-crm",
  aud: "odysight-web",
  exp: future(),
};

describe("verifySessionToken", () => {
  it("accepts a staff token", () => {
    const claims = verifySessionToken(sign(staffClaims), SECRET);
    expect(claims).not.toBeNull();
    expect(claims?.sub).toBe("1");
    expect(claims?.role).toBe("SUPER_ADMIN");
  });

  it("rejects a customer-portal token signed with the same secret", () => {
    // The API and web container share one JWT_SECRET, so this token's
    // signature is valid — only the audience separates it from a staff token.
    const portalToken = sign({
      sub: "42",
      iss: "odysight-crm",
      aud: "odysight-portal",
      exp: future(),
    });
    expect(verifySessionToken(portalToken, SECRET)).toBeNull();
  });

  it("rejects a token from another issuer", () => {
    expect(
      verifySessionToken(sign({ ...staffClaims, iss: "someone-else" }), SECRET),
    ).toBeNull();
  });

  it("rejects a token with no audience or issuer claim", () => {
    expect(
      verifySessionToken(sign({ sub: "1", exp: future() }), SECRET),
    ).toBeNull();
  });

  it("accepts an audience array that includes the staff audience", () => {
    const token = sign({
      ...staffClaims,
      aud: ["odysight-portal", "odysight-web"],
    });
    expect(verifySessionToken(token, SECRET)).not.toBeNull();
  });

  it("rejects an audience array without the staff audience", () => {
    const token = sign({ ...staffClaims, aud: ["odysight-portal"] });
    expect(verifySessionToken(token, SECRET)).toBeNull();
  });

  it("rejects an expired token", () => {
    expect(
      verifySessionToken(sign({ ...staffClaims, exp: past() }), SECRET),
    ).toBeNull();
  });

  it("rejects a token with no expiry", () => {
    const { exp: _exp, ...noExp } = staffClaims;
    expect(verifySessionToken(sign(noExp), SECRET)).toBeNull();
  });

  it("rejects a token signed with a different secret", () => {
    expect(
      verifySessionToken(sign(staffClaims, "another-secret-entirely"), SECRET),
    ).toBeNull();
  });

  it("rejects a tampered payload", () => {
    const token = sign(staffClaims);
    const [header, , signature] = token.split(".");
    const forged = b64url(JSON.stringify({ ...staffClaims, sub: "999" }));
    expect(
      verifySessionToken(`${header}.${forged}.${signature}`, SECRET),
    ).toBeNull();
  });

  it("rejects alg=none", () => {
    const header = b64url(JSON.stringify({ alg: "none", typ: "JWT" }));
    const payload = b64url(JSON.stringify(staffClaims));
    expect(verifySessionToken(`${header}.${payload}.`, SECRET)).toBeNull();
  });

  it("rejects a malformed token or missing secret", () => {
    expect(verifySessionToken("not-a-jwt", SECRET)).toBeNull();
    expect(verifySessionToken(sign(staffClaims), undefined)).toBeNull();
    expect(verifySessionToken("", SECRET)).toBeNull();
  });
});
