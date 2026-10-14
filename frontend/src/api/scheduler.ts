import { jsonBody } from './http'
import { authorizedRequest } from './session'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_SCHEDULER_API_URL ?? 'http://localhost:8085/api/v1'

export interface Hub {
  id: number
  name: string
  address: string
  // "HH:MM" in the hub's time zone
  open_hour: string
  close_hour: string
  timezone: string
  bay_count: number
}

// Fees and totals are decimal strings ("45.00") so no precision is lost.
export interface ServiceOffer {
  id: number
  name: string
  service_type: string
  fee: string
  duration_minutes: number
}

export interface Slot {
  start: string
  end: string
  available: boolean
}

export interface Availability {
  hub_id: number
  date: string
  timezone: string
  duration_minutes: number
  slots: Slot[]
}

export type AppointmentStatus =
  | 'BOOKED'
  | 'CHECKED_IN'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'CANCELLED'
  | 'NO_SHOW'

export interface Appointment {
  id: number
  hub_id: number
  customer_id: number
  vehicle_id: number
  bay_id: number
  start: string
  end: string
  status: AppointmentStatus
  total_amount: string
  services: { service_id: number; name: string; fee: string }[]
  created_at: string
  updated_at: string
}

export interface BookPayload {
  hub_id: number
  vehicle_id: number
  service_ids: number[]
  start: string
}

export function listHubs(): Promise<{ hubs: Hub[] }> {
  return authorizedRequest(`${BASE_URL}/hubs`)
}

export function listServices(): Promise<{ services: ServiceOffer[] }> {
  return authorizedRequest(`${BASE_URL}/services`)
}

export function getAvailability(
  hubId: number,
  date: string,
  serviceIds: number[],
): Promise<Availability> {
  const params = new URLSearchParams({ date })
  for (const id of serviceIds) params.append('service_id', String(id))
  return authorizedRequest(`${BASE_URL}/hubs/${hubId}/availability?${params}`)
}

export function bookAppointment(payload: BookPayload): Promise<Appointment> {
  return authorizedRequest(`${BASE_URL}/appointments`, jsonBody('POST', payload))
}

export function listMyAppointments(): Promise<{ appointments: Appointment[] }> {
  return authorizedRequest(`${BASE_URL}/me/appointments`)
}

export function cancelAppointment(appointmentId: number): Promise<Appointment> {
  return authorizedRequest(`${BASE_URL}/appointments/${appointmentId}/cancel`, { method: 'POST' })
}

// Times are shown in the hub's own zone, since that's where the customer
// has to turn up.
export function formatTime(value: string, timeZone: string): string {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', timeZone })
}

export function formatDate(value: string, timeZone: string): string {
  return new Date(value).toLocaleDateString([], {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone,
  })
}

// The calendar date it is right now in timeZone, as YYYY-MM-DD.
export function todayIn(timeZone: string): string {
  return new Date().toLocaleDateString('en-CA', { timeZone })
}

export interface HubDay {
  hub_id: number
  date: string
  timezone: string
  appointments: Appointment[]
}

// Staff only: a hub's appointments on one local day, earliest first.
export function listHubDay(hubId: number, date: string): Promise<HubDay> {
  return authorizedRequest(`${BASE_URL}/hubs/${hubId}/appointments?date=${encodeURIComponent(date)}`)
}

// Staff only: marks the work done. scheduler publishes ServiceCompleted and
// billing-service prices the bill from it a moment later.
export function completeAppointment(appointmentId: number): Promise<Appointment> {
  return authorizedRequest(`${BASE_URL}/appointments/${appointmentId}/complete`, { method: 'POST' })
}
