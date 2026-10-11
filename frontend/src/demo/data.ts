// Sample data for the demo pages. scheduler-, dealership-, billing- and
// report-service don't exist yet (docs/architecture-design.md §0), so the
// booking → service → bill → report flow runs in the browser against this.

export function formatVnd(amount: number): string {
  return `${amount.toLocaleString('vi-VN')} ₫`
}

export interface Hub {
  id: number
  name: string
  bays: number
  technicians: number
}

export const HUBS: Hub[] = [
  { id: 1, name: 'District 7 Hub', bays: 4, technicians: 6 },
  { id: 2, name: 'Thu Duc Hub', bays: 3, technicians: 4 },
]

export interface ServiceType {
  id: number
  name: string
  minutes: number
  skill: string
  price: number
}

export const SERVICE_CATALOG: ServiceType[] = [
  { id: 1, name: 'Oil change', minutes: 30, skill: 'Engine L1', price: 350_000 },
  { id: 2, name: 'Brake inspection', minutes: 45, skill: 'Brakes L2', price: 200_000 },
  { id: 3, name: 'Tyre rotation', minutes: 30, skill: 'Tyres L1', price: 150_000 },
  { id: 4, name: 'A/C service', minutes: 60, skill: 'HVAC L2', price: 600_000 },
]

// Applied when the vehicle's warranty has expired or it has none
// (business-requirement §6). The real amount is an open question.
export const WARRANTY_FEE = 250_000

// Share of the estimate taken as a deposit. Open question #5 in
// business-requirement §10; 30% is a placeholder.
export const DEPOSIT_RATE = 0.3

export const SLOT_TIMES = ['08:00', '09:30', '11:00', '13:30', '15:00']

// Next five weekdays from today, each with a fixed pattern of taken slots so
// the grid looks lived-in.
export interface SlotDay {
  key: string
  weekday: string
  label: string
  taken: boolean[]
}

const TAKEN_PATTERNS = [
  [true, true, false, true, false],
  [false, false, false, false, true],
  [false, false, true, false, false],
  [true, false, false, false, false],
  [false, true, true, false, false],
]

export function upcomingDays(from = new Date()): SlotDay[] {
  const days: SlotDay[] = []
  const d = new Date(from)
  while (days.length < 5) {
    d.setDate(d.getDate() + 1)
    if (d.getDay() === 0 || d.getDay() === 6) continue
    days.push({
      key: d.toISOString().slice(0, 10),
      weekday: d.toLocaleDateString(undefined, { weekday: 'short' }),
      label: d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' }),
      taken: TAKEN_PATTERNS[days.length],
    })
  }
  return days
}

export interface Material {
  name: string
  qty: number
  unit: string
  unitPrice: number
}

// Materials a booking consumes, by service id.
export const MATERIALS_BY_SERVICE: Record<number, Material[]> = {
  1: [
    { name: 'Engine oil 5W-30', qty: 4, unit: 'L', unitPrice: 70_000 },
    { name: 'Oil filter', qty: 1, unit: '', unitPrice: 40_000 },
  ],
  2: [{ name: 'Brake cleaner', qty: 1, unit: 'can', unitPrice: 60_000 }],
  3: [],
  4: [{ name: 'Refrigerant R134a', qty: 0.4, unit: 'kg', unitPrice: 450_000 }],
}

// The extra work the technician finds mid-service (business-requirement §5.6).
export const CHANGE_REQUEST = {
  service: 'Replace front brake pads',
  price: 300_000,
  extraMinutes: 40,
  material: { name: 'Brake pad set', qty: 1, unit: '', unitPrice: 450_000 } as Material,
  note: 'Pads are at 2 mm, below the safe limit.',
}

export const TECHNICIAN = {
  name: 'Minh Tran',
  skills: [
    { name: 'Engine', level: 2.3, count: 7 },
    { name: 'Brakes', level: 2.1, count: 4 },
    { name: 'Tyres', level: 1.8, count: 9 },
  ],
}

