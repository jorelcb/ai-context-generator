# API Design Skill

> An API is a **contract with people you can't refactor**. Design it spec-first, keep it boringly consistent, and treat every breaking change as a versioned event. REST-centric; the contract rules apply equally to gRPC/RPC. Go + TypeScript used as reference stacks.

## When to use

Designing new endpoints, defining or evolving request/response schemas, choosing a versioning approach, reviewing an API for consistency, or writing the OpenAPI document.

## Resource modeling (normative)

- URLs are **nouns for resources**, plural, kebab-case: `/purchase-orders/{id}`, never `/getPurchaseOrder`.
- Nest only for true ownership, max one level: `/orders/{id}/items`; relate everything else by ID or query param.
- Actions that don't map to CRUD become **sub-resources or controller verbs at the end**: `POST /orders/{id}/cancellation` (preferred) or `POST /orders/{id}:cancel` — never verbs in the middle of paths.
- Responses expose a **representation DTO**, never the internal/DB model — fields are deliberately chosen, internal IDs/flags stay internal.
- One JSON field-naming convention API-wide (pick `camelCase` or `snake_case`, then never mix).

## Methods and status codes — decision table

| Operation              | Method | Success             | Notes                                              |
| ---------------------- | ------ | ------------------- | -------------------------------------------------- |
| Read one / list        | GET    | 200                 | Safe, cacheable, never mutates                     |
| Create                 | POST   | 201 + `Location`    | Support an idempotency key for retries             |
| Full replace           | PUT    | 200 / 204           | Idempotent by definition                           |
| Partial update         | PATCH  | 200                 | JSON Merge Patch unless you need JSON Patch        |
| Delete                 | DELETE | 204                 | Idempotent: repeat delete → 204 or 404, never 500  |
| Validation failure     | —      | 422 (or 400)        | Field-level details, see error format              |
| Auth missing / wrong   | —      | 401 / 403           | 401 = who are you; 403 = you can't do that         |
| Conflict / stale write | —      | 409                 | Pair with `ETag`/`If-Match` for optimistic locking |
| Rate limited           | —      | 429 + `Retry-After` |                                                    |

## Error format (one shape, everywhere)

Use **RFC 9457 Problem Details** (`application/problem+json`): `type`, `title`, `status`, `detail`, `instance`, plus an `errors[]` array with `{field, code, message}` for validation. Machine-readable `code`s are part of the contract — clients branch on them, so renaming one is a breaking change. Never leak stack traces, SQL, or internal class names.

## Collections (normative)

- **Every list endpoint paginates from day one** — retrofitting pagination is a breaking change. Default cursor-based (`?cursor=...&limit=50`, response carries `nextCursor`); offset only for small, stable datasets.
- Filtering via query params (`?status=open`), sorting via `?sort=-createdAt` (leading `-` = descending). Document the allowed fields; reject unknown ones with 400.
- Cap `limit` server-side; an uncapped list endpoint is a self-inflicted DoS.

## Versioning and evolution

- Version from day one: URL path `/v1/` (visible, cache-friendly) is the pragmatic default; media-type/header versioning if the org already does it.
- **Non-breaking** (no new version): adding optional request fields, adding response fields, new endpoints. Clients must be told to ignore unknown response fields.
- **Breaking** (new version or deprecation cycle): removing/renaming fields, type changes, semantics changes, new required params, tightening validation, changing error `code`s.
- Deprecate with `Deprecation`/`Sunset` headers + a dated migration note; keep N-1 alive for the published window. Detect accidental breaks mechanically with `oasdiff` — see Verification.

## Anti-patterns

- Exposing the database row as the response schema — couples every consumer to your storage.
- Verbs in URLs (`/createUser`), GET that mutates, POST used for everything.
- 200 with `{"error": ...}` in the body — breaks every HTTP-aware client and cache.
- List endpoints without pagination or limit caps.
- Per-endpoint bespoke error shapes; clients end up with a parser per route.
- Designing in code first and "generating the spec later" — the contract becomes whatever the implementation accidentally does (review this in [[code-review]]).

## Verification (executable)

The contract is linted and diffed, not eyeballed: lint the OpenAPI doc with `spectral lint openapi.yaml` (custom ruleset = your conventions codified) or `vacuum lint`; detect breaking changes in CI with `oasdiff breaking base.yaml revision.yaml`; for gRPC, `buf lint` + `buf breaking`. Keep server and spec from drifting by generating one from the other (`oapi-codegen` in Go, `openapi-typescript`/`zod-openapi` in TS) and add a contract test per endpoint ([[test-strategy]]). Full tooling table in **[reference.md](reference.md)**.

## Going deeper

- **[reference.md](reference.md)** — spec-first tooling per ecosystem, extended status-code and header guidance, pagination patterns, design-review checklist.
- **[examples.md](examples.md)** — one resource end-to-end: OpenAPI fragment, Problem-Details error, cursor pagination, Go and TypeScript handlers.
