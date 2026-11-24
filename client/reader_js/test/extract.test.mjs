// Runs the built client/assets/reader/extract.js, the script the app ships,
// against saved pages under jsdom. Run with `make reader-js-test`.
//
// These tests cover the script's logic only. How a real WebView injects it and
// delivers the message is covered by the Patrol test in client/e2e.

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { JSDOM } from "jsdom";

const script = readFileSync(
  new URL("../../assets/reader/extract.js", import.meta.url),
  "utf8",
);

/// Loads a fixture page at [url] with the `PaperdollReader` channel stubbed,
/// injects the script, and returns the window and the posted messages.
function load(fixture, url = "https://coastal.example/posts/1") {
  const html = readFileSync(new URL(`fixtures/${fixture}`, import.meta.url));
  const dom = new JSDOM(html, { url, runScripts: "outside-only" });
  const messages = [];
  dom.window.PaperdollReader = { postMessage: (m) => messages.push(m) };
  dom.window.eval(script);
  return { window: dom.window, messages };
}

function extract(window, messages, messageId = "m1") {
  window.eval(`window.__paperdollReader.extract(${JSON.stringify(messageId)})`);
  assert.equal(messages.length, 1, "posts exactly one message");
  assert.equal(typeof messages[0], "string");
  return JSON.parse(messages[0]);
}

test("an article page yields the article and its metadata", () => {
  const { window, messages } = load("article.html");
  const message = extract(window, messages, "id-42");

  assert.equal(message.v, 1);
  assert.equal(message.messageId, "id-42");
  assert.equal(message.status, "ok");
  assert.equal(message.url, "https://coastal.example/posts/1");
  assert.equal(message.title, "How Tide Pools Survive the Summer");
  assert.equal(message.byline, "Mina Sato");
  assert.equal(message.siteName, "Coastal Notes");
  assert.equal(message.error, null);
  assert.match(message.content, /Anemones close up/);
  assert.match(message.content, /<figure>/);
  assert.doesNotMatch(message.content, /Archive/, "navigation is dropped");
  // Readability resolves relative URLs against the page URL.
  assert.match(
    message.content,
    /src="https:\/\/coastal\.example\/images\/pool\.jpg"/,
  );
  assert.match(
    message.content,
    /href="https:\/\/coastal\.example\/data\/pools\.csv"/,
  );
});

test("a page rendered by JavaScript yields its article", () => {
  const { window, messages } = load("js_rendered.html");
  const message = extract(window, messages);

  assert.equal(message.status, "ok");
  assert.match(message.content, /Gulls that nest near the harbour/);
  assert.doesNotMatch(message.content, /Search/, "toolbar is dropped");
});

test("figures in blocks with media in their class names are kept", () => {
  const { window, messages } = load("media_column.html");
  const message = extract(window, messages);

  assert.equal(message.status, "ok");
  assert.match(message.content, /rubble-mound\.png/);
  assert.match(message.content, /composite\.png/);
});

for (const fixture of ["index.html", "app_shell.html"]) {
  test(`${fixture} yields no article`, () => {
    const { window, messages } = load(fixture);
    const message = extract(window, messages);

    assert.deepEqual(message, {
      v: 1,
      messageId: "m1",
      status: "empty",
      url: "https://coastal.example/posts/1",
      title: null,
      byline: null,
      siteName: null,
      content: null,
      error: null,
    });
  });
}

test("an exception is reported as an error", () => {
  const { window, messages } = load("article.html");
  window.document.cloneNode = () => {
    throw new Error("clone failed");
  };
  const message = extract(window, messages);

  assert.equal(message.status, "error");
  assert.equal(message.error, "clone failed");
  assert.equal(message.content, null);
  assert.equal(message.title, null);
});

test("the URL after a redirect or pushState is reported", () => {
  const { window, messages } = load("article.html");
  window.history.pushState(null, "", "/posts/1?page=2");
  const message = extract(window, messages);

  assert.equal(message.url, "https://coastal.example/posts/1?page=2");
});

test("injecting the script twice keeps the first definition", () => {
  const { window } = load("article.html");
  const first = window.__paperdollReader;
  window.eval(script);

  assert.equal(window.__paperdollReader, first);
});

test("extraction leaves the live document unchanged", () => {
  const { window, messages } = load("article.html");
  const before = window.document.documentElement.outerHTML;
  extract(window, messages);

  assert.equal(window.document.documentElement.outerHTML, before);
});
