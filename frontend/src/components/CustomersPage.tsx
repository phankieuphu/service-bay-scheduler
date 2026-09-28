import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  getCustomer,
  listCustomers,
  type Customer,
  type CustomerPage,
} from '../api/customer'
import { CreateCustomer } from './CreateCustomer'
import { CustomerDetail } from './CustomerDetail'

const PAGE_SIZE = 10

export function CustomersPage({ onOpenVehicle }: { onOpenVehicle: (vehicleId: number) => void }) {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [nextCursor, setNextCursor] = useState<number | null>(null)
  const [selected, setSelected] = useState<Customer | null>(null)
  const [creating, setCreating] = useState(false)
  const [lookupId, setLookupId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  function applyPage(page: CustomerPage, cursor: number) {
    setCustomers((prev) => (cursor === 0 ? page.customers : [...prev, ...page.customers]))
    setNextCursor(page.has_more && page.next_cursor ? page.next_cursor : null)
  }

  function applyError(err: unknown) {
    setError(err instanceof Error ? err.message : 'Failed to load customers.')
  }

  async function loadPage(cursor: number) {
    setError(null)
    setLoading(true)
    try {
      applyPage(await listCustomers(cursor, PAGE_SIZE), cursor)
    } catch (err) {
      applyError(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    listCustomers(0, PAGE_SIZE)
      .then((page) => applyPage(page, 0))
      .catch(applyError)
      .finally(() => setLoading(false))
  }, [])

  function select(customer: Customer | null) {
    setError(null)
    setNotice(null)
    setCreating(false)
    setSelected(customer)
  }

  async function handleLookup(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setNotice(null)

    const id = Number(lookupId)
    if (!Number.isInteger(id) || id <= 0) {
      setError('Enter a valid customer ID.')
      return
    }

    try {
      select(await getCustomer(id))
    } catch (err) {
      setSelected(null)
      if (err instanceof ApiError && err.status === 404) {
        setError(`No customer found with ID ${id}.`)
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    }
  }

  function handleUpdated(updated: Customer) {
    setSelected(updated)
    setCustomers((prev) => prev.map((c) => (c.id === updated.id ? updated : c)))
  }

  function handleDeleted(customerId: number) {
    setSelected(null)
    setCustomers((prev) => prev.filter((c) => c.id !== customerId))
    setNotice(`Customer ${customerId} deleted.`)
  }

  function handleCreated(created: Customer) {
    setCustomers((prev) => [created, ...prev])
    select(created)
  }

  return (
    <div className="two-col">
      <section className="panel stack" aria-labelledby="customers-heading">
        <div className="panel-header">
          <h2 id="customers-heading">Customers</h2>
          <form className="inline-form" onSubmit={handleLookup}>
            <label htmlFor="customer-lookup-id" className="visually-hidden">
              Customer ID
            </label>
            <input
              id="customer-lookup-id"
              type="number"
              min={1}
              value={lookupId}
              onChange={(e) => setLookupId(e.target.value)}
              placeholder="Look up by ID"
            />
            <button type="submit">Look up</button>
          </form>
        </div>

        {error && <p className="message error flush">{error}</p>}
        {notice && <p className="message success flush">{notice}</p>}

        <div className="row-list customers" role="list" aria-label="Customers">
          <div className="row-head" aria-hidden="true">
            <span>ID</span>
            <span>Name</span>
            <span>Email</span>
            <span>Status</span>
          </div>
          {customers.map((customer) => (
            <div role="listitem" key={customer.id}>
              <button
                type="button"
                className={`row${selected?.id === customer.id ? ' selected' : ''}`}
                aria-pressed={selected?.id === customer.id}
                onClick={() => select(customer)}
              >
                <span>{customer.id}</span>
                <span>{customer.name}</span>
                <span className="truncate">{customer.email}</span>
                <span>
                  <span className={`badge badge-${customer.status.toLowerCase()}`}>
                    {customer.status}
                  </span>
                </span>
              </button>
            </div>
          ))}
          {!loading && customers.length === 0 && <p className="muted empty">No customers yet.</p>}
        </div>

        <div className="actions">
          {nextCursor !== null && (
            <button
              type="button"
              className="secondary"
              disabled={loading}
              onClick={() => loadPage(nextCursor)}
            >
              {loading ? 'Loading…' : 'Load more'}
            </button>
          )}
          <button
            type="button"
            className="secondary"
            disabled={loading}
            onClick={() => loadPage(0)}
          >
            Refresh
          </button>
          <button
            type="button"
            className="secondary push-right"
            onClick={() => {
              select(null)
              setCreating(true)
            }}
          >
            New customer
          </button>
        </div>
      </section>

      {creating ? (
        <CreateCustomer onCreated={handleCreated} onCancel={() => setCreating(false)} />
      ) : selected ? (
        <CustomerDetail
          key={selected.id}
          customer={selected}
          onUpdated={handleUpdated}
          onDeleted={handleDeleted}
          onOpenVehicle={onOpenVehicle}
        />
      ) : (
        <section className="card placeholder">
          <p className="muted">Select a customer to see their details and vehicles.</p>
        </section>
      )}
    </div>
  )
}
