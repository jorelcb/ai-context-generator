# domain-event — Reference

## Transactional outbox in depth

The outbox solves the **dual-write problem**: you cannot atomically write to a database _and_ publish to a broker. Anything that tries (publish-then-commit, commit-then-publish, best-effort retries) has a window where one side succeeds and the other does not.

**Table shape** (minimum): `id`, `event_id`, `event_type`, `aggregate_id`, `payload` (JSON/bytes), `occurred_at`, `published_at NULL`.

**Relay options:**

| Relay                        | How                                              | Trade-off                                            |
| ---------------------------- | ------------------------------------------------ | ---------------------------------------------------- |
| Polling publisher            | Periodic `SELECT ... WHERE published_at IS NULL` | Simple, portable; adds polling latency               |
| CDC (e.g. Debezium)          | Tail the DB log, stream outbox inserts to broker | Near-real-time, no app code; heavier infra           |
| In-process after-commit hook | Publish after tx commit, outbox as fallback      | Lower latency; still needs the poller for crash gaps |

Whichever relay: delivery is **at-least-once**. The relay may crash after publishing but before marking `published_at` — the row is re-published. This is by design; consumers dedup ([[event-idempotency]]).

Ordering: publish per-aggregate in `occurred_at`/sequence order (partition by `aggregate_id` on partitioned brokers like Kafka). Cross-aggregate global ordering is not promised — don't design consumers that need it.

## Fat vs thin events — decision table

| Question                                                    | Lean thin (IDs + few fields)   | Lean fat (carry the data)                 |
| ----------------------------------------------------------- | ------------------------------ | ----------------------------------------- |
| Can consumers query the producer cheaply and synchronously? | Yes → thin acceptable          | No / cross-team boundary → fat            |
| Is the data needed at _reaction time_ or _display time_?    | Display → thin + projection    | Reaction → fat                            |
| Does the payload change often?                              | Yes → thin (less schema churn) | No → fat                                  |
| Is the consumer outside your bounded context?               | —                              | Fat, with a **published-language** schema |

Default for integration events between contexts: **moderately fat** — the fields of the fact, none of the aggregate's internals.

## Versioning and upcasters

- `event_version` starts at 1. Additive changes don't bump it.
- A breaking restructure = new version (or a new event type, e.g. `OrderPlacedV2`). Producers switch; old events stay as written.
- An **upcaster** is a read-time translator: when a consumer/store reads `OrderPlaced v1`, the upcaster maps it to the v2 in-memory shape (filling defaults, renaming fields). Chain upcasters v1→v2→v3; never rewrite stored events.

## PII and crypto-shredding

Events outlive users and GDPR deletion requests, and most brokers/stores are append-only. Strategies:

1. **Tokenize**: store a reference to PII held in a mutable, deletable store.
2. **Crypto-shredding**: encrypt PII fields with a per-subject key kept in a key store; "delete the user" = delete the key — ciphertext in the immutable stream becomes unreadable.
3. **Don't put it in events** at all when consumers don't need it (the best option, often available).

## Schema registries and contracts

- On Kafka-style platforms, register event schemas (Avro/Protobuf/JSON Schema) in a **schema registry** with compatibility mode `BACKWARD` (new schema can read old data) — this mechanically enforces the evolution table in the SKILL.
- Without a registry, treat the event schema as a versioned contract in the repo and add consumer-driven contract tests.

## Tooling for local development and tests

- **testcontainers** (Go: `testcontainers-go`; TS: `testcontainers`) to spin up Postgres + the broker (Kafka via `redpanda` image for speed, or RabbitMQ) in integration tests — assert outbox rows become broker messages.
- Outbox relay tests: kill the relay between "publish" and "mark published", restart, assert the consumer saw the event twice and deduped ([[event-idempotency]]).
- Keep a **canonical example event** per type as a fixture; contract tests diff real payloads against it.
