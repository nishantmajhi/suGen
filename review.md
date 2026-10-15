# Code Review — suGen

Reviewed 2026-09-04 against the working tree. Scope: `cmd/server/main.go`, `internal/generator/`,
`internal/handler/`, `internal/httputil/`, `internal/randutil/`, `internal/wordlist/`, `go.mod`,
`README.md`.

**Updated 2026-10-01 after the fixes on branch `cleanup/2026-10-01` (unpushed).** The first
review's P1 was real and is fixed. But it called this "a clean project" with findings "about the
deployment shape rather than the cryptography", and that undersold it. **One unauthenticated GET
stopped the server**, every password whose minimums were set near the length came out the wrong
length, and the page could show a password for settings other than the ones on screen. The
cryptography itself (`crypto/rand`, the unbiased `Intn`, the Fisher-Yates shuffle) was right, and
is untouched.

## Corrections to the first review

1. **"No tests" was wrong.** There were 37 tests in a top-level `test/` tree, and they passed.
   They covered the happy paths and never a boundary, so every bug below got through them. One of
   them asserted a bug as correct behaviour (minimums counted for classes that were switched
   off). With the tests outside the packages, `go test ./internal/...` said "no test files" and
   coverage read 0%. Moved beside the packages, coverage read 89.5% for the generator, which held
   the length bug the whole time.
2. **Finding 3's premise does not hold.** The generation options are never in the path. Both
   handlers read only the query string, and the logger never printed the query. The logger had a
   different problem: it printed the *decoded* path, so `%0A` in a URL wrote a forged line into
   the log. That is fixed, and a test pins that the query is never logged.
3. **Finding 4 ("no rate limiting... just a CPU and bandwidth sink", P3) missed the real issue.**
   No flood was needed. `passwordLength` went straight into `make([]byte, 0, n)`, and
   `GET /password/?passwordLength=1000000000000` ended the process with `fatal error: out of
   memory`, which `net/http` cannot recover. Reproduced against the original binary: the next
   request got no answer at all.
4. **The P1's remedy was half right.** `Cache-Control: no-store` is right and is done.
   `Pragma: no-cache` was not added: it is defined only as a request header (RFC 9111 §5.4,
   deprecated) and does nothing on a response.

## What was done

| # | Finding | State |
|---|---|---|
| 1 | P1: no `Cache-Control` on generated secrets | **Fixed** in `3ab4f69`. Checked by its effect: against the original, the browser suite finds both passwords and the passphrase a page showed in Chrome's disk cache; now it finds none of them |
| 2 | P2: server-side generation | **Documented, not changed.** The README's "Deploying it" says the server sees every secret, and that HTTPS and HSTS belong in a proxy. Moving generation into the browser would be a redesign of the author's app |
| 3 | P2: log the route, not the path | **Premise wrong** (correction 2). Log forging fixed instead: the path is logged escaped (`61096fc`) |
| 4 | P3: rate limiting | **Not added**; the README says to put it in a proxy. The real issue, unbounded sizes, is fixed (`232fdb2`) |
| 5 | P3: TLS/HSTS | **Documented** in the README |
| 6 | P3: tests | Already existed (correction 1). Moved (`be7985b`), 37 → 65, plus 11 real-Chrome checks |
| 7 | P3: `LICENSE` | **Not added**: the author's choice. The README says no licence has been granted |

Not in the review, all reproduced against the original first:

- **One GET stopped the server** (`232fdb2`): `passwordLength` is now at most 1024 and
  `totalWords` at most 64, checked before anything is allocated or drawn. Over the limit is
  refused with 200 and an `error` field, the API's existing shape for a refusal. The limits refuse
  rather than clamp, so nothing is ever silently weaker than asked.
- **Passwords of the wrong length** (`a1fdf03`). The length check left out the 1-2 guaranteed
  letters, so 8 characters with 3 digits and 5 special characters gave 10, under a slider reading
  8. It counted minimums for classes that were switched off, so with "Numbers" unticked (the page
  still sends its disabled minimum) 8 characters with 6 special characters was refused. A negative
  minimum let a negative length reach `make()` and panic. The check now counts what the generator
  guarantees, subtracts rather than adds so two huge minimums cannot overflow past it, and the
  draws use the same numbers.
- **An empty passphrase returned as success** (`30c745a`): `totalWords` of 0, negative or
  non-numeric gave `""`, or only the digits. Raised to 1 word, as `adjectiveNoun` already raised
  its count to 2. Both styles now share one `assemble()`.
- **`intval()` that was not** (`53cc578`): an overflowing number parsed as 0, where PHP
  saturates, so `passwordLength=99999999999999999999` gave a 2-character password. It now
  saturates, and is refused as too long.
- **No idle timeout** (`61096fc`): a keep-alive connection was held open indefinitely. Now 60 s.
- **Page: answers shown out of order** (`3353b32`). Whichever answer arrived last was shown. The
  browser suite caught this locally with nothing slowed down: the box ended on the previous
  settings' password. Each update now drops answers to anything but its latest request.
