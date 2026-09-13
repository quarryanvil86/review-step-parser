package schedstep

import (
	"strings"
	"testing"
)

func TestParseScheduleValid(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []Step
	}{
		{
			name:  "single step",
			input: "1m",
			want:  []Step{{1, Minute}},
		},
		{
			name:  "typical learning steps",
			input: "10m 1h 1d",
			want:  []Step{{10, Minute}, {1, Hour}, {1, Day}},
		},
		{
			name:  "runs of whitespace and tabs collapse",
			input: "10m   1h\t1d",
			want:  []Step{{10, Minute}, {1, Hour}, {1, Day}},
		},
		{
			name:  "leading and trailing whitespace is trimmed",
			input: "  1m 1h  ",
			want:  []Step{{1, Minute}, {1, Hour}},
		},
		{
			name:  "month uses the mo suffix, distinct from minutes",
			input: "1m 1mo",
			want:  []Step{{1, Minute}, {1, Month}},
		},
		{
			name:  "equal durations across units are allowed",
			input: "60m 1h",
			want:  []Step{{60, Minute}, {1, Hour}},
		},
		{
			name:  "equal durations the other direction are also allowed",
			input: "1h 60m",
			want:  []Step{{1, Hour}, {60, Minute}},
		},
		{
			name:  "value at the maximum is allowed",
			input: "9999d",
			want:  []Step{{9999, Day}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSchedule(tc.input)
			if err != nil {
				t.Fatalf("ParseSchedule(%q) returned error: %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseSchedule(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ParseSchedule(%q)[%d] = %v, want %v", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseScheduleInvalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   \t  "},
		{"zero value", "0d"},
		{"unknown unit", "5w"},
		{"missing unit", "5"},
		{"missing number", "d"},
		{"uppercase unit is not the same as lowercase", "5M"},
		{"negative number", "-5d"},
		{"decimal value", "1.5d"},
		{"decreasing interval", "1d 1h"},
		{"decreasing interval later in the list", "1m 1h 30m"},
		{"number overflows the step maximum", "10000d"},
		{"number overflows the parser entirely", "99999999999999999999d"},
		{"stray token mixed with valid ones", "1m foo 1h"},
		{"too many steps", strings.TrimSpace(strings.Repeat("1m ", MaxSteps+1))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSchedule(tc.input)
			if err == nil {
				t.Fatalf("ParseSchedule(%q) = %v, want error", tc.input, got)
			}
		})
	}
}

func TestParseErrorReportsPosition(t *testing.T) {
	_, err := ParseSchedule("1m 1h 30m")
	if err == nil {
		t.Fatal("expected an error")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Index != 3 {
		t.Fatalf("Index = %d, want 3", pe.Index)
	}
	if pe.Token != "30m" {
		t.Fatalf("Token = %q, want %q", pe.Token, "30m")
	}
}

func TestCanonicalPicksLargestExactUnit(t *testing.T) {
	cases := []struct {
		input Step
		want  string
	}{
		{Step{60, Minute}, "1h"},
		{Step{1440, Minute}, "1d"},
		{Step{43200, Minute}, "1mo"},
		{Step{525600, Minute}, "1y"},
		{Step{90, Minute}, "90m"},   // not an exact hour, stays in minutes
		{Step{120, Minute}, "2h"},   // exact hour, but not an exact day
		{Step{1, Day}, "1d"},        // already canonical
		{Step{24, Hour}, "1d"},      // rewritten to the larger unit
	}

	for _, tc := range cases {
		got := tc.input.Canonical().String()
		if got != tc.want {
			t.Errorf("Step%v.Canonical() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"1m", "1m"},
		{"60m", "1h"},
		{"10m 60m 1440m", "10m 1h 1d"},
		{"  10m    1h ", "10m 1h"},
		{"1mo", "1mo"},
	}

	for _, tc := range cases {
		got, err := Normalize(tc.input)
		if err != nil {
			t.Fatalf("Normalize(%q) returned error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	inputs := []string{"1m 10m 1d 3d 7d 14d", "90m 25h", "9999y"}
	for _, in := range inputs {
		once, err := Normalize(in)
		if err != nil {
			t.Fatalf("Normalize(%q) returned error: %v", in, err)
		}
		twice, err := Normalize(once)
		if err != nil {
			t.Fatalf("Normalize(%q) (second pass) returned error: %v", once, err)
		}
		if once != twice {
			t.Errorf("Normalize not idempotent: %q -> %q -> %q", in, once, twice)
		}
	}
}
