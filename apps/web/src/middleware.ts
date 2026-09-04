import { defineMiddleware } from "astro:middleware";

const PUBLIC_PATHS = new Set(["/login"]);

export const onRequest = defineMiddleware((context, next) => {
  const { cookies, redirect, request } = context;
  const pathname = new URL(request.url).pathname;

  const isPublic = PUBLIC_PATHS.has(pathname);
  const hasSession = cookies.has("odysight_session");

  if (!hasSession && !isPublic) {
    return redirect(`/login?next=${encodeURIComponent(pathname)}`);
  }

  if (hasSession && isPublic) {
    return redirect("/dashboard");
  }

  return next();
});
