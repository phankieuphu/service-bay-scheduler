import { useEffect, useState } from 'react'
import { formatMoney, waitForSettlement, type Payment } from '../api/billing'

// Where Stripe brings the customer back after a redirect-based payment
// method. The redirect itself proves nothing: this polls billing-service,
// which only marks a payment paid once Stripe's webhook says so.
export function PaymentReturn({ paymentId, onDone }: { paymentId: number; onDone: () => void }) {
  const [payment, setPayment] = useState<Payment | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    waitForSettlement(paymentId)
      .then((p) => !cancelled && setPayment(p))
      .catch((err) => !cancelled && setError(err instanceof Error ? err.message : 'Couldn’t check the payment.'))
    return () => {
      cancelled = true
    }
  }, [paymentId])

  return (
    <div className="card narrow stack">
      <h2>Payment</h2>
      {!payment && !error && <p className="muted">Confirming your payment…</p>}
      {error && <p className="message error flush">{error}</p>}
      {payment?.status === 'SUCCEEDED' && (
        <p className="message success flush">Paid {formatMoney(payment.amount_minor, payment.currency)}. Thank you!</p>
      )}
      {payment && payment.status !== 'SUCCEEDED' && (
        <p className="message info flush">
          {payment.failure_message ??
            (payment.status === 'PROCESSING' || payment.status === 'REQUIRES_PAYMENT'
              ? 'Your payment is still being confirmed. Your appointments page will show it once it goes through.'
              : `The payment ended as ${payment.status.toLowerCase()}.`)}
        </p>
      )}
      <div className="actions">
        <button type="button" className="secondary" onClick={onDone}>
          Back to my appointments
        </button>
      </div>
    </div>
  )
}
