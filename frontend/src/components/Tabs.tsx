interface TabsProps<T extends string> {
  tabs: { id: T; label: string }[]
  active: T
  onChange: (id: T) => void
  className?: string
}

export function Tabs<T extends string>({ tabs, active, onChange, className = 'tabs' }: TabsProps<T>) {
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
