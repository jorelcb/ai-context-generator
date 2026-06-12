# Domain Event Skill

> **Scope:** events as **facts** — immutable records of something that already happened in the domain. This skill covers naming, schema design, evolution, and publishing through the transactional outbox. Producing events is the job of aggregates invoked by a [[command-handler]]; consuming them safely is [[event-idempotency]] and [[event-projection]].

## When to use

Designing the event a new use case will emit, adding an event for a feature, refactoring a status flag or CRUD "updated" notification into proper events, or evolving the schema of an event already in production.

## Naming (normative)

- **Past tense, domain-meaningful:** `OrderPlaced`, `OrderShipped`, `PaymentApproved`, `ReservationCancelled`.
- NEVER generic CRUD names: `OrderCreated` says nothing a domain expert would say; `OrderUpdated` has no semantic meaning at all.
- NEVER imperative names: `PlaceOrder` is a command (a request that can be rejected); `OrderPlaced` is a fact (it cannot be rejected by consumers — only reacted to).
- One event = one fact. If a name needs "And", split it.

## Schema design

1. **Envelope** (same for every event): `event_id` (unique), `event_type`, `event_version` (default 1), `aggregate_id`, `occurred_at`, `correlation_id`, `causation_id`.
2. **Payload**: only what subscribers need to **react** — not the entire aggregate state.
   - Too thin (only an ID) forces consumers to query back, defeating async decoupling.
   - Too fat (full aggregate dump) couples every consumer to your internal structure.
   - Rule of thumb: the fields a domain expert would mention when describing the fact aloud.
3. **Self-contained**: no references to internal services, no URLs to fetch the "real" data.
4. **No raw PII** — events are retained long-term and replayed; encrypt or tokenize sensitive fields (see crypto-shredding in [reference.md](reference.md)).

## Schema evolution (normative)

| Change              | Allowed?  | How                                                            |
| ------------------- | --------- | -------------------------------------------------------------- |
| Add optional field  | Yes       | Compatible; readers ignore unknown fields                      |
| Remove a field      | **Never** | Events are immutable once produced; deprecate, stop writing it |
| Rename a field      | No        | Introduce a new event type (or version) and migrate producers  |
| Restructure payload | No        | New event type/version; upcasters translate old events at read |

Consumers MUST tolerate unknown fields. Producers MUST never break a published shape.

## Publishing — transactional outbox (mandatory)

1. The aggregate produces the event when its invariant-protected method succeeds.
2. The [[command-handler]] persists aggregate state **and** the event into an `outbox` table **in the same database transaction**.
3. A separate relay process (poller or CDC) reads the outbox and publishes to the broker, marking rows as sent.
4. This guarantees **at-least-once** delivery — never zero, possibly more than once. Consumers handle redelivery via [[event-idempotency]].

Never publish directly to the broker from the handler ("dual write"): if the transaction commits but the publish fails (or vice versa), state and stream diverge silently.

## Anti-patterns to avoid

- **Anemic events** carrying only an ID — consumers must query back, recoupling everything synchronously.
- **State-dump events** (`OrderState`) instead of change-expressing facts (`OrderShipped`).
- **Mutable events** — editing fields after publish; fix forward with a new event.
- **Schema = aggregate internals** — exposing private structure couples all consumers to your refactors.
- **Dual writes** — broker publish outside the state transaction.
- **PII in clear text** in long-retention streams.

## Verification

- [ ] Name is past tense and a domain expert would recognize it.
- [ ] A subscriber can react without querying the producer back.
- [ ] Envelope carries event_id, aggregate_id, occurred_at, correlation/causation IDs, version.
- [ ] Event is written via the outbox, in the same transaction as state.
- [ ] Reading the event ten years later still makes domain sense.

## Going deeper

- **[reference.md](reference.md)** — outbox relay options in depth, fat vs thin event decision table, upcasters, crypto-shredding for PII, schema registries, local-broker tooling.
- **[examples.md](examples.md)** — `OrderPlaced` envelope + payload, raising from the aggregate, and the outbox write, in Go and TypeScript.
