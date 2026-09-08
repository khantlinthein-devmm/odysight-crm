import { defineMiddleware } from "astro:middleware";
import { createHmac, timingSafeEqual } from "node:crypto";

const PUBLIC_PATHS = new Set(["/", "/login", "/signup", "/portal"]);
const AUTH_PATHS = new Set(["/login", "/signup"]);

function isStaticAsset(pathname: string): boolean {
  return (
    pathname.startsWith("/_astro/") ||
    pathname.startsWith("/assets/") ||
    pathname === "/favicon.svg" ||
    pathname === "/favicon.ico" ||
    pathname === "/robots.txt" ||
    /\.[a-z0-9]+$/i.test(pathname)
  );
}

function base64UrlDecode(input: string): Buffer {
  const pad = input.length % 4 === 0 ? "" : "=".repeat(4 - (input.length % 4));
  return Buffer.from(
    input.replace(/-/g, "+").replace(/_/g, "/") + pad,
    "base64",
  );
}

interface TokenClaims {
  sub?: string;
  exp?: number;
}

// Verifies the HS256 JWT signature and expiry using the shared JWT_SECRET.
// Mirrors the API's signing scheme (iss=odysight-crm, aud=odysight-web).
function verifyJWT(token: string): TokenClaims | null {
  const secret = process.env.JWT_SECRET;
  if (!secret) return null;
  const parts = token.split(".");
  if (parts.length !== 3) return null;

  const data = `${parts[0]}.${parts[1]}`;
  const signature = base64UrlDecode(parts[2]!);
  const expected = createHmac("sha256", secret).update(data).digest();
  if (
    signature.length !== expected.length ||
    !timingSafeEqual(signature, expected)
  ) {
    return null;
  }

  try {
    const payload = JSON.parse(
      base64UrlDecode(parts[1]!).toString("utf8"),
    ) as TokenClaims;
    if (
      typeof payload.exp !== "number" ||
      payload.exp * 1000 <= Date.now()
    ) {
      return null;
    }
    return payload;
  } catch {
    return null;
  }
}

export const onRequest = defineMiddleware((context, next) => {
  const { cookies, redirect, request } = context;
  const rawPathname = new URL(request.url).pathname;
  // Normalize trailing slash (except root) so /login/ and /signup/ match.
  const pathname =
    rawPathname.length > 1 && rawPathname.endsWith("/")
      ? rawPathname.slice(0, -1)
      : rawPathname;

  if (isStaticAsset(pathname)) {
    return next();
  }

  const isPublic = PUBLIC_PATHS.has(pathname);
  const session = cookies.get("odysight_session");
  let hasValidSession = false;
  if (session?.value) {
    hasValidSession =
      // Dev-only mock tokens (never valid in production builds).
      import.meta.env.DEV && session.value.startsWith("mock-token-")
        ? true
        : verifyJWT(session.value) !== null;
  }

  if (!hasValidSession && session) {
    // Clear an invalid or expired session cookie.
    cookies.delete("odysight_session");
  }

  if (!hasValidSession && !isPublic) {
    return redirect(`/login?next=${encodeURIComponent(pathname)}`);
  }

  if (hasValidSession && AUTH_PATHS.has(pathname)) {
    return redirect("/dashboard");
  }

  return next();
});