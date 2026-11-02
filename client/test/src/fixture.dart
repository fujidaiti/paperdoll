import 'package:openapi/api.dart' as api;

/// A shared universe of fixtures for the widget tests.
final fixture = _Fixture();

class _Fixture {
  final feeds = _feeds;
  final entries = _entries;
  final webClips = _webClips;
  final feedCandidates = _feedCandidates;
  final stories = _stories;
  final readingList = _readingList;
  final pages = _pages;
}

final _feeds = _Feeds();

class _Feeds {
  final bbcNews = api.Feed(
    id: 1,
    url: 'http://feeds.bbci.co.uk/news/rss.xml',
    siteUrl: 'https://www.bbc.co.uk/news',
    iconUrl: 'https://news.bbcimg.co.uk/nol/shared/img/bbc_news_120x60.gif',
    title: 'BBC News',
    description: 'BBC News - News Front Page',
    subscribed: true,
    hasSelectors: false,
  );

  final nasa = api.Feed(
    id: 2,
    url: 'http://www.nasa.gov/news-release/feed/',
    siteUrl: 'https://www.nasa.gov',
    title: 'NASA',
    description:
        'Official National Aeronautics and Space Administration Website',
    subscribed: true,
    hasSelectors: false,
  );

  final stackOverflow = api.Feed(
    id: 3,
    url: 'http://stackoverflow.com/feeds',
    siteUrl: 'https://stackoverflow.com/questions',
    title: 'Recent Questions - Stack Overflow',
    description: 'most recent 30 from stackoverflow.com',
    subscribed: true,
    hasSelectors: false,
  );

  final wikipedia = api.Feed(
    id: 4,
    url: 'http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss',
    siteUrl: 'https://en.wikipedia.org/wiki/Special:RecentChanges',
    title: 'Wikipedia  - Recent changes [en]',
    description: 'Track the most recent changes to the wiki in this feed.',
    subscribed: true,
    hasSelectors: false,
  );

  // A web page without an RSS feed, whose URL and site URL are the page's.
  final exampleBlog = api.Feed(
    id: 5,
    url: 'https://blog.example.com/',
    siteUrl: 'https://blog.example.com/',
    title: 'Example Blog',
    subscribed: true,
    hasSelectors: false,
  );
}

final _entries = _Entries();

class _Entries {
  final nuclearDeal = api.FeedEntry(
    id: 11,
    feedId: _feeds.bbcNews.id,
    url: 'https://www.bbc.co.uk/news/articles/cj03r59z73po',
    title: 'US signs landmark nuclear deal with Saudi Arabia',
    content:
        '<article><p>The US Department of Energy says the "peaceful" '
        'co-operation agreement will give US firms great access to the Saudi '
        'nuclear energy programme.</p></article>',
    snapshotAt: DateTime.utc(2026, 7, 23),
  );

  final houthiStrikes = api.FeedEntry(
    id: 12,
    feedId: _feeds.bbcNews.id,
    url: 'https://www.bbc.co.uk/news/articles/cpw9xzx9r4ko',
    title:
        'Houthis claim attack on oil tankers as US launches more strikes on '
        'Iran',
    snapshotAt: DateTime.utc(2026, 7, 23),
  );

  final moonshotAi = api.FeedEntry(
    id: 13,
    feedId: _feeds.bbcNews.id,
    url: 'https://www.bbc.co.uk/news/articles/c5ye2gyz0x4o',
    title: "China's Moonshot AI stole from Anthropic, Trump tech adviser says",
    snapshotAt: DateTime.utc(2026, 7, 23),
  );
}

final _webClips = _WebClips();

class _WebClips {
  final buildingEffectiveAgents = api.GetWebClip200Response(
    id: 7,
    url: 'https://www.anthropic.com/engineering/building-effective-agents',
    title: 'Building effective agents',
    content:
        '<article><p>The most successful implementations use simple, '
        'composable patterns rather than complex frameworks.</p></article>',
  );

  // A clip whose content hasn't been fetched.
  final claudeCodeBestPractices = api.GetWebClip200Response(
    id: 8,
    url: 'https://www.anthropic.com/engineering/claude-code-best-practices',
    title: 'Claude Code: Best practices for agentic coding',
  );
}

final _feedCandidates = _FeedCandidates();

class _FeedCandidates {
  // What /feeds/search returns before NASA is subscribed to.
  final nasa = api.FeedCandidate(
    url: _feeds.nasa.url,
    siteUrl: _feeds.nasa.siteUrl,
    iconUrl: 'https://www.google.com/s2/favicons?domain=nasa.gov&sz=64',
    title: _feeds.nasa.title,
    description: _feeds.nasa.description,
  );

