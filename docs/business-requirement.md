# Business Requirement

## 1. Overview

Service Bay is a management platform for a vehicle **service hub** (dealership). It manages customers' vehicles, materials, technicians, and service bays, and coordinates the full lifecycle of a maintenance visit — from booking to pickup and billing.

## 2. Actors

| Actor            | Description                                                                 |
| ----------------- | ---------------------------------------------------------------------------- |
| Customer          | Owns one or more vehicles; books, pays for, and tracks service appointments |
| Service Hub       | A physical dealership location with its own bays, technicians, and materials |
| Technician        | Performs services; holds skills that improve with practice                  |
| Dealership Manager | Reviews end-of-day reports for their hub (implied by §7)                    |

## 3. Core Entities

| Entity        | Purpose                                                        |
| ------------- | --------------------------------------------------------------- |
| Vehicle       | Belongs to a customer; has a warranty status                    |
| Service Hub   | A location the customer books at                                 |
| Service       | A type of maintenance/repair work; requires a specific skill    |
| Service Bay   | Physical capacity at a hub, prepared before the customer arrives |
| Technician    | Assigned by matching required skill to the requested service(s) |
| Skill         | A capability a technician holds, with a numeric level           |
| Material      | Consumed during a service; tracked in hub inventory              |
| Booking       | The customer's scheduled appointment                             |
| Bill          | Generated after service completion                               |

## 4. Customer Journey (Booking)

1. Customer views the list of their vehicles.
2. Customer chooses a service hub (location).
3. Customer chooses a vehicle and the maintenance/service(s) needed.
4. Customer chooses a time slot and books the appointment.
5. Customer may pay at the time of booking, or later (see §6).

## 5. Hub & Technician Journey (Fulfillment)

1. The hub receives the new booking from the customer.
2. The system assigns a technician whose skill matches the requested service(s).
3. The technician prepares the service bay and the required materials ahead of time.
4. The technician waits for the customer to arrive.
5. On arrival, the technician receives the vehicle and begins the service.
6. **Mid-service change:** if a new issue is discovered during the work, the technician requests the customer's approval to add extra services and/or materials to the booking.
7. The technician finishes the inspection and maintenance work.

## 6. Billing & Payment

After the service is completed, the system generates a bill.

```text
Bill total = warranty/expiration fee (only if the vehicle's warranty has expired or it has none)
           + service fees
           + material costs
```

- The customer pays once at the end, **or**
- If the customer already made a payment at booking time (a deposit), they pay only the remaining balance.

## 7. Technician Skill Progression

- Each technician holds a level per skill.
- Performing a service that uses a given skill **10 times** increases that skill's level by **+0.1**.

## 8. Materials Management

- Each hub tracks its own material inventory.
- Materials consumed during a service are deducted from the hub's stock and included in the customer's bill.

## 9. Reporting

- At the end of each day, the system produces a report for every dealership hub.

## 10. Open Questions

These need to be clarified before the requirement can be considered final (see `responsibility-planning.md` — "Clarify requirements" / "Double-check requirements"):

1. **Technician assignment** — is it automatic (system picks best-matching/least-loaded technician) or can hub staff override it manually?
2. **No availability** — if no bay or no qualified technician is free at the requested time, what should the customer see (alternative slots, waitlist, nothing)?
3. **Mid-service approval (§5.6)** — if the customer rejects the extra service/material request, what happens to the original booking? Can the customer be reached in real time, or is this handled asynchronously?
4. **Cancellation / no-show** — is there a policy, deadline, or fee?
5. **Deposit (§6)** — is the pre-payment a fixed amount, a percentage, or the full estimated cost?
6. **Warranty (§6)** — how is a vehicle's warranty tracked (start date + duration? per-service warranty?), and who sets it?
7. **Skill levels (§7)** — what is a technician's starting level, is there a maximum, and does a level affect anything (e.g. eligibility for harder services, pay)?
8. **Materials (§8)** — what happens if a hub runs out of a required material mid-service or before an appointment?
9. **Report content (§9)** — what does the end-of-day report contain (revenue, bookings completed, materials consumed, technician utilization), and who receives it?
10. **Multi-hub customers** — can one customer book the same vehicle at different hubs, and is service history shared across hubs?
