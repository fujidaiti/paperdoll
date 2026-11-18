---
name: testing
description: Consult this skill whenever writing, running, editing, debugging, or reviewing ANY test in this project — unit tests, integration tests, backend tests, client tests, or any other test type. Always check this skill first for testing-related tasks, even if the user doesn't explicitly say "test" but is clearly working with test files or test directories. It routes to the authoritative reference for each test category.
---

# Testing 

Before touching any test, identify the category and read the matching reference below first. Each reference is the source of truth for that category (setup, conventions, commands, rules).

- Client widget tests: `client/test/README.md`
- Server integration tests: `server/itest/README.md`
- Server unit tests of the post selectors (txtar cases and web pages): `server/feature/feed/testdata/README.md`
- E2E tests: `client/e2e/README.md`

## Basic rules

- NEVER generate fixture in test code. Store them as static files.
- While sharing the same fixture with multiple test cases is fine, prefer duplicating fixtures over creating complex fixtures that covers all cases.
- Do not create helper functions without an explicit need. If there's only one call site for a helper, we don't need it.
- Do not bypass business logics as much as possible.
- "Redundant but explicit" is better than "short but implicit".
