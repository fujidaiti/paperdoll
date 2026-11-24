// Entry point of the script the reader injects into the original page. It is
// bundled with Readability into client/assets/reader/extract.js by
// `make reader-js`; see that target for how to rebuild it.
//
// Dart calls `window.__paperdollReader.extract(messageId)`, and the script
// posts exactly one JSON string to the `PaperdollReader` channel. The message
// format (contract version 1) is documented on `ReaderPageController` in the
// Dart code, which treats every field as untrusted: the page's own scripts
// share this JavaScript context and can tamper with anything here.

import { Readability, isProbablyReaderable } from "@mozilla/readability";

// The class and ID names that count against a block as article content.
// Readability's own list also has "media" and "scroll", which match blocks of
// the article itself, such as a "media-column" holding its figures, so those
// figures were removed. This is the list of go-readability, which the server
// extracts articles with.
const NEGATIVE_CLASSES =
  /-ad-|hidden|^hid$| hid$| hid |^hid |banner|combx|comment|com-|contact|footer|gdpr|masthead|meta|outbrain|promo|related|share|shoutbox|sidebar|skyscraper|sponsor|shopping|tags|widget/i;

// Injecting the script twice keeps the first definition.
if (!window.__paperdollReader) {
  window.__paperdollReader = { extract };
}

function extract(messageId) {
  const message = {
    v: 1,
    messageId,
    status: "empty",
    url: document.URL,
    title: null,
    byline: null,
    siteName: null,
    content: null,
    error: null,
  };
  try {
    // Pages without article-like text (index pages, app shells, paywall
    // stubs) still give Readability something to return, so they are
    // reported as empty before it runs.
    if (isProbablyReaderable(document)) {
      // Readability modifies the document it parses. A clone keeps the live
      // page intact, because the reader may still show it as a fallback.
      const reader = new Readability(document.cloneNode(true));
      // REGEXPS is not a public option: the extract.js tests fail if an
      // upgrade of Readability stops reading it.
      reader.REGEXPS = { ...reader.REGEXPS, negative: NEGATIVE_CLASSES };
      const article = reader.parse();
      if (article && article.content) {
        message.status = "ok";
        message.title = article.title || null;
        message.byline = article.byline || null;
        message.siteName = article.siteName || null;
        message.content = article.content;
      }
    }
  } catch (e) {
    message.status = "error";
    message.error = String((e && e.message) || e);
  }
  window.PaperdollReader.postMessage(JSON.stringify(message));
}
