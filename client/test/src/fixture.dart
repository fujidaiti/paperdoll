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
  );

  final nasa = api.Feed(
    id: 2,
    url: 'http://www.nasa.gov/news-release/feed/',
    siteUrl: 'https://www.nasa.gov',
    title: 'NASA',
    description:
        'Official National Aeronautics and Space Administration Website',
  );

  final stackOverflow = api.Feed(
    id: 3,
    url: 'http://stackoverflow.com/feeds',
    siteUrl: 'https://stackoverflow.com/questions',
    title: 'Recent Questions - Stack Overflow',
    description: 'most recent 30 from stackoverflow.com',
  );

  final wikipedia = api.Feed(
    id: 4,
    url: 'http://en.wikipedia.org/w/api.php?limit=50&action=feedrecentchanges&feedformat=rss',
    siteUrl: 'https://en.wikipedia.org/wiki/Special:RecentChanges',
    title: 'Wikipedia  - Recent changes [en]',
    description: 'Track the most recent changes to the wiki in this feed.',
  );

  // A plain HTML page with no feed, subscribed to with selectors.
  final exampleBlog = api.Feed(
    id: 5,
    url: 'https://blog.example.com/',
    siteUrl: 'https://blog.example.com/',
    title: 'Example Blog',
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

  // What /feeds/search returns for a plain HTML page with no feed. Each group
  // exercises a different rule of the subscription flow:
  // - group 0 is the normal path: one link candidate, so the link question
  //   is skipped, and a `p` row that only one of the sampled items carries.
  // - group 1 has no link candidate, so it cannot be ticked.
  // - group 2 has two link candidates, so the link question is asked.
  final exampleBlog = api.FeedCandidate(
    url: _feeds.exampleBlog.url,
    siteUrl: _feeds.exampleBlog.siteUrl,
    title: _feeds.exampleBlog.title,
    postGroups: [
      api.PostGroup(
        id: 0,
        selector: 'main > article',
        count: 24,
        // Three sampled items rather than two, because the third is the one
        // that carries the `p` row. The server grows the sample until every
        // row it reports has a value somewhere in it.
        sampled: 3,
        links: [
          api.AttributeCandidate(
            selector: ':scope',
            values: [
              api.AttributeValue(value: 'https://blog.example.com/posts/1'),
              api.AttributeValue(value: 'https://blog.example.com/posts/2'),
              api.AttributeValue(value: 'https://blog.example.com/posts/3'),
            ],
          ),
        ],
        texts: [
          api.AttributeCandidate(
            selector: 'div > h3',
            values: [
              api.AttributeValue(value: 'Why we moved to a monorepo'),
              api.AttributeValue(value: 'Notes from the September release'),
              api.AttributeValue(value: 'A week of incident reviews'),
            ],
          ),
          api.AttributeCandidate(
            selector: 'div > time',
            values: [
              api.AttributeValue(value: '2026-09-12'),
              api.AttributeValue(value: '2026-09-03'),
              api.AttributeValue(value: '2026-08-28'),
            ],
          ),
          // The optional description. Only the third post carries it, so the
          // first two entries are empty.
          api.AttributeCandidate(
            selector: 'p',
            values: [
              api.AttributeValue(),
              api.AttributeValue(),
              api.AttributeValue(
                value: 'What three days of incident reviews taught us.',
              ),
            ],
          ),
        ],
        images: [
          api.AttributeCandidate(
            selector: 'img',
            values: [
              api.AttributeValue(
                value: 'https://blog.example.com/images/1.png',
                alt: 'A diagram of the repository layout',
              ),
              api.AttributeValue(
                value: 'https://blog.example.com/images/2.png',
                alt: 'The release banner',
              ),
              api.AttributeValue(
                value: 'https://blog.example.com/images/3.png',
                alt: 'A wall of sticky notes',
              ),
            ],
          ),
        ],
      ),
      api.PostGroup(
        id: 1,
        selector: 'aside > ul > li',
        count: 6,
        sampled: 2,
        texts: [
          api.AttributeCandidate(
            selector: ':scope',
            values: [
              api.AttributeValue(value: 'Engineering'),
              api.AttributeValue(value: 'Announcements'),
            ],
          ),
        ],
      ),
      api.PostGroup(
        id: 2,
        selector: 'section.news > div',
        count: 10,
        sampled: 2,
        links: [
          api.AttributeCandidate(
            selector: 'a.headline',
            values: [
              api.AttributeValue(value: 'https://blog.example.com/news/1'),
              api.AttributeValue(value: 'https://blog.example.com/news/2'),
            ],
          ),
          api.AttributeCandidate(
            selector: 'a.more',
            values: [
              api.AttributeValue(value: 'https://blog.example.com/news/1#more'),
              api.AttributeValue(value: 'https://blog.example.com/news/2#more'),
            ],
          ),
        ],
        texts: [
          api.AttributeCandidate(
            selector: 'a.headline',
            values: [
              api.AttributeValue(value: 'Office hours move to Thursdays'),
              api.AttributeValue(value: 'New contributors this month'),
            ],
          ),
        ],
      ),
    ],
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

  final nuclearDeal = api.ReadingListItem(
    id: 5,
    resourceId: _entries.nuclearDeal.id,
    kind: api.ReadingListItemKindEnum.feedEntry,
    title: _entries.nuclearDeal.title,
    savedAt: DateTime.utc(2026, 7, 1),
  );
}
