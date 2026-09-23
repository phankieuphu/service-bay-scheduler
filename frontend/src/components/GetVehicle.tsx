import { useState, type FormEvent } from 'react'
import { ApiError, getVehicle, type Vehicle } from '../api/vehicle'

export function GetVehicle() {
  const [vehicleId, setVehicleId] = useState('')
  const [vehicle, setVehicle] = useState<Vehicle | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setVehicle(null)

    const id = Number(vehicleId)
    if (!Number.isInteger(id) || id <= 0) {
      setError('Enter a valid vehicle ID.')
      return
    }

    setLoading(true)
    try {
      setVehicle(await getVehicle(id))
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError(`No vehicle found with ID ${id}.`)
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="panel">
      <h2>Get Vehicle</h2>
      <form className="form-row" onSubmit={handleSubmit}>
        <label htmlFor="vehicle-id">Vehicle ID</label>
        <input
          id="vehicle-id"
          type="number"
          min={1}
          value={vehicleId}
          onChange={(e) => setVehicleId(e.target.value)}
          placeholder="e.g. 1"
          required
        />
        <button type="submit" disabled={loading}>
          {loading ? 'Looking up…' : 'Look up'}
        </button>
      </form>

      {error && <p className="message error">{error}</p>}

      {vehicle && (
        <table className="details">
          <tbody>
            <tr>
              <th>ID</th>
              <td>{vehicle.id}</td>
            </tr>
            <tr>
              <th>VIN</th>
              <td>{vehicle.vin}</td>
            </tr>
            <tr>
              <th>License plate</th>
              <td>{vehicle.license_plate}</td>
            </tr>
            <tr>
              <th>Status</th>
              <td>
                <span className={`badge badge-${vehicle.status.toLowerCase()}`}>
                  {vehicle.status}
                </span>
              </td>
            </tr>
            <tr>
              <th>Warranty end date</th>
              <td>{new Date(vehicle.warranty_end_date).toLocaleDateString()}</td>
            </tr>
            <tr>
              <th>Created</th>
              <td>{new Date(vehicle.created_at).toLocaleString()}</td>
            </tr>
            <tr>
              <th>Updated</th>
              <td>{new Date(vehicle.updated_at).toLocaleString()}</td>
            </tr>
          </tbody>
        </table>
      )}
    </div>
  )
}
