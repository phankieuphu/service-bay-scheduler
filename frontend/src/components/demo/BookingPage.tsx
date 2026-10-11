import { useEffect, useMemo, useState } from 'react'
import { dayFromApi, listVehicles, todayInput } from '../../api/vehicle'
import {
  DEPOSIT_RATE,
  HUBS,
  SAMPLE_VEHICLE,
  SERVICE_CATALOG,
  SLOT_TIMES,
  WARRANTY_FEE,
  formatVnd,
  upcomingDays,
} from '../../demo/data'
import { useDemo } from '../../demo/demo-context'
import type { DemoSection } from './sections'

interface VehicleOption {
  id: number
  label: string
  plate: string
  inWarranty: boolean
  warrantyNote: string
}

function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h === 0) return `${m} min`
  return m === 0 ? `${h} h` : `${h} h ${m} min`
}

export function BookingPage({ onNavigate }: { onNavigate: (section: DemoSection) => void }) {
  const { state, confirmBooking } = useDemo()
  const [vehicles, setVehicles] = useState<VehicleOption[] | null>(null)
  const [vehicleId, setVehicleId] = useState<number | null>(null)
  const [hubId, setHubId] = useState<number>(HUBS[0].id)
  const [serviceIds, setServiceIds] = useState<number[]>([1, 2, 3])
  const [slot, setSlot] = useState<{ day: string; time: string } | null>(null)
  const [payDeposit, setPayDeposit] = useState(true)
  const [justBooked, setJustBooked] = useState(false)
  const days = useMemo(() => upcomingDays(), [])

  // Real vehicles from vehicle-service; the sample one keeps the demo usable
  // when the registry is empty or the service is down.
  useEffect(() => {
    let cancelled = false
    const today = todayInput()
    listVehicles({ status: 'ACTIVE' }, 0, 20)
      .then((page) => page.vehicles)
      .catch(() => [])
      .then((list) => {
        if (cancelled) return
        const options: VehicleOption[] = list.map((v) => {
          const end = v.warranty_end_date ? dayFromApi(v.warranty_end_date) : null
          const inWarranty = end !== null && end >= today
          return {
            id: v.id,
            label: v.license_plate ? `${v.license_plate} · VIN ${v.vin}` : `VIN ${v.vin}`,
            plate: v.license_plate || v.vin,
            inWarranty,
            warrantyNote: end
              ? inWarranty
                ? `Under warranty until ${end}`
                : `Warranty ended ${end}`
              : 'No warranty on record',
          }
        })
        const all = options.length > 0 ? options : [SAMPLE_VEHICLE]
        setVehicles(all)
        setVehicleId(all[0].id)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const vehicle = vehicles?.find((v) => v.id === vehicleId) ?? null
  const hub = HUBS.find((h) => h.id === hubId) ?? HUBS[0]
  const services = SERVICE_CATALOG.filter((s) => serviceIds.includes(s.id))
  const servicesTotal = services.reduce((a, s) => a + s.price, 0)
  const warrantyFee = vehicle && !vehicle.inWarranty ? WARRANTY_FEE : 0
  const estimate = servicesTotal + warrantyFee
  const deposit = payDeposit ? Math.round((estimate * DEPOSIT_RATE) / 1000) * 1000 : 0
  const minutes = services.reduce((a, s) => a + s.minutes, 0)
  const slotDay = days.find((d) => d.key === slot?.day)
  const canConfirm = vehicle !== null && services.length > 0 && slot !== null

  const wizard = [
    { label: 'Vehicle', done: vehicle !== null },
    { label: 'Hub', done: true },
    { label: 'Services', done: services.length > 0 },
    { label: 'Time', done: slot !== null },
  ]
  const current = wizard.findIndex((w) => !w.done)

  function toggleService(id: number) {
    setServiceIds((ids) => (ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]))
  }

  function confirm() {
    if (!vehicle || !slot || !slotDay) return
    confirmBooking({
      vehicleLabel: vehicle.label,
      plate: vehicle.plate,
      inWarranty: vehicle.inWarranty,
      hub,
      services,
      dayLabel: `${slotDay.weekday} ${slotDay.label}`,
      time: slot.time,
      estimate,
      deposit,
    })
    setJustBooked(true)
  }

  return (
    <div className="stack roomy demo">
      <div>
        <h2 className="demo-title">Book a service</h2>
        <p className="muted flush">
          Pick a vehicle, a hub, the work you need and a time. The slot is held for 10 minutes
          while you confirm.
        </p>
      </div>

      <ol className="wizard" aria-label="Booking steps">
        {wizard.map((w, i) => (
          <li key={w.label} className={w.done ? 'done' : i === current ? 'current' : ''}>
            <span className="wizard-dot">{w.done ? '✓' : i + 1}</span>
            {w.label}
          </li>
        ))}
      </ol>

      <div className="split">
        <div className="split-main stack">
          <section className="card stack">
            <label className="field">
              <span className="label">1 · Vehicle</span>
              <select
                value={vehicleId ?? ''}
                disabled={!vehicles}
                onChange={(e) => setVehicleId(Number(e.target.value))}
              >
                {!vehicles && <option>Loading vehicles…</option>}
                {vehicles?.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.label}
                  </option>
                ))}
              </select>
            </label>
            {vehicle && (
              <p className={`small flush ${vehicle.inWarranty ? 'good' : 'warn-text'}`}>
                {vehicle.warrantyNote}
              </p>
            )}
          </section>

          <section className="card stack">
            <label className="field">
              <span className="label">2 · Service hub</span>
              <select value={hubId} onChange={(e) => setHubId(Number(e.target.value))}>
                {HUBS.map((h) => (
                  <option key={h.id} value={h.id}>
                    {h.name}
                  </option>
                ))}
              </select>
            </label>
            <p className="small muted flush">
              {hub.bays} bays · {hub.technicians} technicians on shift
            </p>
          </section>

          <fieldset className="card stack demo-fieldset">
            <legend className="label">3 · Services</legend>
            {SERVICE_CATALOG.map((s) => {
              const checked = serviceIds.includes(s.id)
              return (
                <label key={s.id} className={`choice${checked ? ' selected' : ''}`}>
                  <input type="checkbox" checked={checked} onChange={() => toggleService(s.id)} />
                  <span className="choice-main">
                    <span className="strong">{s.name}</span>
                    <span className="small muted">
                      {s.minutes} min · needs {s.skill}
                    </span>
                  </span>
                  <span className="strong num">{formatVnd(s.price)}</span>
                </label>
              )
            })}
          </fieldset>

          <section className="card stack focus-card" aria-labelledby="slot-heading">
            <div className="section-head wrap">
              <h3 id="slot-heading" className="demo-h3">
                4 · Choose a time
              </h3>
              <span className="small muted">
                Slots with a free bay and a technician skilled in every service
              </span>
            </div>
            <div className="scroll-x">
              <div className="slot-grid">
                {days.map((d) => (
                  <div key={d.key} className="slot-day">
                    <div className="slot-day-head">
                      <strong>{d.weekday}</strong>
                      <span className="small muted">{d.label}</span>
                    </div>
                    {SLOT_TIMES.map((t, i) => {
                      const picked = slot?.day === d.key && slot.time === t
                      return (
                        <button
                          key={t}
                          type="button"
                          className={`slot${picked ? ' picked' : ''}`}
                          disabled={d.taken[i]}
                          aria-pressed={picked}
                          aria-label={`${d.weekday} ${d.label} ${t}${d.taken[i] ? ', taken' : ''}`}
                          onClick={() => setSlot({ day: d.key, time: t })}
                        >
                          {t}
                        </button>
                      )
                    })}
                  </div>
                ))}
              </div>
            </div>
          </section>
        </div>

        <aside className="split-side panel stack" aria-labelledby="summary-heading">
          <h3 id="summary-heading" className="demo-h3">
            Summary
          </h3>
          <dl className="kv">
            <dt>When</dt>
            <dd className="strong">{slotDay && slot ? `${slotDay.weekday} ${slotDay.label} · ${slot.time}` : '—'}</dd>
            <dt>Duration</dt>
            <dd>{minutes > 0 ? `about ${formatDuration(minutes)}` : '—'}</dd>
            <dt>Hub</dt>
            <dd>{hub.name}</dd>
          </dl>
          <hr className="rule" />
          <dl className="kv num">
            {services.map((s) => (
              <div key={s.id} className="kv-row">
                <dt className="plain">{s.name}</dt>
                <dd>{formatVnd(s.price)}</dd>
              </div>
            ))}
            <dt>Warranty fee</dt>
            <dd className={warrantyFee === 0 ? 'good' : ''}>
              {warrantyFee === 0 ? 'Waived' : formatVnd(warrantyFee)}
            </dd>
            <dt>Materials</dt>
            <dd className="muted">Billed after service</dd>
            <dt className="total">Estimate</dt>
            <dd className="total">{formatVnd(estimate)}</dd>
          </dl>
          <fieldset className="pay-choice">
            <legend className="small muted">Payment</legend>
            <label>
              <input type="radio" name="pay" checked={payDeposit} onChange={() => setPayDeposit(true)} />
              Pay a {Math.round(DEPOSIT_RATE * 100)}% deposit now ({formatVnd(Math.round((estimate * DEPOSIT_RATE) / 1000) * 1000)})
            </label>
            <label>
              <input type="radio" name="pay" checked={!payDeposit} onChange={() => setPayDeposit(false)} />
              Pay everything after service
            </label>
          </fieldset>
          <button type="button" className="primary big" disabled={!canConfirm} onClick={confirm}>
            Hold slot &amp; confirm
          </button>
          {justBooked && state.booking ? (
            <div className="message success flush stack">
              <span>
                Booking #{state.booking.id} confirmed for {state.booking.dayLabel} at{' '}
                {state.booking.time}.
              </span>
              <button type="button" className="secondary align-start" onClick={() => onNavigate('hub')}>
                Continue as technician →
              </button>
            </div>
          ) : (
            <p className="small muted flush">
              Confirming publishes <code>BookingConfirmed</code>; notification-service sends you the
              confirmation.
            </p>
          )}
        </aside>
      </div>
    </div>
  )
}
