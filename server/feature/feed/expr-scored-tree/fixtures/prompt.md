You are recording a test fixture by reading a saved web page. Work only from the
file given below. Do not read any other file in the repository, and do not look
for an existing fixture.

INPUT FILE (read it fully): {INPUT}

The page URL is {PAGEURL} . The file is that page with scripts, styles and
irrelevant attributes removed. Every href and src in it is already an absolute
URL. Nothing else was changed.

YOUR TASK The page shows a list of posts. Find that list and record one entry
per post, in the order the page lists them.

Write the result as JSON to: {OUTPUT}

Shape:

{ "url": "{PAGEURL}", "posts": [ { "link": "<absolute URL of the post itself>",
"texts": [{"kind": "title", "value": "...", "required": true}], "images":
[{"kind": "thumbnail", "value": "<absolute URL>", "required": true}] } ] }

RULES

1. Which blocks count as a post. Only the entries of the page's own main list. A
   navigation menu, a sidebar or a footer may hold blocks that look like a post,
   with a title, a description and an image. Those are not posts. If you are
   unsure whether a block is a post, leave it out and say so in your final
   report.

2. Copy values exactly. Copy each value character for character as written in
   the file. Never paraphrase, never fix punctuation, never translate, never add
   or remove words. Collapse runs of whitespace to single spaces; change nothing
   else. In particular, keep every typographic character as the page writes it:
   a curly quote stays curly, an apostrophe written as ’ does not become ', and
   an en or em dash does not become a hyphen.

3. Never join and never split. Each value is the text of one element, as
   written. If an element's text is interrupted by a nested element, record each
   uninterrupted run as its own value. In
   `<p>Read our <a>new guide</a> today</p>` you record three values: "Read our",
   "new guide", "today". Do not join them into one sentence. If a title is
   written as "Some title, Smith et al., FAT*'20", record it whole, exactly like
   that, and do not cut it down to the title alone.

4. A value is always a whole text run, never a part of one. If a run carries a
   connective word or a punctuation mark together with the value, for example
   `<p>By Laurie Voss |</p>`, record the whole run exactly as written,
   `By Laurie Voss |`, and give it the kind of the value it carries. Never trim
   a run down to the part you consider the value. Rule 10 removes a text only
   when that whole text is a run of its own.

5. The kinds you may use, and nothing else:
   - texts: title, description, pub-date, update-date, author, category,
     reading-time, body
   - images: thumbnail, avatar `kind` is optional for a text. If a text clearly
     belongs to the post but none of the kinds fits, record it with no `kind`
     field. Every image must have a kind; an image that is neither a thumbnail
     nor an author avatar is not recorded at all.

6. Record every value of the post, not only the obvious ones. Work through the
   card element by element. Every text inside the card is either recorded or
   left out by rule 10, and there is no third option. Authors, categories, tags,
   section names, difficulty levels and reading times are values of the post and
   must be recorded with their kind. A short label naming what the entry is,
   such as "Video", "article", "Podcast" or the name of the section it belongs
   to, is a category and must be recorded. The name of the source or
   publication, a video duration and a price have no matching kind, so record
   each as a text with no `kind` field. Rule 10 is a closed list of what to
   leave out, and a value that is not on it is recorded. Recording only the
   title, the date and the image is wrong.

7. `required` is true only for kind title, description, pub-date and thumbnail.
   Every other value, and every value with no kind, has required false.

8. Dates. If a post shows two dates, the earlier one is pub-date and the later
   one is update-date. If the date sits in a `datetime` attribute, record two
   pub-date values: one holding the attribute's value and one holding the
   visible text. If there is no `datetime` attribute, record only the visible
   text. A post may therefore have two pub-date values, and that is correct.

9. A post may have several thumbnail values, for example the same picture at two
   sizes. Record each one. A post has at most one title.

10. Do NOT record any of these, at all:

- any URL other than the post's own link. No author links, no category links, no
  comment links, no related links. The `link` field is the only URL of that
  kind, and there is no `links` list in the schema.
- any image that is not the post's thumbnail or its author's avatar. A site
  logo, a section icon, a decorative graphic, a badge and a placeholder are all
  left out even when they sit inside the card.
- interface controls, connective words and labels that are not a property of the
  post: "by", "on", "in", "Permalink": "Read more", "Learn more", "Share",
  "Subscribe", pagination numbers, comment counts, like counts.
- separators such as "·", "|" or "—" on their own.

11. If the page shows no date, no description or no image for a post, leave
    those values out. Do not invent them.

12. Each `link` must appear exactly once in your output.

{SCOPE}

BEFORE YOU WRITE THE FILE Pick one card in the middle of the list. Go through
every text and image inside it one by one, and confirm that each is either in
your output or named by rule 10. If you find one that is neither, you are
under-recording: go back over every card and add what you missed.

When you are done, report: how many posts you recorded, which blocks you
deliberately excluded and why, and any value you were unsure about.
