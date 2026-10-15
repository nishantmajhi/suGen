# suGen

A password and passphrase generator: a small Go HTTP server with a one-page
web UI. It makes three kinds of secret:

- **Passwords**: random characters, with options for length, upper case,
  digits, special characters, and leaving out characters that are easy to
  misread (`I`, `l`, `O`, `0`).
- **Word passphrases**: random words from a 367,522-word dictionary, joined
  with spaces: `Gremmie Heartfully Teletypewriters`.
- **Adjective + noun passphrases**: one or more adjectives followed by one
  noun, from short lists of everyday words: `Rustic Dazzling Plain Shepherd`.

Everything random comes from Go's `crypto/rand`. The word lists and the web
page are compiled into the binary, so the server is one file with no
database and no third-party dependencies.

**The server generates the secret and sends it to the browser.** Read
[Deploying it](#deploying-it) before putting it anywhere other people can
reach.

## Running it

You need Go 1.22 or newer. If `go version` does not work, install Go from
<https://go.dev/dl/> first (the Windows `.msi` and macOS `.pkg` installers
put it on your `PATH`; restart the terminal afterwards).

From the folder that contains `go.mod`:

```sh
go run ./cmd/server
```

```
2026/10/01 19:30:00 suGen listening on :3000
```

Open <http://localhost:3000>. Stop it with `Ctrl+C`.

To listen on another port, set `PORT`. In a POSIX shell:

```sh
PORT=8080 go run ./cmd/server
```

In PowerShell, `$env:PORT = "8080"` first. `PORT` is the only setting.

To build a single file you can copy to another machine that has no Go:

```sh
go build -o sugen-server ./cmd/server
GOOS=linux GOARCH=amd64 go build -o sugen-server-linux ./cmd/server   # cross-compile
```

The page is embedded at build time, so after changing anything under
`web/static/`, run `go run` again or rebuild.

## The page

The page shows one password and one passphrase, each with a copy button,
and the options for each. Changing an option fetches a new one. The round
button at the bottom fetches both again (and puts the word count back to
2). Settings with no room in them, such as 8 characters with at least 3
digits and 5 special characters, show the server's reason in the
password box, and the copy button refuses until there is a password again.

The page uses the same two endpoints described next.

## API

Two `GET` endpoints, both answering JSON. Every parameter is optional.
`true` and `false` are case-insensitive; anything else counts as `false`.
Numbers are read the way PHP's `intval()` reads them (`12abc` is 12, `abc`
is 0, and a number too large for an integer is the largest integer).

Every answer from these two endpoints carries `Cache-Control: no-store`,
so neither a browser nor a proxy keeps a copy of it.

### `GET /password/`

| Parameter | Default | Meaning |
|---|---|---|
| `passwordLength` | `12` | Exact length of the password. At most 1024. |
| `mixedCase` | `true` | Use upper-case letters as well as lower-case. |
| `numbers` | `true` | Use digits. |
| `minimumDigits` | `0` | At least this many digits. Ignored when `numbers` is false. |
| `specialCharacters` | `true` | Use `~!@#$%^&-=_+?`. |
| `minimumSpecialCharacters` | `0` | At least this many special characters. Ignored when `specialCharacters` is false. |
| `avoidAmbiguousCharacters` | `true` | Never use `I`, `l`, `O` or `0`. |

Every password contains at least one lower-case letter, and at least one
upper-case letter when `mixedCase` is true. The positions of all the
guaranteed characters are shuffled.

```sh
curl "http://localhost:3000/password/?passwordLength=16&minimumDigits=3&minimumSpecialCharacters=2"
```

```json
{"password":"rB#?1e6_8qYJ$p5K"}
```

### `GET /passphrase/`

| Parameter | Default | Meaning |
|---|---|---|
| `style` | `words` | `words` or `adjectiveNoun` (case-insensitive). Anything else means `words`. |
| `totalWords` | `2` | How many words, at most 64. For `words`, fewer than 1 means 1. For `adjectiveNoun`, fewer than 2 means 2, and the last word is the noun. |
| `capitalizeWords` | `true` | Capitalize the first letter of each word. |
| `includeNumbers` | `false` | Add a space and 1 to 4 random digits at the end. |

```sh
curl "http://localhost:3000/passphrase/?totalWords=3"
```

```json
{"passphrase":"Gremmie Heartfully Teletypewriters"}
```

```sh
curl "http://localhost:3000/passphrase/?style=adjectiveNoun&includeNumbers=true"
```

```json
{"passphrase":"Odd Nebula 5"}
```

