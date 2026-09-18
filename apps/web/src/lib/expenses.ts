import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export interface Expense {
  id: number;
  spentOn: string;
  category: string;
  amount: number;
  note: string;
  createdBy: number;
  createdAt: string;
}

export interface ExpenseListParams {
  from?: string;
  to?: string;
  category?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export type CreateExpenseInput = Pick<Expense, "spentOn" | "category" | "amount" | "note">;

export type UpdateExpenseInput = Partial<Pick<Expense, "spentOn" | "category" | "amount" | "note">>;

let mockId = 700;

const mockExpenses: Expense[] = [
  {
    id: 701,
    spentOn: "2026-09-12",
    category: "Supplies",
    amount: 2450,
    note: "Cleaning chemicals restock",
    createdBy: 1,
    createdAt: "2026-09-12T10:00:00Z",
  },
  {
    id: 702,
    spentOn: "2026-09-05",
    category: "Transport",
    amount: 1800,
    note: "Fuel for team vans",
    createdBy: 1,
    createdAt: "2026-09-05T09:30:00Z",
  },
  {
    id: 703,
    spentOn: "2026-08-22",
    category: "Salary",
    amount: 45000,
    note: "August payroll — cleaners",
    createdBy: 1,
    createdAt: "2026-08-22T08:00:00Z",
  },
  {
    id: 704,
    spentOn: "2026-08-15",
    category: "Rent",
    amount: 12000,
    note: "Office rent August",
    createdBy: 1,
    createdAt: "2026-08-15T08:00:00Z",
  },
  {
    id: 705,
    spentOn: "2026-07-28",
    category: "Equipment",
    amount: 8900,
    note: "Vacuum cleaner replacement",
    createdBy: 1,
    createdAt: "2026-07-28T11:00:00Z",
  },
  {
    id: 706,
    spentOn: "2026-07-10",
    category: "Utilities",
    amount: 1600,
    note: "Electricity + water",
    createdBy: 1,
    createdAt: "2026-07-10T09:00:00Z",
  },
  {
    id: 707,
    spentOn: "2026-09-02",
    category: "Food",
    amount: 950,
    note: "Team lunch",
    createdBy: 1,
    createdAt: "2026-09-02T12:00:00Z",
  },
];

function clone(expense: Expense): Expense {
  return { ...expense };
}

function filterMocks(params: ExpenseListParams): Expense[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  const cat = params.category?.trim().toLowerCase() ?? "";
  let rows = mockExpenses.filter((e) => {
    if (params.from && e.spentOn < params.from) return false;
    if (params.to && e.spentOn > params.to) return false;
    if (cat && e.category.toLowerCase() !== cat) return false;
    if (!q) return true;
    return `${e.category} ${e.note}`.toLowerCase().includes(q);
  });
  rows = [...rows].sort((a, b) => (a.spentOn < b.spentOn ? 1 : -1));
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  rows = rows.slice(offset, offset + limit);
  return rows.map(clone);
}

export async function getExpenses(params: ExpenseListParams = {}): Promise<Expense[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Expense[] | Page<Expense>>("/api/v1/expenses" + toQuery(params as Record<string, string | number | undefined>));
  return unwrapPage(json);
}

export async function createExpense(input: CreateExpenseInput): Promise<Expense> {
  if (USE_MOCKS) {
    await delay(400);
    const expense: Expense = {
      ...input,
      id: ++mockId,
      createdBy: 1,
      createdAt: new Date().toISOString(),
    };
    mockExpenses.unshift(expense);
    return clone(expense);
  }
  return apiFetch<Expense>("/api/v1/expenses", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateExpense(id: number, input: UpdateExpenseInput): Promise<Expense> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockExpenses.findIndex((e) => e.id === id);
    if (index === -1) throw new ApiError(404, `Expense ${id} not found`);
    const expense = { ...mockExpenses[index]!, ...input };
    mockExpenses[index] = expense;
    return clone(expense);
  }
  return apiFetch<Expense>(`/api/v1/expenses/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteExpense(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockExpenses.findIndex((e) => e.id === id);
    if (index === -1) throw new ApiError(404, `Expense ${id} not found`);
    mockExpenses.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/expenses/${id}`, { method: "DELETE" });
}
