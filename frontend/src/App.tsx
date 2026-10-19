import { useState } from 'react'
import type { User } from './api/identity'
import { useAuth } from './auth/auth-context'
import { AccountPanel } from './components/AccountPanel'
import { AuthScreen } from './components/AuthScreen'
import { Brand } from './components/Brand'
import { CustomerArea } from './components/CustomerArea'
import { CustomersPage } from './components/CustomersPage'
import { Landing, type AuthMode } from './components/Landing'
import { PaymentReturn } from './components/PaymentReturn'
import { StaffAppointments } from './components/StaffAppointments'
import { Tabs } from './components/Tabs'
import { UserMenu } from './components/UserMenu'
import { VehiclesPage } from './components/VehiclesPage'
import './App.css'

type Section = 'customers' | 'vehicles' | 'hub-day' | 'book' | 'appointments' | 'account'

// The Customers and Vehicles sections are staff tools: customer-service
// only serves staff, and vehicle changes need a staff role. A CUSTOMER gets
// booking and their own appointments instead.
function isStaff(user: User): boolean {
  return user.role !== 'CUSTOMER'
}

function Workspace({ user }: { user: User }) {
  const staff = isStaff(user)
  const [section, setSection] = useState<Section>(staff ? 'customers' : 'book')
  // Set when another section links to a vehicle ("Open" on a customer's
  // vehicle list); VehiclesPage opens it on mount.
  const [vehicleToOpen, setVehicleToOpen] = useState<number | null>(null)
  // Stripe sends the browser back here (?payment_id=…) after a payment
  // method that redirects.
  const [returningPayment, setReturningPayment] = useState<number | null>(() => {
    const id = Number(new URLSearchParams(window.location.search).get('payment_id'))
    return Number.isInteger(id) && id > 0 ? id : null
  })

  function leavePaymentReturn() {
    window.history.replaceState(null, '', window.location.pathname)
    setReturningPayment(null)
    setSection('appointments')
  }

  function openVehicle(vehicleId: number) {
    setVehicleToOpen(vehicleId)
    setSection('vehicles')
  }

  const tabs: { id: Section; label: string }[] = staff
    ? [
        { id: 'customers', label: 'Customers' },
        { id: 'vehicles', label: 'Vehicles' },
        { id: 'hub-day', label: 'Appointments' },
        { id: 'account', label: 'Account' },
      ]
    : [
        { id: 'book', label: 'Book a service' },
        { id: 'appointments', label: 'My appointments' },
        { id: 'account', label: 'Account' },
      ]

  return (
    <>
      <header className="app-header">
        <div className="header-bar">
          <Brand as="h1" />
          <UserMenu user={user} />
        </div>
        <Tabs
          label="Sections"
          tabs={tabs}
          active={section}
          onChange={(next) => {
            setVehicleToOpen(null)
            setSection(next)
          }}
        />
      </header>

      <main className="app-main">
        {returningPayment !== null && <PaymentReturn paymentId={returningPayment} onDone={leavePaymentReturn} />}
        {returningPayment === null && section === 'customers' && <CustomersPage onOpenVehicle={openVehicle} />}
        {returningPayment === null && section === 'vehicles' && <VehiclesPage initialVehicleId={vehicleToOpen} />}
        {returningPayment === null && section === 'hub-day' && <StaffAppointments />}
        {returningPayment === null && (section === 'book' || section === 'appointments') && (
          <CustomerArea section={section} onNavigate={setSection} />
        )}
        {returningPayment === null && section === 'account' && <AccountPanel user={user} />}
      </main>
    </>
  )
}

function App() {
  const { state, retry, signOut } = useAuth()
  // Signed-out visitors see the landing page until they pick sign in or
  // sign up. A session that just expired skips it and goes straight to the
  // sign-in form, where the notice explains why, unless they ask for home.
  const [authMode, setAuthMode] = useState<AuthMode | 'home' | null>(null)

  if (state.status === 'signed-in') {
    return (
      <div className="app">
        {/* key: a different user signing in starts from a fresh workspace */}
        <Workspace key={state.user.id} user={state.user} />
      </div>
    )
  }

  const notice = state.status === 'signed-out' ? state.notice : undefined
  if (state.status === 'signed-out' && (authMode === 'home' || (authMode === null && !notice))) {
    return <Landing onAuth={setAuthMode} />
  }

  return (
    <div className="app auth">
      <header className="auth-header">
        {state.status === 'signed-out' && (
          <button type="button" className="link auth-back" onClick={() => setAuthMode('home')}>
            ← Back to home
          </button>
        )}
        <Brand as="h1" />
        <p className="muted">Book, track and pay for vehicle service across every hub.</p>
      </header>
      <main>
        {state.status === 'loading' && <p className="muted">Restoring your session…</p>}
        {state.status === 'unreachable' && (
          <div className="panel">
            <h2>Can’t reach the identity service</h2>
            <p className="muted">
              Your session couldn’t be restored ({state.message}). Check that identity-service is
              running, then try again.
            </p>
            <div className="actions">
              <button type="button" className="secondary" onClick={retry}>
                Try again
              </button>
              <button type="button" className="secondary" onClick={() => void signOut()}>
                Sign in as someone else
              </button>
            </div>
          </div>
        )}
        {state.status === 'signed-out' && (
          <AuthScreen key={authMode ?? 'sign-in'} initialMode={authMode === 'sign-up' ? 'sign-up' : 'sign-in'} />
        )}
      </main>
    </div>
  )
}

export default App
