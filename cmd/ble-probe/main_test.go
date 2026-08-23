package main

import (
	"flag"
	"testing"
)

func TestParseAnyOrder(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"flags after positional", []string{"8614", "-wifi", "-speed"}},
		{"flags before positional", []string{"-wifi", "-speed", "8614"}},
		{"flags around positional", []string{"-wifi", "8614", "-speed"}},
	}
	for _, c := range cases {
		fs := flag.NewFlagSet("validate", flag.ContinueOnError)
		wifi := fs.Bool("wifi", false, "")
		speed := fs.Bool("speed", false, "")
		positional := parseAnyOrder(fs, c.args)
		if len(positional) != 1 || positional[0] != "8614" {
			t.Errorf("%s: positional = %v", c.name, positional)
		}
		if !*wifi || !*speed {
			t.Errorf("%s: wifi=%v speed=%v, want both true", c.name, *wifi, *speed)
		}
	}
}
