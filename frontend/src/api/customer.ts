import { jsonBody, request } from './http'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_CUSTOMER_API_URL ?? 'http://localhost:8080/api/v1'

export type CustomerStatus = 'ACTIVE' | 'INACTIVE' | 'BANNED'

export interface Customer {
  id: number
  name: string
  email: string
  phone: string
  birth_day: string
  status: CustomerStatus
  created_at: string
  // Keep this as the exact string the API returned: it's echoed back on
  // update as an optimistic lock, and round-tripping it through Date would
  // drop the sub-millisecond precision and cause a spurious 409.
  updated_at: string
}

export interface CustomerPage {
  customers: Customer[]
  next_cursor?: number
  has_more: boolean
}

export interface CreateCustomerPayload {
  name: string
  email: string
  phone?: string
  birth_day: string
}

export interface UpdateCustomerPayload {
  name?: string
  birth_day?: string
  updated_at: string
}

// The API's birth_day is a full RFC3339 timestamp at midnight UTC; these
// convert to and from the YYYY-MM-DD value a date <input> uses.
export function birthDayToInput(birthDay: string): string {
  return birthDay.slice(0, 10)
}

export function inputToBirthDay(date: string): string {
  return `${date}T00:00:00Z`
}

export function listCustomers(cursor = 0, limit = 20): Promise<CustomerPage> {
  const params = new URLSearchParams({ cursor: String(cursor), limit: String(limit) })
  return request(`${BASE_URL}/customer?${params}`)
}

export function getCustomer(customerId: number): Promise<Customer> {
  return request(`${BASE_URL}/customer/${customerId}`)
}

export function createCustomer(payload: CreateCustomerPayload): Promise<Customer> {
  return request(`${BASE_URL}/customer`, jsonBody('POST', payload))
}

export function updateCustomer(
  customerId: number,
  payload: UpdateCustomerPayload,
): Promise<void> {
  return request(`${BASE_URL}/customer/${customerId}`, jsonBody('PUT', payload))
}

export function deleteCustomer(customerId: number): Promise<void> {
  return request(`${BASE_URL}/customer/${customerId}`, { method: 'DELETE' })
}
