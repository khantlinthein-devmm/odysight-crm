import { defineMiddleware } from "astro:middleware";
import { verifySessionToken } from "./lib/session-token";

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
  let needsTwoFactorSetup = false;
  if (session?.value) {
    if (import.meta.env.DEV && session.value.startsWith("mock-token-")) {
      // Dev-only mock tokens (never valid in production builds).
      hasValidSession = true;
    } else {
      const claims = verifySessionToken(session.value, process.env.JWT_SECRET);
      hasValidSession = claims !== null;
      needsTwoFactorSetup = claims?.mfa_setup === true;
    }
  }

  if (!hasValidSession && session) {
    // Clear an invalid or expired session cookie.
    cookies.delete("odysight_session");
  }

  if (!hasValidSession && !isPublic) {
    return redirect(`/login?next=${encodeURIComponent(pathname)}`);
  }

  // Office roles must finish 2FA setup before using anything else.
  if (hasValidSession && needsTwoFactorSetup && pathname !== "/setup-2fa") {
    return redirect("/setup-2fa");
  }

  if (hasValidSession && AUTH_PATHS.has(pathname)) {
    return redirect("/dashboard");
  }

  return next();
});