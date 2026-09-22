# Sanitize fixtures

Each file holds the output of `sanitize` for the saved page of the same name in
the parent directory, produced by running that function itself. They record what
the cleaning rules remove today, so a change in those rules is reviewed as a
diff of these files. They are formatted with
`npx prettier --embedded-language-formatting=off`, which is required because
prettier otherwise rewrites the CSS inside `style` attributes.
