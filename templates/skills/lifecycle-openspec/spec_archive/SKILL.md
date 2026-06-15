# Spec Archive (OpenSpec)

Finalize an applied OpenSpec change. Consolidate the change's spec deltas into the source-of-truth `openspec/specs/<capability>/spec.md`, merge the feature branch, move the change workspace into the dated archive, and clean up. This is phase 3 of 3 in the OpenSpec lifecycle: `/spec-propose` -> `/spec-apply` -> `/spec-archive`. It requires an approved, mergeable PR from [[spec-apply]].

## When to use

- A change's PR is approved and ready to merge, and its deltas must become canonical
- You need to fold `## ADDED|MODIFIED|REMOVED|RENAMED Requirements` into the live specs
- You want the change preserved as an audit record rather than deleted
- Closing the OpenSpec lifecycle that began with [[spec-propose]] and ran through [[spec-apply]]

This is the only phase that writes to the source-of-truth `openspec/specs/` — until now everything lived in deltas under the change workspace.

## Process

### 1. Verify the apply phase is complete

Do not archive prematurely. Confirm every task in `tasks.md` is checked or explicitly skipped, all tests pass on the feature branch, and the PR is approved and ready to merge. If any condition fails, return to [[spec-apply]].

### 2. Consolidate deltas into the source-of-truth specs

For each delta at `openspec/changes/<change-id>/specs/<capability>/spec.md`, open the matching `openspec/specs/<capability>/spec.md` (create it if the capability is new) and fold the deltas in:

- **ADDED** — append each new `### Requirement:` block, scenarios included
- **MODIFIED** — replace the existing requirement with the revised version; drop the `(Previously: …)` annotation, since the canonical spec records only the current state
- **REMOVED** — delete those requirements from the spec entirely
- **RENAMED** — update the requirement heading to the new name, preserving its scenarios

Leave every untouched requirement exactly as it was.

### 3. Validate the merged specs

Confirm the consolidated specs are well-formed before committing: every `### Requirement:` has at least one GIVEN/WHEN/THEN `#### Scenario:`, no two requirements share a name within a capability, every capability directory has a `spec.md`, and the markdown is structurally clean. Run `openspec validate` if available.

### 4. Commit the consolidation

Stage `openspec/specs/` and commit as `archive: <change-id> - consolidate spec deltas`. This commit lives on the feature branch alongside the implementation.

### 5. Move the change into the dated archive

Capture today's date as `YYYY-MM-DD` and move `openspec/changes/<change-id>/` to `openspec/changes/archive/YYYY-MM-DD-<change-id>/`. The date-stamp preserves the proposal artifacts and history indefinitely. Commit as `archive: move <change-id> to archive/`. Never delete the change directory — archiving keeps the audit trail intact.

### 6. Merge the feature branch

Merge using the repo's preferred strategy (squash, merge commit, or rebase), with the change-id and one-line summary as the merge title and any closed issues referenced in the body. Verify the merge landed on the base branch. Do not force-push the base branch.

### 7. Delete the feature branch

Once merged, remove the local branch (`git branch -d feature/<change-id>`) and the remote (`git push origin --delete feature/<change-id>`), then confirm it is gone with `git branch -a`.

### 8. Verify deployment

If the project has a CD pipeline, watch it complete, run smoke tests against the deployed environment, and monitor error rates and key metrics for regressions. If problems surface, prepare a rollback or a fresh change via [[spec-propose]].

### 9. Communicate completion

Notify stakeholders: update the original ticket or issue with the merge link, note any follow-up tasks that emerged during apply, and document deployment caveats or required user actions.

## Anti-patterns

- Archiving while tasks are still unchecked or the PR is unmerged
- Merging deltas into the specs without validating the result
- Deleting the change directory instead of moving it to `archive/` (loses history)
- Skipping the date-stamped archive move, leaving specs without an audit trail
- Leaving `(Previously: …)` annotations in the canonical spec
- Merging the feature branch before the specs are consolidated
- Force-pushing the base branch, or skipping deployment verification on production-affecting changes

## Verification

Before closing out:

- [ ] Every delta is folded into `openspec/specs/<capability>/spec.md` (ADDED/MODIFIED/REMOVED/RENAMED handled)
- [ ] Merged specs validate — one scenario per requirement, no duplicate names, clean markdown
- [ ] Consolidation committed and the change moved to `openspec/changes/archive/YYYY-MM-DD-<change-id>/`
- [ ] Feature branch merged to base and deleted locally and remotely
- [ ] Deployment verified (if applicable) and stakeholders notified

Previous phases: [[spec-propose]] then [[spec-apply]]. The change is now part of the canonical specs.
