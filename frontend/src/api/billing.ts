import { jsonBody } from './http'
import { authorizedRequest } from './session'

export { ApiError } from './http'

const BASE_URL = import.meta.env.VITE_BILLING_API_URL ?? 'http://localhost:8084/api/v1'

// Money is integer minor units plus an ISO 4217 code (billing-service #87):
// 15550 USD is $155.50, 150000 VND is ₫150,000. formatMoney does the
// conversion; never do arithmetic on the formatted string.
export type BillStatus = 'OPEN' | 'PENDING' | 'PARTIALLY_PAID' | 'PAID' | 'VOID'
export type PaymentStatus =
  | 'REQUIRES_PAYMENT'
  | 'PROCESSING'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'CANCELED'
  | 'REFUNDED'
  | 'PARTIALLY_REFUNDED'

export interface BillLine {
  id: number
  kind: 'SERVICE' | 'MATERIAL' | 'WARRANTY_FEE'
  source_id: number | null
  name: string
  quantity: number
  unit_price_minor: number
  line_total_minor: number
}

export interface Payment {
  id: number
  bill_id: number
  type: 'DEPOSIT' | 'FINAL'
  amount_minor: number
  currency: string
  status: PaymentStatus
  failure_code: string | null
  failure_message: string | null
  card_brand: string | null
  card_last4: string | null
  paid_at: string | null
}

export interface Bill {
  id: number
  appointment_id: number
  customer_id: number
  vehicle_id: number
  status: BillStatus
  currency: string
  lines: BillLine[]
  warranty_fee_minor: number
  service_fee_minor: number
  material_cost_minor: number
  // null while OPEN: the bill isn't priced until the service is completed,
  // so there's no total or balance to show yet (not "nothing due").
  total_minor: number | null
  payments: Payment[]
  amount_paid_minor: number
  balance_due_minor: number | null
  refund_due_minor: number
  finalized_at: string | null
}

export interface BillPage {
  bills: Bill[]
  next_cursor: number | null
  has_more: boolean
}

// A payment the browser confirms with Stripe's Payment Element.
export interface StartedPayment {
  payment_id: number
  bill_id: number
  type: 'DEPOSIT' | 'FINAL'
  amount_minor: number
  currency: string
  status: PaymentStatus
  client_secret: string | null
}

export function getBill(billId: number): Promise<Bill> {
  return authorizedRequest(`${BASE_URL}/bills/${billId}`)
}

// 404 until the booking has a deposit or its service is completed.
export function getBookingBill(appointmentId: number): Promise<Bill> {
  return authorizedRequest(`${BASE_URL}/bookings/${appointmentId}/bill`)
}

export function listMyBills(cursor?: number): Promise<BillPage> {
  const params = new URLSearchParams({ limit: '20' })
  if (cursor !== undefined) params.set('cursor', String(cursor))
  return authorizedRequest(`${BASE_URL}/me/bills?${params}`)
}

export function getPayment(paymentId: number): Promise<Payment> {
  return authorizedRequest(`${BASE_URL}/payments/${paymentId}`)
}

// idempotencyKey: one per user action (newIdempotencyKey()), reused if the
// same action is retried, so a lost response never charges twice.
export function startDeposit(appointmentId: number, idempotencyKey: string): Promise<StartedPayment> {
  return authorizedRequest(`${BASE_URL}/bookings/${appointmentId}/deposit`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}

// The server charges the bill's current balance; the client never sends an amount.
export function payBill(billId: number, idempotencyKey: string): Promise<StartedPayment> {
  return authorizedRequest(`${BASE_URL}/bills/${billId}/payments`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  })
}

// Stripe test-mode keys only: pays with a test card without the Payment Element.
export function testConfirm(paymentId: number, paymentMethod = 'pm_card_visa'): Promise<Payment> {
  return authorizedRequest(
    `${BASE_URL}/payments/${paymentId}/test-confirm`,
    jsonBody('POST', { payment_method: paymentMethod }),
  )
}

export function newIdempotencyKey(): string {
  return crypto.randomUUID()
}

// Stripe's zero- and three-decimal currencies (same lists as billing-service's Money).
const ZERO_DECIMAL = new Set(['BIF', 'CLP', 'DJF', 'GNF', 'JPY', 'KMF', 'KRW', 'MGA', 'PYG', 'RWF', 'UGX', 'VND', 'VUV', 'XAF', 'XOF', 'XPF'])
const THREE_DECIMAL = new Set(['BHD', 'JOD', 'KWD', 'OMR', 'TND'])

export function currencyDecimals(currency: string): number {
  const code = currency.toUpperCase()
  if (ZERO_DECIMAL.has(code)) return 0
  if (THREE_DECIMAL.has(code)) return 3
  return 2
}

export function formatMoney(amountMinor: number, currency: string): string {
  const decimals = currencyDecimals(currency)
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency.toUpperCase(),
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(amountMinor / 10 ** decimals)
}

// How long to wait for billing-service to record Stripe's webhook.
const POLL_INTERVAL_MS = 1000
const POLL_ATTEMPTS = 30

// Where Stripe sends the browser back after a payment method that redirects
// (bank redirects; cards and 3-D Secure finish in place). App reads
// payment_id from it and shows PaymentReturn.
export function returnUrl(paymentId: number): string {
  const url = new URL(window.location.href)
  url.search = ''
  url.hash = ''
  url.searchParams.set('payment_id', String(paymentId))
  return url.toString()
}

// Polls billing-service until the payment settles. What the browser saw from
// Stripe doesn't count: only the webhook, recorded by billing, does. A
// declined attempt stops the wait (REQUIRES_PAYMENT with a failure message).
export async function waitForSettlement(paymentId: number): Promise<Payment> {
  let latest = await getPayment(paymentId)
  for (let i = 0; i < POLL_ATTEMPTS && (latest.status === 'REQUIRES_PAYMENT' || latest.status === 'PROCESSING'); i++) {
    if (latest.status === 'REQUIRES_PAYMENT' && latest.failure_message) break
    await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS))
    latest = await getPayment(paymentId)
  }
  return latest
}
