import type { ReactNode } from 'react'
import { Brand } from './Brand'
import './Landing.css'

export type AuthMode = 'sign-in' | 'sign-up'

const REPO_URL = 'https://github.com/phankieuphu/service-bay-scheduler'

function Icon({ children }: { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  )
}

const steps: { title: string; body: string; icon: ReactNode }[] = [
  {
    title: 'Pick a hub and a slot',
    body: 'Choose your vehicle, the services it needs and a free time at the hub that suits you.',
    icon: (
      <Icon>
        <rect x="3.5" y="5" width="17" height="15" rx="2" />
        <path d="M3.5 10h17M8 3v4M16 3v4M8 14h3" />
      </Icon>
    ),
  },
  {
    title: 'Hold it with a deposit',
    body: 'Pay a deposit by card when you book, or skip it and settle everything at the end.',
    icon: (
      <Icon>
        <rect x="3" y="6" width="18" height="13" rx="2" />
        <path d="M3 10.5h18M7 15h4" />
      </Icon>
    ),
  },
  {
    title: 'Drop the car off',
    body: 'Your bay is reserved for you. When the work is done, staff mark the appointment complete.',
    icon: (
      <Icon>
        <path d="M14.5 6.5a4 4 0 0 0-5.3 5.3L4 17l3 3 5.2-5.2a4 4 0 0 0 5.3-5.3l-2.4 2.4-2.4-.6-.6-2.4z" />
      </Icon>
    ),
  },
  {
    title: 'Pay the balance',
    body: 'The bill prices itself from the work done and your warranty, minus what you already paid.',
    icon: (
      <Icon>
        <path d="M6 3h12v18l-3-2-3 2-3-2-3 2z" />
        <path d="M9 8h6M9 12h6M9 16h3" />
      </Icon>
    ),
  },
]

const customerFeatures = [
  'Book any service at any hub in a couple of clicks',
  'Live free-slot search, so you never book a taken bay',
  'See, track and cancel your own appointments',
  'Card payments and deposits through Stripe',
  'One account for every vehicle you own',
]

const staffFeatures = [
  'Customer directory with search and full profiles',
  'Vehicle register: VIN and plate lookup, warranty, history',
  'Safe ownership transfers between customers',
  'A hub’s day of appointments at a glance',
  'Complete a job and the bill is raised automatically',
]

const services: { name: string; stack: string; does: string }[] = [
  { name: 'identity', stack: 'Go', does: 'Accounts, JWT, refresh tokens' },
  { name: 'customer', stack: 'Go', does: 'Customer profiles' },
  { name: 'vehicle', stack: 'Go', does: 'Vehicles, owners, warranty' },
  { name: 'scheduler', stack: 'Go', does: 'Hubs, slots, bookings' },
  { name: 'billing', stack: 'Java', does: 'Deposits, bills, Stripe' },
]

const guarantees: { title: string; body: string }[] = [
  {
    title: 'No double-booked bays',
    body: 'A Postgres exclusion constraint rejects overlapping bookings for the same bay, even under concurrent load.',
  },
  {
    title: 'Events never lost',
    body: 'Every state change writes its Kafka event to a transactional outbox in the same commit, then a relay publishes it in order.',
  },
  {
    title: 'One owner at a time',
    body: 'Ownership transfers take a row lock, so racing requests for the same vehicle serialize and exactly one wins.',
  },
  {
    title: 'Payments you can trust',
    body: 'Stripe PaymentIntents and signed webhooks, with idempotent consumers and dead-letter topics for anything that fails.',
  },
]

const stack = [
  'Go',
  'Java 25 · Spring Boot 4',
  'React 19 · Vite',
  'PostgreSQL 16',
  'Kafka (KRaft)',
  'Redis',
  'Stripe',
  'Prometheus',
  'Grafana',
  'Loki · Alloy',
  'Kubernetes',
  'k6',
]

