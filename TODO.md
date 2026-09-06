# Sign-up email verification specification

Scope: server only. Client work (verification screen, ticket handoff, failure
surfacing) is tracked separately.

`POST /sign-up` changes from 201 with an auth token to 202 with a ticket. Ship
the server and the client together, or version the endpoint.

## Parameters

| Setting     | Value                                                      |
| ----------- | ---------------------------------------------------------- |
| Ticket      | 32 random bytes, the same construction as `AuthToken`      |
| Code length | 6 digits                                                   |
| Code hash   | bcrypt at cost 12, the same `bcryptCost` the passwords use |
| Code TTL    | 10 minutes                                                 |
| Throttle    | 3 sends per address per hour                               |
| Fail cap    | 5 wrong codes per attempt                                  |

- Build the ticket exactly as `AuthToken` in `server/feature/user/auth.go` does:
  32 bytes from `crypto/rand`, `base64.RawURLEncoding` on the wire, `sha256` at
  rest.
- Generate the code with `rand.Int` over `10^6`, not a modulo of a random
  integer, and format it with `%06d` so leading zeros survive.
- bcrypt is salted, so a row cannot be looked up by its code hash: fetch the
  attempt by ticket, then compare the submitted code against the stored hash.
- The sixth wrong guess makes the attempt inert.

## Data model

Create the `pending_signup_attempts` table. All columns are mandatory.

| Column                   | Meaning                                         |
| ------------------------ | ----------------------------------------------- |
| `id`                     | Primary key                                     |
| `email`                  | Submitted email address                         |
| `password_hash`          | Submitted password, hashed as in `users`        |
| `verification_code_hash` | One-time passcode, hashed as `password_hash` is |
| `ticket_hash`            | Hashed secret tying the code to the attempt     |
| `expires_at`             | Deadline of this attempt                        |
| `fail_count`             | How many times verification has failed          |
| `created_at`             | When the attempt was made and its code mailed   |

Only the user who made the sign-up request knows the ticket secret.

Indexes:

- `UNIQUE` on `ticket_hash`, the same way `auth_tokens.token_hash` is declared
- `(email, created_at)`, for the throttle's `COUNT(*)`

### One row per send

- Every send inserts a row: sign-up and resend both create a new attempt, and
  nothing updates an existing one in place.
- The throttle is a `COUNT(*)` over `created_at` filtered by address. There is
  no separate send log.
- `fail_count` is per row and starts at 0 on every new attempt. It is not
  carried over from the attempt a resend was based on.

### Attempt lifetime

- Delete a row only when the email is verified and promoted to a user.
- Leave rows that expire or exhaust their fail cap in place, still readable, so
  the API can tell the client why the attempt is dead.
- Do not revoke old attempts superseded by a resend; let them expire.
- Retain inert rows for `max(code TTL, throttle window)` plus a margin -- with
  the values above, the 1-hour throttle window -- not merely past `expires_at`.

## API

Error bodies keep the existing `{message}` shape. The status code carries the
distinction the client needs.

### Edit `POST /sign-up`

Registers a pending attempt and mails a code, instead of creating the user.

- Stop registering a new user.
- Stop issuing a new auth token.
- Drop `device` from the request.
- Reject the request if the email is already registered (unchanged 409).
- Apply the per-address send throttle before sending anything.
- Register the submitted email and password to the pending-attempts table.
- Return 202 with a ticket instead of 201 with an auth token.

| Status | Meaning                                      |
| ------ | -------------------------------------------- |
| 202    | `Body.ticket`                                |
| 400    | The email or password is invalid             |
| 409    | The email is already registered              |
| 429    | The address has been mailed too often lately |
| 500    | Unknown error                                |

### Add `POST /sign-up/verify-email`

Verifies the submitted email. Client apps call this after the user signs up and
receives an email with a verification code. The attempt is queried by ticket.

- Verify only if the pair of ticket and code is correct. Then move the email and
  password to `users`, issue a new auth token, and delete the attempt row.
- On a wrong code, increase the fail count. Once it reaches the cap, the attempt
  is dead: verification never succeeds again, even with the correct code.
- Increment in a single statement --
  `UPDATE ... SET fail_count = fail_count + 1 ... RETURNING fail_count` -- and
  check the cap against the returned value, not against a separately read one.

Request:

- `Body.ticket`: a secret returned by `POST /sign-up`
- `Body.verification_code`: the passcode sent via the email
- `Body.device`: same meaning as in `POST /sign-in`

