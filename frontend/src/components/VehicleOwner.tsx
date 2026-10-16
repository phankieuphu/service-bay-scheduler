import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  assignOwner,
  dayFromApi,
  dayToApi,
  getVehicleOwner,
  todayInput,
  transferVehicle,
  type VehicleOwnership,
} from '../api/vehicle'

type Owner = NonNullable<VehicleOwnership['owner']>

function positiveId(value: string): number | null {
  const id = Number(value)
  return Number.isInteger(id) && id > 0 ? id : null
}

// VehicleOwner shows who currently owns the vehicle and changes it: a vehicle
// with no owner gets the "assign first owner" form, one with an owner gets a
// transfer form starting from that owner. After any change, or a 409 that
// means someone else changed it first, the owner is reloaded from the API.
export function VehicleOwner({ vehicleId }: { vehicleId: number }) {
  // undefined: still loading; null: no current owner.
  const [owner, setOwner] = useState<Owner | null | undefined>(undefined)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [customerId, setCustomerId] = useState('')
  const [to, setTo] = useState('')
  const [date, setDate] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function showOwner(result: VehicleOwnership) {
    setOwner(result.owner)
    setLoadError(null)
  }

  function showLoadError(err: unknown) {
    setLoadError(err instanceof Error ? err.message : 'Could not load the current owner.')
  }

  // VehicleDetail keys this component by vehicle, so a different vehicle
  // remounts it and starts from "loading" again.
  useEffect(() => {
    let cancelled = false
    getVehicleOwner(vehicleId)
      .then((result) => {
        if (!cancelled) showOwner(result)
      })
      .catch((err) => {
        if (!cancelled) showLoadError(err)
      })
    return () => {
      cancelled = true
    }
  }, [vehicleId])

  async function reload() {
    try {
      showOwner(await getVehicleOwner(vehicleId))
    } catch (err) {
      showLoadError(err)
    }
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
      await reload()
    } catch (err) {
      // 409 also covers a scrapped vehicle (invalid state); only an existing
      // owner means someone assigned one first.
      if (err instanceof ApiError && err.status === 409 && err.message.includes('already has a current owner')) {
        setError('Someone else just assigned an owner. Transfer it instead.')
        setTo(String(id))
        await reload()
      } else if (err instanceof ApiError && err.status === 404) {
        setError('Vehicle or customer not found.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleTransfer(event: FormEvent, from: Owner) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    const toId = positiveId(to)
    if (toId === null) {
      setError('Enter a valid customer ID to transfer to.')
      return
    }
    if (toId === from.customer_id) {
      setError(`Customer ${toId} already owns this vehicle.`)
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
        from: from.customer_id,
        to: toId,
        date: date ? dayToApi(date) : undefined,
      })
      setSuccess(`Transferred from customer ${from.customer_id} to customer ${toId}.`)
      setTo('')
      setDate('')
      await reload()
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('Vehicle or customer not found.')
      } else if (err instanceof ApiError && err.status === 409) {
        setError('The vehicle was just transferred by someone else. The current owner has been reloaded.')
        await reload()
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  let body
  if (loadError) {
    body = <p className="message error flush">{loadError}</p>
  } else if (owner === undefined) {
    body = <p className="muted">Loading owner…</p>
  } else if (owner === null) {
    body = (
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
    )
  } else {
    body = (
      <>
        <p className="flush">
          Customer <strong>#{owner.customer_id}</strong> since {dayFromApi(owner.owned_from)}
        </p>
        <form className="field-grid inset" onSubmit={(e) => handleTransfer(e, owner)}>
          <label className="field" htmlFor="transfer-to">
            Transfer to customer ID
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
              min={dayFromApi(owner.owned_from)}
              max={todayInput()}
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? 'Transferring…' : 'Transfer'}
          </button>
        </form>
      </>
    )
  }

  return (
    <div className="subsection">
      <div className="subsection-header">
        <h3>Owner</h3>
      </div>

      {body}

      {error && <p className="message error flush">{error}</p>}
      {success && <p className="message success flush">{success}</p>}
    </div>
  )
}
