import { useState } from 'react'
import type { User } from './api/identity'
import { useAuth } from './auth/auth-context'
import { AccountPanel } from './components/AccountPanel'
import { AuthScreen } from './components/AuthScreen'
import { Brand } from './components/Brand'
import { CustomersPage } from './components/CustomersPage'
import { Tabs } from './components/Tabs'
import { UserMenu } from './components/UserMenu'
import { VehiclesPage } from './components/VehiclesPage'
import './App.css'

type Section = 'customers' | 'vehicles' | 'account'

function Workspace({ user }: { user: User }) {
  const [section, setSection] = useState<Section>('customers')
  // Set when another section links to a vehicle ("Open" on a customer's
  // vehicle list); VehiclesPage opens it on mount.
  const [vehicleToOpen, setVehicleToOpen] = useState<number | null>(null)

  function openVehicle(vehicleId: number) {
    setVehicleToOpen(vehicleId)
    setSection('vehicles')
  }

  return (
    <>
      <header className="app-header">
        <div className="header-bar">
          <Brand as="h1" />
          <UserMenu user={user} />
        </div>
        <Tabs
          label="Sections"
          tabs={[
            { id: 'customers', label: 'Customers' },
            { id: 'vehicles', label: 'Vehicles' },
            { id: 'account', label: 'Account' },
          ]}
          active={section}
          onChange={(next) => {
            setVehicleToOpen(null)
            setSection(next)
          }}
        />
      </header>

      <main>
        {section === 'customers' && <CustomersPage onOpenVehicle={openVehicle} />}
        {section === 'vehicles' && <VehiclesPage initialVehicleId={vehicleToOpen} />}
        {section === 'account' && <AccountPanel user={user} />}
      </main>
    </>
  )
}

function App() {
  const { state, retry, signOut } = useAuth()

  if (state.status === 'signed-in') {
    return (
      <div className="app">
        {/* key: a different user signing in starts from a fresh workspace */}
        <Workspace key={state.user.id} user={state.user} />
      </div>
    )
  }

  return (
    <div className="app auth">
      <header className="auth-header">
        <Brand as="h1" />
        <p className="muted">Book, track and pay for vehicle service across every hub.</p>
      </header>
      <main>
        {state.status === 'loading' && <p className="muted">Restoring your session…</p>}
        {state.status === 'unreachable' && (
          <div className="panel">
            <h2>Can’t reach the identity service</h2>
            <p className="muted">
              Your session couldn’t be restored ({state.message}). Check that identity-service
              is running, then try again.
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
        {state.status === 'signed-out' && <AuthScreen />}
      </main>
    </div>
  )
}

export default App
