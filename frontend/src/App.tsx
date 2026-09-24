import { useState } from 'react'
import { CreateCustomer } from './components/CreateCustomer'
import { CustomerList } from './components/CustomerList'
import { GetVehicle } from './components/GetVehicle'
import { TransferVehicle } from './components/TransferVehicle'
import './App.css'

type Section = 'customers' | 'vehicles'
type CustomerTab = 'list' | 'create'
type VehicleTab = 'get' | 'transfer'

interface TabsProps<T extends string> {
  tabs: { id: T; label: string }[]
  active: T
  onChange: (id: T) => void
  className?: string
}

function Tabs<T extends string>({ tabs, active, onChange, className = 'tabs' }: TabsProps<T>) {
  return (
    <nav className={className}>
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          className={active === tab.id ? 'active' : ''}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </nav>
  )
}

function App() {
  const [section, setSection] = useState<Section>('customers')
  const [customerTab, setCustomerTab] = useState<CustomerTab>('list')
  const [vehicleTab, setVehicleTab] = useState<VehicleTab>('get')

  return (
    <div className="app">
      <header>
        <h1>Service Bay</h1>
        <Tabs
          tabs={[
            { id: 'customers', label: 'Customers' },
            { id: 'vehicles', label: 'Vehicles' },
          ]}
          active={section}
          onChange={setSection}
        />
      </header>

      <main>
        {section === 'customers' ? (
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
        ) : (
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
      </main>
    </div>
  )
}

export default App
