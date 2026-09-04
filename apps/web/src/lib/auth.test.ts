import { beforeEach, describe, expect, it } from "vitest";
import { getSessionUser, login, logout } from "./auth";

describe("auth", () => {
  beforeEach(() => {
    localStorage.clear();
    document.cookie = "odysight_session=; path=/; max-age=0";
  });

  it("logs in with valid mock credentials", async () => {
    const user = await login("admin@example.com", "admin123");

    expect(user.role).toBe("SUPER_ADMIN");
    expect(getSessionUser()?.email).toBe("admin@example.com");
    expect(document.cookie).toContain("odysight_session=");
  });

  it("rejects invalid credentials", async () => {
    await expect(login("admin@example.com", "wrong")).rejects.toThrow(
      /invalid email or password/i,
    );
    expect(getSessionUser()).toBeNull();
  });

  it("logout clears the session", async () => {
    await login("dispatch@example.com", "dispatch123");
    expect(getSessionUser()).not.toBeNull();
    logout();
    expect(getSessionUser()).toBeNull();
    expect(document.cookie).not.toContain("odysight_session=mock-token");
  });
});
