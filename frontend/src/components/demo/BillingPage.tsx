import { useState } from 'react'
import { PAST_BILLS, formatVnd } from '../../demo/data'
import { buildBill, useDemo } from '../../demo/demo-context'
import type { DemoSection } from './sections'

const METHODS = ['Bank transfer (QR)', 'Card', 'Cash at the hub']

export function BillingPage({ onNavigate }: { onNavigate: (section: DemoSection) => void }) {
  const { state, payBalance } = useDemo()
  const { booking, stage, changeDecision, paidMethod } = state
  const [method, setMethod] = useState(METHODS[0])
  const ready = booking !== null && stage === 'completed'
  const bill = ready ? buildBill(booking, changeDecision === 'approved') : null
  const paid = paidMethod !== null

  return (
    <div className="split demo">
      <nav className="bill-list stack" aria-labelledby="bills-heading">
        <h2 id="bills-heading" className="demo-h3">
          Your bills
        </h2>
        {booking && (
          <div className="bill-item selected">
            <span className="section-head">
              <strong>{booking.plate}</strong>
              <span className={`badge ${paid ? 'badge-active' : ready ? 'badge-warn' : 'badge-inactive'}`}>
                {paid ? 'Paid' : ready ? 'Due' : 'Pending'}
              </span>
            </span>
            <span className="small muted">
              {booking.dayLabel}
              {bill ? ` · ${formatVnd(bill.total)}` : ''}
            </span>
          </div>
        )}
        {PAST_BILLS.map((b) => (
          <div key={b.id} className="bill-item">
            <span className="section-head">
              <strong>{b.plate}</strong>
              <span className="badge badge-active">Paid</span>
            </span>
            <span className="small muted">
              {b.date} · {formatVnd(b.total)}
            </span>
          </div>
        ))}
      </nav>

      {!booking || !bill ? (
        <section className="split-main card placeholder">
          <div className="stack">
            <p className="muted">
              {booking
                ? 'The bill is generated when the service is completed.'
                : 'No demo booking yet — book and service a car to get a bill.'}
            </p>
            <div>
              <button
                type="button"
                className="primary"
                onClick={() => onNavigate(booking ? 'hub' : 'booking')}
              >
                {booking ? 'Go to the hub board' : 'Book a service'}
              </button>
            </div>
          </div>
        </section>
      ) : (
        <>
          <article className="split-main card stack roomy" aria-labelledby="bill-heading">
            <div className="section-head wrap">
              <div>
                <h2 id="bill-heading" className="demo-title">
                  Bill for booking #{booking.id}
                </h2>
                <p className="muted small flush">
                  {booking.plate} · {booking.hub.name} · {booking.dayLabel}
                </p>
              </div>
              <span className={`badge ${paid ? 'badge-active' : 'badge-warn'}`}>
                {paid ? 'Paid' : 'Balance due'}
              </span>
            </div>
            <div className="scroll-x">
              <table className="demo-table">
                <thead>
                  <tr>
                    <th scope="col">Item</th>
                    <th scope="col">Type</th>
                    <th scope="col" className="right">
                      Qty
                    </th>
                    <th scope="col" className="right">
                      Amount
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {bill.lines.map((l) => (
                    <tr key={`${l.kind}-${l.name}`}>
                      <td>
                        {l.name}
                        {l.addedMidService && (
                          <span className="badge step-next tag">Approved mid-service</span>
                        )}
                      </td>
                      <td className="muted">{l.kind}</td>
                      <td className="right">{l.qty}</td>
                      <td className="right">{formatVnd(l.amount)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <dl className="kv num totals">
              <dt>Services</dt>
              <dd>{formatVnd(bill.services)}</dd>
              <dt>Materials</dt>
              <dd>{formatVnd(bill.materials)}</dd>
              <dt>Warranty fee</dt>
              <dd className={bill.warrantyFee === 0 ? 'good' : ''}>
                {bill.warrantyFee === 0 ? '0 ₫ · in warranty' : formatVnd(bill.warrantyFee)}
              </dd>
              <dt className="total ruled">Total</dt>
              <dd className="total ruled">{formatVnd(bill.total)}</dd>
              {bill.deposit > 0 && (
                <>
                  <dt>Deposit paid</dt>
                  <dd className="muted">− {formatVnd(bill.deposit)}</dd>
                </>
              )}
              <dt className="total big-total">{paid ? 'Paid' : 'Balance due'}</dt>
              <dd className="total big-total accent">{formatVnd(bill.balance)}</dd>
            </dl>
            <p className="small muted flush">
              Total = warranty fee (only if warranty expired or none) + service fees + material
              costs. Any deposit is subtracted.
            </p>
          </article>

          <aside className="split-side panel stack" aria-labelledby="pay-heading">
            <h3 id="pay-heading" className="demo-h3">
              {paid ? 'Paid' : 'Pay balance'}
            </h3>
            {paid ? (
              <p className="message success flush">
                Paid {formatVnd(bill.balance)} by {paidMethod}. <code>PaymentReceived</code> published.
              </p>
            ) : (
              <>
                <fieldset className="pay-choice boxed">
                  <legend className="small muted">Method</legend>
                  {METHODS.map((m) => (
                    <label key={m} className={method === m ? 'selected' : ''}>
                      <input type="radio" name="method" checked={method === m} onChange={() => setMethod(m)} />
                      {m}
                    </label>
                  ))}
                </fieldset>
                <button type="button" className="primary big" onClick={() => payBalance(method)}>
                  Pay {formatVnd(bill.balance)}
                </button>
                <p className="small muted flush">
                  Demo payment — nothing is charged. Paying publishes <code>PaymentReceived</code>.
                </p>
              </>
            )}
            <hr className="rule" />
            <h4 className="demo-h4">Payment history</h4>
            <ul className="plain-list small num">
              {bill.deposit > 0 && (
                <li className="section-head">
                  <span>Deposit · at booking</span>
                  <span>{formatVnd(bill.deposit)}</span>
                </li>
              )}
              {paid && (
                <li className="section-head">
                  <span>Balance · {paidMethod}</span>
                  <span>{formatVnd(bill.balance)}</span>
                </li>
              )}
              {bill.deposit === 0 && !paid && <li className="muted">No payments yet.</li>}
            </ul>
            {paid && (
              <button type="button" className="secondary" onClick={() => onNavigate('reports')}>
                See the day-end report →
              </button>
            )}
          </aside>
        </>
      )}
    </div>
  )
}