function HeroPreview() {
  return (
    <div className="lp-preview" aria-hidden="true">
      <div className="lp-mock lp-mock-booking">
        <div className="lp-mock-head">
          <span className="lp-dot" />
          <span className="strong">Book a service</span>
        </div>
        <div className="lp-field">
          <span className="lp-label">Hub</span>
          <span>Downtown Service Hub</span>
        </div>
        <div className="lp-field">
          <span className="lp-label">Vehicle</span>
          <span>
            Toyota Camry · <span className="mono">51A-123.45</span>
          </span>
        </div>
        <div className="lp-field">
          <span className="lp-label">Services</span>
          <span className="lp-chips">
            <span className="lp-chip">Oil change</span>
            <span className="lp-chip">Brake check</span>
          </span>
        </div>
        <span className="lp-label">Thursday, free slots</span>
        <div className="lp-slots">
          <span className="lp-slot taken">08:00</span>
          <span className="lp-slot">09:30</span>
          <span className="lp-slot selected">11:00</span>
          <span className="lp-slot">13:30</span>
          <span className="lp-slot taken">15:00</span>
          <span className="lp-slot">16:30</span>
        </div>
        <span className="lp-fake-button">Book 11:00</span>
      </div>

      <div className="lp-mock lp-mock-bill">
        <div className="lp-mock-head">
          <span className="strong">Bill #1042</span>
          <span className="lp-badge">Paid</span>
        </div>
        <div className="lp-line">
          <span>Oil change</span>
          <span>$45.00</span>
        </div>
        <div className="lp-line">
          <span>Brake check</span>
          <span>$30.00</span>
        </div>
        <div className="lp-line muted">
          <span>Deposit at booking</span>
          <span>−$20.00</span>
        </div>
        <div className="lp-line lp-total">
          <span>Balance paid</span>
          <span>$55.00</span>
        </div>
      </div>
    </div>
  )
}

function ArchitectureDiagram() {
  return (
    <div className="lp-arch" role="img" aria-label="The web app calls five services; each owns its Postgres database and publishes events to Kafka through an outbox.">
      <div className="lp-arch-tier">
        <div className="lp-node lp-node-client">
          <span className="strong">Web app</span>
          <span className="small muted">React 19 · Stripe Elements</span>
        </div>
      </div>
      <div className="lp-arch-link">
        <span className="small muted">REST + JWT</span>
      </div>
      <div className="lp-arch-tier lp-arch-services">
        {services.map((s) => (
          <div key={s.name} className="lp-node">
            <span className="lp-node-title">
              <span className="strong">{s.name}</span>
              <span className={`lp-tag ${s.stack === 'Java' ? 'java' : ''}`}>{s.stack}</span>
            </span>
            <span className="small muted">{s.does}</span>
            <span className="lp-db">
              <Icon>
                <ellipse cx="12" cy="6" rx="7" ry="2.5" />
                <path d="M5 6v12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5V6M5 12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5" />
              </Icon>
              {s.name}_db
            </span>
          </div>
        ))}
      </div>
      <div className="lp-arch-link">
        <span className="small muted">transactional outbox</span>
      </div>
      <div className="lp-arch-tier">
        <div className="lp-node lp-node-bus">
          <span className="strong">Kafka event bus</span>
          <span className="small muted">
            UserCreated · AppointmentBooked · ServiceCompleted · PaymentReceived
          </span>
        </div>
      </div>
    </div>
  )
}

