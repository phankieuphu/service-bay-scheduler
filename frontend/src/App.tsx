import { useState } from 'react'
import { GetVehicle } from './components/GetVehicle'
import { TransferVehicle } from './components/TransferVehicle'
import './App.css'

type Tab = 'get' | 'transfer'

function App() {
  const [tab, setTab] = useState<Tab>('get')

  return (
    <div className="app">
      <header>
        <h1>Service Bay — Vehicles</h1>
        <nav className="tabs">
          <button
            type="button"
            className={tab === 'get' ? 'active' : ''}
            onClick={() => setTab('get')}
          >
            Get Vehicle
          </button>
          <button
            type="button"
            className={tab === 'transfer' ? 'active' : ''}
            onClick={() => setTab('transfer')}
          >
            Transfer Vehicle
          </button>
        </nav>
      </header>

      <main>{tab === 'get' ? <GetVehicle /> : <TransferVehicle />}</main>
    </div>
  )
}

export default App
