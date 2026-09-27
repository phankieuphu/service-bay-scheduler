import { useState } from 'react'
import type { User } from './api/identity'
import { useAuth } from './auth/auth-context'
import { AccountPanel } from './components/AccountPanel'
import { AuthScreen } from './components/AuthScreen'
import { CreateCustomer } from './components/CreateCustomer'
import { CustomerList } from './components/CustomerList'
import { GetVehicle } from './components/GetVehicle'
import { Tabs } from './components/Tabs'
import { TransferVehicle } from './components/TransferVehicle'
import { UserMenu } from './components/UserMenu'
import './App.css'

type Section = 'customers' | 'vehicles' | 'account'
type CustomerTab = 'list' | 'create'
type VehicleTab = 'get' | 'transfer'

function Workspace({ user }: { user: User }) {
  const [section, setSection] = useState<Section>('customers')
  const [customerTab, setCustomerTab] = useState<CustomerTab>('list')
  const [vehicleTab, setVehicleTab] = useState<VehicleTab>('get')

  return (
    <>
      <header>
        <div className="header-bar">
          <h1>Service Bay</h1>
          <UserMenu user={user} />
        </div>
        <Tabs
          tabs={[
            { id: 'customers', label: 'Customers' },
            { id: 'vehicles', label: 'Vehicles' },
            { id: 'account', label: 'Account' },
          ]}
          active={section}
          onChange={setSection}
        />
      </header>

      <main>
        {section === 'customers' && (
          <>
            <Tabs
              className="subtabs"
              tabs={[
                { id: 'list', label: 'Browse' },
                { id: 'create', label: 'Create' },
              ]}
              active={customerTab}
              onChange={setCustomerTab}
            />
            {customerTab === 'list' ? <CustomerList /> : <CreateCustomer />}
          </>
        )}
        {section === 'vehicles' && (
          <>
            <Tabs
              className="subtabs"
              tabs={[
                { id: 'get', label: 'Get Vehicle' },
                { id: 'transfer', label: 'Transfer Vehicle' },
              ]}
              active={vehicleTab}
              onChange={setVehicleTab}
            />
            {vehicleTab === 'get' ? <GetVehicle /> : <TransferVehicle />}
          </>
        )}
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
    <div className="app">
      <header>
        <h1>Service Bay</h1>
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
