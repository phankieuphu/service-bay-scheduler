import { jsonBody } from './http'
import { authorizedRequest } from './session'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_VEHICLE_API_URL ?? 'http://localhost:8081/api/v1'

export type VehicleStatus = 'ACTIVE' | 'SOLD' | 'SCRAPPED'

export const VEHICLE_STATUSES: VehicleStatus[] = ['ACTIVE', 'SOLD', 'SCRAPPED']

export interface Vehicle {
  id: number
  vin: string
  license_plate: string
  vehicle_model_id: number
  // null when the vehicle has no warranty on record
  warranty_end_date: string | null
  status: VehicleStatus
  created_at: string
  // Kept as the exact string the API returned: it's echoed back on PATCH as
  // an optimistic lock (see customer.ts).
  updated_at: string
}

export interface VehiclePage {
  vehicles: Vehicle[]
  next_cursor?: number
  has_more: boolean
}

export interface VehicleFilters {
  vin?: string
  plate?: string
  status?: VehicleStatus
}

export interface RegisterVehiclePayload {
  vin: string
  vehicle_model_id: number
  license_plate?: string
  warranty_end_date?: string
  status?: VehicleStatus
}

// Omitted fields are left unchanged; the API has no way to clear a warranty.
export interface UpdateVehiclePayload {
  status?: VehicleStatus
  warranty_end_date?: string
  updated_at: string
}

export interface CustomerVehicle extends Vehicle {
  owned_from: string
}

export interface TransferVehiclePayload {
  vehicle_id: number
  from: number
  to: number
  date?: string
}

export interface Warranty {
  vehicle_id: number
  warranty_end_date: string | null
  as_of: string
  in_warranty: boolean
  days_remaining: number
}

export interface VehicleMaterial {
  id: number
  material_id: number
  description?: string
  count: number
  installed_at: string
}

export interface ServiceHistoryEntry {
  id: number
  appointment_id: number
  dealership_id: number
  completed_at: string
  services: { service_id: number; name: string }[]
}

// The API's calendar-day fields (warranty_end_date, owned_from, transfer and
// owner dates) are RFC3339 timestamps whose day is read in the offset sent;
// these convert to and from the YYYY-MM-DD value a date <input> uses.
export function dayFromApi(value: string | null): string {
  return value ? value.slice(0, 10) : ''
}

export function dayToApi(date: string): string {
  return `${date}T00:00:00Z`
}

// todayInput is the user's local date as YYYY-MM-DD.
export function todayInput(): string {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

export function listVehicles(
  filters: VehicleFilters = {},
  cursor = 0,
  limit = 20,
): Promise<VehiclePage> {
  const params = new URLSearchParams({ cursor: String(cursor), limit: String(limit) })
  if (filters.vin) params.set('vin', filters.vin)
  if (filters.plate) params.set('plate', filters.plate)
  if (filters.status) params.set('status', filters.status)
  return authorizedRequest(`${BASE_URL}/vehicle?${params}`)
}

export function getVehicle(vehicleId: number): Promise<Vehicle> {
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}`)
}

export function registerVehicle(payload: RegisterVehiclePayload): Promise<Vehicle> {
  return authorizedRequest(`${BASE_URL}/vehicle`, jsonBody('POST', payload))
}

export function updateVehicle(vehicleId: number, payload: UpdateVehiclePayload): Promise<Vehicle> {
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}`, jsonBody('PATCH', payload))
}

export function assignOwner(
  vehicleId: number,
  payload: { customer_id: number; date?: string },
): Promise<void> {
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}/owner`, jsonBody('POST', payload))
}

export function transferVehicle(payload: TransferVehiclePayload): Promise<void> {
  return authorizedRequest(`${BASE_URL}/transfer`, jsonBody('POST', payload))
}

export function getCustomerVehicles(customerId: number): Promise<{ vehicles: CustomerVehicle[] }> {
  return authorizedRequest(`${BASE_URL}/customers/${customerId}/vehicles`)
}

export function getWarranty(vehicleId: number, date?: string): Promise<Warranty> {
  const query = date ? `?${new URLSearchParams({ date })}` : ''
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}/warranty${query}`)
}

export function getVehicleMaterials(vehicleId: number): Promise<{ materials: VehicleMaterial[] }> {
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}/materials`)
}

export function getServiceHistory(vehicleId: number): Promise<{ history: ServiceHistoryEntry[] }> {
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}/history`)
}
