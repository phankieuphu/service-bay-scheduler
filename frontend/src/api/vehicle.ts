import { jsonBody } from './http'
import { authorizedRequest } from './session'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_VEHICLE_API_URL ?? 'http://localhost:8081/api/v1'

export type VehicleStatus = 'ACTIVE' | 'SOLD' | 'SCRAPPED'

export interface Vehicle {
  id: number
  vin: string
  license_plate: string
  vehicle_model_id: number
  // null when the vehicle has no warranty on record
  warranty_end_date: string | null
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
  return authorizedRequest(`${BASE_URL}/vehicle/${vehicleId}`)
}

export function transferVehicle(payload: TransferVehiclePayload): Promise<void> {
  return authorizedRequest(`${BASE_URL}/transfer`, jsonBody('POST', payload))
}