export function Landing({ onAuth }: { onAuth: (mode: AuthMode) => void }) {
  return (
    <div className="landing">
      <header className="lp-nav">
        <div className="lp-container lp-nav-bar">
          <Brand />
          <nav className="lp-nav-links" aria-label="Page sections">
            <a href="#how">How it works</a>
            <a href="#features">Features</a>
            <a href="#architecture">Under the hood</a>
          </nav>
          <div className="actions">
            <button type="button" className="secondary" onClick={() => onAuth('sign-in')}>
              Sign in
            </button>
            <button type="button" className="primary lp-nav-cta" onClick={() => onAuth('sign-up')}>
              Get started
            </button>
          </div>
        </div>
      </header>

      <main>
        <section className="lp-hero">
          <div className="lp-container lp-hero-grid">
            <div className="lp-hero-copy">
              <span className="lp-eyebrow">Vehicle service, end to end</span>
              <h1>
                From booking to <span className="lp-accent">paid bill</span>, without the phone calls.
              </h1>
              <p className="lp-lead">
                Service Bay lets drivers book a bay at any hub, pay a deposit, and settle the bill
                online, while staff run customers, vehicles and the day’s appointments from one
                place.
              </p>
              <div className="actions">
                <button type="button" className="primary lp-big" onClick={() => onAuth('sign-up')}>
                  Book your first service
                </button>
                <button type="button" className="secondary lp-big" onClick={() => onAuth('sign-in')}>
                  I already have an account
                </button>
              </div>
              <ul className="lp-hero-points">
                <li>Real-time free slots</li>
                <li>Secure card payments</li>
                <li>Warranty-aware billing</li>
              </ul>
            </div>
            <HeroPreview />
          </div>
        </section>

        <section className="lp-section" id="how">
          <div className="lp-container">
            <div className="lp-section-head">
              <span className="lp-eyebrow">How it works</span>
              <h2>Four steps, one visit</h2>
            </div>
            <ol className="lp-steps">
              {steps.map((step, i) => (
                <li key={step.title} className="lp-step">
                  <span className="lp-step-icon">{step.icon}</span>
                  <span className="lp-step-num">0{i + 1}</span>
                  <h3>{step.title}</h3>
                  <p className="muted">{step.body}</p>
                </li>
              ))}
            </ol>
          </div>
        </section>

        <section className="lp-section lp-tinted" id="features">
          <div className="lp-container">
            <div className="lp-section-head">
              <span className="lp-eyebrow">Features</span>
              <h2>Built for both sides of the counter</h2>
            </div>
            <div className="lp-audiences">
              <article className="lp-audience">
                <h3>For drivers</h3>
                <p className="muted">Everything you need to get your car looked after.</p>
                <ul className="lp-checks">
                  {customerFeatures.map((f) => (
                    <li key={f}>{f}</li>
                  ))}
                </ul>
                <button type="button" className="primary" onClick={() => onAuth('sign-up')}>
                  Create a customer account
                </button>
              </article>
              <article className="lp-audience">
                <h3>For hub staff</h3>
                <p className="muted">The tools to run a busy service floor.</p>
                <ul className="lp-checks">
                  {staffFeatures.map((f) => (
                    <li key={f}>{f}</li>
                  ))}
                </ul>
                <button type="button" className="secondary" onClick={() => onAuth('sign-in')}>
                  Staff sign in
                </button>
              </article>
            </div>
          </div>
        </section>

        <section className="lp-section" id="architecture">
          <div className="lp-container">
            <div className="lp-section-head">
              <span className="lp-eyebrow">Under the hood</span>
              <h2>Microservices that keep their promises</h2>
              <p className="muted lp-section-lead">
                Five independent services, each with its own database, talking through Kafka
                events. Hexagonal architecture throughout, observable from day one.
              </p>
            </div>
            <ArchitectureDiagram />
            <div className="lp-guarantees">
              {guarantees.map((g) => (
                <div key={g.title} className="lp-guarantee">
                  <h3>{g.title}</h3>
                  <p className="muted">{g.body}</p>
                </div>
              ))}
            </div>
            <dl className="lp-stats">
              <div>
                <dt>Services live</dt>
                <dd>5</dd>
              </div>
              <div>
                <dt>Databases per service</dt>
                <dd>1</dd>
              </div>
              <div>
                <dt>Load-test tier</dt>
                <dd>1M</dd>
              </div>
              <div>
                <dt>Outbox relay poll</dt>
                <dd>2s</dd>
              </div>
            </dl>
            <ul className="lp-stack" aria-label="Technology stack">
              {stack.map((s) => (
                <li key={s}>{s}</li>
              ))}
            </ul>
          </div>
        </section>

        <section className="lp-section lp-tinted">
          <div className="lp-container">
            <div className="lp-section-head">
              <span className="lp-eyebrow">Roadmap</span>
              <h2>What’s coming next</h2>
            </div>
            <div className="lp-roadmap">
              <div className="lp-road">
                <h3>Dealership</h3>
                <p className="muted">Hubs, bays, technicians and skills, with materials tracked per hub.</p>
              </div>
              <div className="lp-road">
                <h3>Notifications</h3>
                <p className="muted">Booking confirmations, reminders and receipts by email.</p>
              </div>
              <div className="lp-road">
                <h3>Reports</h3>
                <p className="muted">End-of-day hub reports and a warehouse for revenue and utilization.</p>
              </div>
            </div>
          </div>
        </section>

        <section className="lp-section">
          <div className="lp-container">
            <div className="lp-cta">
              <h2>Your car’s next service is a few clicks away.</h2>
              <p>Create an account, add your vehicle, and pick a slot.</p>
              <div className="actions">
                <button type="button" className="lp-cta-primary lp-big" onClick={() => onAuth('sign-up')}>
                  Get started free
                </button>
                <button type="button" className="lp-cta-secondary lp-big" onClick={() => onAuth('sign-in')}>
                  Sign in
                </button>
              </div>
            </div>
          </div>
        </section>
      </main>

      <footer className="lp-footer">
        <div className="lp-container lp-footer-bar">
          <Brand />
          <span className="small muted">Book, track and pay for vehicle service across every hub.</span>
          <a href={REPO_URL} target="_blank" rel="noreferrer" className="small">
            Source on GitHub
          </a>
        </div>
      </footer>
    </div>
  )
}
