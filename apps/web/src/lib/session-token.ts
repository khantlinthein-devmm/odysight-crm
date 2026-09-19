import { createHmac, timingSafeEqual } from "node:crypto";

// Mirrors the claims the Go API enforces in internal/auth/middleware.go.
// The API and the web container share one JWT_SECRET, so the signature alone
// does not prove a token was minted for staff: a customer-portal token
// (audience odysight-portal) is signed with the very same key. Issuer and
// audience are what separate the two, and both must be checked here.
export const TOKEN_ISSUER = "odysight-crm";
export const STAFF_AUDIENCE = "odysight-web";

export interface SessionClaims {
  sub?: string;
  exp?: number;
  iss?: string;
  aud?: string | string[];
  role?: string;
}

interface TokenHeader {
  alg?: string;
  typ?: string;
}

function base64UrlDecode(input: string): Buffer {
  const pad = input.length % 4 === 0 ? "" : "=".repeat(4 - (input.length % 4));
  return Buffer.from(
    input.replace(/-/g, "+").replace(/_/g, "/") + pad,
    "base64",
  );
}

function audienceMatches(aud: SessionClaims["aud"], expected: string): boolean {
  if (typeof aud === "string") return aud === expected;
  if (Array.isArray(aud)) return aud.includes(expected);
  return false;
}

/**
 * Verifies an HS256 staff session token: signature, expiry, issuer and
 * audience. Returns the claims, or null when the token fails any check.
 */
export function verifySessionToken(
  token: string,
  secret: string | undefined,
): SessionClaims | null {
  if (!secret) return null;
  const parts = token.split(".");
  if (parts.length !== 3) return null;

  let header: TokenHeader;
  try {
    header = JSON.parse(
      base64UrlDecode(parts[0]!).toString("utf8"),
    ) as TokenHeader;
  } catch {
    return null;
  }
  if (header.alg !== "HS256") return null;

  const data = `${parts[0]}.${parts[1]}`;
  const signature = base64UrlDecode(parts[2]!);
  const expected = createHmac("sha256", secret).update(data).digest();
  if (
    signature.length !== expected.length ||
    !timingSafeEqual(signature, expected)
  ) {
    return null;
  }

  let payload: SessionClaims;
  try {
    payload = JSON.parse(
      base64UrlDecode(parts[1]!).toString("utf8"),
    ) as SessionClaims;
  } catch {
    return null;
  }

  if (typeof payload.exp !== "number" || payload.exp * 1000 <= Date.now()) {
    return null;
  }
  if (payload.iss !== TOKEN_ISSUER) return null;
  if (!audienceMatches(payload.aud, STAFF_AUDIENCE)) return null;

  return payload;
}
