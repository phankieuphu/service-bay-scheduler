/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_VEHICLE_API_URL?: string
  readonly VITE_CUSTOMER_API_URL?: string
  readonly VITE_IDENTITY_API_URL?: string
  readonly VITE_SCHEDULER_API_URL?: string
  readonly VITE_BILLING_API_URL?: string
  readonly VITE_STRIPE_PUBLISHABLE_KEY?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
