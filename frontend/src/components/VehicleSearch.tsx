import { useEffect, useState, type FormEvent } from 'react'
import {
  listVehicles,
  VEHICLE_STATUSES,
  type Vehicle,
  type VehicleFilters,
  type VehiclePage,
  type VehicleStatus,
} from '../api/vehicle'

const PAGE_SIZE = 10

interface Props {
  selectedId: number | null
  updated: Vehicle | null
  onSelect: (vehicleId: number) => void
}

export function VehicleSearch({ selectedId, updated, onSelect }: Props) {
  const [plate, setPlate] = useState('')
  const [vin, setVin] = useState('')
  const [status, setStatus] = useState<VehicleStatus | ''>('')
  // The filters the current results were fetched with; "Load more" pages
  // through these, not whatever is typed in the form now.
  const [filters, setFilters] = useState<VehicleFilters>({})
  const [vehicles, setVehicles] = useState<Vehicle[]>([])
  const [nextCursor, setNextCursor] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  function applyPage(page: VehiclePage, cursor: number) {
    setVehicles((prev) => (cursor === 0 ? page.vehicles : [...prev, ...page.vehicles]))
    setNextCursor(page.has_more && page.next_cursor ? page.next_cursor : null)
  }

  function applyError(err: unknown) {
    setError(err instanceof Error ? err.message : 'Failed to load vehicles.')
  }

  async function load(nextFilters: VehicleFilters, cursor: number) {
    setError(null)
    setLoading(true)
    try {
      applyPage(await listVehicles(nextFilters, cursor, PAGE_SIZE), cursor)
      setFilters(nextFilters)
    } catch (err) {
      applyError(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    listVehicles({}, 0, PAGE_SIZE)
      .then((page) => applyPage(page, 0))
      .catch(applyError)
      .finally(() => setLoading(false))
  }, [])

  function handleSearch(event: FormEvent) {
    event.preventDefault()
    void load(
      { plate: plate.trim() || undefined, vin: vin.trim() || undefined, status: status || undefined },
      0,
    )
  }

  return (
    <section className="panel stack" aria-labelledby="vehicle-search-heading">
      <h2 id="vehicle-search-heading">Find a vehicle</h2>
      <form className="field-grid" onSubmit={handleSearch}>
        <label className="field" htmlFor="search-plate">
          License plate
          <input id="search-plate" value={plate} onChange={(e) => setPlate(e.target.value)} />
        </label>
        <label className="field" htmlFor="search-vin">
          VIN
          <input
            id="search-vin"
            placeholder="17 characters"
            value={vin}
            onChange={(e) => setVin(e.target.value)}
          />
        </label>
        <label className="field" htmlFor="search-status">
          Status
          <select
            id="search-status"
            value={status}
            onChange={(e) => setStatus(e.target.value as VehicleStatus | '')}
          >
            <option value="">Any</option>
            {VEHICLE_STATUSES.map((s) => (
              <option key={s}>{s}</option>
            ))}
          </select>
        </label>
        <button type="submit" disabled={loading}>
          Search
        </button>
      </form>
      <p className="muted small flush">
        Plates match however they’re typed: “51a-123.45” finds 51A12345.
      </p>

      {error && <p className="message error flush">{error}</p>}

      <div className="row-list vehicles" role="list" aria-label="Results">
        <div className="row-head" aria-hidden="true">
          <span>Plate</span>
          <span>VIN</span>
          <span>Status</span>
        </div>
        {vehicles.map((listed) => {
          const vehicle = updated?.id === listed.id ? updated : listed
          return (
            <div role="listitem" key={vehicle.id}>
              <button
                type="button"
                className={`row${selectedId === vehicle.id ? ' selected' : ''}`}
                aria-pressed={selectedId === vehicle.id}
                onClick={() => onSelect(vehicle.id)}
              >
                <span className="strong">{vehicle.license_plate || 'No plate'}</span>
                <span className="mono small truncate">{vehicle.vin}</span>
                <span>
                  <span className={`badge badge-${vehicle.status.toLowerCase()}`}>
                    {vehicle.status}
                  </span>
                </span>
              </button>
            </div>
          )
        })}
        {!loading && vehicles.length === 0 && <p className="muted empty">No vehicles found.</p>}
      </div>

      <div className="actions">
        <span className="muted small">
          Showing {vehicles.length}
          {nextCursor !== null ? ' so far' : ''}
        </span>
        {nextCursor !== null && (
          <button
            type="button"
            className="secondary push-right"
            disabled={loading}
            onClick={() => load(filters, nextCursor)}
          >
            {loading ? 'Loading…' : 'Load more'}
          </button>
        )}
      </div>
    </section>
  )
}
