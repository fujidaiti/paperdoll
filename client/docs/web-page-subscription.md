# Subscribing to a web page that has no feed

## Context

Today the client subscribes to a feed in one tap: the user types a URL on the
Feed Search screen, `GET /feeds/search` returns a `FeedCandidate`, and
`PUT /feeds` with `{ url }` finishes the job. That works only when the URL is an
RSS/Atom feed, or an HTML page that links to one.

The server is being extended to handle a plain HTML page with no feed. It cannot
decide by itself which elements of the page are posts, so it stops deciding and
starts enumerating: it returns every list of post-like elements it found, and
the user resolves the ambiguity in the app. The server design and the
measurements behind it are in the feed detector document.

- Design: file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/IDEA2.md
- API: file:///Users/fujidaiti/Dev/paperdoll/api/paths/feeds.yaml
- Schemas: file:///Users/fujidaiti/Dev/paperdoll/api/components/schemas.yaml

The API contract is already merged and the Dart models are already generated, so
the client work does not depend on the server implementation. Every screen below
can be built and tested against `StubServer`.

## Goal

A user can subscribe to a web page that publishes no feed.

Acceptance criteria:

1. Searching a URL that has a feed behaves exactly as it does today. One tap on
   Subscribe, no extra screen. No regression in `test/features/feed_test.dart`.
2. Searching a URL with no feed, whose response carries `post_groups`, opens a
   subscription screen instead of subscribing directly.
3. On that screen the user can tick one or more groups, and for each ticked
   group choose which element is the title, the description, the timestamp and
   the image. The link needs no choice when the group offers exactly one link
   candidate, and is chosen the same way when it offers several.
4. A group whose `links` list is empty cannot be ticked, because a post with no
   URL cannot be stored.
5. Choices survive navigation between the two screens, and ticking or unticking
   a group does not clear them.
6. Subscribe sends `PUT /feeds` with the URL and one `selectors` entry per
   ticked group, and the app returns to the feed list with the new feed in it.
7. A rejected subscription (400) keeps the user on the screen with their choices
   intact and shows the server's message.

## Approach

The client holds all of the selection state in one Riverpod notifier for the
duration of the flow, and converts it into the request body only when the user
taps Subscribe. The screens read that notifier and write to it; they hold no
state of their own.

Three ideas carry the whole design.

- **A choice is a selector, not a value.** Every row the user sees is one
  `AttributeCandidate`, that is, one relative selector plus the values it
  produces in the sampled items. The user picks a row, and the client stores its
  `selector` string. The client never parses a selector, never builds one, and
  never compares one against the page.
- **A group is answered as a whole.** The selection for a group is a record of
  at most five selector strings (`link`, `title`, `description`, `image`,
  `timestamp`). There is no per-post state anywhere.
- **The preview is built by index.** `AttributeCandidate.values` holds exactly
  `PostGroup.sampled` entries, in item order. The preview of item _i_ is built
  by reading `values[i]` of each selected row, so the same data drives the
  preview and the rows.

## The API the client codes against

`GET /feeds/search` answers with `SearchFeeds200Response`, whose `feeds` are
`FeedCandidate`s. The new field is `post_groups`:

- empty or absent → the URL has a feed. Subscribe with `{ url }` as today.
- non-empty → the URL is a plain page. Open the subscription flow.

```
FeedCandidate.postGroups : List<PostGroup>

PostGroup
  id        int              unique inside this response, not stored server side
  selector  String           send back as PostSelectors.root
  count     int              how many items the group holds in the page
  sampled   int              how many items the values arrays describe
  links     List<AttributeCandidate>
  texts     List<AttributeCandidate>   title, description and timestamp come from here
  images    List<AttributeCandidate>

AttributeCandidate
  selector  String           relative to the item, ":scope" means the item itself
  matched   int              how many of the group's count items it reaches
  values    List<AttributeValue>   exactly `sampled` entries, in item order

AttributeValue
  value     String?          text, resolved href, or resolved image URL. Null when
                             the selector reaches nothing in that item
  alt       String?          images only
```

`PUT /feeds` takes the same `url` plus the new `selectors`:

```
SubscribeToFeedRequest
  url        String
  selectors  List<PostSelectors>      omit for a real feed

PostSelectors
  root        String  required, from PostGroup.selector
  link        String  required
  title       String?
  description String?
  image       String?
  timestamp   String?
```

The server fetches the page again and validates the selectors before saving. It
answers 400 when a `root` matches nothing or a `link` matches nothing in any
item. On success it extracts the posts immediately, so the feed's timeline is
populated before the first poll.

Two properties of the response shape matter for the UI:

- **`post_groups` is not capped.** The server returns every group it found,
  which reaches 135 on one of the measured pages. The list must be rendered
  lazily (`ListView.builder`) and must never be loaded into a `Column`.
- **`matched` may be lower than `count`.** On six of the measured pages the best
  selector does not reach every item of its group. The row has to show that, for
  example "reaches 89 of 103 posts", because it is the user's only warning that
  the choice will leave some posts incomplete.

## Screens

### Screen 1: the group picker

