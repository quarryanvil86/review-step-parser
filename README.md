# review-step-parser

A small Go library for parsing and pretty-printing spaced-repetition
review-step schedules: the short interval lists that decide how a card
moves through the early "learning" phase before it graduates to normal
review, e.g. `1m 10m 1d 3d`.

## The problem

Most spaced-repetition tools (Anki and its clones, mainly) let a user
type a schedule like this straight into a config field:

```
1m 10m 1d 3d 7d
```

That string is free text as far as the app is concerned until something
parses it, and the failure modes are the boring, predictable kind:

- a typo turns `1d` into `1.d` and the app silently drops the step
- someone writes `1M` meaning minutes and it gets read as months (or the
  other way around), and a card that should come back in a minute comes
  back in thirty days instead
- steps end up out of order (`1d 1h`) and the schedule effectively gets
  shorter as the card advances, which defeats the point of spaced review
- the same schedule gets typed two different ways (`60m` vs `1h`) and now
  two cards that should be identical look different in storage, diffs,
  and support tickets

`review-step-parser` turns that string into a validated `[]Step` and back
into one canonical string, so a config can be checked on save and shown
back to the user in a single consistent form.

## Grammar

A schedule is one or more whitespace-separated steps. Each step is a
positive integer immediately followed by a unit:

| suffix | unit    |
|--------|---------|
| `m`    | minutes |
| `h`    | hours   |
| `d`    | days    |
| `mo`   | months  |
| `y`    | years   |

`mo` is spelled out on purpose. `m` was already taken by minutes, and
guessing which one a bare `m` means is exactly the kind of ambiguity this
package exists to remove.

Rules a schedule has to satisfy to be valid:

- at least one step, at most `MaxSteps` (32)
- every step's value is between 1 and `MaxStepValue` (9999)
- each step's duration must be at least as long as the one before it
  (repeating a duration, like `60m 1h`, is fine; going backwards isn't)

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/quarryanvil86/review-step-parser"
)

func main() {
	steps, err := schedstep.ParseSchedule("10m 1h 1440m")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(schedstep.FormatSchedule(steps)) // "10m 1h 1d"

	if _, err := schedstep.ParseSchedule("1d 1h"); err != nil {
		fmt.Println(err) // step 2 ("1h"): interval is shorter than the previous step
	}
}
```

`Normalize` is a shortcut for the common case of parsing and immediately
re-printing in canonical form:

```go
clean, err := schedstep.Normalize("  10m   1h  ")
// clean == "10m 1h", err == nil
```

## Status

Early. The parser and printer cover the core grammar above; see the
issue tracker for what's planned next. Integer-only step values are a
deliberate simplification for now, not a final design decision.

## License

MIT, see [LICENSE](LICENSE).
