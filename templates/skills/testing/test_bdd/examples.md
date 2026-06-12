# test-bdd — Examples

## 1. Example Mapping output (Discovery)

The artifact a Three Amigos session produces before any Gherkin:

```
Story: Withdraw cash from an ATM
  Rule: Cannot withdraw more than the balance
    e.g. balance 100, withdraw 200 → declined, balance unchanged
    e.g. balance 100, withdraw 100 → approved, balance 0
  Rule: Cannot withdraw more than the daily limit
    e.g. limit 300, already withdrew 250, withdraw 100 → declined
  ? Does a declined withdrawal still print a receipt?   ← open question
```

Too many `?` ⇒ the story isn't understood; too many Rules ⇒ split the story.

## 2. Feature file (Formulation)

```gherkin
@withdrawal
Feature: Cash withdrawal
  As a customer I want to withdraw cash so that I have money on hand.

  Background:
    Given I am an authenticated customer

  Rule: Cannot withdraw more than the balance
    Scenario: Withdrawal within balance is approved
      Given my account balance is 100
      When I withdraw 100
      Then the withdrawal is approved
      And my balance is 0

    Scenario: Withdrawal over balance is declined
      Given my account balance is 100
      When I withdraw 200
      Then the withdrawal is declined
      And my balance is unchanged

    Scenario Outline: Balance boundary
      Given my account balance is <initial>
      When I withdraw <amount>
      Then the withdrawal is <result>

      Examples:
        | initial | amount | result   |
        | 100     | 100    | approved |
        | 100     | 200    | declined |
```

Note: declarative steps (no buttons, no fields) — one `When` per scenario.

## 3. Step definitions (Automation)

### Go — godog

```go
func InitializeScenario(ctx *godog.ScenarioContext) {
    var acc *account.Account
    var result string
    ctx.Step(`^my account balance is (\d+)$`, func(b int) error {
        acc = account.NewWithBalance(int64(b)); return nil
    })
    ctx.Step(`^I withdraw (\d+)$`, func(a int) error {
        result = acc.Withdraw(int64(a)); return nil // domain enforces the rule
    })
    ctx.Step(`^the withdrawal is (approved|declined)$`, func(want string) error {
        if result != want { return fmt.Errorf("got %s, want %s", result, want) }
        return nil
    })
}
```

### TypeScript — @cucumber/cucumber

```typescript
import { Given, When, Then } from "@cucumber/cucumber";
import assert from "assert";

let account: Account;
let result: string;

Given("my account balance is {int}", (b: number) => {
  account = Account.withBalance(b);
});
When("I withdraw {int}", (a: number) => {
  result = account.withdraw(a);
}); // domain rule
Then("the withdrawal is {word}", (want: string) => {
  assert.equal(result, want);
});
```

The step definitions are thin: they translate Gherkin to calls on the domain ([[ddd-entity]]) and assert outcomes. Business rules live in `Account`, never in the steps.
