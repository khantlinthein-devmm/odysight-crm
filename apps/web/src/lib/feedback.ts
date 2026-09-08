import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export interface Feedback {
  id: number;
  bookingId: number;
  bookingNumber: string;
  customerId: number | null;
  customerName: string;
  rating: number;
  comment: string;
  createdAt: string;
}

export interface CreateFeedbackInput {
  bookingId: number;
  rating: number;
  comment: string;
}

export interface UpdateFeedbackInput {
  rating?: number;
  comment?: string;
}

export interface ListParams {
  search?: string;
  limit?: number;
  offset?: number;
}

let mockId = 90;

const mockFeedback: Feedback[] = [
  {
    id: 81,
    bookingId: 204,
    bookingNumber: "BK-2026-0204",
    customerId: 104,
    customerName: "Maria Gomez",
    rating: 5,
    comment: "Beautiful job, kitchen sparkles!",
    createdAt: "2026-08-11T08:00:00Z",
  },
  {
    id: 82,
    bookingId: 201,
    bookingNumber: "BK-2026-0201",
    customerId: 101,
    customerName: "Somchai Prasert",
    rating: 4,
    comment: "Great service, a little late.",
    createdAt: "2026-08-19T09:12:00Z",
  },
];

function clone(f: Feedback): Feedback {
  return { ...f };
}

function filterMocks(params: ListParams): Feedback[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  const rows = mockFeedback.filter((f) => {
    if (!q) return true;
    return (
      f.customerName.toLowerCase().includes(q) ||
      f.bookingNumber.toLowerCase().includes(q) ||
      f.comment.toLowerCase().includes(q)
    );
  });
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  return rows.slice(offset, offset + limit).map(clone);
}

export async function getFeedback(params: ListParams = {}): Promise<Feedback[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Feedback[] | Page<Feedback>>(
    "/api/v1/feedback" + toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}

export async function getFeedbackItem(id: number): Promise<Feedback> {
  if (USE_MOCKS) {
    await delay(200);
    const item = mockFeedback.find((f) => f.id === id);
    if (!item) throw new ApiError(404, `Feedback ${id} not found`);
    return clone(item);
  }
  return apiFetch<Feedback>(`/api/v1/feedback/${id}`);
}

export async function createFeedback(input: CreateFeedbackInput): Promise<Feedback> {
  if (USE_MOCKS) {
    await delay(400);
    const item: Feedback = {
      ...input,
      id: ++mockId,
      bookingNumber: "BK-2026-0999",
      customerId: null,
      customerName: "Unknown",
      createdAt: new Date().toISOString(),
    };
    mockFeedback.unshift(item);
    return clone(item);
  }
  return apiFetch<Feedback>("/api/v1/feedback", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateFeedback(id: number, input: UpdateFeedbackInput): Promise<Feedback> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockFeedback.findIndex((f) => f.id === id);
    if (index === -1) throw new ApiError(404, `Feedback ${id} not found`);
    const item = { ...mockFeedback[index]!, ...input };
    mockFeedback[index] = item;
    return clone(item);
  }
  return apiFetch<Feedback>(`/api/v1/feedback/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteFeedback(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockFeedback.findIndex((f) => f.id === id);
    if (index === -1) throw new ApiError(404, `Feedback ${id} not found`);
    mockFeedback.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/feedback/${id}`, { method: "DELETE" });
}

export function ratingStars(rating: number): string {
  return "★".repeat(Math.max(0, Math.min(5, rating))) +
    "☆".repeat(Math.max(0, 5 - rating));
}