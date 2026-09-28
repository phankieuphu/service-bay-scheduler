import { useEffect, useState, type FormEvent } from 'react'
import {
  dayFromApi,
  getServiceHistory,
  getVehicleMaterials,
  getWarranty,
  todayInput,
  type ServiceHistoryEntry,
  type VehicleMaterial,
  type Warranty,
} from '../api/vehicle'

type RecordTab = 'warranty' | 'materials' | 'history'

const records: { id: RecordTab; label: string }[] = [
  { id: 'warranty', label: 'Warranty' },
  { id: 'materials', label: 'Materials' },
  { id: 'history', label: 'Service history' },
]

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : 'Something went wrong.'
}

// useLoad fetches load(vehicleId) and tracks its result; data is null while
// loading. load must be a stable (module-level) function.
function useLoad<T>(
  load: (vehicleId: number) => Promise<T>,
  vehicleId: number,
): { data: T | null; error: string | null } {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => {
    let cancelled = false
    load(vehicleId)
      .then((result) => {
        if (!cancelled) setData(result)
      })
      .catch((err) => {
        if (!cancelled) setError(errorText(err))
      })
    return () => {
      cancelled = true
    }
  }, [load, vehicleId])
  return { data, error }
}

export function VehicleRecords({ vehicleId }: { vehicleId: number }) {
  const [active, setActive] = useState<RecordTab>('warranty')

  return (
    <div className="subsection">
      <div className="record-tabs" role="tablist" aria-label="Vehicle records">
        {records.map((record) => (
          <button
            key={record.id}
            id={`record-tab-${record.id}`}
            type="button"
            role="tab"
            aria-selected={active === record.id}
            aria-controls="record-panel"
            className={active === record.id ? 'active' : ''}
            onClick={() => setActive(record.id)}
          >
            {record.label}
          </button>
        ))}
      </div>
      <div id="record-panel" role="tabpanel" aria-labelledby={`record-tab-${active}`}>
        {active === 'warranty' && <WarrantyRecord vehicleId={vehicleId} />}
        {active === 'materials' && <MaterialsRecord vehicleId={vehicleId} />}
        {active === 'history' && <HistoryRecord vehicleId={vehicleId} />}
      </div>
    </div>
  )
}

function WarrantyRecord({ vehicleId }: { vehicleId: number }) {
  const [date, setDate] = useState(todayInput())
  const [warranty, setWarranty] = useState<Warranty | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  async function check(day: string) {
    setError(null)
    setLoading(true)
    try {
      setWarranty(await getWarranty(vehicleId, day))
    } catch (err) {
      setError(errorText(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    getWarranty(vehicleId, todayInput())
      .then(setWarranty)
      .catch((err) => setError(errorText(err)))
      .finally(() => setLoading(false))
  }, [vehicleId])

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (date) void check(date)
  }

  const endDay = dayFromApi(warranty?.warranty_end_date ?? null)

  return (
    <div className="stack">
      {warranty && (
        <div className="actions">
          <span className={`badge ${warranty.in_warranty ? 'badge-active' : 'badge-warn'}`}>
            {warranty.in_warranty ? 'In warranty' : endDay ? 'Expired' : 'No warranty'}
          </span>
          <span className="warranty-text">
            {warranty.in_warranty
              ? `Ends ${endDay} · ${warranty.days_remaining} days left`
              : `${endDay ? `Ended ${endDay}` : 'No warranty on record'} — a warranty fee applies to service on ${warranty.as_of}`}
          </span>
        </div>
      )}
      <form className="inline-form" onSubmit={handleSubmit}>
        <label className="field" htmlFor="warranty-date">
          Check on date
          <input
            id="warranty-date"
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            required
          />
        </label>
        <button type="submit" className="secondary" disabled={loading}>
          {loading ? 'Checking…' : 'Check'}
        </button>
      </form>
      {error && <p className="message error flush">{error}</p>}
      <p className="muted small flush">
        Billing checks the warranty on the day the service was completed. A warranty covers its
        whole last day.
      </p>
    </div>
  )
}

function MaterialsRecord({ vehicleId }: { vehicleId: number }) {
  const { data, error } = useLoad<{ materials: VehicleMaterial[] }>(getVehicleMaterials, vehicleId)

  if (error) return <p className="message error flush">{error}</p>
  if (!data) return <p className="muted small">Loading…</p>
  if (data.materials.length === 0) return <p className="muted small">No materials installed yet.</p>

  return (
    <ul className="item-list">
      {data.materials.map((material) => (
        <li key={material.id} className="item material">
          <span>
            {material.description || 'Material'}{' '}
            <span className="muted small">#{material.material_id}</span>
          </span>
          <span className="muted">×{material.count}</span>
          <span className="muted small">
            {new Date(material.installed_at).toLocaleDateString()}
          </span>
        </li>
      ))}
    </ul>
  )
}

function HistoryRecord({ vehicleId }: { vehicleId: number }) {
  const { data, error } = useLoad<{ history: ServiceHistoryEntry[] }>(getServiceHistory, vehicleId)

  if (error) return <p className="message error flush">{error}</p>
  if (!data) return <p className="muted small">Loading…</p>
  if (data.history.length === 0) return <p className="muted small">No completed services yet.</p>

  return (
    <ol className="item-list">
      {data.history.map((entry) => (
        <li key={entry.id} className="item history">
          <div className="item-row">
            <span className="strong">
              {entry.services.map((s) => s.name).join(', ') || 'Service'}
            </span>
            <span className="muted small">{new Date(entry.completed_at).toLocaleDateString()}</span>
          </div>
          <span className="muted small">
            Appointment #{entry.appointment_id} · Hub #{entry.dealership_id}
          </span>
        </li>
      ))}
    </ol>
  )
}
