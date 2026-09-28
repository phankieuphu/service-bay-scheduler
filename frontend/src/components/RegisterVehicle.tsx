import { useState, type FormEvent } from 'react'
import {
  ApiError,
  dayToApi,
  registerVehicle,
  VEHICLE_STATUSES,
  type Vehicle,
  type VehicleStatus,
} from '../api/vehicle'

interface FormState {
  vin: string
  plate: string
  modelId: string
  warranty: string
  status: VehicleStatus
}

const initialState: FormState = { vin: '', plate: '', modelId: '', warranty: '', status: 'ACTIVE' }

// A VIN is 17 characters from A-Z and 0-9, never I, O or Q.
const VIN_PATTERN = /^[A-HJ-NPR-Z0-9]{17}$/

export function RegisterVehicle({ onOpen }: { onOpen: (vehicleId: number) => void }) {
  const [form, setForm] = useState<FormState>(initialState)
  const [vinError, setVinError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [registered, setRegistered] = useState<Vehicle | null>(null)
  const [loading, setLoading] = useState(false)

  function updateField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setVinError(null)
    setError(null)
    setRegistered(null)

    const vin = form.vin.trim().toUpperCase()
    if (!VIN_PATTERN.test(vin)) {
      setVinError('A VIN is 17 letters and digits, without I, O or Q.')
      return
    }
    const modelId = Number(form.modelId)
    if (!Number.isInteger(modelId) || modelId <= 0) {
      setError('Enter a valid model ID.')
      return
    }

    setLoading(true)
    try {
      const created = await registerVehicle({
        vin,
        vehicle_model_id: modelId,
        license_plate: form.plate.trim() || undefined,
        warranty_end_date: form.warranty ? dayToApi(form.warranty) : undefined,
        status: form.status,
      })
      setRegistered(created)
      setForm(initialState)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        if (/\bVIN\b/.test(err.message)) setVinError('A vehicle with this VIN is already registered.')
        else setError('A vehicle with this license plate is already registered.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="panel narrow stack" aria-labelledby="register-vehicle-heading">
      <h2 id="register-vehicle-heading">Register a vehicle</h2>
      <form className="form-grid" onSubmit={handleSubmit}>
        <label htmlFor="register-vin">VIN</label>
        <div className="field-with-error">
          <input
            id="register-vin"
            className={`mono${vinError ? ' invalid' : ''}`}
            value={form.vin}
            onChange={(e) => updateField('vin', e.target.value)}
            aria-invalid={vinError !== null}
            aria-describedby={vinError ? 'register-vin-error' : undefined}
            required
          />
          {vinError && (
            <span id="register-vin-error" className="field-error">
              {vinError}
            </span>
          )}
        </div>

        <label htmlFor="register-plate">License plate (optional)</label>
        <input
          id="register-plate"
          placeholder="e.g. 51A-123.45"
          value={form.plate}
          onChange={(e) => updateField('plate', e.target.value)}
        />

        <label htmlFor="register-model">Model ID</label>
        <input
          id="register-model"
          type="number"
          min={1}
          value={form.modelId}
          onChange={(e) => updateField('modelId', e.target.value)}
          required
        />

        <label htmlFor="register-warranty">Warranty ends (optional)</label>
        <input
          id="register-warranty"
          type="date"
          value={form.warranty}
          onChange={(e) => updateField('warranty', e.target.value)}
        />

        <label htmlFor="register-status">Status</label>
        <select
          id="register-status"
          value={form.status}
          onChange={(e) => updateField('status', e.target.value as VehicleStatus)}
        >
          {VEHICLE_STATUSES.map((s) => (
            <option key={s}>{s}</option>
          ))}
        </select>

        <div className="span-2 actions">
          <button type="submit" disabled={loading}>
            {loading ? 'Registering…' : 'Register'}
          </button>
          <span className="muted small">
            VINs and plates are saved in capitals; plates keep only letters and digits.
          </span>
        </div>
      </form>

      {error && <p className="message error flush">{error}</p>}
      {registered && (
        <div className="message success confirm flush">
          <span>Vehicle #{registered.id} registered. Next: assign its first owner.</span>
          <button type="button" className="secondary" onClick={() => onOpen(registered.id)}>
            Open vehicle
          </button>
        </div>
      )}
    </section>
  )
}
