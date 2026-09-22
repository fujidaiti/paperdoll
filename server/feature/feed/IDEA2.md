# Convert web pages into RSS feeds

An overview of the workflow would looks like this:

1. users make a request to GET /feeds/search with a URL of the page they want to
   observe.
2. the server samples chunks of elements that look like feed posts from the
   page, and return them to the users as a part of feed candidate.
3. users select which chunks should be handled as posts, which image is the
   thumbnail, which one is the title, etc...
4. users subscribe to the page via PUT /feeds with that URL and post selectors
   that is constructed from choices they made in the previous step.
5. In a feed polling, the server fetches the page, uses the selector to extract
   posts. The subsequent process is the same as RSS/Atom feeds.

## Step 1

Users use the same flow as today's feed subscription: tap the + button in the
feed list page, and enter the URL.

## Step 2

Given the url, the server fetch the resource. If it's a RSS/Atom feed, it falls
into the existing flow. If it's an HTML, the server searches for a link to the
feed in the document header:

```
    <link
      rel="alternate"
      type="application/rss+xml"
      title="RSS"
      href="feed://developer.apple.com/news/rss/news.rss"
    />
```

If it finds it, this case also falls into the existing flow.

Otherwise, the server extracts chunks of post-like DOM nodes with candidate
attributes for the link, title, description, timestamp, and the thumbnail. The
results are returned as part of the response of GET /feeds/search, which would
look something like this:

```json
{
  "feeds": [
    {
      "url": "https://claude.com/blog",
      "site_url": "https://claude.com/blog",
      "title": "Blog | Claude by Anthropic",
      "candidates": [ // new field. an empty list if the RSS/Atom xml is found.
        {
          "id": 0, // an integer that is unique among the same site
          "selector": "" // selector to the parent group of post nodes
          "posts": [
            {
              "texts": [ // text elements in the chunk
                {
                  "selector": "" // selector to this element, relative to the group
                  "value": "Claude Cowork and chat are now one Claude",
                },
                {
                  "selector": "...",
                  "value": "Read more",
                },
                {
                  "selector": "...",
                  "value": "September 16, 2026",
                }
              ],
              "images": [ // image elements in the chunk
                {
                  "selector": "...",
                  "src": "https://...",
                  "alt": "...", // if any
                }
              ],
              "links": [ // link elements in the chunk
                {
                  "selector": "...",
                  "value": "https://", // the href value
                }
              ]
            }
          ]
        }
      ],
    }
  ]
}
```

The candidates is a list of groups that contain one or more post-like chunks
shareing the same DOM structure. Each group has candidates for the post's
attributes, for example, the texts it a list of possible texts that may be the
title, description, or the timestamp. Once the ambiguity is resolved, the server
can query posts by constructed selectors.

## Step 3

On the client side, users need to resolve the ambiguity before making a
subscription. The subscribe button navigates the users to a new screen, where
candidate chunks and attributes are shown and let users select correct ones
step-by-step.

In the first step, users select a listed group, which displays one or two
previews of the candidate posts, including all possible images and texts:

```
Group 1         [ ]

[img1], [igm2]
Claude Cowork and chat are now one Claude
Read more
September 16, 2026
-------------------
[img1], [igm2]
Claude in Chrome is generally available
Read more
Aug 10, 2026

--------------------
Group 2          [x]

...

[Subscribe]
```

Users can mark correct groups (x). Tapping a group navigate the user to the
second step, in which they are asked to select a correct title:

```
Which group contains the titles?

[x]
Claude Cowork and chat are now one Claude
Claude in Chrome is generally available
---
[ ]
Read more
Read more
---
[ ]
September 16, 2026
Aug 10, 2026
---

[None of them]
[Next]
```

They can claim there's no correct candidate in the list (except for the links).

The same flow follows. Candidate groups for an attribute are displayed, users
select one, go to next step, and so on. Selected candidates are filtered out in
subsequent steps so that users don't select the same element for multiple
attributes (claim the same text for both the title and description).

When everything finishes, the app brings the users back to the first step page,
where the user's selections are reflected on the group preview card (display
only selected ones:

```
Group 1         [ ]

[img2]
Claude Cowork and chat are now one Claude
September 16, 2026
-------------------
[img2]
Claude in Chrome is generally available
Aug 10, 2026

--------------------
Group 2          [x]
...

[Subscribe]
```

Toggling a group checkbox doesn't clear the candidate selections; which groups
to include and which element to include are two seprate things. Tapping a group
navigates the user to the second step page again, in which previous selections
are remembered.

## Step 4

the users then hit the subscribe button to subscribe to the page via PUT /feeds,
with group selectors constructed from the candidate selections:

```json
{
  "url": "https://claude.com/blog",
  "selectors": [
    // new field. omit when posting an rss feed link
    {
      "root": "...", // selector to the group's root
      "title": "...", // relative selector to the post title, if any
      "description": "...", // relative selector to the post description, if any
      "image": "...", // image selector, if any
      "link": "..." // post link selector (required)
    }
  ]
}
```

The server fetches the url and determine if it's an RSS/Atom feed in the same
way as step 2, so the two requests should end up in the same result. Save the
feed and selectors in the DB, then wait for the next polling.

## Step 5

When polling the page, fetch the HTML and extract posts using the saved
selectors. The posts are then processed just like usual items from RSS/Atom
feed: being saved as entries, and converted to stories if needed.

---

## Notes

- The selectors are for sanitized HTML, not for raw HTML.
- A group with only one post is valid. Think of a hero post that is displayed at
  the top of the blog site with a large thumbnail, having no sibling.

## Open questions

### How to parse unstructured timestamps?

TBD. The publish date is optional, but having it is better than missing if
visual timestamp exists.
