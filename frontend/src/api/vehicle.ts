import { jsonBody, request } from './http'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_VEHICLE_API_URL ?? 'http://localhost:8081/api/v1'

export type VehicleStatus = 'ACTIVE' | 'SOLD' | 'SCRAPPED'

export interface Vehicle {
  id: number
  vin: string
  license_plate: string
  warranty_end_date: string
  status: VehicleStatus
  created_at: string
  updated_at: string
}

export interface TransferVehiclePayload {
  vehicle_id: number
  from: number
  to: number
  date?: string
}

export function getVehicle(vehicleId: number): Promise<Vehicle> {
  return request(`${BASE_URL}/vehicle/${vehicleId}`)
}

export function transferVehicle(payload: TransferVehiclePayload): Promise<void> {
  return request(`${BASE_URL}/transfer`, jsonBody('POST', payload))
}
