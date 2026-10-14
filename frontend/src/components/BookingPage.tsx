import { useEffect, useMemo, useState } from 'react'
import type { Customer } from '../api/customer'
import {
  ApiError,
  bookAppointment,
  formatDate,
  formatTime,
  getAvailability,
  listHubs,
  listServices,
  todayIn,
  type Appointment,
  type Availability,
  type Hub,
  type ServiceOffer,
} from '../api/scheduler'
import { getCustomerVehicles, type CustomerVehicle } from '../api/vehicle'
import { PageHeader } from './PageHeader'

interface Props {
  customer: Customer
  onViewAppointments: () => void
}

interface Catalog {
  vehicles: CustomerVehicle[]
  hubs: Hub[]
  services: ServiceOffer[]
}

// scheduler-service prefixes its messages with the error kind ("conflict:
// no bay is free at that time"); the kind is already clear from context.
function errorMessage(err: unknown, fallback: string): string {
  if (!(err instanceof Error)) return fallback
  return err.message.replace(/^(conflict|invalid input|not found): /, '')
}

export function BookingPage({ customer, onViewAppointments }: Props) {
  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [vehicleId, setVehicleId] = useState<number | null>(null)
  const [hubId, setHubId] = useState<number | null>(null)
  const [serviceIds, setServiceIds] = useState<number[]>([])
  const [date, setDate] = useState('')
  // Bumped to refetch availability after a booking attempt.
  const [availabilityVersion, setAvailabilityVersion] = useState(0)
  // Availability and the picked slot belong to one request (hub, date,
  // services, version); changing any of them leaves them stale, and they're
  // ignored until the matching response arrives.
  const requestKey =
    hubId !== null && date && serviceIds.length > 0
      ? `${hubId}|${date}|${serviceIds.join(',')}|${availabilityVersion}`
      : null
  const [result, setResult] = useState<
    { key: string; availability: Availability } | { key: string; error: string } | null
  >(null)
  const [picked, setPicked] = useState<{ key: string; start: string } | null>(null)
  const current = result && result.key === requestKey ? result : null
  const availability = current && 'availability' in current ? current.availability : null
  const availabilityError = current && 'error' in current ? current.error : null
  const slotStart = picked && picked.key === requestKey ? picked.start : null

  const [booking, setBooking] = useState(false)
  const [bookError, setBookError] = useState<string | null>(null)
  const [booked, setBooked] = useState<Appointment | null>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([getCustomerVehicles(customer.id), listHubs(), listServices()])
      .then(([v, h, s]) => {
        if (cancelled) return
        setCatalog({ vehicles: v.vehicles, hubs: h.hubs, services: s.services })
        if (v.vehicles.length > 0) setVehicleId(v.vehicles[0].id)
        if (h.hubs.length > 0) {
          setHubId(h.hubs[0].id)
          setDate(todayIn(h.hubs[0].timezone))
        }
      })
      .catch((err) => {
        if (!cancelled) setLoadError(errorMessage(err, 'Failed to load booking options.'))
      })
    return () => {
      cancelled = true
    }
  }, [customer.id])

  const hub = catalog?.hubs.find((h) => h.id === hubId) ?? null
  const selectedServices = useMemo(
    () => catalog?.services.filter((s) => serviceIds.includes(s.id)) ?? [],
    [catalog, serviceIds],
  )
  const total = selectedServices.reduce((sum, s) => sum + Number(s.fee), 0)
  const minutes = selectedServices.reduce((sum, s) => sum + s.duration_minutes, 0)

  useEffect(() => {
    if (requestKey === null || hubId === null) return

    let cancelled = false
    getAvailability(hubId, date, serviceIds)
      .then((res) => {
        if (!cancelled) setResult({ key: requestKey, availability: res })
      })
      .catch((err) => {
        if (!cancelled) setResult({ key: requestKey, error: errorMessage(err, 'Failed to load free times.') })
      })
    return () => {
      cancelled = true
    }
  }, [requestKey, hubId, date, serviceIds])

  function toggleService(id: number) {
    setBooked(null)
    setServiceIds((prev) => (prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id]))
  }

  async function handleBook() {
    if (vehicleId === null || hubId === null || slotStart === null) return
    setBooking(true)
    setBookError(null)
    setBooked(null)
    try {
      const appointment = await bookAppointment({
        hub_id: hubId,
        vehicle_id: vehicleId,
        service_ids: serviceIds,
        start: slotStart,
      })
      setBooked(appointment)
      setServiceIds([])
    } catch (err) {
      const message = errorMessage(err, 'Booking failed.')
      setBookError(err instanceof ApiError && err.status === 409 ? `${message}. Pick another time.` : message)
      setAvailabilityVersion((v) => v + 1)
    } finally {
      setBooking(false)
    }
  }

  const header = (
    <PageHeader
      title="Book a service"
      description="Choose your vehicle, a hub and the services you need, then pick a free time."
    />
  )

  if (loadError) {
    return (
      <div className="stack roomy">
        {header}
        <p className="message error flush">{loadError}</p>
      </div>
    )
  }
  if (!catalog) {
    return (
      <div className="stack roomy">
        {header}
        <p className="muted">Loading…</p>
      </div>
    )
  }
  if (catalog.vehicles.length === 0) {
    return (
      <div className="stack roomy">
        {header}
        <p className="message info">
          You don’t have a vehicle registered to you yet. Ask the dealership to register your
          vehicle with your name, then come back to book.
        </p>
      </div>
    )
  }

  const freeSlots = availability?.slots.filter((s) => s.available).length ?? 0

  return (
    <div className="stack roomy">
      {header}
      <div className="two-col">
        <section className="card stack" aria-labelledby="booking-what-heading">
          <h2 id="booking-what-heading">What and where</h2>

          <label className="field">
            Vehicle
            <select
              value={vehicleId ?? ''}
              onChange={(e) => {
                setBooked(null)
                setVehicleId(Number(e.target.value))
              }}
            >
              {catalog.vehicles.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.license_plate || 'No plate'} · {v.vin}
                </option>
              ))}
            </select>
          </label>

          <label className="field">
            Service hub
            <select
              value={hubId ?? ''}
              onChange={(e) => {
                setBooked(null)
                setHubId(Number(e.target.value))
              }}
            >
              {catalog.hubs.map((h) => (
                <option key={h.id} value={h.id}>
                  {h.name}
                </option>
              ))}
            </select>
          </label>
          {hub && (
            <p className="muted small flush">
              {hub.address} · open {hub.open_hour}–{hub.close_hour} · {hub.bay_count} bays
            </p>
          )}

          <fieldset className="choice-list">
            <legend className="label">Services</legend>
            {catalog.services.map((s) => (
              <label key={s.id} className={`choice${serviceIds.includes(s.id) ? ' active' : ''}`}>
                <input
                  type="checkbox"
                  checked={serviceIds.includes(s.id)}
                  onChange={() => toggleService(s.id)}
                />
                <span className="choice-main">
                  <span className="strong">{s.name}</span>
                  <span className="muted small">
                    {s.service_type} · {s.duration_minutes} min
                  </span>
                </span>
                <span className="choice-price">{s.fee}</span>
              </label>
            ))}
          </fieldset>

          <label className="field">
            Date
            <input
              type="date"
              value={date}
              min={hub ? todayIn(hub.timezone) : undefined}
              onChange={(e) => {
                setBooked(null)
                setDate(e.target.value)
              }}
            />
          </label>
        </section>

        <section className="card stack" aria-labelledby="booking-when-heading">
          <h2 id="booking-when-heading">When</h2>

          {serviceIds.length === 0 && (
            <p className="muted flush">Pick at least one service to see free times.</p>
          )}
          {availabilityError && <p className="message error flush">{availabilityError}</p>}
          {serviceIds.length > 0 && !availability && !availabilityError && (
            <p className="muted flush">Loading free times…</p>
          )}
          {availability && hub && (
            <>
              <p className="muted small flush">
                {formatDate(`${availability.date}T12:00:00Z`, availability.timezone)} ·{' '}
                {availability.duration_minutes} min · {freeSlots} free{' '}
                {freeSlots === 1 ? 'time' : 'times'} · times in {availability.timezone}
              </p>
              {availability.slots.length === 0 ? (
                <p className="muted flush">These services take longer than the hub is open.</p>
              ) : (
                <div className="slot-grid" role="radiogroup" aria-label="Start time">
                  {availability.slots.map((slot) => (
                    <button
                      key={slot.start}
                      type="button"
                      role="radio"
                      aria-checked={slotStart === slot.start}
                      className={`slot${slotStart === slot.start ? ' active' : ''}`}
                      disabled={!slot.available}
                      onClick={() => {
                        setBooked(null)
                        if (requestKey) setPicked({ key: requestKey, start: slot.start })
                      }}
                    >
                      {formatTime(slot.start, availability.timezone)}
                    </button>
                  ))}
                </div>
              )}
            </>
          )}

          {selectedServices.length > 0 && (
            <div className="inset stack">
              <div className="item-row">
                <span className="strong">Estimated total</span>
                <span className="strong">{total.toFixed(2)}</span>
              </div>
              <span className="muted small">
                {selectedServices.map((s) => s.name).join(', ')} · {minutes} min
                {slotStart && availability
                  ? ` · ${formatTime(slotStart, availability.timezone)} on ${formatDate(slotStart, availability.timezone)}`
                  : ''}
              </span>
            </div>
          )}

          <div className="actions">
            <button type="button" disabled={slotStart === null || booking} onClick={handleBook}>
              {booking ? 'Booking…' : 'Confirm booking'}
            </button>
          </div>

          {bookError && <p className="message error flush">{bookError}</p>}
          {booked && hub && (
            <div className="message success confirm flush">
              <span>
                Booked #{booked.id}: {formatTime(booked.start, hub.timezone)} on{' '}
                {formatDate(booked.start, hub.timezone)} at {hub.name}.
              </span>
              <button type="button" className="secondary" onClick={onViewAppointments}>
                View my appointments
              </button>
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
