# Code Coverage Best Practices for Agentic Development

Research and analysis on optimal code coverage strategies when using AI coding agents.

## The Core Problem

When AI agents generate code, two opposing forces create tension:

1. **High coverage slows down** - More tests mean longer iteration cycles and higher token costs
2. **Low coverage risks regressions** - Agents lack institutional memory; they don't know if they broke something previously built

## Key Insight: Tests Are Institutional Memory

Human developers remember past bugs and context. Agents don't. **Tests become the primary mechanism for regression detection**, not a secondary safety net.

This shifts the value proposition: testing agent-generated code is MORE important than testing human-written code, because it's the only way agents learn what to preserve.

## The Middle Ground: Tiered Coverage Strategy

| Risk Tier | Coverage Target | What Goes Here |
|-----------|-----------------|----------------|
| Critical | 85-95% | Business logic, security, data integrity, money flows, API contracts |
| Core | 70-80% | Main workflows, state management, integrations |
| Low-risk | 50-60% | Getters/setters, DTOs, glue code, logging |

## Principles

### 1. Intention-Based > High Percentage

Every test should answer: **"What regression would this catch if it failed?"**

Tests that can't name a concrete regression are coverage-chasing noise. Focus on:
- Core business rules
- Edge cases that have caused bugs before
- Integration points between components
- Security and data integrity boundaries

### 2. The Real Cost Is Churn, Not Writing Tests

Implementation-coupled tests that break on refactors cause agent thrashing. This is far more expensive than writing fewer, behavior-focused tests.

**Behavior-focused tests** verify observable outputs and side effects rather than implementation details. They test WHAT the code does, not HOW it does it - so they survive refactoring.

| Behavior-Focused | Implementation-Coupled |
|------------------|------------------------|
| Tests inputs → outputs | Tests internal method calls |
| Breaks when behavior changes | Breaks when code is refactored |
| Mocks external dependencies only | Mocks internal collaborators |

### 3. Every Bug Becomes a Regression Test

When something breaks, the fix includes a test that would have caught it. This encodes institutional memory into the test suite.

### 4. Coverage as Diagnostic, Not KPI

Use coverage reports to **find untested critical paths**, not to hit arbitrary numbers.

- 80% with meaningful tests beats 95% with brittle ones
- Low coverage in a critical module is a signal to investigate
- Improving code (deleting dead branches) can legitimately lower coverage while increasing safety

### 5. Fast Feedback Loop Is Non-Negotiable

If your test suite runs under 3 minutes, run it on every agent iteration. The cost is worth the regression protection.

For longer suites:
- Run focused tests during iteration
- Run full suite before commit/merge
- Consider test parallelization

## Diminishing Returns After 80%

Research consistently shows diminishing returns kick in after 80-90% coverage:

| Aspect | 80-90% Coverage | 100% Coverage |
|--------|-----------------|---------------|
| Bug-finding value | High early, then flattens | Minimal for final 10-20% |
| Engineering effort | Proportional to benefit | Disproportionately high |
| Test complexity | Moderate, behavior-focused | Often brittle, over-specified |
| Design impact | Encourages testable code | Can incentivize design compromises |

The last 10-20% requires:
- Intricate test setups
- Heavy mocking
- Tests tightly coupled to implementation
- Covering defensive/error paths rarely hit in practice

## Agent-Specific Practices

### Minimum Rule

**ALL agent-generated code must have at least one intention-revealing test before merge.**

Not necessarily high coverage, but something that would fail if the core behavior broke.

### Have Agents Write Tests Too

Since agents can generate tests, the marginal cost of test creation is low. Prompt agents to:
- Write tests alongside implementation
- Generate edge case tests for their own code
- Create regression tests when fixing bugs

### Run Tests in the Agent Loop

Include test execution as part of the agent's implementation cycle:
```
implement → run tests → fix failures → verify tests pass → commit
```

This catches regressions before they propagate.

### Test Known Issue Scenarios

Maintain a list of known system failures and ensure tests cover these scenarios. Agents don't know your system's historical pain points unless tests encode them.

## Practical Recommendation

**Aim for 75-85% coverage on critical code with behavior-focused tests. Accept 50-70% elsewhere. Always run tests in the agent loop.**

The balance between cost/duration and regression protection favors **FAST, MEANINGFUL tests** over comprehensive slow tests.

## When Higher Coverage Makes Sense

Push toward 95%+ in these contexts:
- Safety-critical or regulated domains (medical, fintech, automotive)
- Strong TDD culture where tests come naturally with design
- Public APIs with stability guarantees
- Security-sensitive code paths

## Sources

- [Gremlin - Reliability Best Practices for AI Agents](https://www.gremlin.com/blog/three-reliability-best-practices-when-using-ai-agents-for-coding)
- [Google Testing Blog - Coverage Best Practices](https://testing.googleblog.com/2020/08/code-coverage-best-practices.html)
- [CMU SEI - Developer Testing Practices](https://www.sei.cmu.edu/blog/six-best-practices-for-developer-testing/)
- [Stack Overflow - Coverage Tradeoffs](https://stackoverflow.blog/2025/12/22/making-your-code-base-better-will-make-your-code-coverage-worse/)
- [Codacy - Best Practices for AI Coding](https://blog.codacy.com/best-practices-for-coding-with-ai)
- [Augment - Best Practices for AI Coding Agents](https://www.augmentcode.com/blog/best-practices-for-using-ai-coding-agents)