| Status | Meaning                                                    |
| ------ | ---------------------------------------------------------- |
| 200    | Verified; `Body.token` carries a new auth token            |
| 400    | Malformed request (bad code format, empty device, ...)     |
| 401    | The code is wrong and attempts remain                      |
| 404    | No attempt matches the ticket                              |
| 409    | The address was registered by someone else in the meantime |
| 410    | The attempt is no longer usable (expired, or exhausted)    |
| 500    | Unknown error                                              |

### Add `POST /sign-up/resend-verification`

Re-sends a verification code to the address behind the given ticket, without the
client holding the password a second time.

- Do **not** update the existing attempt. Read the email and password hash off
  the attempt the ticket points at, then insert a brand new attempt from them:
  new code, new ticket, fresh `expires_at`, `fail_count` back to 0.
- Require a live attempt for the ticket (not expired, not exhausted).
- Reject the request if the address has been registered in the meantime.
- Apply the same per-address send throttle as `POST /sign-up`.
- Leave the previous attempt alone; it expires on its own.
- Return a new ticket. The client must replace the one it holds, and the code
  from the older email will not verify against it.

| Status | Meaning                                                 |
| ------ | ------------------------------------------------------- |
| 202    | `Body.ticket`, the same shape `POST /sign-up` returns   |
| 404    | No attempt matches the ticket                           |
| 409    | The address was registered in the meantime              |
| 410    | The attempt is no longer usable (expired, or exhausted) |
| 429    | The address has been mailed too often lately            |
| 500    | Unknown error                                           |

## Service layer

Add `user.Service.SendEmail` of type
`func(ctx context.Context, to, subject, body string) error`. The sender address
comes from config, not from the caller.

Change `user.Service.SignUp` to:

```go
SignUp(ctx context.Context, email CanonicalEmail, pswd ValidPassword) (Ticket, error)
```

The `device` parameter goes away, and the return value becomes a ticket instead
of an `AuthToken`. Steps:

1. Reject the email if it is already registered.
2. Apply the per-address send throttle.
3. Insert a new attempt to `pending_signup_attempts`.
4. Generate a verification code.
5. Send the email with the code.
6. Return a ticket for the attempt.

Add:

```go
ResendSignUpVerificationEmail(ctx context.Context, ticket string) (Ticket, error)
```

1. Query a live attempt for the given ticket, for its email and password hash.
2. Apply the per-address send throttle.
3. Insert a new attempt from them, exactly as `SignUp` does.
4. Generate a verification code.
5. Send the email with the code.
6. Return the new ticket.

Add:

```go
VerifySignUpEmailAddress(ctx context.Context, ticket, code, device string) (AuthToken, error)
```

1. Check if a live attempt exists for the ticket and code.
2. Register the verified email address and password hash to `users`.
3. Issue an auth token for the device.
4. Remove the attempt from DB.

### Send failures

- Call `SendEmail` synchronously, but never let its error change what the caller
  does: keep the attempt row and return 202 either way.
- Log the failure with the attempt id and the address.
- Do not send asynchronously. The user recovers through resend.

## Background workers

Clean up inert attempts, periodically. Retention is
`max(code TTL, throttle window)` plus a margin, not `expires_at`.

## Mail servers

- Use Resend in production, and Mailpit in development.
- Configure the server host (and secrets) via env variable(s).

### Email content

Fixed template. Keep the code on a line of its own, with no other 6-digit number
anywhere in the message:

```
Subject: Your verification code

Enter this code to finish creating your account:

123456

The code expires in 10 minutes. If you did not request it, ignore this email.
```

The test stub matches `\b(\d{6})\b`, so the template and the stub have to be
edited together.

## Testing

Generate the verification code with the real generator in tests, not a mock.
Install a stub `SendEmail` that extracts the code from the email body with the
regex given under "Email content" above, stores it in a variable, and forwards
it to `VerifySignUpEmailAddress`.

Update existing tests:

- Update `provisionTestAccount` (and the other direct `SignUp` call sites in
  `server/itest/` and `client/e2e/`) to run `SignUp` +
  `VerifySignUpEmailAddress` against the stub `SendEmail`.
- Rewrite the `SignUp` tests for the new behaviors.

Add tests for:

- `ResendSignUpVerificationEmail`
- `VerifySignUpEmailAddress`
- The throttle: the fourth send to an address inside an hour is refused, and a
  send is allowed again once the window has passed
- Expiry: an attempt older than the code TTL is refused, and its row is still
  readable so the refusal is distinguishable from an unknown ticket
- The fail cap: the sixth wrong code makes the attempt inert, and the correct
  code no longer verifies afterwards
- The GC worker: inert rows inside the retention window survive
