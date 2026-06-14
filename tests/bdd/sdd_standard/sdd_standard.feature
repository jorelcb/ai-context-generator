Feature: SDD standard pluggable selection
  As a developer using codify
  I want to choose between Spec-Kit and OpenSpec (and future standards)
  So that codify spec produces the file set my team has standardized on

  Background:
    Given the SDD registry is loaded with default adapters

  # ===========================================================================
  # Spec-Kit adapter — the DEFAULT standard since v4.0.0 (audit decision D1)
  # ===========================================================================

  Scenario: Spec-Kit adapter is registered as the default
    When I look up SDD standard "spec-kit"
    Then the lookup should succeed
    And the standard's display name should be "GitHub Spec-Kit"
    And the standard's template directory should be "spec-kit"

  Scenario: Spec-Kit produces lowercase-named files
    When I look up SDD standard "spec-kit"
    Then the bootstrap artifacts should include exactly these files:
      | constitution.md |
      | spec.md         |
      | plan.md         |
      | tasks.md        |
      | research.md     |
      | data-model.md   |
      | quickstart.md   |
    And every required artifact should have a lowercase file name

  Scenario: Spec-Kit includes a project-level constitution outside the feature dir
    When I look up SDD standard "spec-kit"
    Then the artifact "constitution.md" should resolve to path ".specify/memory/constitution.md" for feature "checkout"
    And the artifact "constitution.md" should be optional and skip-if-exists

  Scenario: Spec-Kit feature artifacts live under specs/<feature>/
    When I look up SDD standard "spec-kit"
    Then the artifact "spec.md" should resolve to path "specs/checkout/spec.md" for feature "checkout"
    And the artifact "plan.md" should resolve to path "specs/checkout/plan.md" for feature "checkout"

  Scenario: Spec-Kit's spec, plan, tasks are required and the rest are optional
    When I look up SDD standard "spec-kit"
    Then the required artifact files should be exactly:
      | spec.md  |
      | plan.md  |
      | tasks.md |
    And the optional artifact files should be exactly:
      | constitution.md |
      | research.md     |
      | data-model.md   |
      | quickstart.md   |

  Scenario: Spec-Kit ships specify/plan/tasks lifecycle workflows
    When I look up SDD standard "spec-kit"
    Then the lifecycle workflow IDs should be:
      | speckit_specify |
      | speckit_plan    |
      | speckit_tasks   |

  Scenario: Spec-Kit hints point at the constitution location and the clarification marker
    When I look up SDD standard "spec-kit"
    Then the system prompt hints in "en" should mention "lowercase"
    And the system prompt hints in "en" should mention ".specify/memory/constitution.md"
    And the system prompt hints in "en" should mention "NEEDS CLARIFICATION"
    And the system prompt hints in "es" should mention ".specify/memory/constitution.md"

  # ===========================================================================
  # OpenSpec adapter — real OpenSpec structure (openspec/ root)
  # ===========================================================================

  Scenario: OpenSpec adapter is registered
    When I look up SDD standard "openspec"
    Then the lookup should succeed
    And the standard's display name should be "OpenSpec"
    And the standard's template directory should be "openspec"

  Scenario: OpenSpec produces a project doc and a capability spec in the openspec/ tree
    When I look up SDD standard "openspec"
    Then the bootstrap artifacts should include exactly these files:
      | project.md |
      | spec.md    |
    And every bootstrap artifact should be marked required
    And the artifact "project.md" should resolve to path "openspec/project.md" for feature "payments"
    And the artifact "spec.md" should resolve to path "openspec/specs/payments/spec.md" for feature "payments"

  Scenario: OpenSpec hints carry the literal requirement and scenario syntax
    When I look up SDD standard "openspec"
    Then the system prompt hints in "en" should mention "### Requirement:"
    And the system prompt hints in "en" should mention "#### Scenario:"
    And the system prompt hints in "en" should mention "RENAMED"

  Scenario: OpenSpec ships the propose/apply/archive lifecycle workflows
    When I look up SDD standard "openspec"
    Then the lifecycle workflow IDs should be:
      | spec_propose |
      | spec_apply   |
      | spec_archive |

  # ===========================================================================
  # Resolución por precedencia (ADR-0011) — default ahora spec-kit (D1)
  # ===========================================================================

  Scenario: Resolution falls through to the default when nothing is set
    When I resolve with flag "" project "" user ""
    Then the resolved standard ID should be "spec-kit"

  Scenario: User config alone selects the standard
    When I resolve with flag "" project "" user "openspec"
    Then the resolved standard ID should be "openspec"

  Scenario: Project config overrides user config
    When I resolve with flag "" project "openspec" user "spec-kit"
    Then the resolved standard ID should be "openspec"

  Scenario: CLI flag overrides project and user config
    When I resolve with flag "spec-kit" project "openspec" user "openspec"
    Then the resolved standard ID should be "spec-kit"

  Scenario: Unknown flag fails with explicit error listing available standards
    When I resolve with flag "does-not-exist" project "" user ""
    Then resolution should fail with error containing "does-not-exist"
    And resolution should fail with error containing "openspec"
    And resolution should fail with error containing "spec-kit"

  Scenario: Unknown project config value fails before silently falling back
    When I resolve with flag "" project "phantom" user "openspec"
    Then resolution should fail with error containing "phantom"

  # ===========================================================================
  # Templates físicas en el embedded FS
  # ===========================================================================

  Scenario: OpenSpec spec templates exist in the embedded filesystem (en)
    Then the embedded FS should contain template "templates/en/sdd/openspec/spec/openspec_project.template"
    And the embedded FS should contain template "templates/en/sdd/openspec/spec/openspec_spec.template"

  Scenario: Spec-Kit spec templates exist in the embedded filesystem (en)
    Then the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_constitution.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_spec.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_plan.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_tasks.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_research.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_data_model.template"
    And the embedded FS should contain template "templates/en/sdd/spec-kit/spec/speckit_quickstart.template"

  Scenario: Both standards have parallel ES templates
    Then the embedded FS should contain template "templates/es/sdd/openspec/spec/openspec_spec.template"
    And the embedded FS should contain template "templates/es/sdd/spec-kit/spec/speckit_spec.template"
