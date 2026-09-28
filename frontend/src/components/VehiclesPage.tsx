import { useState } from 'react'
import type { Vehicle } from '../api/vehicle'
import { RegisterVehicle } from './RegisterVehicle'
import { Tabs } from './Tabs'
import { VehicleDetail } from './VehicleDetail'
import { VehicleSearch } from './VehicleSearch'

type Mode = 'search' | 'register'

export function VehiclesPage({ initialVehicleId }: { initialVehicleId: number | null }) {
  const [mode, setMode] = useState<Mode>('search')
  const [selectedId, setSelectedId] = useState<number | null>(initialVehicleId)
  // The last vehicle saved from the detail pane, so its search row shows the
  // new status without a refetch.
  const [updated, setUpdated] = useState<Vehicle | null>(null)

  function openVehicle(vehicleId: number) {
    setSelectedId(vehicleId)
    setMode('search')
  }

  return (
    <div className="stack">
      <Tabs
        className="subtabs"
        tabs={[
          { id: 'search', label: 'Search' },
          { id: 'register', label: 'Register' },
        ]}
        active={mode}
        onChange={setMode}
      />

      {mode === 'search' ? (
        <div className="two-col">
          <VehicleSearch selectedId={selectedId} updated={updated} onSelect={setSelectedId} />
          {selectedId !== null ? (
            <VehicleDetail key={selectedId} vehicleId={selectedId} onUpdated={setUpdated} />
          ) : (
            <section className="card placeholder">
              <p className="muted">Select a vehicle to see its warranty, materials and history.</p>
            </section>
          )}
        </div>
      ) : (
        <RegisterVehicle onOpen={openVehicle} />
      )}
    </div>
  )
}
