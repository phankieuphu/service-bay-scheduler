import { useEffect, useState } from 'react'
import {
  cancelAppointment,
  formatDate,
  formatTime,
  listHubs,
  listMyAppointments,
  type Appointment,
  type AppointmentStatus,
  type Hub,
} from '../api/scheduler'
import { AppointmentBilling } from './AppointmentBilling'
import { PageHeader } from './PageHeader'

const statusBadge: Record<AppointmentStatus, string> = {
  BOOKED: 'badge-active',
  CHECKED_IN: 'badge-warn',
  IN_PROGRESS: 'badge-warn',
  COMPLETED: 'badge-inactive',
  CANCELLED: 'badge-scrapped',
  NO_SHOW: 'badge-scrapped',
}

// Fallback when an appointment's hub isn't in the hub list any more.
const DEFAULT_TIME_ZONE = 'Asia/Ho_Chi_Minh'

export function MyAppointments({ onBook }: { onBook: () => void }) {
  const [appointments, setAppointments] = useState<Appointment[] | null>(null)
  const [hubs, setHubs] = useState<Map<number, Hub>>(new Map())
  const [error, setError] = useState<string | null>(null)
  const [cancelling, setCancelling] = useState<number | null>(null)
  const [cancelError, setCancelError] = useState<string | null>(null)
  // The appointment whose deposit or bill is open below it.
  const [billingFor, setBillingFor] = useState<number | null>(null)
  // What counts as upcoming is fixed when the page opens.
  const [now] = useState(() => Date.now())

  useEffect(() => {
    let cancelled = false
    Promise.all([listMyAppointments(), listHubs()])
      .then(([a, h]) => {
        if (cancelled) return
        setAppointments(a.appointments)
        setHubs(new Map(h.hubs.map((hub) => [hub.id, hub])))
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load appointments.')
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function handleCancel(appointment: Appointment) {
    if (!window.confirm(`Cancel appointment #${appointment.id}?`)) return
    setCancelling(appointment.id)
    setCancelError(null)
    try {
      const updated = await cancelAppointment(appointment.id)
      setAppointments((prev) => prev?.map((a) => (a.id === updated.id ? updated : a)) ?? null)
    } catch (err) {
      setCancelError(err instanceof Error ? err.message : 'Cancelling failed.')
    } finally {
      setCancelling(null)
    }
  }

  const upcoming =
    appointments
      ?.filter((a) => new Date(a.end).getTime() > now && a.status !== 'CANCELLED')
      .reverse() ?? []
  const past = appointments?.filter((a) => !upcoming.includes(a)) ?? []

  function renderList(list: Appointment[], empty: string) {
    if (list.length === 0) return <p className="muted small flush">{empty}</p>
    return (
      <ul className="item-list">
        {list.map((a) => {
          const hub = hubs.get(a.hub_id)
          const tz = hub?.timezone ?? DEFAULT_TIME_ZONE
          const cancellable = a.status === 'BOOKED' && new Date(a.start).getTime() > now
          const billable = a.status === 'BOOKED' || a.status === 'COMPLETED'
          return (
            <li key={a.id} className={`item${billingFor === a.id ? ' expanded' : ''}`}>
              <div className="item-main">
                <span className="strong">
                  {formatDate(a.start, tz)} · {formatTime(a.start, tz)}–{formatTime(a.end, tz)}
                </span>
                <span className="muted small">
                  {hub?.name ?? `Hub #${a.hub_id}`} · vehicle #{a.vehicle_id} · #{a.id}
                </span>
                <span className="small">
                  {a.services.map((s) => s.name).join(', ')} · total {a.total_amount}
                </span>
              </div>
              <div className="item-meta">
                <span className={`badge ${statusBadge[a.status]}`}>{a.status.replace('_', ' ')}</span>
                {billable && (
                  <button
                    type="button"
                    className="link"
                    aria-expanded={billingFor === a.id}
                    onClick={() => setBillingFor(billingFor === a.id ? null : a.id)}
                  >
                    {billingFor === a.id ? 'Hide billing' : a.status === 'COMPLETED' ? 'Bill' : 'Billing'}
                  </button>
                )}
                {cancellable && (
                  <button
                    type="button"
                    className="link"
                    disabled={cancelling === a.id}
                    onClick={() => handleCancel(a)}
                  >
                    {cancelling === a.id ? 'Cancelling…' : 'Cancel'}
                  </button>
                )}
              </div>
              {billingFor === a.id && (
                <div className="item-detail">
                  <AppointmentBilling appointment={a} />
                </div>
              )}
            </li>
          )
        })}
      </ul>
    )
  }

  return (
    <div className="stack roomy">
      <PageHeader title="My appointments" description="Your bookings at Service Bay hubs." />
      {error && <p className="message error flush">{error}</p>}
      {!error && appointments === null && <p className="muted">Loading…</p>}
      {cancelError && <p className="message error flush">{cancelError}</p>}
      {appointments && (
        <div className="two-col">
          <section className="card stack" aria-labelledby="upcoming-heading">
            <div className="subsection-header">
              <h2 id="upcoming-heading">Upcoming</h2>
              <button type="button" className="secondary" onClick={onBook}>
                Book a service
              </button>
            </div>
            {renderList(upcoming, 'Nothing booked. Book a service to see it here.')}
          </section>
          <section className="card stack" aria-labelledby="past-heading">
            <h2 id="past-heading">Past and cancelled</h2>
            {renderList(past, 'No past appointments yet.')}
          </section>
        </div>
      )}
    </div>
  )
}
