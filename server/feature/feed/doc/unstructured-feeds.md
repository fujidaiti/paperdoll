# Unstructured feeds

Users can subscribe to a web page that publishes no feed, such as a blog index
without RSS, by following these steps:

1. **Subscribe.** The user enters a URL. If the URL is a feed, or the page links
   to one, the user subscribes to that feed as usual. Otherwise, the page itself
   becomes an _unstructured feed_.
2. **Teach.** The user opens a _selector editor_ on the page and _marks_ a few
   posts. From the marks, the server generates _post selectors_ that also find
   the other posts of the same shape. The user checks what they find, and
   _commits_ them.
3. **Extract.** The saved selectors extract the posts from the page, and each
   post becomes an entry of the feed, as an item of an RSS feed does.

## Terms

### Unstructured feed

A feed whose source is a web page instead of a feed document. Its URL and its
site URL are both the URL of the page, and this is how the system tells it apart
from a web feed. Like every feed, it is shared by all users, and so are its
extraction rules.

### Feed discovery

The steps that turn a URL into a feed. The URL is tried in this order:

1. as a feed document;
2. as a page that declares its feed in its head;
3. as a page that links to its feed from its body, at a usual feed path of the
   same site, such as `/feed`;
4. as the page itself, which becomes an unstructured feed.

### Simplified page

The page reduced to its visible content. Scripts, styles, hidden elements and
text formatting are removed. When the page marks its main content, the rest of
the page, such as navigation and footers, is removed too. What remains becomes a
tree of two kinds of nodes:

- A _value_ is a single piece of content: a text, a link, an image or a time.
- A _container_ groups other nodes and keeps the tag and the classes of the
  original element.

Selectors run on the simplified page, and the editor shows it to the user. Each
node has a _node ID_, its position in the page in document order. Marks refer to
nodes by their IDs, so an ID is meaningful only within one simplified page.

### Post, box and fields

A _post_ is one item that the page lists, such as one card of a blog index. Its
_box_ is the node that holds the whole post. Its _fields_ are the nodes inside
the box that hold its values:

- _title_ and _link_, which every post has;
- _body_, _thumbnail_ and _pub date_, which a post may have.

### Post selector

A rule that finds the posts of one kind on a page. It has a path to the boxes,
and a path from a box to the node of each field. A path is a list of steps that
select nodes by their tags, classes and positions, much like a CSS selector. A
page can need several selectors, such as one for a featured post and one for the
list below it. Users never see selectors: they see only the posts that the
selectors find.

### Exclusions

Some nodes have the shape of a post but are not posts, and some parts of a page
should be ignored. The user removes them in two ways:

- An _excluded link_ removes every post that links to it, such as a "Join our
  newsletter" card in a list of articles.
- An _excluded node_ removes a part of the page as a whole, such as a menu or a
  sidebar.

Exclusions apply to all selectors of a feed.

### Extraction rules

The post selectors and the exclusions of a feed together. They are what is saved
on the feed. Running them on a simplified page is called _extraction_. Posts
with the same link are merged into one, since some pages show one post in
several places.

### Marks

What the user tells the editor: the _marked posts_, each with its box and its
field nodes, the excluded links, and the excluded nodes. Marks are the input and
the extraction rules are the output. The server generates the rules from the
marks, and the rules do not depend on the order in which the user made them.

Marks are not saved with the feed. When an editor is opened on a feed that
already has rules, the server derives a small set of marks that reproduce what
the saved rules find, so the user can continue from there.

### Found posts

The posts that the current rules find on the page, including the ones that the
user did not mark. The editor shows each found post as a card of its values, so
the user only needs to mark the posts that are missing or found with wrong
values. The editor also shows the _excluded posts_, which an excluded link
removes, and a _preview_ of the posts as the feed will store them.

### Selector editor

A temporary editing session on the server, in which one user builds the
extraction rules of one unstructured feed. When it opens, it fetches the page
and keeps a snapshot of the simplified page, so that the node IDs stay valid
while the user edits. It holds the marks and the rules generated from them.

- **Update.** On each change, the app sends all marks, and the server generates
  the rules again and returns what they find. The app never runs selectors
  itself.
- **Commit.** Saves the rules to the feed, replacing the previous rules, stores
  the posts that they find as entries of the feed, and ends the editor.
- **Close.** Ends the editor without saving.

An editor expires after a while, so that rules built on an old snapshot of the
page are not saved.
