import { useState, type FormEvent } from 'react'
import { ApiError, assignOwner, dayToApi, todayInput, transferVehicle } from '../api/vehicle'

type Mode = 'assign' | 'transfer'

function positiveId(value: string): number | null {
  const id = Number(value)
  return Number.isInteger(id) && id > 0 ? id : null
}

// VehicleOwner changes who owns the vehicle. The API doesn't report the
// current owner of a vehicle (only a customer's vehicles), so both actions
// are offered: assigning a first owner 409s if there already is one, and the
// form then switches to a transfer.
export function VehicleOwner({ vehicleId }: { vehicleId: number }) {
  const [mode, setMode] = useState<Mode>('assign')
  const [customerId, setCustomerId] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [date, setDate] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function switchMode(next: Mode) {
    setMode(next)
    setError(null)
    setSuccess(null)
  }

  async function handleAssign(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    const id = positiveId(customerId)
    if (id === null) {
      setError('Enter a valid customer ID.')
      return
    }

    setLoading(true)
    try {
      await assignOwner(vehicleId, { customer_id: id })
      setSuccess(`Customer ${id} is now the owner.`)
      setCustomerId('')
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setMode('transfer')
        setTo(String(id))
        setError('This vehicle already has an owner. Transfer it instead.')
      } else if (err instanceof ApiError && err.status === 404) {
        setError('Vehicle or customer not found.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleTransfer(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    const fromId = positiveId(from)
    const toId = positiveId(to)
    if (fromId === null || toId === null) {
      setError('From and To customer IDs must both be positive numbers.')
      return
    }
    if (fromId === toId) {
      setError('From and To customers must be different.')
      return
    }
    if (date && date > todayInput()) {
      setError('The transfer date can’t be in the future.')
      return
    }

    setLoading(true)
    try {
      await transferVehicle({
        vehicle_id: vehicleId,
        from: fromId,
        to: toId,
        date: date ? dayToApi(date) : undefined,
      })
      setSuccess(`Transferred from customer ${fromId} to customer ${toId}.`)
      setFrom('')
      setTo('')
      setDate('')
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('Vehicle or customer not found.')
      } else if (err instanceof ApiError && err.status === 409) {
        setError(
          `Customer ${fromId} isn’t the current owner, or the vehicle was just transferred by someone else.`,
        )
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="subsection">
      <div className="subsection-header">
        <h3>Owner</h3>
        {mode === 'assign' ? (
          <button type="button" className="secondary" onClick={() => switchMode('transfer')}>
            Transfer…
          </button>
        ) : (
          <button type="button" className="secondary" onClick={() => switchMode('assign')}>
            Assign first owner…
          </button>
        )}
      </div>

      {mode === 'assign' ? (
        <form className="inline-form dashed" onSubmit={handleAssign}>
          <label className="field grow" htmlFor="owner-customer-id">
            No owner yet — customer ID
            <input
              id="owner-customer-id"
              type="number"
              min={1}
              placeholder="e.g. 101"
              value={customerId}
              onChange={(e) => setCustomerId(e.target.value)}
              required
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? 'Assigning…' : 'Assign first owner'}
          </button>
        </form>
      ) : (
        <form className="field-grid inset" onSubmit={handleTransfer}>
          <label className="field" htmlFor="transfer-from">
            From customer ID
            <input
              id="transfer-from"
              type="number"
              min={1}
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              required
            />
          </label>
          <label className="field" htmlFor="transfer-to">
            To customer ID
            <input
              id="transfer-to"
              type="number"
              min={1}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              required
            />
          </label>
          <label className="field" htmlFor="transfer-date">
            Date (optional)
            <input
              id="transfer-date"
              type="date"
              max={todayInput()}
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? 'Transferring…' : 'Transfer'}
          </button>
        </form>
      )}

      {error && <p className="message error flush">{error}</p>}
      {success && <p className="message success flush">{success}</p>}
    </div>
  )
}
