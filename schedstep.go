// Package schedstep parses and prints the review-step schedules used by
// spaced-repetition learning phases: short, space-separated lists of
// intervals such as "1m 10m 1d 3d" describing how long to wait before a
// card comes back for review again.
//
// The grammar for a single step is a positive integer immediately followed
// by a unit: m (minutes), h (hours), d (days), mo (months), y (years).
// "mo" is spelled out on purpose, since "m" is already taken by minutes and
// silently guessing which one a user meant is how these configs end up
// scheduling a card for 30 minutes instead of 30 months.
package schedstep

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Unit identifies the time unit of a Step. The zero value is Minute.
type Unit uint8

const (
	Minute Unit = iota
	Hour
	Day
	Month
	Year
)

// unitMinutes gives the length of each unit in minutes. Month and year are
// fixed-length (30 and 365 days) rather than calendar-aware, which is the
// right tradeoff for a step list: it describes an offset, not a date.
var unitMinutes = [...]uint64{
	Minute: 1,
	Hour:   60,
	Day:    1440,
	Month:  43200,
	Year:   525600,
}

var unitSuffixes = [...]string{
	Minute: "m",
	Hour:   "h",
	Day:    "d",
	Month:  "mo",
	Year:   "y",
}

// canonicalOrder is the order Canonical walks when picking the largest unit
// that divides a duration evenly, largest first.
var canonicalOrder = [...]Unit{Year, Month, Day, Hour, Minute}

func unitFromSuffix(sfx string) (Unit, bool) {
	switch sfx {
	case "m":
		return Minute, true
	case "h":
		return Hour, true
	case "d":
		return Day, true
	case "mo":
		return Month, true
	case "y":
		return Year, true
	default:
		return 0, false
	}
}

// MaxStepValue bounds the numeric part of a step. It exists so a typo like
// a stray extra digit produces a validation error instead of a schedule
// that quietly waits for centuries.
const MaxStepValue = 9999

// MaxSteps bounds how many steps a single schedule may contain.
const MaxSteps = 32

// Step is one interval in a schedule, e.g. "3d" is Step{Value: 3, Unit: Day}.
type Step struct {
	Value uint32
	Unit  Unit
}

func (s Step) minutes() uint64 {
	return uint64(s.Value) * unitMinutes[s.Unit]
}

// String renders the step in its own unit, e.g. Step{3, Day}.String() == "3d".
// It does not canonicalize; use Canonical first if you want the shortest
// exact representation.
func (s Step) String() string {
	return fmt.Sprintf("%d%s", s.Value, unitSuffixes[s.Unit])
}

// Canonical returns the step rewritten in the largest unit that represents
// its duration without a remainder, e.g. 90 minutes stays "90m" but 120
// minutes becomes "2h". This is what gives the pretty printer a single
// stable form regardless of which unit the input happened to use.
func (s Step) Canonical() Step {
	mins := s.minutes()
	for _, u := range canonicalOrder {
		um := unitMinutes[u]
		if mins%um == 0 {
			return Step{Value: uint32(mins / um), Unit: u}
		}
	}
	// unreachable: Minute has unitMinutes 1, so it always divides evenly.
	return s
}

// ParseError describes why a single step in a schedule failed to parse or
// validate. Index is 1-based, matching how a person would count the steps
// when reading the schedule left to right.
type ParseError struct {
	Token  string
	Index  int
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("step %d (%q): %s", e.Index, e.Token, e.Reason)
}

var stepPattern = regexp.MustCompile(`^([0-9]+)(mo|m|h|d|y)$`)

func parseStep(token string) (Step, error) {
	m := stepPattern.FindStringSubmatch(token)
	if m == nil {
		return Step{}, errors.New("not a valid step (want a number followed by m, h, d, mo, or y)")
	}

	value, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil {
		return Step{}, fmt.Errorf("number out of range: %w", err)
	}
	if value == 0 {
		return Step{}, errors.New("interval must be greater than zero")
	}
	if value > MaxStepValue {
		return Step{}, fmt.Errorf("value exceeds maximum of %d", MaxStepValue)
	}

	unit, ok := unitFromSuffix(m[2])
	if !ok {
		// stepPattern only captures known suffixes, so this cannot happen.
		return Step{}, fmt.Errorf("unknown unit %q", m[2])
	}

	return Step{Value: uint32(value), Unit: unit}, nil
}

// ParseSchedule parses a whitespace-separated list of steps and validates
// it as a usable spaced-repetition schedule: every token must be a valid
// step, there must be at least one and at most MaxSteps, and each step's
// duration must be no shorter than the one before it. Schedules are allowed
// to repeat a duration (e.g. "1h 60m") but never go backwards.
func ParseSchedule(s string) ([]Step, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil, errors.New("schedule has no steps")
	}
	if len(fields) > MaxSteps {
		return nil, fmt.Errorf("schedule has %d steps, max is %d", len(fields), MaxSteps)
	}

	steps := make([]Step, 0, len(fields))
	var prevMinutes uint64
	for i, f := range fields {
		st, err := parseStep(f)
		if err != nil {
			return nil, &ParseError{Token: f, Index: i + 1, Reason: err.Error()}
		}
		mins := st.minutes()
		if i > 0 && mins < prevMinutes {
			return nil, &ParseError{Token: f, Index: i + 1, Reason: "interval is shorter than the previous step"}
		}
		prevMinutes = mins
		steps = append(steps, st)
	}
	return steps, nil
}

// FormatSchedule pretty-prints a slice of steps as a single space-separated
// line, canonicalizing each step's unit. It is the inverse of ParseSchedule
// for the purpose of normalizing whatever a user typed.
func FormatSchedule(steps []Step) string {
	parts := make([]string, len(steps))
	for i, st := range steps {
		parts[i] = st.Canonical().String()
	}
	return strings.Join(parts, " ")
}

// Normalize parses s and re-prints it in canonical form. It is a
// convenience for the common case of cleaning up user input before storing
// it, and it fails the same way ParseSchedule does on invalid input.
func Normalize(s string) (string, error) {
	steps, err := ParseSchedule(s)
	if err != nil {
		return "", err
	}
	return FormatSchedule(steps), nil
}
