import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  dayFromApi,
  dayToApi,
  getVehicle,
  updateVehicle,
  VEHICLE_STATUSES,
  type Vehicle,
  type VehicleStatus,
} from '../api/vehicle'
import { VehicleOwner } from './VehicleOwner'
import { VehicleRecords } from './VehicleRecords'

interface Props {
  vehicleId: number
  onUpdated: (vehicle: Vehicle) => void
}

export function VehicleDetail({ vehicleId, onUpdated }: Props) {
  const [vehicle, setVehicle] = useState<Vehicle | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [notice, setNotice] = useState<{ kind: 'success' | 'error'; text: string } | null>(null)

  useEffect(() => {
    let cancelled = false
    getVehicle(vehicleId)
      .then((v) => {
        if (!cancelled) setVehicle(v)
      })
      .catch((err) => {
        if (cancelled) return
        setLoadError(
          err instanceof ApiError && err.status === 404
            ? `No vehicle found with ID ${vehicleId}.`
            : err instanceof Error
              ? err.message
              : 'Something went wrong.',
        )
      })
    return () => {
      cancelled = true
    }
  }, [vehicleId])

  function handleSaved(saved: Vehicle, reloaded: boolean) {
    setVehicle(saved)
    onUpdated(saved)
    setNotice(
      reloaded
        ? {
            kind: 'error',
            text: 'This vehicle was changed by someone else since you opened it. The latest version has been loaded — review and save again.',
          }
        : { kind: 'success', text: 'Vehicle updated.' },
    )
  }

  if (loadError) {
    return (
      <section className="card">
        <p className="message error flush">{loadError}</p>
      </section>
    )
  }
  if (!vehicle) {
    return (
      <section className="card">
        <p className="muted">Loading vehicle…</p>
      </section>
    )
  }

  const scrapped = vehicle.status === 'SCRAPPED'

  return (
    <section className="card stack roomy" aria-labelledby="vehicle-detail-heading">
      <div className="detail-header">
        <div className="title-block">
          <h2 id="vehicle-detail-heading">
            {vehicle.license_plate || 'No plate'} <span className="muted">#{vehicle.id}</span>
          </h2>
          <span className="mono muted small">
            {vehicle.vin} · Model #{vehicle.vehicle_model_id}
          </span>
        </div>
        <span className={`badge badge-${vehicle.status.toLowerCase()}`}>{vehicle.status}</span>
      </div>

      {scrapped ? (
        <p className="message error flush">
          This vehicle is scrapped. It can no longer be edited or given an owner.
        </p>
      ) : (
        // key: reset the form's fields whenever a save (or reload) changes the vehicle
        <EditVehicleForm
          key={vehicle.updated_at}
          vehicle={vehicle}
          onSaved={handleSaved}
          onEdit={() => setNotice(null)}
        />
      )}
      {notice && <p className={`message ${notice.kind} flush`}>{notice.text}</p>}

      {!scrapped && <VehicleOwner vehicleId={vehicle.id} />}

      <VehicleRecords key={vehicle.updated_at} vehicleId={vehicle.id} />
    </section>
  )
}

function EditVehicleForm({
  vehicle,
  onSaved,
  onEdit,
}: {
  vehicle: Vehicle
  onSaved: (vehicle: Vehicle, reloaded: boolean) => void
  onEdit: () => void
}) {
  const [status, setStatus] = useState<VehicleStatus>(vehicle.status)
  const [warranty, setWarranty] = useState(dayFromApi(vehicle.warranty_end_date))
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    onEdit()

    const statusChanged = status !== vehicle.status
    // An emptied date field means "leave as is": the API can't clear a warranty.
    const warrantyChanged = warranty !== '' && warranty !== dayFromApi(vehicle.warranty_end_date)
    if (!statusChanged && !warrantyChanged) {
      setError('Nothing to update.')
      return
    }

    setLoading(true)
    try {
      onSaved(
        await updateVehicle(vehicle.id, {
          status: statusChanged ? status : undefined,
          warranty_end_date: warrantyChanged ? dayToApi(warranty) : undefined,
          updated_at: vehicle.updated_at,
        }),
        false,
      )
    } catch (err) {
      setLoading(false)
      if (err instanceof ApiError && err.status === 409) {
        // Either the vehicle changed since it was loaded, or it's been
        // scrapped: reload so the pane shows the current state either way.
        try {
          const latest = await getVehicle(vehicle.id)
          if (latest.updated_at !== vehicle.updated_at) {
            onSaved(latest, true)
            return
          }
        } catch {
          // fall through to the API's message
        }
      }
      setError(err instanceof Error ? err.message : 'Something went wrong.')
    }
  }

  return (
    <form className="field-grid three inset" onSubmit={handleSubmit}>
      <label className="field" htmlFor="edit-vehicle-status">
        Status
        <select
          id="edit-vehicle-status"
          value={status}
          onChange={(e) => setStatus(e.target.value as VehicleStatus)}
        >
          {VEHICLE_STATUSES.map((s) => (
            <option key={s}>{s}</option>
          ))}
        </select>
      </label>
      <label className="field" htmlFor="edit-vehicle-warranty">
        Warranty ends
        <input
          id="edit-vehicle-warranty"
          type="date"
          value={warranty}
          onChange={(e) => setWarranty(e.target.value)}
        />
      </label>
      <button type="submit" disabled={loading}>
        {loading ? 'Saving…' : 'Save changes'}
      </button>
      <span className="hint span-all">
        {status === 'SCRAPPED' && vehicle.status !== 'SCRAPPED'
          ? 'Scrapping is final: the vehicle can’t be edited or given an owner afterwards.'
          : 'If someone else changed this vehicle since you opened it, it’s reloaded and you can save again.'}
      </span>
      {error && <p className="message error flush span-all">{error}</p>}
    </form>
  )
}
