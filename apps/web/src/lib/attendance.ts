import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

// Attendance is kept for two workforces: field cleaners and office/team staff.
export type PersonType = "cleaner" | "staff";

export interface AttendanceRecord {
  id: number;
  personType: PersonType;
  personId: number;
  personName: string;
  workDate: string;
  checkInAt: string | null;
  checkOutAt: string | null;
  note: string;
  createdAt: string;
}

export interface ListParams {
  type?: PersonType;
  person?: number;
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
    personType: "cleaner",
    personId: 601,
    personName: "Nok Srisuwan",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "08:30"),
    checkOutAt: null,
    note: "",
    createdAt: atTime(dayOffset(0), "08:30"),
  },
  {
    id: 802,
    personType: "cleaner",
    personId: 602,
    personName: "Pim Jiraroj",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "08:15"),
    checkOutAt: atTime(dayOffset(0), "17:05"),
    note: "",
    createdAt: atTime(dayOffset(0), "08:15"),
  },
  {
    id: 803,
    personType: "cleaner",
    personId: 604,
    personName: "Mali Kongdee",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "09:02"),
    checkOutAt: null,
    note: "Late arrival — traffic",
    createdAt: atTime(dayOffset(0), "09:02"),
  },
  {
    id: 804,
    personType: "cleaner",
    personId: 603,
    personName: "Daeng Chaiya",
    workDate: dayOffset(-1),
    checkInAt: atTime(dayOffset(-1), "08:45"),
    checkOutAt: atTime(dayOffset(-1), "17:10"),
    note: "",
    createdAt: atTime(dayOffset(-1), "08:45"),
  },
  {
    id: 805,
    personType: "cleaner",
    personId: 601,
    personName: "Nok Srisuwan",
    workDate: dayOffset(-1),
    checkInAt: atTime(dayOffset(-1), "08:20"),
    checkOutAt: atTime(dayOffset(-1), "16:55"),
    note: "",
    createdAt: atTime(dayOffset(-1), "08:20"),
  },
  {
    id: 806,
    personType: "staff",
    personId: 1,
    personName: "Admin User",
    workDate: dayOffset(0),
    checkInAt: atTime(dayOffset(0), "08:55"),
    checkOutAt: null,
    note: "",
    createdAt: atTime(dayOffset(0), "08:55"),
  },
  {
    id: 807,
    personType: "staff",
    personId: 2,
    personName: "Dispatcher",
    workDate: dayOffset(-1),
    checkInAt: atTime(dayOffset(-1), "08:40"),
    checkOutAt: atTime(dayOffset(-1), "17:30"),
    note: "",
    createdAt: atTime(dayOffset(-1), "08:40"),
  },
];

function clone(record: AttendanceRecord): AttendanceRecord {
  return { ...record };
}

function filterMocks(params: ListParams): AttendanceRecord[] {
  let rows = mockAttendance.filter((r) => {
    if (params.type !== undefined && r.personType !== params.type) return false;
    if (params.person !== undefined && r.personId !== params.person) return false;
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

function mockPerson(personType: PersonType, personId: number): string {
  const known = mockAttendance.find(
    (r) => r.personType === personType && r.personId === personId,
  );
  if (known) return known.personName;
  return personType === "staff" ? `Staff ${personId}` : `Cleaner ${personId}`;
}

export async function checkIn(
  personType: PersonType,
  personId: number,
): Promise<AttendanceRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const today = dayOffset(0);
    const existing = mockAttendance.find(
      (r) => r.personType === personType && r.personId === personId && r.workDate === today,
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
      personType,
      personId,
      personName: mockPerson(personType, personId),
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
    body: JSON.stringify({ personType, personId }),
  });
}

export async function checkOut(
  personType: PersonType,
  personId: number,
): Promise<AttendanceRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const today = dayOffset(0);
    const existing = mockAttendance.find(
      (r) => r.personType === personType && r.personId === personId && r.workDate === today,
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
    body: JSON.stringify({ personType, personId }),
  });
}
