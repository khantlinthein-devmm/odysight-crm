import { apiFetch } from "./api";
import { persistSettingsSnapshot, readSettingsSnapshot } from "./offline";

export interface CompanySettings {
  name: string;
  phone: string;
  address: string;
  invoiceFooter: string;
  logoUrl: string;
  /** Registered legal name printed on tax invoices. */
  legalName: string;
  /** 13-digit Thai tax ID. */
  taxId: string;
  /** Branch code; "00000" = head office. */
  taxBranch: string;
  vatRegistered: boolean;
}

export interface LocalizationSettings {
  timezone: string;
  dateFormat: string;
  language: string;
  currency: string;
}

export interface BookingSettings {
  defaultDurationMinutes: number;
  bufferMinutes: number;
  workStart: string;
  workEnd: string;
  holidays: string[];
  /** Cleaners must be this close to a job site to check in (0 = off). */
  checkInRadiusMeters: number;
}

export interface PaymentSettings {
  taxRatePercent: number;
  methods: string[];
  /** PromptPay mobile / tax ID / e-wallet ID; empty = no QR on invoices. */
  promptPayId: string;
  /** Free-text bank details printed on invoices. */
  bankAccount: string;
}

export interface ServiceItem {
  id: string;
  name: string;
  durationMinutes: number;
  basePrice: number;
  pricePerSqm: number;
  active: boolean;
}

export interface NotificationSettings {
  newLeadEmail: boolean;
  bookingChangeEmail: boolean;
  paymentFailEmail: boolean;
  overdueReminderEmail: boolean;
  overdueReminderDays: number;
  recipients: string[];
}

export interface SmtpSettings {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  password: string;
  fromEmail: string;
  fromName: string;
  encryption: "none" | "starttls" | "ssl";
}

export interface SmsSettings {
  enabled: boolean;
  provider: string;
  webhookUrl: string;
  apiKey: string;
  fromNumber: string;
}

export interface WorkspaceSettings {
  company: CompanySettings;
  localization: LocalizationSettings;
  booking: BookingSettings;
  payments: PaymentSettings;
  services: ServiceItem[];
  notifications: NotificationSettings;
  smtp: SmtpSettings;
  sms: SmsSettings;
  whatsapp: SmsSettings;
}

export const DEFAULT_SETTINGS: WorkspaceSettings = {
  company: {
    name: "Smile Clean Thailand",
    phone: "",
    address: "",
    invoiceFooter: "Thank you for your business!",
    logoUrl: "",
    legalName: "",
    taxId: "",
    taxBranch: "00000",
    vatRegistered: false,
  },
  localization: {
    timezone: "Asia/Bangkok",
    dateFormat: "DD/MM/YYYY",
    language: "en",
    currency: "THB",
  },
  booking: {
    defaultDurationMinutes: 120,
    bufferMinutes: 30,
    workStart: "08:00",
    workEnd: "18:00",
    holidays: [],
    checkInRadiusMeters: 0,
  },
  payments: {
    promptPayId: "",
    bankAccount: "",
    taxRatePercent: 7,
    methods: [
      "cash",
      "bank_transfer",
      "promptpay",
      "credit_card",
      "line_pay",
      "online_wallet",
    ],
  },
  services: [
    { id: "house_cleaning", name: "House Cleaning", durationMinutes: 180, basePrice: 1500, pricePerSqm: 25, active: true },
    { id: "condo_cleaning", name: "Condo Cleaning", durationMinutes: 120, basePrice: 1200, pricePerSqm: 25, active: true },
    { id: "deep_cleaning", name: "Deep Cleaning", durationMinutes: 240, basePrice: 2500, pricePerSqm: 35, active: true },
    { id: "move_in_out", name: "Move In/Out", durationMinutes: 300, basePrice: 3000, pricePerSqm: 35, active: true },
    { id: "after_renovation", name: "After Renovation", durationMinutes: 240, basePrice: 2800, pricePerSqm: 40, active: true },
    { id: "office_cleaning", name: "Office Cleaning", durationMinutes: 210, basePrice: 2200, pricePerSqm: 30, active: true },
    { id: "junk_removal", name: "Junk Removal", durationMinutes: 120, basePrice: 1000, pricePerSqm: 0, active: true },
    { id: "aircon_service", name: "Aircon Service", durationMinutes: 90, basePrice: 800, pricePerSqm: 0, active: true },
  ],
  notifications: {
    newLeadEmail: true,
    bookingChangeEmail: true,
    paymentFailEmail: true,
    overdueReminderEmail: true,
    overdueReminderDays: 7,
    recipients: [],
  },
  smtp: {
    enabled: false,
    host: "",
    port: 587,
    username: "",
    password: "",
    fromEmail: "",
    fromName: "",
    encryption: "starttls",
  },
  sms: {
    enabled: false,
    provider: "",
    webhookUrl: "",
    apiKey: "",
    fromNumber: "",
  },
  whatsapp: {
    enabled: false,
    provider: "",
    webhookUrl: "",
    apiKey: "",
    fromNumber: "",
  },
};

