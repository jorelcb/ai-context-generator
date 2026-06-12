# bdd-scenario — Examples

> Mechanics of a well-formed feature file. For the discovery that precedes this and the full step-definition treatment, see [[test-bdd]].

## Feature file with Background, Rule and Scenario Outline

```gherkin
@withdrawal
Feature: Cash withdrawal
  As a customer I want to withdraw cash so that I have money on hand.

  Background:                        # shared by every scenario — keep tiny
    Given I am an authenticated customer

  Rule: Cannot withdraw more than the balance
    Scenario: Withdrawal within balance is approved
      Given my account balance is 100
      When I withdraw 100            # exactly one When
      Then the withdrawal is approved
      And my balance is 0

    Scenario: Withdrawal over balance is declined
      Given my account balance is 100
      When I withdraw 200
      Then the withdrawal is declined
      And my balance is unchanged    # no incidental detail, behavior-focused

    Scenario Outline: Balance boundary   # same behavior, varies only by data
      Given my account balance is <initial>
      When I withdraw <amount>
      Then the withdrawal is <result>

      Examples:
        | initial | amount | result   |
        | 100     | 100    | approved |
        | 100     | 200    | declined |
        | 0       | 50     | declined |
```

Why this is good: declarative (no buttons/fields), one `When` each, ≤ 4 steps, scenarios independent, names read like business rules.

## Thin step-definition skeleton

### Go — godog

```go
func InitializeScenario(ctx *godog.ScenarioContext) {
    var acc *account.Account
    var result string
    ctx.Step(`^my account balance is (\d+)$`, func(b int) error {
        acc = account.NewWithBalance(int64(b)); return nil       // arrange
    })
    ctx.Step(`^I withdraw (\d+)$`, func(a int) error {
        result = acc.Withdraw(int64(a)); return nil               // act (domain rule)
    })
    ctx.Step(`^the withdrawal is (approved|declined)$`, func(want string) error {
        if result != want { return fmt.Errorf("got %s, want %s", result, want) } // assert
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

The steps only arrange/act/assert and share state through a context — the rule lives in `Account` ([[ddd-entity]]), never in the step.
