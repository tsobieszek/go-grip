# Demonstration: Lists Without Blank Lines

This file demonstrates that lists are now properly rendered even when not preceded by a blank line.

## Before the fix
Previously, the following markdown would NOT render as a list:
```markdown
This is some text.
- Item 1
- Item 2
```

It would be rendered as plain text within the paragraph.

## After the fix
Now it works! Here's an example:
This is some text immediately followed by a list.
- First item
- Second item
- Third item

## Another example with ordered lists
Here is a paragraph of text without a blank line before the list.
1. First numbered item
2. Second numbered item
3. Third numbered item

## Works in blockquotes too
> This is a blockquote with text.
> - List item A
> - List item B

## Mixed list markers
Text before an asterisk list.
* Item with asterisk
* Another asterisk item

Text before a plus list.
+ Item with plus
+ Another plus item

## The traditional way still works
This is text with a blank line before the list.

- This has always worked
- And still does
- No changes here

## Summary
The preprocessor adds blank lines before lists that need them, making the markdown parser recognize them correctly. This implements GitHub Flavored Markdown behavior while maintaining backward compatibility.