```sh
curl "http://localhost:3000/passphrase/?style=adjectiveNoun&capitalizeWords=false&includeNumbers=true"
```

```json
{"passphrase":"stormy diamond 336"}
```

### When a request cannot be met

The answer is still `200`, with an `error` field instead of the secret:

```json
{"error":"password length too short for the specified minimum requirements"}
{"error":"password length exceeds the maximum of 1024"}
{"error":"number of words exceeds the maximum of 64"}
```

A password is too short when `passwordLength` is less than the characters
it must contain: 1 lower-case letter, plus 1 upper-case letter with
`mixedCase`, plus each minimum that is switched on. So 8 characters with at
least 3 digits and 5 special characters is refused: with the two letters
that makes 10.

A `500` with `{"error":"internal server error"}` means the system's random
source failed; the details go to the server's log.

## How strong is what it makes

Strength is the number of equally likely results, written as bits (each bit
doubles the guesses an attacker needs). These assume the attacker knows
exactly how suGen works, which is the safe assumption.

| What | Bits |
|---|---|
| Password, 12 characters, every class, ambiguous characters left out (the page's default) | about 70 |
| Password, 24 characters, same | about 145 |
| Word passphrase, 2 words (the default) | 37 |
| Word passphrase, 5 words | 92 |
| Adjective + noun, 2 words | **15** |
| Adjective + noun, 5 words | 37 |
| `includeNumbers` adds | at most 13 |

The dictionary has 367,522 words, so each word adds 18.5 bits. The
adjective and noun lists have 163 and 158 words, so each of those adds
about 7.3. **A two-word adjective + noun passphrase is one of 25,754
possibilities**: an attacker can simply try every one. That style is meant
to be easy to remember, not to stand alone as a password; use five words
or more, or the `words` style. Capitalizing adds nothing, since it is
always all or none. The digit suffix is one of 11,110 values, but it is a
single digit a quarter of the time.

## Deploying it

The design has consequences worth knowing:

- **The server sees every secret it generates**, and so does anything
  between it and the browser. Serve it only over HTTPS. If a proxy
  terminates TLS in front of it, redirect HTTP to HTTPS and send
  `Strict-Transport-Security` there; suGen itself only speaks plain HTTP.
- **Nothing is logged that would reveal a secret.** The log has the method,
  the path and the time taken, never the query string or the response.
- **There is no rate limiting.** Each request is cheap and bounded (at most
  1024 characters or 64 words), but nothing stops one client sending many.
  Put a limit in the proxy if the server is public.
- Connections time out: 5 seconds to send the request headers, 60 seconds
  idle between requests.

## Tests

```sh
go test ./...
```

The tests sit beside the code they test, as Go expects. They cover the
generators (including every combination of password options over small
lengths, and chi-square checks that the random choices and the shuffle are
fair), the query parsing, both handlers, and the server's routes and log.

There is also a check of the page itself, in real headless Chrome:

```sh
node test/browser/run.js
```

It builds the server, starts it on a free port, and drives the page: the
boxes, the copy buttons, answers arriving out of order, the browser
restoring the form on Back, and that no secret it showed is left in
Chrome's disk cache. It needs Node 18 or newer, Go, and Chrome (set
`CHROME` if Chrome is somewhere unusual, and `GO` if `go` is not on your
`PATH`). `--root <folder>` checks another copy of the project instead.

## Layout

```
cmd/server/          main.go: routes, request log, timeouts, startup and shutdown
internal/generator/  passwords and both passphrase styles (no HTTP in here)
internal/handler/    turns a query string into generator options, and the result into JSON
internal/httputil/   query parsing and the JSON response
internal/randutil/   crypto/rand helpers: Intn, Digit, Shuffle
internal/wordlist/   loads the three embedded word lists (data/*.txt)
web/static/          the page: index.html, CSS, JS
web/embed.go         embeds web/static into the binary
test/browser/        the headless Chrome check
```

## Background

suGen began as a PHP script. That version read through the whole dictionary
file for every word it picked, which the author measured at about 135 ms
per word; this one loads the lists once at startup. The Go version keeps
the PHP API's endpoints, parameter names, defaults and its habit of
answering an impossible request with `200` and an `error` field. It also
fixed one PHP bug (`avoidAmbiguousCharacters` did not apply to the
guaranteed characters) and added the adjective + noun style.

## Licence

Free for personal, non-commercial use. You may use, study and modify this
software for yourself, but you may not use it commercially, and you may not
share or publish it, in its original form or modified. See [LICENSE](LICENSE)
for the full terms.
