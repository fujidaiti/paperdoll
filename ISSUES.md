# Known issues

Findings from the review of `server/feature/user/auth.go` against `TODO.md`.
These are accepted as real problems, to be fixed later.

## 1. The fail-count increment is not atomic, so the cap can be exceeded

Location: `server/feature/user/auth.go`, `Service.VerifySignUpEmailAddress`.

`fail_count` is read in the `SELECT` that loads the attempt, compared against
`maxFailCount`, and then incremented by a separate `UPDATE` whose error is only
logged. There are two consequences:

- Two concurrent requests with a wrong code both read the same `fail_count`,
  both pass the cap check, and both consume a guess. With enough concurrency, an
  attacker gets more than 5 guesses against a 6-digit code.
- If the `UPDATE` fails, the failure is logged and the request continues, so
  that guess is not counted.

Fix: increment and read back in a single statement, as `TODO.md` specifies:
`UPDATE pending_signup_attempts SET fail_count = fail_count + 1 WHERE id = $1 RETURNING fail_count`,
then compare the returned value against the cap instead of the separately read
one. Treat a failure of that statement as an error rather than continuing.

## 2. The attempt row is never deleted after successful verification

Location: `server/feature/user/auth.go`, `Service.VerifySignUpEmailAddress`.

`TODO.md` requires deleting the attempt row once the email is verified and
promoted to `users`. The current implementation leaves it, so the password hash
and the code hash of every verified account stay in `pending_signup_attempts`
indefinitely. The planned GC worker will not remove them either, because it only
deletes inert rows past the retention window.

Fix: delete the attempt row after the `users` insert succeeds.

Note that this changes an existing behavior. Today, verifying twice with the
same ticket returns `ErrEmailTaken`, because the row is still present, and
`TestAuth_VerifySignUpEmailAddress_DuplicateVerifications` in
`server/itest/user_test.go` asserts that. After the row is deleted, the second
call finds no attempt for the ticket and must return `ErrTokenInvalid`. Update
that test together with the fix.

## 3. Resend does not require a non-exhausted attempt

Location: `server/feature/user/auth.go`, `ResendSignUpVerificationEmail`.

`TODO.md` requires a live attempt, meaning neither expired nor exhausted. The
lookup selects `email`, `password_hash`, and `expires_at`, but not `fail_count`,
so a ticket whose fail cap is already exhausted still produces a new attempt.
Consequently, the API can never return 410 for the exhausted case on this
endpoint.

Fix: select `fail_count` as well and return `ErrTokenExpired` when it has
reached the cap. The cap is currently a local constant inside
`Service.VerifySignUpEmailAddress`, so it has to be shared between the two
functions.

## 4. Email uniqueness check and attempt insertion are not atomic

Location: `server/feature/user/auth.go`, `issueSignUpTicket`.

The `SELECT EXISTS(... FROM users WHERE email = $1)` check and the `INSERT` into
`pending_signup_attempts` run as separate statements, so two concurrent sign-ups
for an address that was just registered can both pass the check.

The impact is limited: `Service.VerifySignUpEmailAddress` re-checks uniqueness
through `INSERT ... ON CONFLICT (email) DO NOTHING`, so the only result is a
pending attempt row that can never be promoted. A comment recording this
reasoning may be enough, instead of a code change.

## 5. `VerificationCode.Match` returns true when both hashes are empty

Location: `server/feature/user/auth.go`, `VerificationCode.Hash` and
`VerificationCode.Match`.

`Hash` returns `nil` for an empty code, and
`subtle.ConstantTimeCompare(nil, nil)` returns 1. Therefore an empty submitted
code matches an empty stored hash.

This is currently unreachable, because `verification_code_hash` is declared
`NOT NULL` and inserting a `nil` hash fails. Still, the method should not report
a match for an empty code.

Fix: return false from `Match` when the code is empty or `codeHash` is empty.
