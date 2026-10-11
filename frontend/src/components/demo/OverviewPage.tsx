import { useEffect, useState } from 'react'
import { listVehicles } from '../../api/vehicle'
import { useDemo } from '../../demo/demo-context'
import type { DemoSection } from './sections'

type StepState = 'done' | 'next' | 'todo'

const STEP_LABELS: Record<StepState, string> = { done: 'Done', next: 'Next', todo: 'To do' }

const SERVICES = [
  { name: 'identity-service', owns: 'Users, credentials, roles, tokens', topic: '→ identity.user-events', live: true },
  { name: 'customer-service', owns: 'Customer profiles and contact info', topic: '→ customer.events', live: true },
  { name: 'vehicle-service', owns: 'Vehicles, owners, warranty, history', topic: '→ vehicle.*.v1', live: true },
  { name: 'scheduler-service', owns: 'Bookings, slot holds, mid-service changes', topic: '→ BookingConfirmed, ServiceCompleted', live: false },
  { name: 'dealership-service', owns: 'Hubs, bays, technicians, skills, stock', topic: '→ MaterialsConsumed, ShiftChanged', live: false },
  { name: 'billing-service', owns: 'Bills, deposits, payments', topic: '→ PaymentReceived', live: false },
  { name: 'notification-service', owns: 'Email / SMS / push delivery log', topic: '← BookingConfirmed, PaymentReceived', live: false },
  { name: 'report-service', owns: 'Daily per-hub rollups', topic: '← ServiceCompleted, MaterialsConsumed', live: false },
]

function timeAgo(at: Date, now: number): string {
  const seconds = Math.max(0, Math.round((now - at.getTime()) / 1000))
  if (seconds < 5) return 'just now'
  if (seconds < 60) return `${seconds} s ago`
  return `${Math.round(seconds / 60)} min ago`
}

export function OverviewPage({ onNavigate }: { onNavigate: (section: DemoSection) => void }) {
  const { state, reset } = useDemo()
  const [hasVehicle, setHasVehicle] = useState<boolean | null>(null)
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    let cancelled = false
    listVehicles({}, 0, 1)
      .then((page) => !cancelled && setHasVehicle(page.vehicles.length > 0))
      .catch(() => !cancelled && setHasVehicle(false))
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 5000)
    return () => clearInterval(timer)
  }, [])

  const doneFlags = [
    true, // signed in to see this page
    hasVehicle === true,
    state.booking !== null,
    state.stage === 'completed',
    state.paidMethod !== null,
    false,
  ]
  const firstOpen = doneFlags.indexOf(false)
  const steps: { title: string; svc: string; target: DemoSection }[] = [
    { title: 'Sign up', svc: 'identity-service', target: 'account' },
    { title: 'Add a vehicle', svc: 'vehicle-service', target: 'vehicles' },
    { title: 'Book a slot', svc: 'scheduler-service', target: 'booking' },
    { title: 'Service the car', svc: 'dealership-service', target: 'hub' },
    { title: 'Pay the bill', svc: 'billing-service', target: 'billing' },
    { title: 'Day-end report', svc: 'report-service', target: 'reports' },
  ]

  return (
    <div className="stack roomy demo">
      <section className="demo-hero">
        <div>
          <h2 className="demo-title">Follow one booking through every service</h2>
          <p className="muted flush">
            Walk the full journey — register, add a vehicle, book a slot, service it, pay, report —
            and watch each step land as an event from the service that owns it.
          </p>
        </div>
        <div className="actions">
          <button type="button" className="primary" onClick={() => onNavigate('booking')}>
            Start demo scenario
          </button>
          <button type="button" className="secondary" onClick={reset}>
            Reset sample data
          </button>
        </div>
      </section>

      <section aria-labelledby="scenario-heading" className="stack">
        <h3 id="scenario-heading" className="demo-h3">
          Demo scenario
        </h3>
        <ol className="scenario">
          {steps.map((step, i) => {
            const stepState: StepState = doneFlags[i] ? 'done' : i === firstOpen ? 'next' : 'todo'
            return (
              <li key={step.title}>
                <button type="button" className="scenario-step" onClick={() => onNavigate(step.target)}>
                  <span className="scenario-meta">
                    <span>Step {i + 1}</span>
                    <span className={`badge step-${stepState}`}>{STEP_LABELS[stepState]}</span>
                  </span>
                  <span className="strong">{step.title}</span>
                  <span className="small muted">{step.svc}</span>
                </button>
              </li>
            )
          })}
        </ol>
      </section>

      <div className="split">
        <section aria-labelledby="services-heading" className="split-main stack">
          <div className="section-head">
            <h3 id="services-heading" className="demo-h3">
              Services
            </h3>
            <span className="legend">
              <span>
                <span className="dot dot-live" /> Live API
              </span>
              <span>
                <span className="dot dot-sim" /> Simulated in browser
              </span>
            </span>
          </div>
          <div className="service-grid">
            {SERVICES.map((s) => (
              <article key={s.name} className={`service-card${s.live ? '' : ' simulated'}`}>
                <div className="section-head">
                  <h4 className="mono">{s.name}</h4>
                  <span className={`badge ${s.live ? 'badge-active' : 'badge-warn'}`}>
                    {s.live ? 'Live' : 'Simulated'}
                  </span>
                </div>
                <p className="muted small flush">{s.owns}</p>
                <p className="mono small flush neutral">{s.topic}</p>
              </article>
            ))}
          </div>
        </section>

        <aside aria-labelledby="events-heading" className="split-side panel stack">
          <div className="section-head">
            <h3 id="events-heading" className="demo-h3">
              Event stream
            </h3>
            <span className="small live-note">
              <span className="dot dot-live" /> Outbox relay · 2 s poll
            </span>
          </div>
          {state.events.length === 0 ? (
            <p className="muted small flush">
              No events yet. Book a slot to see <code>BookingConfirmed</code> appear here.
            </p>
          ) : (
            <ul className="event-list">
              {state.events.slice(0, 8).map((e) => (
                <li key={e.id}>
                  <span className="section-head">
                    <strong>{e.type}</strong>
                    <span className="small muted">{timeAgo(e.at, now)}</span>
                  </span>
                  <span className="mono small neutral">
                    {e.topic} · key {e.key}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="small muted flush">
            Events from simulated services are generated in the browser; topic names for
            services that aren’t built yet are provisional.
          </p>
        </aside>
      </div>
    </div>
  )
}
