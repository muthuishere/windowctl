# clipboard

## ADDED Requirements

### Requirement: Read and write the system clipboard

`windowctl clipboard get` SHALL print the current text contents of the system pasteboard;
`windowctl clipboard set --text <s>` (or stdin) SHALL replace it. This is the reliable,
layout/IME-independent path for inserting text into a focused field (set clipboard, then
paste), sidestepping synthetic-keystroke drops.

#### Scenario: Round-trip

- **WHEN** `windowctl clipboard set --text "hello"` then `windowctl clipboard get`
- **THEN** the second command prints `hello`.

#### Scenario: Set from stdin

- **WHEN** text is piped to `windowctl clipboard set` with no `--text`
- **THEN** the piped bytes become the clipboard contents.

#### Scenario: Reliable insertion path

- **WHEN** a caller does `clipboard set` then `key --combo cmd+v` into a focused field
- **THEN** the full text lands even for content lengths where synthetic typing drops
  characters.
