import { apiFetch } from "./api";

export interface TwoFactorStatus {
  enabled: boolean;
  /** Office roles must keep it on. */
  required: boolean;
  backupCodesLeft: number;
  trustedDevices: number;
}

export interface TwoFactorSetup {
  secret: string;
  uri: string;
  /** PNG data URL of the enrolment QR code. */
  qr: string;
}

export function getTwoFactorStatus(): Promise<TwoFactorStatus> {
  return apiFetch<TwoFactorStatus>("/api/v1/auth/2fa");
}

export function beginTwoFactorSetup(): Promise<TwoFactorSetup> {
  return apiFetch<TwoFactorSetup>("/api/v1/auth/2fa/setup", { method: "POST" });
}

/** Turns 2FA on; returns the one-time backup codes. */
export async function enableTwoFactor(code: string): Promise<string[]> {
  const r = await apiFetch<{ backupCodes: string[] }>("/api/v1/auth/2fa/enable", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
  return r.backupCodes;
}

export async function regenerateBackupCodes(code: string): Promise<string[]> {
  const r = await apiFetch<{ backupCodes: string[] }>("/api/v1/auth/2fa/backup-codes", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
  return r.backupCodes;
}

export async function disableTwoFactor(code: string): Promise<void> {
  await apiFetch<void>("/api/v1/auth/2fa/disable", { method: "POST", body: JSON.stringify({ code }) });
}

export async function forgetTrustedDevices(): Promise<void> {
  await apiFetch<void>("/api/v1/auth/2fa/forget-devices", { method: "POST" });
}

/** Super admin: clear a user's 2FA after a lost phone. */
export async function resetUserTwoFactor(userId: number): Promise<void> {
  await apiFetch<void>(`/api/v1/auth/2fa/reset/${userId}`, { method: "POST" });
}
