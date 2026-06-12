# api-design — Reference

## Contract tooling per ecosystem

| Concern                 | Tool                                  | Invocation / note                                        |
| ----------------------- | ------------------------------------- | -------------------------------------------------------- |
| Spec linting (REST)     | Spectral                              | `spectral lint openapi.yaml --ruleset .spectral.yaml`    |
| Spec linting (alt)      | vacuum, Redocly CLI                   | `vacuum lint openapi.yaml` / `redocly lint`              |
| Breaking-change diff    | oasdiff                               | `oasdiff breaking main.yaml head.yaml` in CI             |
| gRPC / protobuf         | buf                                   | `buf lint` + `buf breaking --against '.git#branch=main'` |
| Go server from spec     | oapi-codegen                          | generate handlers + types; spec is source of truth       |
| Go spec from code (alt) | swaggo/swag                           | only if code-first is already entrenched                 |
| TS types from spec      | openapi-typescript, orval             | generated client types; no hand-written DTOs             |
| TS schema-first (alt)   | zod + zod-openapi / @hono/zod-openapi | runtime validation and spec from one schema              |
| Mock server             | Prism                                 | `prism mock openapi.yaml` — front-end unblocks early     |
| Docs                    | Redoc / Scalar / Swagger UI           | rendered from the same spec                              |

Codify your conventions as Spectral rules (naming case, pagination params required on lists, Problem Details on errors) so the linter — not reviewers — enforces them.

## Status codes — extended guidance

| Code | Use precisely for                                                             |
| ---- | ----------------------------------------------------------------------------- |
| 200  | GET/PATCH/PUT success with body                                               |
| 201  | Resource created; include `Location` header                                   |
| 202  | Accepted for async processing; return a status-poll URL                       |
| 204  | Success, no body (DELETE, sometimes PUT)                                      |
| 304  | Conditional GET, `ETag`/`If-None-Match` matched                               |
| 400  | Malformed request (unparseable, unknown params)                               |
| 401  | Not authenticated (missing/invalid credentials)                               |
| 403  | Authenticated but not authorized                                              |
| 404  | Resource absent — also for authz-hiding if that's your policy (be consistent) |
| 409  | State conflict: duplicate create, stale `If-Match`, illegal transition        |
| 412  | Precondition (`If-Match`) failed — optimistic locking                         |
| 422  | Well-formed but semantically invalid (validation errors)                      |
| 429  | Rate limited; include `Retry-After`                                           |
| 500  | Bug. Never use for expected conditions                                        |
| 503  | Dependency down / maintenance; include `Retry-After`                          |

## Headers that are part of the contract

- `ETag` + `If-Match` — optimistic concurrency on updates.
- `Idempotency-Key` — client-supplied key on POST so retries don't double-create.
- `Deprecation` + `Sunset` + `Link rel="successor-version"` — deprecation lifecycle.
- `RateLimit-*` / `Retry-After` — quota visibility.
- `Cache-Control` on GETs — decide cacheability deliberately, don't inherit defaults.

## Pagination patterns — choosing

| Pattern | Shape                                    | Use when                              | Caveats                                                  |
| ------- | ---------------------------------------- | ------------------------------------- | -------------------------------------------------------- |
| Cursor  | `?cursor=opaque&limit=50` → `nextCursor` | Default; large/changing datasets      | Cursor must be opaque (encode, don't expose SQL offsets) |
| Offset  | `?offset=100&limit=50` + `total`         | Small stable sets; UIs needing page N | Skews under concurrent writes; slow deep pages           |
| Keyset  | `?after=2026-06-01T00:00:00Z`            | Time-ordered feeds                    | Needs a unique, indexed sort key                         |

Rules: `limit` capped server-side (document the cap); the response envelope for lists is uniform API-wide (`{ data: [...], nextCursor }`); cursors expire gracefully (400 with a clear code, not 500).

## Design-review checklist

- [ ] Spec exists and lints clean (`spectral`/`buf lint`); CI diffs it for breaking changes (`oasdiff`/`buf breaking`).
- [ ] URLs: plural nouns, kebab-case, ≤1 nesting level, no verbs mid-path.
- [ ] One JSON naming convention; response DTOs decoupled from storage models.
- [ ] Every list endpoint: pagination + capped limit + documented filters/sort.
- [ ] Every error: Problem Details shape with stable machine-readable `code`s.
- [ ] Writes: POST creates support idempotency keys; updates support `ETag`/`If-Match` where lost-update matters.
- [ ] AuthN/AuthZ requirement stated per endpoint in the spec (`security` section), not just in middleware.
- [ ] Versioning declared; deprecation policy written; unknown-field tolerance documented for clients.
- [ ] Each endpoint has a contract test asserting status codes + schema ([[test-strategy]]).
- [ ] Naming and shape changes reviewed as breaking-change candidates in [[code-review]].

## gRPC quick mapping

Same contract discipline, different mechanics: proto files are the spec; `buf lint` enforces style (package versioning `acme.orders.v1`), `buf breaking` gates evolution; errors use `google.rpc.Status` + typed details instead of Problem Details; pagination via `page_token`/`next_page_token` (AIP-158). Never reuse field numbers; mark removed fields `reserved`.
