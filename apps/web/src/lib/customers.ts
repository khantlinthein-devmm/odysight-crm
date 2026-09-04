import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type CustomerStatus = "active" | "inactive" | "blocked";

export type PropertyType =
  | "house"
  | "condo"
  | "office"
  | "apartment"
  | "other";

export interface Customer {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  address: string;
  propertyType: PropertyType;
  area: string;
  status: CustomerStatus;
  createdAt: string;
}

export type CreateCustomerInput = Omit<Customer, "id" | "createdAt">;

export type UpdateCustomerInput = Partial<CreateCustomerInput>;

let mockId = 200;

const mockCustomers: Customer[] = [
  {
    id: 101,
    firstName: "Somchai",
    lastName: "Prasert",
    email: "somchai.prasert@example.com",
    phone: "+66 91 234 5678",
    address: "Sukhumvit 38, Klongton, Bangkok",
    propertyType: "condo",
    area: "Sukhumvit",
    status: "active",
    createdAt: "2026-08-19T10:00:00Z",
  },
  {
    id: 102,
    firstName: "Jane",
    lastName: "Smith",
    email: "jane.smith@example.com",
    phone: "+66 81 555-0102",
    address: "On Nut 10, Suan Luang, Bangkok",
    propertyType: "house",
    area: "On Nut",
    status: "active",
    createdAt: "2026-08-17T09:30:00Z",
  },
  {
    id: 103,
    firstName: "Omar",
    lastName: "Farouk",
    email: "omar.farouk@example.com",
    phone: "+66 92 555-0103",
    address: "Thonglor 13, Wattana, Bangkok",
    propertyType: "apartment",
    area: "Thonglor",
    status: "active",
    createdAt: "2026-08-14T15:20:00Z",
  },
  {
    id: 104,
    firstName: "Maria",
    lastName: "Gomez",
    email: "maria.gomez@example.com",
    phone: "+66 84 555-0104",
    address: "Asok 4, Khlong Toei, Bangkok",
    propertyType: "condo",
    area: "Asok",
    status: "active",
    createdAt: "2026-08-11T12:00:00Z",
  },
  {
    id: 105,
    firstName: "Wei",
    lastName: "Chen",
    email: "wei.chen@example.com",
    phone: "+66 93 555-0105",
    address: "Sathorn 12, Sathorn, Bangkok",
    propertyType: "office",
    area: "Sathorn",
    status: "inactive",
    createdAt: "2026-08-09T08:45:00Z",
  },
  {
    id: 106,
    firstName: "Aisha",
    lastName: "Khan",
    email: "aisha.khan@example.com",
    phone: "+66 85 555-0106",
    address: "Rama 9 23, Huai Kwang, Bangkok",
    propertyType: "house",
    area: "Rama 9",
    status: "active",
    createdAt: "2026-08-07T17:10:00Z",
  },
];

function clone(customer: Customer): Customer {
  return { ...customer };
}

export async function getCustomers(): Promise<Customer[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockCustomers.map(clone);
  }
  return apiFetch<Customer[]>("/api/v1/customers");
}

export async function getCustomer(id: number): Promise<Customer> {
  if (USE_MOCKS) {
    await delay(200);
    const customer = mockCustomers.find((c) => c.id === id);
    if (!customer) throw new ApiError(404, `Customer ${id} not found`);
    return clone(customer);
  }
  return apiFetch<Customer>(`/api/v1/customers/${id}`);
}

export async function createCustomer(
  input: CreateCustomerInput,
): Promise<Customer> {
  if (USE_MOCKS) {
    await delay(400);
    const customer: Customer = {
      ...input,
      id: ++mockId,
      createdAt: new Date().toISOString(),
    };
    mockCustomers.unshift(customer);
    return clone(customer);
  }
  return apiFetch<Customer>("/api/v1/customers", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateCustomer(
  id: number,
  input: UpdateCustomerInput,
): Promise<Customer> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockCustomers.findIndex((c) => c.id === id);
    if (index === -1) throw new ApiError(404, `Customer ${id} not found`);
    const customer = { ...mockCustomers[index], ...input };
    mockCustomers[index] = customer;
    return clone(customer);
  }
  return apiFetch<Customer>(`/api/v1/customers/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteCustomer(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockCustomers.findIndex((c) => c.id === id);
    if (index === -1) throw new ApiError(404, `Customer ${id} not found`);
    mockCustomers.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/customers/${id}`, { method: "DELETE" });
}
