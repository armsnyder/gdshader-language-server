# Rules for Agents

## General Rules

1. Run `make reviewable` after every task, to run the tests and the linter.

## Code Patterns

1. Always wrap errors, like `fmt.Errorf("validate data: %w", err)`.
2. Use `log/slog` for logging.
3. Use modern Go features. The Go language and standard libraries have been updated significantly since the last time you looked at Go code. There are likely newer Go patterns that make Go easier to write. Examples: `cmp.Or(val, fallbackVal)`, `slices.Contains(slice, value)`,. `slices.Min(slice)`, use `any` and not `interface{}`, use `range x` to iterate x number of times.
4. Code structure: Public types go near the top. Helpers go below where they're called. Whenever you are finished with a coding task, look back at your code and re-organize the top-level declarations to adhere to this ordering rule.

## Code Comments

1. Comment WHY, not WHAT. Explain reasoning, include links to docs and other context. Do not state the obvious or summarize the code.
2. NEVER put specific values in comments that can become out of date, such as variable values.
3. DO use `HACK` and `TODO` comments liberally in situations where a feature is incomplete or does not meet the highest level production code standards.
4. GoDocs: Documentation should explain how to use the thing. Never document internal implementation details. Wrap GoDoc comments at 80 characters. When mentioning other functions/types, wrap in square brackets, e.g. `returns a [slog.Logger]`

## Test Code Patterns

1. Prefer blackbox tests: Test code should use `package name_test` and only test the public API of the package. Do not export types solely to be able to test them.
2. Prefer table-driven tests: Use table-driven tests with contrasting test cases to cover multiple scenarios with the same test function. If you are ever working on test code, whenever you finish adding tests, check to see if existing tests can be refactored to table-driven tests and do so.
3. Use Gomega: Do use the Gomega assertion library, but do not use Ginkgo. Use the dot-import `. "github.com/onsi/gomega"` in test code.
4. Use PascalCase for test names, the `t.Run("TestName")` argument, and `name` field in table-driven tests.
