import { createContext, useContext } from 'react'
import {
  CHANGE_REQUEST,
  MATERIALS_BY_SERVICE,
  WARRANTY_FEE,
  type Hub,
  type Material,
  type ServiceType,
} from './data'

export interface DemoBooking {
  id: number
  vehicleLabel: string
  plate: string
  inWarranty: boolean
  hub: Hub
  services: ServiceType[]
  dayLabel: string
  time: string
  estimate: number
  deposit: number
}

// Where the booking is on the hub's side (business-requirement §5).
export type ServiceStage = 'scheduled' | 'in-service' | 'waiting' | 'completed'

export interface DemoEvent {
  id: number
  type: string
  topic: string
  key: string
  at: Date
}

export interface DemoState {
  booking: DemoBooking | null
  stage: ServiceStage
  changeDecision: 'approved' | 'declined' | null
  paidMethod: string | null
  events: DemoEvent[]
}

export interface DemoContextValue {
  state: DemoState
  confirmBooking: (booking: Omit<DemoBooking, 'id'>) => void
  startService: () => void
  requestChange: () => void
  answerChange: (approved: boolean) => void
  completeService: () => void
  payBalance: (method: string) => void
  reset: () => void
}

export const DemoContext = createContext<DemoContextValue | null>(null)

export function useDemo(): DemoContextValue {
  const value = useContext(DemoContext)
  if (!value) throw new Error('useDemo must be used inside <DemoProvider>')
  return value
}

export interface BillLine {
  name: string
  kind: 'Service' | 'Material'
  qty: string
  amount: number
  addedMidService: boolean
}

export interface Bill {
  lines: BillLine[]
  services: number
  materials: number
  warrantyFee: number
  total: number
  deposit: number
  balance: number
}

function materialLine(m: Material, addedMidService = false): BillLine {
  return {
    name: m.name,
    kind: 'Material',
    qty: `${m.qty.toLocaleString()}${m.unit ? ` ${m.unit}` : ''}`,
    amount: Math.round(m.qty * m.unitPrice),
    addedMidService,
  }
}

// buildBill applies business-requirement §6: warranty fee (only when out of
// warranty) + service fees + material costs, less any deposit.
export function buildBill(booking: DemoBooking, changeApproved: boolean): Bill {
  const lines: BillLine[] = booking.services.map((s) => ({
    name: s.name,
    kind: 'Service',
    qty: '1',
    amount: s.price,
    addedMidService: false,
  }))
  if (changeApproved) {
    lines.push({
      name: CHANGE_REQUEST.service,
      kind: 'Service',
      qty: '1',
      amount: CHANGE_REQUEST.price,
      addedMidService: true,
    })
  }
  for (const s of booking.services) {
    for (const m of MATERIALS_BY_SERVICE[s.id] ?? []) lines.push(materialLine(m))
  }
  if (changeApproved) lines.push(materialLine(CHANGE_REQUEST.material, true))

  const services = lines.filter((l) => l.kind === 'Service').reduce((a, l) => a + l.amount, 0)
  const materials = lines.filter((l) => l.kind === 'Material').reduce((a, l) => a + l.amount, 0)
  const warrantyFee = booking.inWarranty ? 0 : WARRANTY_FEE
  const total = services + materials + warrantyFee
  return {
    lines,
    services,
    materials,
    warrantyFee,
    total,
    deposit: booking.deposit,
    balance: Math.max(0, total - booking.deposit),
  }
}
