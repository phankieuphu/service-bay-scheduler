export function PageHeader({ title, description }: { title: string; description?: string }) {
  return (
    <div className="page-header">
      <h2 className="page-title">{title}</h2>
      {description && <p className="muted flush">{description}</p>}
    </div>
  )
}
