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

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const body = await response.json()
    if (typeof body?.error === 'string') return body.error
  } catch {
    // response had no JSON body; fall through to the status text
  }
  return response.statusText || `Request failed with status ${response.status}`
}

export async function getVehicle(vehicleId: number): Promise<Vehicle> {
  const response = await fetch(`${BASE_URL}/vehicle/${vehicleId}`)
  if (!response.ok) {
    throw new ApiError(response.status, await readErrorMessage(response))
  }
  return response.json()
}

export async function transferVehicle(
  payload: TransferVehiclePayload,
): Promise<void> {
  const response = await fetch(`${BASE_URL}/transfer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!response.ok && response.status !== 204) {
    throw new ApiError(response.status, await readErrorMessage(response))
  }
}
