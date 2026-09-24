import { useState, type FormEvent } from 'react'
import { ApiError, transferVehicle } from '../api/vehicle'

interface FormState {
  vehicleId: string
  from: string
  to: string
  date: string
}

const initialState: FormState = { vehicleId: '', from: '', to: '', date: '' }

export function TransferVehicle() {
  const [form, setForm] = useState<FormState>(initialState)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function updateField<K extends keyof FormState>(key: K, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    const vehicleId = Number(form.vehicleId)
    const from = Number(form.from)
    const to = Number(form.to)

    if (![vehicleId, from, to].every((n) => Number.isInteger(n) && n > 0)) {
      setError('Vehicle ID, From customer ID, and To customer ID must all be positive numbers.')
      return
    }
    if (from === to) {
      setError('From and To customers must be different.')
      return
    }

    setLoading(true)
    try {
      await transferVehicle({
        vehicle_id: vehicleId,
        from,
        to,
        date: form.date ? new Date(form.date).toISOString() : undefined,
      })
      setSuccess(`Vehicle ${vehicleId} transferred from customer ${from} to customer ${to}.`)
      setForm(initialState)
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('Vehicle or customer not found.')
      } else if (err instanceof ApiError && err.status === 409) {
        setError('Transfer conflicts with the vehicle’s current ownership state — it may have just been transferred by another request.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="panel">
      <h2>Transfer Vehicle</h2>
      <form className="form-grid" onSubmit={handleSubmit}>
        <label htmlFor="transfer-vehicle-id">Vehicle ID</label>
        <input
          id="transfer-vehicle-id"
          type="number"
          min={1}
          value={form.vehicleId}
          onChange={(e) => updateField('vehicleId', e.target.value)}
          required
        />

        <label htmlFor="transfer-from">From customer ID</label>
        <input
          id="transfer-from"
          type="number"
          min={1}
          value={form.from}
          onChange={(e) => updateField('from', e.target.value)}
          required
        />

        <label htmlFor="transfer-to">To customer ID</label>
        <input
          id="transfer-to"
          type="number"
          min={1}
          value={form.to}
          onChange={(e) => updateField('to', e.target.value)}
          required
        />

        <label htmlFor="transfer-date">Date (optional)</label>
        <input
          id="transfer-date"
          type="datetime-local"
          value={form.date}
          onChange={(e) => updateField('date', e.target.value)}
        />

        <button type="submit" disabled={loading} className="span-2">
          {loading ? 'Transferring…' : 'Transfer'}
        </button>
      </form>

      {error && <p className="message error">{error}</p>}
      {success && <p className="message success">{success}</p>}
    </div>
  )
}
