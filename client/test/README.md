# Widget tests

Tests live in `test/` and run on the Flutter test framework — **no device
needed**. The whole app is booted with `patrolWidgetTest`; HTTP is answered
in-process by a `StubServer` Dio interceptor, and platform plugins (secure
storage, device info) are replaced with in-memory stand-ins.

```sh
make test        # or: fvm flutter test  (run from the project root)
```

The real-backend E2E suite lives separately in `client/e2e/` (Patrol on a real
device, driven by the Go runner in `../e2e`).

## File organisation

Feature tests live under `test/features/`, one file per app feature:

```
test/
  features/
    auth_test.dart          # sign-in / sign-up
    newspaper_test.dart     # today / newspaper
    feed_test.dart          # feeds
    reading_list_test.dart  # reading list
  core/
    error_interceptor_test.dart  # plain unit tests for lib/core/
  src/
    boilerplate.dart        # pumpApp + in-memory platform fakes
    fixture.dart            # shared test data (feeds, entries, stories, …)
    stub_server.dart        # StubServer HTTP interceptor
```

Add a new `<feature>_test.dart` for each new feature; don't group unrelated
features into one file. Shared setup belongs in `src/boilerplate.dart`.
`test/core/` holds plain unit tests for the shared layer under `lib/core/`,
which need no app boot.

## How mocking works

Build a `StubServer`, register the routes the test needs on it, then boot the
app with it:

```dart
patrolWidgetTest('...', (t) async {
  final server = StubServer.withDefaultResponses()
    ..stubGet('/feeds', body: feedsBody)
    ..stubPost('/reading-list', status: 201, body: itemBody);
  await pumpAppWithAuth(t, server);

  // Test actions follow.
});
```

- **Stub before booting.** The shell builds all three tabs on startup, so the
  data a screen shows on first load (Today's stories, the feeds list, the
  reading list) is fetched _during_ boot. That's why the test owns the server
  and stubs it up front — a route registered after `pumpApp` returns is too late
  for that first fetch, and the screen has already rendered its empty default.
- `pumpApp(t, server)` starts **signed out** (lands on the sign-in screen).
  `pumpAppWithAuth(t, server)` starts signed in, landing straight on Today —
  feature tests that aren't about the auth flow use it. (`pumpApp` also takes a
  `token:` when a test needs a specific one.)
- `StubServer.withDefaultResponses()` pre-stubs the three shell tabs
  (`/newspapers/today`, `/reading-list`, `/feeds`) with realistic data from
  `src/fixture.dart`, so any test can boot and navigate without stubbing them
  itself. Override a tab (register it again) when a test needs it in a specific
  state — empty, or holding particular items.
- Any request no stub matches fails the test at teardown, naming the endpoint.
- The stubbed body is JSON round-tripped before it reaches the app, exactly like
  real transport — so passing generated `api.*` models (whose `toJson()` is
  shallow) through `.toJson()` works, nested models included.

### Building response bodies

Use the generated `api.*` models (from `package:openapi`) for type safety:

```dart
final feed = api.Feed(id: 1, url: '...', title: 'Anthropic Engineering Blog');
server.stubGet('/feeds', body: api.GetFeeds200Response(feeds: [feed]).toJson());
```

### Overriding routes — last registration wins

`StubServer` is **not a queue**: when several registrations match the same
request, the last one registered wins, and it answers _every_ matching request
(there's no "first call returns X, second returns Y"). Two registrations for the
same path coexist when their `bodyMatcher`s differ, though — that's how a test
answers one request body with a failure and another with a success. Register a
route again to override an empty default:

```dart
// Overrides the empty default; answers every /feeds load with this feed.
server.stubGet('/feeds', body: api.GetFeeds200Response(feeds: [feed]).toJson());
```

### Matching PUT / POST / PATCH with a body

Pass the exact expected body as `bodyMatcher` when the shape is known — this
turns the stub into an implicit assertion that the app sends the right payload.
`bodyMatcher` matches as a **subset** of the request body (extra keys ignored);
omit it to match any body.

```dart
server.stubPut(
  '/feeds',
  body: feed.toJson(),
  bodyMatcher: {'url': 'https://dart.dev/blog/feed.xml'},
);
```

Prefer an explicit `bodyMatcher` for `stubPost`/`stubPatch` — those mutations
are exactly where body correctness matters.

### Stateful stubs — responses that change across a request

`stub*` answers with a body fixed at registration time. When a route must answer
differently _after_ some request (e.g. `GET /feeds` is empty until a
`PUT /feeds` subscribes), use `onGet` / `onPost` / `onPut` instead: they take a
`respond` closure evaluated per request, so it can read a variable the test
captured. Have the `PUT` handler mutate that variable and the `GET` handler read
it — no re-registering routes mid-test.

```dart
var subscribed = false;
server
  ..onGet('/feeds', respond: (_) => (
        200,
        api.GetFeeds200Response(feeds: subscribed ? [nasa] : []).toJson(),
      ))
  ..onPut('/feeds', bodyMatcher: {'url': nasa.url}, respond: (_) {
    subscribed = true;
    return (200, nasa.toJson());
  });
```

`respond` returns a `(status, body)` record and receives the decoded request
body as its argument (ignored with `_` above, since a `GET` carries none). The
same argument lets a test capture what the app sent, which is how it asserts a
field is _absent_ — something `bodyMatcher` cannot express, as it matches a
subset:

```dart
Object? sentBody;
server.onPost('/signup', respond: (body) {
  sentBody = body;
  return (202, api.SignUpTicket(ticket: 'ticket').toJson());
});
// ... after the tap:
expect(sentBody, {'email': email, 'password': password});
```

## Naming tests

Name each test after the **use case** it covers, not the click sequence — what
the user accomplishes, not what the test taps through:

```dart
// Good
patrolWidgetTest('Subscribe to a known web feed', (t) async { ... });
// Avoid
patrolWidgetTest('Tap feed and scroll to entry', (t) async { ... });
```

Keep names to one short phrase, no punctuation.

## Test key conventions

Keys are defined in `lib/debug_keys.dart` and shared between app code and tests.

- **Static keys** — a fixed `const Key` for nav controls, buttons, text fields,
  destination screens (`Scaffold`s), and feedback widgets (snackbars).
- **Parameterized keys** — for dynamically-generated list items, keyed
  `<identifier>:<display text>`, e.g. `AppDebugKey.feedRow('The Dart Blog')` →
  `Key('feed:The Dart Blog')`. The identifier keeps the widget unambiguous even
  when the same text appears on multiple screens. Assign it at the list-builder
  call site.

## Writing assertions

### Actions auto-settle

`patrolWidgetTest` actions (`tap`, `enterText`, scroll) call `pumpAndSettle`
themselves, so you usually don't need to wait after them — assert directly:

```dart
await t(AppDebugKey.feedsNavDestination).tap();
expect(t(AppDebugKey.feedsScreen), findsOneWidget);
```

For content that lands a frame later (async fetches after navigation), use
`await t(key).waitUntilVisible()` instead of `expect`.

### Tap list items by parameterized key

The parameterized key ties the locator to a specific widget _and_ title, so the
tap doubles as a presence assertion — it fails if no such widget exists:

```dart
// Good — unambiguous, asserts the row exists
await t(AppDebugKey.feedRow('Anthropic Engineering Blog')).tap();
// Avoid — a text finder matches any widget on any screen with that string
await t('Anthropic Engineering Blog').tap();
```
