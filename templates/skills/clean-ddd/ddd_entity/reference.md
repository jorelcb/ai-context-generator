# ddd-entity — Reference

## Entity vs Value Object — decide fast

| Ask                                                   | Entity                   | Value Object                     |
| ----------------------------------------------------- | ------------------------ | -------------------------------- |
| Does it need a stable identity over time?             | Yes                      | No                               |
| Are two instances with equal fields interchangeable?  | No                       | Yes                              |
| Does it have a lifecycle (created, changed, deleted)? | Yes                      | No                               |
| Examples                                              | Order, Customer, Account | Money, Email, Address, DateRange |

When in doubt, prefer a **value object** — they are simpler, immutable and testable.

## Immutability per language

- **Go**: no real immutability; enforce it by convention — unexported fields + constructor + value receivers, no setters. Return new structs instead of mutating.
- **TypeScript**: `readonly` fields, `private constructor` + static factory, return new instances. Consider `Object.freeze` at the boundary if defensive copies matter.

## Factory methods

A factory (named constructor) is the **only** way to build a valid aggregate:

- It validates all invariants before returning.
- On failure it returns an error (Go) / throws a domain error (TS) — never a partially-built object.
- Distinguish _create_ (new identity, may emit a `Created` event) from _reconstitute_ (rebuild from storage, no events) — repositories use reconstitution.

## Domain events

- Raise an event when something business-meaningful happened (`OrderPlaced`), recorded on the aggregate.
- Events are **value objects**: immutable, named in past tense.
- The aggregate records them; the Application Service / infrastructure dispatches them after the transaction commits. The domain does not publish them itself (no infrastructure in Domain).

## Aggregate sizing heuristics

- Smaller is better; a large aggregate kills concurrency (whole thing locks).
- If two parts don't need to be consistent in the _same_ transaction, they are probably **separate aggregates** linked by ID.
- Eventual consistency between aggregates is normal and healthy.

## Where the repository goes

The aggregate's repository is a **driven port (interface) defined in the Domain** next to the aggregate, implemented in Infrastructure. Design it with [[hexagonal-port]]; one repository per aggregate root.