export type JobState = 'confirmed' | 'held' | 'in-service' | 'waiting' | 'completed'

export const JOB_STATE_LABELS: Record<JobState, string> = {
  confirmed: 'Confirmed',
  held: 'Held (unpaid)',
  'in-service': 'In service',
  waiting: 'Waiting approval',
  completed: 'Completed',
}

export interface BayJob {
  plate: string
  what: string
  start: number // hour of day, e.g. 9.5 = 09:30
  end: number
  state: JobState
  mine?: boolean
}

export const OTHER_JOBS: Record<string, BayJob[]> = {
  'Bay 1': [
    { plate: '30F-882.10', what: 'Oil change', start: 8, end: 9, state: 'completed' },
    { plate: '51G-004.77', what: 'A/C service', start: 13.5, end: 14.5, state: 'confirmed' },
  ],
  'Bay 2': [{ plate: '59C-310.02', what: 'Inspection', start: 15, end: 16, state: 'held' }],
  'Bay 3': [
    { plate: '51H-777.31', what: 'Tyre rotation', start: 8, end: 8.5, state: 'completed' },
    { plate: '60A-221.90', what: 'Brake inspection', start: 10, end: 11, state: 'in-service' },
    { plate: '51K-118.06', what: 'Oil change', start: 16, end: 16.5, state: 'confirmed' },
  ],
  'Bay 4': [
    { plate: '43A-560.12', what: 'Engine diagnostic', start: 11, end: 13, state: 'confirmed' },
  ],
}

// Sample vehicle used when vehicle-service has none (or is unreachable).
export const SAMPLE_VEHICLE = {
  id: 0,
  label: 'Toyota Vios 2022 · 51A-123.45',
  plate: '51A-123.45',
  inWarranty: true,
  warrantyNote: 'Under warranty (sample vehicle)',
}

export const PAST_BILLS = [
  { id: 'B-1873', plate: '51A-123.45', date: '02 Jul 2026', total: 650_000 },
  { id: 'B-1544', plate: '30F-882.10', date: '18 Apr 2026', total: 2_300_000 },
]

// End-of-day report sample (business-requirement §9, data-warehouse-design §12).
export const REPORT = {
  kpis: [
    { label: 'Revenue', value: '14,2 M ₫', delta: '+8% vs last week', tone: 'good' },
    { label: 'Bookings completed', value: '9 / 10', delta: '1 no-show', tone: 'neutral' },
    { label: 'Mid-service upsell', value: '3 of 4', delta: 'approved by customers', tone: 'neutral' },
    { label: 'Bay occupancy', value: '71%', delta: '−4 pts vs last week', tone: 'warn' },
  ],
  revenue: [
    { name: 'Engine diagnostic', amount: 4_100_000 },
    { name: 'Brakes', amount: 3_600_000 },
    { name: 'Oil change', amount: 2_800_000 },
    { name: 'A/C service', amount: 2_400_000 },
    { name: 'Tyres', amount: 1_300_000 },
  ],
  technicians: [
    { name: 'Minh Tran', utilization: 88, jobs: 3 },
    { name: 'Lan Pham', utilization: 81, jobs: 3 },
    { name: 'Huy Nguyen', utilization: 74, jobs: 2 },
    { name: 'An Le', utilization: 62, jobs: 2 },
    { name: 'Bao Vo', utilization: 45, jobs: 1 },
    { name: 'Thu Do', utilization: 30, jobs: 1 },
  ],
  materials: [
    { name: 'Engine oil 5W-30', used: '16 L', billed: 1_120_000, stock: '44 L', low: false },
    { name: 'Oil filter', used: '4', billed: 160_000, stock: '22', low: false },
    { name: 'Brake pad set', used: '3', billed: 1_350_000, stock: '2', low: true },
    { name: 'Refrigerant R134a', used: '1,2 kg', billed: 540_000, stock: '0,8 kg', low: true },
  ],
}
