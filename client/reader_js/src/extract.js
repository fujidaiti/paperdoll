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
      const article = new Readability(document.cloneNode(true)).parse();
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
