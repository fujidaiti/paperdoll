# Sanitize fixtures

Each file holds the output of `sanitize` for the saved page of the same name in
the parent directory, produced by running that function itself. They record what
the cleaning rules remove today, so a change in those rules is reviewed as a
diff of these files.

They are formatted with prettier, which must not format the CSS inside `style`
attributes: `TestSanitize` compares the element tree and the attribute values as
they are, so rewriting `style="position:absolute"` into
`style="position: absolute;"` makes the fixture stop matching. The repository's
`.prettierrc` carries an override that turns that off for this directory, so
`npx prettier --write` on these files is safe, and so is formatting them from an
editor.