  final exampleBlog = api.FeedCandidate(
    url: _feeds.exampleBlog.url,
    siteUrl: _feeds.exampleBlog.siteUrl,
    title: _feeds.exampleBlog.title,
  );
}

final _stories = _Stories();

class _Stories {
  final nuclearDeal = api.Story(
    id: 1,
    resourceId: _entries.nuclearDeal.id,
    kind: api.StoryKindEnum.feedEntry,
    title: _entries.nuclearDeal.title,
    source_: _feeds.bbcNews.title,
  );

  // A story whose entry has no content.
  final houthiStrikes = api.Story(
    id: 2,
    resourceId: _entries.houthiStrikes.id,
    kind: api.StoryKindEnum.feedEntry,
    title: _entries.houthiStrikes.title,
    source_: _feeds.bbcNews.title,
  );
}

final _readingList = _ReadingList();

class _ReadingList {
  final buildingEffectiveAgents = api.ReadingListItem(
    id: 1,
    resourceId: _webClips.buildingEffectiveAgents.id,
    kind: api.ReadingListItemKindEnum.webClip,
    title: _webClips.buildingEffectiveAgents.title!,
    savedAt: DateTime.utc(2026, 7, 1),
  );

  final claudeCodeBestPractices = api.ReadingListItem(
    id: 2,
    resourceId: _webClips.claudeCodeBestPractices.id,
    kind: api.ReadingListItemKindEnum.webClip,
    title: _webClips.claudeCodeBestPractices.title!,
    savedAt: DateTime.utc(2026, 7, 2),
  );

  final nuclearDeal = api.ReadingListItem(
    id: 5,
    resourceId: _entries.nuclearDeal.id,
    kind: api.ReadingListItemKindEnum.feedEntry,
    title: _entries.nuclearDeal.title,
    savedAt: DateTime.utc(2026, 7, 1),
  );
}

final _pages = _Pages();

/// The simplified pages and the selector editors of the selector editor
/// endpoints (`/feeds/{id}/selector-editors`). The editors are written as JSON
/// as the server sends them, since the generated models write null for each
/// field that a post does not have.
class _Pages {
  /// A simplified page with node IDs (see withNodeIDs in
  /// server/feature/feed/selector_editor.go), which has no whitespace between
  /// its nodes.
  final exampleBlog = [
    '<!DOCTYPE html><html><head><title>Example Blog</title>',
    '<link rel="canonical" href="https://blog.example.com/"/></head>',
    '<body data-id="0">',
    '<article class="post" data-id="1">',
    '<h2 href="https://blog.example.com/first" data-id="2">First post</h2>',
    '<p data-id="3">The first post.</p>',
    '</article>',
    '<article class="post" data-id="4">',
    '<h2 href="https://blog.example.com/second" data-id="5">Second post</h2>',
    '<p data-id="6">The second post.</p>',
    '</article>',
    '</body></html>',
  ].join();

  /// [exampleBlog] with two newsletter cards like the posts,
  /// whose links differ only in the query.
  final exampleBlogWithNewsletters = [
    '<!DOCTYPE html><html><head><title>Example Blog</title>',
    '<link rel="canonical" href="https://blog.example.com/"/></head>',
    '<body data-id="0">',
    '<article class="post" data-id="1">',
    '<h2 href="https://blog.example.com/first" data-id="2">First post</h2>',
    '<p data-id="3">Read First post.</p>',
    '</article>',
    '<article class="post" data-id="4">',
    '<h2 href="https://blog.example.com/newsletters?weekly" data-id="5">',
    'Weekly digest</h2>',
    '<p data-id="6">Read Weekly digest.</p>',
    '</article>',
    '<article class="post" data-id="7">',
    '<h2 href="https://blog.example.com/newsletters?monthly" data-id="8">',
    'Monthly digest</h2>',
    '<p data-id="9">Read Monthly digest.</p>',
    '</article>',
    '</body></html>',
  ].join();

  /// A new editor of [exampleBlog] for a feed without saved selectors.
  late final exampleBlogEditor = <String, Object?>{
    'id': 42,
    'expires_at': '2026-07-03T13:30:00Z',
    'page': exampleBlog,
    'labeled_posts': <Object?>[],
    'excluded_links': <Object?>[],
    'matched_posts': <Object?>[],
    'excluded_posts': <Object?>[],
    'preview': <Object?>[],
  };

