# DDD Entity Skill

> **Placement:** everything here lives in the **Domain layer**. The driven ports an aggregate needs (repositories) are defined with [[hexagonal-port]]; orchestration around it belongs to an Application Service ([[cqrs-command]]). The layer rules come from [[clean-arch-layer]].

## When to use

Creating a new aggregate, adding a value object, encoding an invariant/business rule, replacing primitives (string/int) with domain types, or emitting domain events.

## Design process

1. **Find the aggregate boundary** — what must stay transactionally consistent together?
2. **Pick the aggregate root** — the only entry point; external code touches the aggregate only through it.
3. **Extract value objects** — replace primitives that carry rules (Email, Money, DateRange).
4. **Encode invariants** — make invalid state unrepresentable; enforce at construction and on every mutation.
5. **Provide factory methods** — named constructors that return a _valid_ aggregate or an error; no half-built objects.
6. **Define domain events** if other parts of the system react to the change.

## Core rules (normative)

**Value Objects**

- Immutable: no setters; "changing" returns a new instance.
- Equality **by value**, not identity.
- **Self-validating**: the constructor rejects invalid input — an existing VO is always valid.
- No identity, no lifecycle.

**Entities & Aggregate Root**

- Identity-based equality (stable ID across state changes).
- All external access goes **through the root**; internal members are not exposed mutable.
- The aggregate is the **transactional consistency boundary** — keep it small.
- Reference **other aggregates by ID only**, never by direct object pointer.

**Invariants**

- Enforced inside the domain, never in the application service or handler.
- A mutation that would break an invariant must fail (return error / throw domain error), not silently correct.

## Anti-patterns to avoid

- **Anemic model**: entity = bag of public getters/setters, logic elsewhere.
- **Primitive obsession**: `string email`, `int cents` instead of `Email`, `Money`.
- **God aggregate**: spanning many transactional boundaries; split it.
- **Leaking domain logic** into application/infrastructure layers.
- Injecting repositories/HTTP/DB into entities (the Domain imports nothing — see [[clean-arch-layer]]).

## Validation checklist

- [ ] Every primitive carrying a rule is a value object.
- [ ] Invalid construction is impossible (factory returns error / throws).
- [ ] All mutation paths re-check invariants.
- [ ] Aggregate references others by ID only.
- [ ] Domain package has zero infrastructure imports.
- [ ] Invariants are covered by unit tests asserting the _failure_ path.

## Examples

Value object + aggregate root with invariants and factory, in Go and TypeScript, with an invariant-failure test: **[examples.md](examples.md)**.

## Reference

Deeper guidance (entity vs VO decision, immutability per language, domain events, aggregate sizing): **[reference.md](reference.md)**.
