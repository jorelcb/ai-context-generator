# code-review — Examples (Go + TypeScript)

> Illustrative — adapt to the project's stack. Each example shows a diff fragment, a bad comment, and the comment this skill requires.

## Go — ignored error + race condition

```go
// diff under review
func (s *Store) Add(item Item) {
    s.items = append(s.items, item)        // s.items shared, no lock
    s.persist()                            // returns error, discarded
}
```

**Bad comment:** "This looks unsafe."

**Good comments:**

> **blocker:** `s.persist()` returns an `error` that is discarded — a failed write leaves the in-memory slice and the store silently diverged. Return the error: `func (s *Store) Add(item Item) error { ...; return s.persist() }`.

> **blocker:** `s.items` is appended without synchronization; `Add` called from concurrent handlers is a data race (`go test -race` will catch it). Guard with `s.mu.Lock()` or document single-goroutine ownership.

## Go — N+1 query

```go
for _, id := range orderIDs {
    o, err := repo.FindByID(ctx, id) // one query per ID
    ...
}
```

> **major:** This issues one query per order (N+1). For a 500-order page that's 500 round-trips. Add `repo.FindByIDs(ctx, orderIDs)` with a single `WHERE id = ANY($1)` and use it here.

## TypeScript — unvalidated input + injection

```typescript
// diff under review
app.get("/users", async (req, res) => {
  const rows = await db.query(
    `SELECT * FROM users WHERE name LIKE '%${req.query.q}%'`,
  );
  res.json(rows);
});
```

**Bad comment:** "Use an ORM."

**Good comments:**

> **blocker:** `req.query.q` is interpolated into SQL — injection (e.g. `q='; DROP TABLE users;--`). Use a parameter: `db.query("SELECT id, name FROM users WHERE name LIKE $1", [`%${q}%`])`.

> **major:** `SELECT *` returns every column, including `password_hash`, straight to the client. Select explicit public fields or map through a response DTO.

> **major:** No pagination — this returns the whole table. Add `LIMIT`/cursor per our list-endpoint convention (see [[api-design]]).

## TypeScript — floating promise

```typescript
function onSignup(user: User) {
  sendWelcomeEmail(user); // async, not awaited, no catch
  metrics.increment("signups");
}
```

> **major:** `sendWelcomeEmail` is async and unawaited — rejections become unhandled and failures are invisible. Either `await` it or explicitly fire-and-forget with logging: `sendWelcomeEmail(user).catch((e) => log.error("welcome email failed", e));` (eslint `@typescript-eslint/no-floating-promises` would flag this — worth enabling).

## Test-gap finding (either language)

> **major:** The fix changes rounding from truncation to half-up, but no test pins the new behavior — the bug can regress silently. Add a case with `2.345 → 2.35` to the existing price tests (see [[test-strategy]] for placement).

## Non-blocking note done right

> **nit (non-blocking):** `processData` and `handleData` in this file do nearly the same thing — pre-existing, not introduced here. Filed #482 to merge them; fine to ignore in this PR ([[refactor-safely]] covers landing that separately).

## Verdict block (end of every review)

```
Verdict: request changes
Blockers: SQL injection in GET /users; discarded persist() error in Store.Add
Majors: missing pagination; no regression test for rounding fix
Reviewed deeply: handlers + store. Skimmed: generated client code.
```
