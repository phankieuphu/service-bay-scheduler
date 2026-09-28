import { useEffect, useState } from 'react'
import { dayFromApi, getCustomerVehicles, type CustomerVehicle } from '../api/vehicle'

interface Props {
  customerId: number
  onOpenVehicle: (vehicleId: number) => void
}

export function CustomerVehicles({ customerId, onOpenVehicle }: Props) {
  const [vehicles, setVehicles] = useState<CustomerVehicle[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    getCustomerVehicles(customerId)
      .then((res) => {
        if (!cancelled) setVehicles(res.vehicles)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load vehicles.')
      })
    return () => {
      cancelled = true
    }
  }, [customerId])

  return (
    <div className="subsection">
      <div className="subsection-header">
        <h3>Vehicles</h3>
        <span className="muted small">Currently owned</span>
      </div>
      {error && <p className="message error flush">Couldn’t load vehicles: {error}</p>}
      {!error && vehicles === null && <p className="muted small">Loading…</p>}
      {vehicles?.length === 0 && (
        <p className="muted small">
          No vehicles yet. Register one from Vehicles and assign this customer as its first owner.
        </p>
      )}
      {vehicles && vehicles.length > 0 && (
        <ul className="item-list">
          {vehicles.map((vehicle) => (
            <li key={vehicle.id} className="item">
              <div className="item-main">
                <span className="strong">{vehicle.license_plate || 'No plate'}</span>
                <span className="mono muted small">{vehicle.vin}</span>
              </div>
              <div className="item-meta">
                <span>since {dayFromApi(vehicle.owned_from)}</span>
                <button type="button" className="link" onClick={() => onOpenVehicle(vehicle.id)}>
                  Open
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
