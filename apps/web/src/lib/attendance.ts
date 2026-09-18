import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export interface AttendanceRecord {
  id: number;
  cleanerId: number;
  cleanerName: string;
  workDate: string;
  checkInAt: string | null;
  checkOutAt: string | null;
  note: string;
  createdAt: string;
}

export interface ListParams {
  cleaner?: number;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

function toDateStr(d: Date): string {
  return d.toISOString().slice(0, 10);
}

function dayOffset(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  return toDateStr(d);
}

function atTime(workDate: string, time: string): string {
  return `${workDate}T${time}:00`;
}

let mockId = 900;

const mockAttendance: AttendanceRecord[] = [
  {
    id: 801,
    cleanerId: 601,
    cleanerName: "Nok Srisuwan",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "08:30"),
    checkOutAt: null,
    note: "",
    createdAt: atTime(dayOffset(0), "08:30"),
  },
  {
    id: 802,
    cleanerId: 602,
    cleanerName: "Pim Jiraroj",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "08:15"),
    checkOutAt: atTime(dayOffset(0), "17:05"),
    note: "",
    createdAt: atTime(dayOffset(0), "08:15"),
  },
  {
    id: 803,
    cleanerId: 604,
    cleanerName: "Mali Kongdee",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "09:02"),
    checkOutAt: null,
    note: "Late arrival — traffic",
    createdAt: atTime(dayOffset(0), "09:02"),
  },
  {
    id: 804,
    cleanerId: 603,
    cleanerName: "Daeng Chaiya",
    workDate: dayOffset(-1),
    checkInAt: atTime(dayOffset(-1), "08:45"),
    checkOutAt: atTime(dayOffset(-1), "17:10"),
    note: "",
    createdAt: atTime(dayOffset(-1), "08:45"),
  },
  {
    id: 805,
    cleanerId: 601,
    cleanerName: "Nok Srisuwan",
    workDate: dayOffset(-1),
    checkInAt: atTime(dayOffset(-1), "08:20"),
    checkOutAt: atTime(dayOffset(-1), "16:55"),
    note: "",
    createdAt: atTime(dayOffset(-1), "08:20"),
  },
  {
    id: 806,
    cleanerId: 605,
    cleanerName: "Som Intarakamhaeng",
    workDate: dayOffset(-2),
    checkInAt: atTime(dayOffset(-2), "08:35"),
    checkOutAt: atTime(dayOffset(-2), "17:00"),
    note: "",
    createdAt: atTime(dayOffset(-2), "08:35"),
  },
  {
    id: 807,
    cleanerId: 602,
    cleanerName: "Pim Jiraroj",
    workDate: dayOffset(-2),
    checkInAt: atTime(dayOffset(-2), "08:10"),
    checkOutAt: atTime(dayOffset(-2), "17:15"),
    note: "",
    createdAt: atTime(dayOffset(-2), "08:10"),
  },
];

const mockCleanerNames: Record<number, string> = {
  601: "Nok Srisuwan",
  602: "Pim Jiraroj",
  603: "Daeng Chaiya",
  604: "Mali Kongdee",
  605: "Som Intarakamhaeng",
};

function clone(record: AttendanceRecord): AttendanceRecord {
  return { ...record };
}

function filterMocks(params: ListParams): AttendanceRecord[] {
  let rows = mockAttendance.filter((r) => {
    if (params.cleaner !== undefined && r.cleanerId !== params.cleaner) return false;
    if (params.from && r.workDate < params.from) return false;
    if (params.to && r.workDate > params.to) return false;
    return true;
  });
  rows = [...rows].sort((a, b) =>
    a.workDate === b.workDate
      ? b.id - a.id
      : b.workDate.localeCompare(a.workDate),
  );
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  rows = rows.slice(offset, offset + limit);
  return rows.map(clone);
}

export async function getAttendance(params: ListParams = {}): Promise<AttendanceRecord[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<AttendanceRecord[] | Page<AttendanceRecord>>(
    "/api/v1/attendance" + toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}

export async function checkIn(cleanerId: number): Promise<AttendanceRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const today = dayOffset(0);
    const existing = mockAttendance.find(
      (r) => r.cleanerId === cleanerId && r.workDate === today,
    );
    if (existing?.checkInAt) {
      throw new ApiError(409, "Already checked in for today");
    }
    const now = new Date().toISOString();
    if (existing) {
      existing.checkInAt = now;
      return clone(existing);
    }
    const record: AttendanceRecord = {
      id: ++mockId,
      cleanerId,
      cleanerName: mockCleanerNames[cleanerId] ?? `Cleaner ${cleanerId}`,
      workDate: today,
      checkInAt: now,
      checkOutAt: null,
      note: "",
      createdAt: now,
    };
    mockAttendance.unshift(record);
    return clone(record);
  }
  return apiFetch<AttendanceRecord>("/api/v1/attendance/check-in", {
    method: "POST",
    body: JSON.stringify({ cleanerId }),
  });
}

export async function checkOut(cleanerId: number): Promise<AttendanceRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const today = dayOffset(0);
    const existing = mockAttendance.find(
      (r) => r.cleanerId === cleanerId && r.workDate === today,
    );
    if (!existing?.checkInAt) {
      throw new ApiError(409, "Not checked in yet");
    }
    if (existing.checkOutAt) {
      throw new ApiError(409, "Already checked out for today");
    }
    existing.checkOutAt = new Date().toISOString();
    return clone(existing);
  }
  return apiFetch<AttendanceRecord>("/api/v1/attendance/check-out", {
    method: "POST",
    body: JSON.stringify({ cleanerId }),
  });
}