export type SettingsKey = keyof WorkspaceSettings;

let cache: WorkspaceSettings | null = null;
let inflight: Promise<WorkspaceSettings> | null = null;

function merge(all: Partial<WorkspaceSettings>): WorkspaceSettings {
  const fallbackRate = (id: string): number =>
    DEFAULT_SETTINGS.services.find((s) => s.id === id)?.pricePerSqm ?? 0;
  const services = (all.services ?? DEFAULT_SETTINGS.services).map((s) => ({
    ...s,
    pricePerSqm:
      typeof s.pricePerSqm === "number" && Number.isFinite(s.pricePerSqm)
        ? s.pricePerSqm
        : fallbackRate(s.id),
  }));
  return {
    company: { ...DEFAULT_SETTINGS.company, ...all.company },
    localization: { ...DEFAULT_SETTINGS.localization, ...all.localization },
    booking: { ...DEFAULT_SETTINGS.booking, ...all.booking },
    payments: { ...DEFAULT_SETTINGS.payments, ...all.payments },
    services,
    notifications: { ...DEFAULT_SETTINGS.notifications, ...all.notifications },
    smtp: { ...DEFAULT_SETTINGS.smtp, ...all.smtp },
    sms: { ...DEFAULT_SETTINGS.sms, ...all.sms },
    whatsapp: { ...DEFAULT_SETTINGS.whatsapp, ...all.whatsapp },
  };
}

export async function getWorkspaceSettings(
  force = false,
): Promise<WorkspaceSettings> {
  if (cache && !force) return cache;
  if (inflight && !force) return inflight;
  inflight = (async () => {
    try {
      const all = await apiFetch<Partial<WorkspaceSettings>>("/api/v1/settings");
      // Persist every successful fetch so the calculator (and other offline
      // surfaces) keep real catalog prices and tax rates with no signal.
      persistSettingsSnapshot(all);
      cache = merge(all);
    } catch {
      // Offline: last-synced snapshot first, built-in defaults last.
      const snapshot = readSettingsSnapshot<WorkspaceSettings>();
      cache = merge(snapshot ?? {});
    }
    return cache;
  })();
  try {
    return await inflight;
  } finally {
    inflight = null;
  }
}

export async function updateWorkspaceSettings(
  patch: Partial<WorkspaceSettings>,
): Promise<WorkspaceSettings> {
  const all = await apiFetch<Partial<WorkspaceSettings>>("/api/v1/settings", {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
  cache = merge(all);
  return cache;
}

export function currencyCode(): string {
  return cache?.localization.currency ?? DEFAULT_SETTINGS.localization.currency;
}

export function formatMoney(amount: number, currency?: string): string {
  const code = currency ?? currencyCode();
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: code,
      maximumFractionDigits: 0,
    }).format(amount);
  } catch {
    return `${code} ${amount.toLocaleString()}`;
  }
}

const FALLBACK_SERVICE_LABELS: Record<string, string> = {
  house_cleaning: "House Cleaning",
  condo_cleaning: "Condo Cleaning",
  deep_cleaning: "Deep Cleaning",
  move_in_out: "Move In/Out",
  after_renovation: "After Renovation",
  office_cleaning: "Office Cleaning",
  junk_removal: "Junk Removal",
  aircon_service: "Aircon Service",
};

export function serviceLabel(id: string): string {
  const found = cache?.services.find((s) => s.id === id);
  if (found) return found.name;
  return FALLBACK_SERVICE_LABELS[id] ?? id;
}

export function activeServices(): ServiceItem[] {
  const list = cache?.services ?? DEFAULT_SETTINGS.services;
  return list.filter((s) => s.active);
}

const FALLBACK_METHOD_LABELS: Record<string, string> = {
  cash: "Cash",
  bank_transfer: "Bank Transfer",
  promptpay: "PromptPay",
  credit_card: "Credit Card",
  line_pay: "LINE Pay",
  online_wallet: "Online Wallet",
};

export function paymentMethodLabel(id: string): string {
  return (
    FALLBACK_METHOD_LABELS[id] ??
    id.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())
  );
}

export function activePaymentMethods(): string[] {
  return cache?.payments.methods ?? DEFAULT_SETTINGS.payments.methods;
}
