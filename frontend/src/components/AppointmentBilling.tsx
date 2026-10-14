import { useCallback, useEffect, useState } from 'react'
import {
  ApiError,
  getBookingBill,
  newIdempotencyKey,
  startDeposit,
  type Bill,
  type Payment,
  type StartedPayment,
} from '../api/billing'
import type { Appointment } from '../api/scheduler'
import { BillView } from './BillView'
import { PaymentForm } from './PaymentForm'

// The money side of one of the customer's appointments: a deposit while it's
// booked, the bill once the service is completed.
export function AppointmentBilling({ appointment }: { appointment: Appointment }) {
  const [bill, setBill] = useState<Bill | null | undefined>(undefined) // undefined: loading, null: none yet
  const [error, setError] = useState<string | null>(null)
  const [started, setStarted] = useState<StartedPayment | null>(null)
  const [depositKey, setDepositKey] = useState<string | null>(null)
  const [starting, setStarting] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const load = useCallback(() => {
    getBookingBill(appointment.id)
      .then(setBill)
      .catch((err) => {
        if (err instanceof ApiError && err.status === 404) setBill(null)
        else setError(err instanceof Error ? err.message : 'Couldn’t load the bill.')
      })
  }, [appointment.id])

  useEffect(load, [load])

  const depositPaid = bill?.payments.some((p) => p.type === 'DEPOSIT' && p.status === 'SUCCEEDED') ?? false
  const canPayDeposit =
    appointment.status === 'BOOKED' && (bill === null || (bill?.status === 'OPEN' && !depositPaid))

  async function payDeposit() {
    const key = depositKey ?? newIdempotencyKey()
    setDepositKey(key)
    setStarting(true)
    setError(null)
    try {
      setStarted(await startDeposit(appointment.id, key))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Couldn’t start the deposit.')
    } finally {
      setStarting(false)
    }
  }

  function settled(payment: Payment) {
    setStarted(null)
    setDepositKey(null)
    setNotice(payment.status === 'SUCCEEDED' ? 'Deposit paid. Thank you!' : null)
    if (payment.status !== 'SUCCEEDED' && payment.failure_message) setError(payment.failure_message)
    load()
  }

  if (bill === undefined && !error) return <p className="muted small">Loading billing…</p>

  return (
    <div className="stack">
      {error && <p className="message error flush">{error}</p>}
      {notice && <p className="message success flush">{notice}</p>}
      {bill && <BillView bill={bill} canPay onChanged={load} />}
      {bill === null && appointment.status !== 'BOOKED' && (
        <p className="muted small flush">
          {appointment.status === 'COMPLETED'
            ? 'Your bill is being prepared. Check back in a moment.'
            : 'There’s no bill for this appointment.'}
        </p>
      )}
      {canPayDeposit && !started && (
        <div className="actions">
          <button type="button" disabled={starting} onClick={payDeposit}>
            {starting ? 'Preparing…' : 'Pay deposit'}
          </button>
          <span className="muted small">A deposit holds your booking; it counts toward the final bill.</span>
        </div>
      )}
      {started && (
        <PaymentForm started={started} label="Deposit" onSettled={settled} onCancel={() => setStarted(null)} />
      )}
    </div>
  )
}