Replaces the direct subscribe when the candidate carries `post_groups`. Reached
by tapping a search result, with the candidate passed through the route as an
`extra`, so the screen needs no fetch of its own.

Each row is a group card:

- a checkbox,
- the item count ("24 posts on this page"),
- a preview of the first two items, built from the current selection for that
  group. Before the user has chosen anything, the preview shows the values of
  the first candidate of each kind, marked as a guess; after they have chosen,
  it shows only the chosen rows.
- tapping the card, not the checkbox, opens screen 2 for that group.

A group with an empty `links` list renders its checkbox disabled, with a line
saying the group has no link to a post.

The bottom bar holds the Subscribe button, enabled when at least one group is
ticked.

### Screen 2: the attribute picker

One screen per group, driven by a step index: title, then description, then
timestamp, then image. Each step shows the same body, only the question and the
source list change.

- the question, for example "Which row holds the title?",
- one selectable row per `AttributeCandidate`, showing its first two values, and
  the coverage line when `matched < count`,
- a "None of them" action, which stores null for that attribute and moves on,
- Back and Next.

The link step is shown first and only when `links.length > 1`. With exactly one
link candidate the client selects it and skips the step.

Rows already chosen in an earlier step are removed from the later steps, so the
same element cannot be both the title and the description. Removing them is a
filter on the list at build time, not a change to the stored candidates.

When the last step is answered the screen pops back to screen 1, where the
preview of that group now reflects the choices.

## Work breakdown

The tasks are ordered so that tasks 2 to 5 can be written in parallel once task
1 is merged. None of them needs a running server.

**1. Domain and repository (blocks the rest, keep it small).**

- `lib/features/feed/domain/post_group.dart`: freezed `PostGroup`,
  `AttributeCandidate`, `AttributeValue`, mirroring the generated models.
- `lib/features/feed/domain/post_selection.dart`: freezed `PostSelection` with
  the five nullable selector fields plus the group id, and a method producing
  `api.PostSelectors`.
- `FeedCandidate` gains `List<PostGroup> postGroups`.
- `FeedRepository.subscribe` gains `{List<PostSelectors> selectors = const []}`;
  `FeedRepositoryImpl` maps the new fields in both directions.
- Extend `test/src/fixture.dart` with a candidate that carries post groups, for
  example a two-group page where one group has two text candidates and one image
  candidate, and a second group with no links. Every later task stubs against
  this fixture.

**2. Selection state.**

- `SubscriptionDraft` notifier in
  `lib/features/feed/presentation/providers/feed_providers.dart`, built from a
  `FeedCandidate`, holding the ticked group ids and a `PostSelection` per group.
- Operations: toggle a group, set an attribute of a group, clear an attribute,
  and build the request body.
- Ticking and unticking must not touch the stored selections, per acceptance
  criterion 5.

**3. Screen 1 and routing.**

- New route under the feeds branch, name `feedSubscription`, path `subscribe`,
  taking the candidate as `extra`.
- `FeedSearchScreen` branches on `candidate.postGroups.isEmpty`: subscribe
  directly as today, or push the new route.
- Group card widget, lazy list, Subscribe button, error handling for 400.
- Debug keys: `feedSubscriptionScreen`, `feedSubscriptionSubscribeButton`, and
  parameterized `postGroupCard(int id)` and `postGroupCheckbox(int id)`.

**4. Screen 2.**

- Route `feedSubscriptionAttributes`, path `groups/:groupId`, under the route
  from task 3.
- The step list, the row widget with the coverage line, "None of them", Back and
  Next.
- Debug keys: `attributePickerScreen`, `attributeRow(String selector)`,
  `attributeNoneButton`, `attributeNextButton`.

**5. Widget tests, in `test/features/feed_test.dart`.**

- Subscribe to a page with no feed, choosing a title and an image. Assert the
  `PUT /feeds` body with an exact `bodyMatcher`, which is the real contract
  check here.
- A group with no link candidate cannot be ticked.
- Choices survive going back to screen 1 and returning to screen 2.
- A 400 keeps the user on the screen and shows the message.
- The existing "Subscribe to a known web feed" test must keep passing unchanged,
  which is how the one-tap path is protected.

## Decisions taken, so nobody has to ask

- **The group id is not sent to the server.** It is assigned per response.
  `PostSelectors.root` carries the selector instead.
- **The draft is not persisted.** Leaving the flow loses the choices. Users
  answer this screen in one sitting, and reopening it needs a fresh
  `GET /feeds/search` anyway, since the selectors are only valid against the
  page the server just fetched.
- **No preview fetch.** The client never loads the target page and never runs a
  selector. Everything shown comes from the search response.
- **Image previews use the `value` of an `images` row directly.** It is an
  absolute URL, resolved by the server.
- **Sampled items are few.** Show at most two in a preview, and expect `sampled`
  to be small. Do not write code that assumes it equals `count`.

## Open points

- The wording of the coverage line is not fixed. "Reaches 89 of 103 posts" is
  the placeholder used above.
- Whether a group with `matched < count` on the link row should warn before
  subscribing is undecided. The server accepts it.