  /// What the selectors find in [exampleBlog] after the heading of the first
  /// post is marked as a post with itself as the title and the link: the
  /// headings of both posts.
  final exampleBlogHeadingsFound = <String, Object?>{
    'matched_posts': [
      {
        'box': 2,
        'title': 2,
        'link': 2,
        'values': {
          'title': 'First post',
          'link': 'https://blog.example.com/first',
        },
      },
      {
        'box': 5,
        'title': 5,
        'link': 5,
        'values': {
          'title': 'Second post',
          'link': 'https://blog.example.com/second',
        },
      },
    ],
    'excluded_posts': <Object?>[],
    'preview': [
      {'title': 'First post', 'link': 'https://blog.example.com/first'},
      {'title': 'Second post', 'link': 'https://blog.example.com/second'},
    ],
  };

  /// A new editor of [exampleBlog] for a feed whose saved selectors find both
  /// posts, which start as labeled.
  late final exampleBlogSavedEditor = <String, Object?>{
    'id': 42,
    'expires_at': '2026-07-03T13:30:00Z',
    'page': exampleBlog,
    'labeled_posts': [
      {'box': 1, 'title': 2, 'link': 2},
      {'box': 4, 'title': 5, 'link': 5},
    ],
    'excluded_links': <Object?>[],
    'matched_posts': [
      {
        'box': 1,
        'title': 2,
        'link': 2,
        'values': {
          'title': 'First post',
          'link': 'https://blog.example.com/first',
        },
      },
      {
        'box': 4,
        'title': 5,
        'link': 5,
        'values': {
          'title': 'Second post',
          'link': 'https://blog.example.com/second',
        },
      },
    ],
    'excluded_posts': <Object?>[],
    'preview': [
      {'title': 'First post', 'link': 'https://blog.example.com/first'},
      {'title': 'Second post', 'link': 'https://blog.example.com/second'},
    ],
  };

  /// What the selectors find in [exampleBlogWithNewsletters] when the first
  /// post is labeled and no link is excluded: all three posts.
  final exampleBlogWithNewslettersAllFound = <String, Object?>{
    'matched_posts': [
      {
        'box': 1,
        'title': 2,
        'link': 2,
        'values': {
          'title': 'First post',
          'link': 'https://blog.example.com/first',
        },
      },
      {
        'box': 4,
        'title': 5,
        'link': 5,
        'values': {
          'title': 'Weekly digest',
          'link': 'https://blog.example.com/newsletters?weekly',
        },
      },
      {
        'box': 7,
        'title': 8,
        'link': 8,
        'values': {
          'title': 'Monthly digest',
          'link': 'https://blog.example.com/newsletters?monthly',
        },
      },
    ],
    'excluded_posts': <Object?>[],
    'preview': [
      {'title': 'First post', 'link': 'https://blog.example.com/first'},
      {
        'title': 'Weekly digest',
        'link': 'https://blog.example.com/newsletters?weekly',
      },
      {
        'title': 'Monthly digest',
        'link': 'https://blog.example.com/newsletters?monthly',
      },
    ],
  };

  /// What the selectors find in [exampleBlogWithNewsletters] when the first
  /// post is labeled and the link of the newsletters is excluded.
  final exampleBlogWithNewslettersExcluded = <String, Object?>{
    'matched_posts': [
      {
        'box': 1,
        'title': 2,
        'link': 2,
        'values': {
          'title': 'First post',
          'link': 'https://blog.example.com/first',
        },
      },
    ],
    'excluded_posts': [
      {'box': 4, 'link': 'blog.example.com/newsletters'},
      {'box': 7, 'link': 'blog.example.com/newsletters'},
    ],
    'preview': [
      {'title': 'First post', 'link': 'https://blog.example.com/first'},
    ],
  };

  /// A new editor of [exampleBlogWithNewsletters] for a feed whose saved
  /// selectors find all three posts, which start as labeled.
  late final exampleBlogWithNewslettersEditor = <String, Object?>{
    'id': 42,
    'expires_at': '2026-07-03T13:30:00Z',
    'page': exampleBlogWithNewsletters,
    'labeled_posts': [
      {'box': 1, 'title': 2, 'link': 2},
      {'box': 4, 'title': 5, 'link': 5},
      {'box': 7, 'title': 8, 'link': 8},
    ],
    'excluded_links': <Object?>[],
    ...exampleBlogWithNewslettersAllFound,
  };
}
