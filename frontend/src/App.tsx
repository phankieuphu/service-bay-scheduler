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
import { BillingPage } from './components/demo/BillingPage'
import { BookingPage } from './components/demo/BookingPage'
import { HubBoardPage } from './components/demo/HubBoardPage'
import { OverviewPage } from './components/demo/OverviewPage'
import { ReportPage } from './components/demo/ReportPage'
import type { DemoSection } from './components/demo/sections'
import { DemoProvider } from './demo/DemoProvider'
import './App.css'
import './components/demo/demo.css'

type Section = DemoSection

function Workspace({ user }: { user: User }) {
  const [section, setSection] = useState<Section>('overview')
  // Set when another section links to a vehicle ("Open" on a customer's
  // vehicle list); VehiclesPage opens it on mount.
  const [vehicleToOpen, setVehicleToOpen] = useState<number | null>(null)

  function openVehicle(vehicleId: number) {
    setVehicleToOpen(vehicleId)
    setSection('vehicles')
  }

  function navigate(next: Section) {
    setVehicleToOpen(null)
    setSection(next)
  }

  return (
    <DemoProvider>
      <header className="app-header">
        <div className="header-bar">
          <div className="brand-row">
            <Brand as="h1" />
            <span className="demo-badge">Demo mode</span>
          </div>
          <UserMenu user={user} />
        </div>
        <Tabs
          label="Sections"
          tabs={[
            { id: 'overview', label: 'Overview' },
            { id: 'customers', label: 'Customers' },
            { id: 'vehicles', label: 'Vehicles' },
            { id: 'booking', label: 'Book service' },
            { id: 'hub', label: 'Hub board' },
            { id: 'billing', label: 'Billing' },
            { id: 'reports', label: 'Reports' },
            { id: 'account', label: 'Account' },
          ]}
          active={section}
          onChange={navigate}
        />
      </header>

      <main>
        {section === 'overview' && <OverviewPage onNavigate={navigate} />}
        {section === 'customers' && <CustomersPage onOpenVehicle={openVehicle} />}
        {section === 'vehicles' && <VehiclesPage initialVehicleId={vehicleToOpen} />}
        {section === 'booking' && <BookingPage onNavigate={navigate} />}
        {section === 'hub' && <HubBoardPage onNavigate={navigate} />}
        {section === 'billing' && <BillingPage onNavigate={navigate} />}
        {section === 'reports' && <ReportPage />}
        {section === 'account' && <AccountPanel user={user} />}
      </main>
    </DemoProvider>
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
