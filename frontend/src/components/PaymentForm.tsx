import { Elements, PaymentElement, useElements, useStripe } from '@stripe/react-stripe-js'
import { loadStripe, type Appearance } from '@stripe/stripe-js'
import { useState, type FormEvent } from 'react'
import {
  formatMoney,
  returnUrl,
  testConfirm,
  waitForSettlement,
  type Payment,
  type StartedPayment,
} from '../api/billing'

const publishableKey = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY
// Loaded once, and only if a key is configured: loadStripe fetches Stripe.js.
const stripePromise = publishableKey ? loadStripe(publishableKey) : null

function appearance(): Appearance {
  const dark = window.matchMedia?.('(prefers-color-scheme: dark)').matches
  return {
    theme: dark ? 'night' : 'stripe',
    variables: { colorPrimary: dark ? '#a78bfa' : '#6d28d9', borderRadius: '8px' },
  }
}

interface Props {
  started: StartedPayment
  label: string
  onSettled: (payment: Payment) => void
  onCancel: () => void
}

export function PaymentForm({ started, label, onSettled, onCancel }: Props) {
  const amount = formatMoney(started.amount_minor, started.currency)

  return (
    <div className="card stack payment-form">
      <div className="subsection-header">
        <h3>
          {label}: {amount}
        </h3>
        <button type="button" className="link" onClick={onCancel}>
          Not now
        </button>
      </div>
      {stripePromise && started.client_secret ? (
        <Elements stripe={stripePromise} options={{ clientSecret: started.client_secret, appearance: appearance() }}>
          <CardCheckout started={started} amount={amount} onSettled={onSettled} />
        </Elements>
      ) : (
        <p className="message info flush">
          Card payments aren’t configured in this web app (VITE_STRIPE_PUBLISHABLE_KEY is not set).
        </p>
      )}
      {import.meta.env.DEV && <TestCardShortcut key={started.payment_id} started={started} onSettled={onSettled} />}
    </div>
  )
}

function CardCheckout({
  started,
  amount,
  onSettled,
}: {
  started: StartedPayment
  amount: string
  onSettled: (payment: Payment) => void
}) {
  const stripe = useStripe()
  const elements = useElements()
  const [state, setState] = useState<'idle' | 'paying' | 'confirming'>('idle')
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (!stripe || !elements) return
    setState('paying')
    setError(null)
    // redirect 'if_required': cards (including 3-D Secure challenges) finish
    // here; only redirect-based methods leave the page.
    const result = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: returnUrl(started.payment_id) },
      redirect: 'if_required',
    })
    if (result.error) {
      // A decline or validation error: Stripe's message is meant for the
      // customer, and the same payment can be retried.
      setError(result.error.message ?? 'The payment didn’t go through.')
      setState('idle')
      return
    }
    setState('confirming')
    try {
      onSettled(await waitForSettlement(started.payment_id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Couldn’t confirm the payment.')
      setState('idle')
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      <PaymentElement />
      {error && <p className="message error flush">{error}</p>}
      <div className="actions">
        <button type="submit" disabled={!stripe || state !== 'idle'}>
          {state === 'paying' ? 'Paying…' : state === 'confirming' ? 'Confirming…' : `Pay ${amount}`}
        </button>
      </div>
    </form>
  )
}

// Dev builds only: billing-service's test-confirm pays with Stripe's test
// cards server-side (test-mode keys only), so the flow can be tried without a
// publishable key.
function TestCardShortcut({ started, onSettled }: { started: StartedPayment; onSettled: (payment: Payment) => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function pay(method: string) {
    setBusy(true)
    setError(null)
    try {
      const payment = await testConfirm(started.payment_id, method)
      if (payment.status === 'REQUIRES_PAYMENT' && payment.failure_message) {
        setError(payment.failure_message)
      } else {
        onSettled(payment)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Test payment failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="inset stack">
      <span className="muted small">Development: pay with a Stripe test card (server test mode only)</span>
      <div className="actions">
        <button type="button" className="secondary" disabled={busy} onClick={() => pay('pm_card_visa')}>
          Test card: success
        </button>
        <button type="button" className="secondary" disabled={busy} onClick={() => pay('pm_card_chargeDeclined')}>
          Test card: declined
        </button>
      </div>
      {error && <p className="message error flush">{error}</p>}
    </div>
  )
}