- **Page: a refusal blanked the box, and Copy copied the blank** (`3353b32`), saying "Password
  copied to clipboard!". The box now shows the server's sentence. Copy takes the password from the
  last good answer, and with none shows the page's existing "Failed to copy password."
- **Page: Back broke the form** (`8a6a061`). Without the back/forward cache, Chrome restores the
  ticks and sliders *after* `load`, with no change events (probed: checked at `DOMContentLoaded`,
  unchecked at `pageshow`). The page read everything at `DOMContentLoaded`, so the minimum inputs
  were enabled under unticked boxes, and then backwards on every click. The first password was
  also made from the defaults: digits under an unticked Numbers box. The first read is now at
  `pageshow`, skipped for a page restored whole from the bfcache.
- **Page: Regenerate's label** (`8a6a061`) said "Length: 12" whatever the slider said.

**What changes on screen**, all in failure states. The markup, CSS and wording are untouched.
It is recorded in `CLEANUP-LOG.md` as a correction, not an exception:

- Settings with no room show "password length too short for the specified minimum requirements"
  (the server's existing text) in the password box, which used to go blank.
- Copy with no password shows "Failed to copy password.", which already existed.
- After Back, the page follows the restored form, and the length label follows its slider.
- The first password now appears after `load` rather than `DOMContentLoaded`. With nothing but
  this page's own small files to load, there is no visible difference.

Tidy (`2cf5479`): three comments that described something else, and a `.gitignore` for the
README's build outputs. The README was rewritten (`48c255b`): it gives the strength of each kind
of secret with the arithmetic, and says how to deploy safely.

**Structure.** The old README defended the separate `test/` tree, knowing Go's convention was
otherwise. It was moved anyway (`be7985b`, a pure rename, revertible on its own) because the cost
was concrete, as correction 1 says.

## Tests

- `go test ./...`: 65 tests, up from 37. Every commit was checked out into its own worktree and
  tested there (37, 44, 50, 52, 54, 56, 62, 62, 62, 62, 65, 65, 65). Against the original logic,
  with only the new names declared in a shim, every new regression test fails.
- `node test/browser/run.js`: 11 checks in real headless Chrome. Needs Node 18+, Go and Chrome. It
  ran 11 of 11 from fresh worktrees of the commit that adds it and of the last commit. Against the
  first commit, exactly its 8 `regression:` checks fail and its 3 controls pass. The disk-cache
  check has a positive control: `password.js` must be found in the cache, so an empty cache cannot
  pass it.
- The three chi-square checks (`2c721e8`) were mutation-tested. An off-by-one shuffle scored
  6,666 against a limit of 31, and a missing shuffle 60,000 against 28.

Go is at `D:\dev-tools\go\bin\go.exe` (1.27.1), not on `PATH`. With `core.autocrlf=true` the
working copies are CRLF, and `gofmt -l` then lists every file; the committed files are LF and
gofmt-clean.

## For the author

- **If suGen is running anywhere reachable, one request can stop it** until it is redeployed from
  this branch. Nothing in the repository says it is deployed (no Dockerfile or platform config), so
  this is only a question.
- **The adjective + noun style at its default of two words is one of 25,754 possibilities,
  about 15 bits.** The README says so and recommends five words or the `words` style. Whether the
  page should default differently is a design decision, so it was left alone.
- A licence.

## Notes for a future session

- The probes that reproduced each bug are in the commit messages. The DoS needs only `curl`
  against a built binary. It ends the process, so use a port of your own.
- The browser suite's `--root` builds any checkout, so `git worktree add <dir> 3d03a8e` and
  `node test/browser/run.js --root <dir>` reproduce the "fails against the original" result.
- Do not replace `crypto/rand` for test determinism. None of the tests needed it: they assert
  properties (lengths, classes, refusals, distributions), not values.

---

The first-pass review follows, unchanged.

## What this project is

A Go HTTP service that generates passwords and passphrases, with an embedded static front end. It
exposes `GET /password/` and `GET /passphrase/` and serves the UI from an embedded filesystem.

## Verdict

The Go here is idiomatic and careful. Specifically worth noting, because these are the things that go
wrong in generators:

- **`crypto/rand` everywhere.** `randutil.Intn` uses `rand.Int(rand.Reader, big.NewInt(n))`, which is
  both cryptographically secure and free of modulo bias. No `math/rand` anywhere.
- **Fisher-Yates shuffle backed by the CSPRNG**, iterating downward with `Intn(i+1)` — the correct
  form, not the subtly-biased variant.
- **Every random call returns an error and every caller checks it**, rather than swallowing failures
  and silently producing a weak result.
- `ReadHeaderTimeout` is set, shutdown is graceful with a context timeout, the static files are
  embedded, and routing uses Go's method-prefixed patterns.

The package layout (`cmd/`, `internal/` split by concern) is the standard one. This is a clean
project.

The findings below are about the deployment shape rather than the cryptography.

## Findings

**1. Generated passwords cross the network, and the server sees every one of them.**
This is the fundamental design question for a web password generator. The server produces the secret
and sends it to the browser, which means:

- The password exists in the server's memory and in its response buffers.
- It travels over the wire — safe under TLS, fully exposed if this is ever served over plain HTTP on a
  network anyone else is on.
- Any reverse proxy, CDN or load balancer in front of it handles the plaintext secret.

The alternative is generating in the browser with `crypto.getRandomValues`, which is available
everywhere and is exactly as secure. The server would then serve only the wordlist and the UI. If you
want to keep the API — it is genuinely useful as an API — then say clearly in the README that the
HTTP endpoints are for programmatic use over TLS, and generate client-side for the web UI.

**2. There are no cache-control headers on the generated responses.** `httputil` sets only
`Content-Type: application/json` before writing the status. A `GET` response with no
`Cache-Control` is cacheable by browsers and intermediaries under default heuristics — which means a
generated password can sit in a disk cache or a proxy. Set `Cache-Control: no-store`, `Pragma:
no-cache` on every generator response. This is a two-line fix and it is the most important one here.

**3. The request logger records the path, which may contain generation parameters.**
`logRequests` logs `r.Method`, `r.URL.Path` and the duration — deliberately not the query string,
which is the right instinct. But the routes are `/password/` and `/passphrase/` with trailing slashes,
suggesting parameters may arrive as path segments. If length, character-set or count end up in the
path, they are being written to the log. That is not the password itself, but it narrows the search
space for anyone who has both the log and the timing. Confirm what the handlers read, and log only
the route pattern rather than the full path.

**4. No rate limiting.** An unauthenticated generator endpoint is cheap to call in a loop. Not a
disclosure risk, just a trivially abusable CPU and bandwidth sink if this is public.

**5. No TLS enforcement or HSTS.** If this is deployed behind a proxy that terminates TLS, add a
redirect and `Strict-Transport-Security` at that layer, and say so in the README so a future deployer
does not put it on plain HTTP.

**6. No tests.** Every generator function is pure apart from the randomness, which is already isolated
behind `randutil` — so injecting a deterministic source makes the whole thing testable. Worth
asserting: requested length is honoured exactly, every required character class appears when
requested, the shuffle touches every position, and `Intn(0)` returns the error rather than panicking.
A statistical smoke test over many samples would also catch a future regression to a biased source.

**7. No `LICENSE`.**

## Cleanup

Nothing to remove. No debug output, no secrets, no committed binaries, no dead code. The `words.txt`
wordlist matched an earlier search for tracker-like strings, but that was a false positive on ordinary
dictionary words.

## Action checklist

- [ ] **P1** Set `Cache-Control: no-store` on every generated-secret response.
- [ ] **P2** Decide server-side versus client-side generation; document the reasoning either way.
- [ ] **P2** Log the route pattern rather than the full path.
- [ ] **P3** Add tests with an injectable random source; add rate limiting.
- [ ] **P3** Document the TLS expectation; add a `LICENSE`.

## Notes for a future session

Build and run with `go run ./cmd/server`. The static UI is embedded, so a rebuild is needed after
changing it. Do not replace `crypto/rand` with `math/rand` for any reason, including test determinism
— inject a `io.Reader` instead, so the production path keeps the CSPRNG.

## Licence, 2026-10-02

The author chose one licence for all of their projects: free for personal,
non-commercial use, modifiable for the user's own use, and not to be
distributed, original or modified. No standard licence says that, so `LICENSE`
is a short custom text, the same in every project apart from the year and what
it excludes. The README's licence section says the same. Anything above that
calls the licence an open question, or calls the project MIT, is superseded.

## CI, 2026-10-02

`.github/workflows/ci.yml` runs `go vet` and `go test ./...` with the Go
version `go.mod` names (1.22.2), and the browser check as a second job, on
`ubuntu-latest`. Nothing has been pushed, so it has not run on GitHub yet.
Every job was run before committing, from a fresh clone of this branch, the
way a runner would run it: Linux jobs in a throwaway Ubuntu 24.04 (the image
`ubuntu-latest` uses) with the same tool versions. all Go tests passing on
1.22.2, the oldest the module allows; this machine verified them on 1.27.

The browser runner fails rather than skipping without Chrome.

**Timing:** the browser check passed 11 of 11 here, but not every time. In the
4 GB Linux VM used to check it, a page load usually took 1.5 to 2 s and now
and then 15 to 17 s, past the 15 s that `cdp.js` waits for
`Page.loadEventFired`; with the wait raised to 90 s it passed every run.
Nothing in the page or the server was slow (both address families answered in
under a millisecond), so this looks like the VM, and a runner has four times
the memory. If the job fails on GitHub with `no Page.loadEventFired`, raise
`timeoutMs` in `once()`.
