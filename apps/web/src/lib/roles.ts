import type { Role } from "./auth";

// Mirrors apps/api/internal/auth/rbac.go — keep in sync when permissions change.
export const ROLE_ORDER: Role[] = [
  "SUPER_ADMIN",
  "ADMIN",
  "MANAGER",
  "DISPATCH",
  "ACCOUNTANT",
  "CLEANER",
];

export interface PermissionGroup {
  resource: string;
  permissions: { key: string; label: string; roles: Role[] }[];
}

export const PERMISSION_MATRIX: PermissionGroup[] = [
  {
    resource: "Leads",
    permissions: [
      { key: "leads.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT"] },
      { key: "leads.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "leads.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
      { key: "leads.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Customers",
    permissions: [
      { key: "customers.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT", "CLEANER"] },
      { key: "customers.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "customers.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "customers.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Bookings",
    permissions: [
      { key: "bookings.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT", "CLEANER"] },
      { key: "bookings.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
      { key: "bookings.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "CLEANER"] },
      { key: "bookings.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Cleaners",
    permissions: [
      { key: "cleaners.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT"] },
      { key: "cleaners.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "cleaners.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
      { key: "cleaners.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Attendance",
    permissions: [
      { key: "attendance.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT"] },
      { key: "attendance.manage", label: "Check in / out", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
    ],
  },
  {
    resource: "Service records",
    permissions: [
      { key: "service_records.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT", "CLEANER"] },
      { key: "service_records.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "CLEANER"] },
      { key: "service_records.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "CLEANER"] },
      { key: "service_records.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Payments",
    permissions: [
      { key: "payments.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
      { key: "payments.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
      { key: "payments.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
    ],
  },
  {
    resource: "Invoices",
    permissions: [
      { key: "invoices.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
      { key: "invoices.create", label: "Create from booking", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
      { key: "invoices.update", label: "Mark paid / void", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
    ],
  },
  {
    resource: "Expenses",
    permissions: [
      { key: "expenses.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
      { key: "expenses.manage", label: "Add / edit / delete", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
    ],
  },
  {
    resource: "Payroll",
    permissions: [
      { key: "payroll.read", label: "View cleaner pay", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
      { key: "payroll.manage", label: "Set pay rates", roles: ["SUPER_ADMIN", "ADMIN", "ACCOUNTANT"] },
    ],
  },
  {
    resource: "Complaints",
    permissions: [
      { key: "complaints.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT"] },
      { key: "complaints.manage", label: "Log / resolve / book re-clean", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
    ],
  },
  {
    resource: "Supplies",
    permissions: [
      { key: "supplies.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT"] },
      { key: "supplies.manage", label: "Catalog / record usage", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"] },
    ],
  },
  {
    resource: "Reports",
    permissions: [
      { key: "reports.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
    ],
  },
  {
    resource: "Feedback",
    permissions: [
      { key: "feedback.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "feedback.create", label: "Add", roles: ["SUPER_ADMIN", "ADMIN"] },
      { key: "feedback.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN"] },
      { key: "feedback.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Notifications",
    permissions: [
      { key: "notifications.read", label: "View log", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
    ],
  },
  {
    resource: "Sites",
    permissions: [
      { key: "sites.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "CLEANER"] },
      { key: "sites.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "sites.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "sites.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Contracts",
    permissions: [
      { key: "contracts.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
      { key: "contracts.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN"] },
      { key: "contracts.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN"] },
      { key: "contracts.delete", label: "Delete", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Quotes",
    permissions: [
      { key: "quotes.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "ACCOUNTANT"] },
      { key: "quotes.create", label: "Create", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "quotes.update", label: "Edit", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "quotes.approve", label: "Approve", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
  {
    resource: "Checklists",
    permissions: [
      { key: "checklists.read", label: "View", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT", "CLEANER"] },
      { key: "checklists.manage", label: "Manage / complete", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "CLEANER"] },
    ],
  },
  {
    resource: "Team & audit",
    permissions: [
      { key: "users.read", label: "View team", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER"] },
      { key: "users.manage", label: "Manage team", roles: ["SUPER_ADMIN"] },
      { key: "audit.read", label: "View audit log", roles: ["SUPER_ADMIN"] },
    ],
  },
  {
    resource: "Workspace settings",
    permissions: [
      { key: "settings.read", label: "View settings", roles: ["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH", "ACCOUNTANT", "CLEANER"] },
      { key: "settings.manage", label: "Edit settings", roles: ["SUPER_ADMIN", "ADMIN"] },
    ],
  },
];

// Grants a single permission key to any role mapped to it in PERMISSION_MATRIX.
export function hasPermission(role: Role | undefined, key: string): boolean {
  if (!role) return false;
  return PERMISSION_MATRIX.some((group) =>
    group.permissions.some(
      (p) => p.key === key && p.roles.includes(role),
    ),
  );
}

// Collection of permission keys for a role, e.g. { "bookings.create": true }.
export function permissionsFor(role: Role | undefined): Record<string, boolean> {
  const out: Record<string, boolean> = {};
  if (!role) return out;
  for (const group of PERMISSION_MATRIX) {
    for (const p of group.permissions) {
      if (p.roles.includes(role)) out[p.key] = true;
    }
  }
  return out;
}
