# Future work on unstructured feeds

Work that the design in `IDEA2.md` calls for and that `PLAN.md` deliberately
leaves out. Each entry says what is missing, why it was dropped, and what it
would take.

- IDEA2.md: file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/IDEA2.md
- PLAN.md: file:///Users/fujidaiti/Dev/paperdoll/server/feature/feed/PLAN.md

## Telling the user that a page feed broke

A site rebuild changes the DOM, the saved keys are no longer written by the
enumerator, and the feed stops producing posts. `IDEA2.md` asks for the user to
be told and sent back through the selection screen for the same URL. See "When
the page changes" in that document.

It is left out because none of the pieces exists yet:

- The polling job has nowhere to record the result of a poll, so a feed that
  returns no post is indistinguishable from a feed that published nothing new.
  This needs a column on `feeds`, or a small table, holding the number of posts
  the last poll produced and when it ran.
- The API carries no field for it, so the client cannot show it. Adding one
  touches `Feed`, which every feed screen reads.
- The client has no screen for it. Sending the user back into the selection flow
  for an existing feed is a new route, and the subscription flow currently only
  creates feeds.

Two cases have to be separated when this is built, and `IDEA2.md` already states
the rule: a `root` that names no group, or a group where no post carries the
link key, is a broken feed and is worth interrupting the user for. An optional
attribute key that stopped being written is not, because the feed keeps working
with that field empty.

Until it exists, a broken page feed is silent: the poll inserts nothing and the
timeline stops growing.
