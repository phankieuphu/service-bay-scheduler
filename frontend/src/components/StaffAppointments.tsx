import { useCallback, useEffect, useState } from 'react'
import { ApiError, getBookingBill, type Bill } from '../api/billing'
import {
  completeAppointment,
  formatTime,
  listHubDay,
  listHubs,
  todayIn,
  type Appointment,
  type Hub,
  type HubDay,
} from '../api/scheduler'
import { BillView } from './BillView'
import { PageHeader } from './PageHeader'

// How long to keep asking billing for a bill after completing: it's priced
// from the ServiceCompleted event, one outbox relay and one consumer hop away.
const BILL_POLL_ATTEMPTS = 15

// Staff view of a hub's day: mark work completed, then see the bill billing
// prices from it.
export function StaffAppointments() {
  const [hubs, setHubs] = useState<Hub[] | null>(null)
  const [hubId, setHubId] = useState<number | null>(null)
  const [date, setDate] = useState('')
  const [day, setDay] = useState<HubDay | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [completing, setCompleting] = useState<number | null>(null)
  const [openBill, setOpenBill] = useState<number | null>(null)

  useEffect(() => {
    listHubs()
      .then(({ hubs }) => {
        setHubs(hubs)
        if (hubs.length > 0) {
          setHubId(hubs[0].id)
          setDate(todayIn(hubs[0].timezone))
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load hubs.'))
  }, [])

  const load = useCallback(() => {
    if (hubId === null || !date) return
    listHubDay(hubId, date)
      .then((d) => {
        setDay(d)
        setError(null)
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load appointments.'))
  }, [hubId, date])

  useEffect(load, [load])

  async function complete(a: Appointment) {
    if (!window.confirm(`Mark appointment #${a.id} completed? Its bill will be issued.`)) return
    setCompleting(a.id)
    setError(null)
    try {
      const updated = await completeAppointment(a.id)
      setDay((prev) => prev && { ...prev, appointments: prev.appointments.map((x) => (x.id === updated.id ? updated : x)) })
      setOpenBill(a.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Completing failed.')
    } finally {
      setCompleting(null)
    }
  }

  const tz = day?.timezone ?? hubs?.find((h) => h.id === hubId)?.timezone ?? 'UTC'

  return (
    <div className="stack roomy">
      <PageHeader title="Appointments" description="A hub’s bookings for the day. Completing one issues its bill." />
      <div className="inline-form">
        <label className="field">
          <span className="label">Hub</span>
          <select value={hubId ?? ''} onChange={(e) => setHubId(Number(e.target.value))} disabled={!hubs}>
            {hubs?.map((h) => (
              <option key={h.id} value={h.id}>
                {h.name}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span className="label">Date</span>
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </label>
      </div>
      {error && <p className="message error flush">{error}</p>}
      {day && day.appointments.length === 0 && <p className="muted">No appointments that day.</p>}
      {day && day.appointments.length > 0 && (
        <ul className="item-list">
          {day.appointments.map((a) => (
            <li key={a.id} className={`item${openBill === a.id ? ' expanded' : ''}`}>
              <div className="item-main">
                <span className="strong">
                  {formatTime(a.start, tz)}–{formatTime(a.end, tz)} · #{a.id}
                </span>
                <span className="muted small">
                  customer #{a.customer_id} · vehicle #{a.vehicle_id} · bay #{a.bay_id}
                </span>
                <span className="small">{a.services.map((s) => s.name).join(', ')}</span>
              </div>
              <div className="item-meta">
                <span className={`badge ${a.status === 'COMPLETED' ? 'badge-inactive' : a.status === 'CANCELLED' ? 'badge-scrapped' : 'badge-active'}`}>
                  {a.status.replace('_', ' ')}
                </span>
                {(a.status === 'BOOKED' || a.status === 'CHECKED_IN' || a.status === 'IN_PROGRESS') && (
                  <button type="button" disabled={completing === a.id} onClick={() => complete(a)}>
                    {completing === a.id ? 'Completing…' : 'Mark completed'}
                  </button>
                )}
                {a.status !== 'CANCELLED' && (
                  <button
                    type="button"
                    className="link"
                    aria-expanded={openBill === a.id}
                    onClick={() => setOpenBill(openBill === a.id ? null : a.id)}
                  >
                    {openBill === a.id ? 'Hide bill' : 'Bill'}
                  </button>
                )}
              </div>
              {openBill === a.id && (
                <div className="item-detail">
                  <StaffBill appointment={a} />
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function StaffBill({ appointment }: { appointment: Appointment }) {
  const [bill, setBill] = useState<Bill | null | undefined>(undefined)
  const [error, setError] = useState<string | null>(null)
  // Bumped after a payment sync so the bill is fetched again.
  const [version, setVersion] = useState(0)

  useEffect(() => {
    let cancelled = false
    // Right after completing, billing hasn't priced the bill yet: keep asking
    // for a few seconds rather than showing an unpriced or missing bill.
    async function poll() {
      for (let i = 0; i < BILL_POLL_ATTEMPTS && !cancelled; i++) {
        try {
          const b = await getBookingBill(appointment.id)
          if (cancelled) return
          setBill(b)
          if (appointment.status !== 'COMPLETED' || b.status !== 'OPEN') return
        } catch (err) {
          if (!(err instanceof ApiError && err.status === 404)) {
            if (!cancelled) setError(err instanceof Error ? err.message : 'Couldn’t load the bill.')
            return
          }
          if (!cancelled) setBill(null)
          if (appointment.status !== 'COMPLETED') return
        }
        await new Promise((r) => setTimeout(r, 1000))
      }
    }
    void poll()
    return () => {
      cancelled = true
    }
  }, [appointment.id, appointment.status, version])

  if (error) return <p className="message error flush">{error}</p>
  if (bill === undefined) return <p className="muted small">Loading bill…</p>
  if (bill === null) {
    return (
      <p className="muted small flush">
        {appointment.status === 'COMPLETED' ? 'The bill is being prepared…' : 'No bill yet: it’s issued when the service is completed.'}
      </p>
    )
  }
  return <BillView bill={bill} canPay={false} canSync onChanged={() => setVersion((v) => v + 1)} />
}
