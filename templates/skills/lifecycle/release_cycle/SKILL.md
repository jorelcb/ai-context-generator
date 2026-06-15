# Release Cycle Lifecycle

Take a codebase through a traceable, reversible release: confirm readiness, choose the version, update references, generate the changelog, commit, tag, push, deploy, and verify. This keeps every release documented and recoverable.

## When to use

- Cutting a new release once features and fixes are merged
- Bumping the version and producing a changelog
- Tagging and pushing a release, then triggering deployment
- Auditing whether a release followed a safe, reversible sequence

As a Claude Code skill, this may declare `allowed-tools` so safe local steps (editing version files, generating the changelog, creating the release commit) run without re-prompting. Keep irreversible, public steps — pushing a tag, publishing a release, deploying — interactive and explicitly confirmed.

## Process

### 1. Verify release readiness

Before anything else:

- All planned features and fixes are merged
- The test suite passes on the release branch
- No open blockers or critical issues remain
- You have reviewed the full set of changes since the last release

### 2. Determine the version number

Follow Semantic Versioning, driven by the commits since the last release:

- **MAJOR** (X.0.0): breaking changes to public APIs or contracts — any `feat!:` or `BREAKING CHANGE:`
- **MINOR** (x.Y.0): new backward-compatible features — any `feat:`
- **PATCH** (x.y.Z): backward-compatible fixes only — `fix:`, `docs:`, `chore:`

### 3. Update version references

Update every place the version appears:

- The version file or constant (`version.go`, `package.json`, `.version`, etc.)
- Documentation badges
- Any configuration that pins the version

### 4. Generate the changelog

Create or update the changelog in the project's format (CHANGELOG.md, GitHub releases, etc.):

- Group changes by type: Breaking Changes, Features, Bug Fixes, Other
- Place breaking changes prominently at the top
- Include commit references for traceability

### 5. Create the release commit

Make a single commit holding only version-related changes:

- Message: `chore: bump version to vX.Y.Z`
- Include the version-file edits and changelog updates
- Do not include any code changes in this commit

### 6. Create the git tag

Tag the release commit:

- Tag format `vX.Y.Z`, matching the project's convention
- Use an annotated tag carrying the changelog summary
- Pushing a tag is irreversible and public — confirm before pushing

### 7. Push the release

Push the release commit, then the tag, and verify both arrive at origin. Always push the commit before the tag so the tag never points at a commit that is not yet on the remote.

### 8. Trigger deployment

When the project has a CI/CD pipeline:

- Confirm CI/CD picks up the new tag
- Monitor the pipeline for failures
- Validate the deployment in the target environment

### 9. Publish release notes

When using GitHub/GitLab releases:

- Target the tag as the release
- Use the changelog entries as the release notes
- Attach relevant artifacts (binaries, packages)
- Mark as pre-release if appropriate

### 10. Verify the release is healthy

- The deployed version matches the release
- Smoke tests pass against the deployed environment
- Error rates and performance metrics show no anomalies
- Package-registry publication succeeded, if applicable

## Anti-patterns

- Releasing without running the full test suite
- Skipping the changelog, making later regression hunts harder
- Tagging before the commit is pushed, risking a tag/commit mismatch
- Including code changes in the version-bump commit
- Walking away after deploy instead of monitoring for early failures
- Drifting from a consistent semver tag format

## Verification

Before declaring the release done:

- [ ] Release branch is green and free of blockers
- [ ] Version bump matches the SemVer impact of the merged commits
- [ ] All version references and badges are updated
- [ ] Changelog is generated with breaking changes surfaced first
- [ ] Version bump commit contains no code changes
- [ ] Annotated tag was pushed only after the commit reached origin
- [ ] Deployment was monitored and the deployed version verified
