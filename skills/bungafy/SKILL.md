---
name: bungafy
description: Compress markdown files to reduce token count. Use when the user says "bungafy", "compress this file", "reduce tokens in", or "shrink this markdown".
allowed-tools: Read Write Edit Glob Grep
effort: medium
argument-hint: [file or glob pattern]
---

# Bungafy

Compress markdown files. Drop articles, simplify grammar, keep meaning.

## Rules

- **Filler words**: delete ("very", "really", "just", "simply", "basically", "actually", "that said", "note that", "please note", "it is worth noting", "in order to" → "to")
- **Shorten phrases**: "is able to" → "can", "make sure to" → "ensure", "due to the fact that" → "because", "at this point in time" → "now", "in the event that" → "if", "for the purpose of" → "to", "with the exception of" → "except"
- **Collapse redundant headers**: merge header + one-sentence section into single line where possible
- **Drop articles**: remove "a", "an", "the" where sentence stays clear
- **Caveman grammar**: strip auxiliaries/connectors — "you should ensure" → "ensure", "it will be used to" → "used to"
- **Remove repetitions**: concept/instruction appearing twice — keep first, delete rest
- **Compress sentences**: active voice, merge short adjacent sentences, drop obvious subjects
- **Whitespace**: no empty lines between list items, no trailing blank lines

Do NOT:
- Change code blocks
- Remove meaning-bearing content
- Break heading navigation
- Remove frontmatter

## Steps

1. **Resolve targets** — use `$ARGUMENTS` as path/glob; if absent, ask user.
2. **Read** each file.
3. **Compress** — apply all rules. Aim 20–40% token reduction.
4. **Write back** — overwrite each file.
5. **Report** — original line count → new line count, estimated token reduction.
