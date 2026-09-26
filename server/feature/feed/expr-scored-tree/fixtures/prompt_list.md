You are listing the posts of a saved web page, so that the value recording can
be split into chunks. Work only from the file given below. Do not read any other
file in the repository, and do not look for an existing fixture.

INPUT FILE (read it fully): {INPUT}

The page URL is {PAGEURL} . The file is that page with scripts, styles and
irrelevant attributes removed. Every href and src in it is already an absolute
URL.

YOUR TASK Find the page's main list of posts and write the link of every post,
in the order the page lists them, as JSON to: {OUTPUT}

Shape:

{ "url": "{PAGEURL}", "links": ["<absolute URL>", "<absolute URL>", ...] }

RULES

1. Only the entries of the page's own main list. A navigation menu, a sidebar, a
   "related" block or a footer may hold blocks that look like a post. Those are
   not posts.
2. A page may show its list in several sections, for example a featured block
   above the main list, or a changelog list beside it. Those sections are part
   of the list. Include them.
3. One link per post, in document order, each appearing exactly once. If a post
   appears twice on the page, list its link once, at the position of its first
   appearance.
4. Record the link exactly as the file writes it, including any query string.
5. Do not record any other URL: no author pages, no category pages, no
   pagination links, no external links.

{SCOPE}

When you are done, report how many links you listed, which sections of the page
they came from, and which blocks you left out and why.
