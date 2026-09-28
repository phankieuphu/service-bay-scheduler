export function Brand({ as: Heading = 'span' }: { as?: 'h1' | 'span' }) {
  return (
    <div className="brand">
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M5 17h14M6 17l1.5-5h9L18 17M8.5 12l1-3h5l1 3" />
        <circle cx="8" cy="18.5" r="1.5" />
        <circle cx="16" cy="18.5" r="1.5" />
      </svg>
      <Heading className="brand-name">Service Bay</Heading>
    </div>
  )
}
