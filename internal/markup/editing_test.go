package markup

import (
	"strings"
	"testing"
)

func TestSafeSplit(t *testing.T) {
	for _, tc := range []struct{ name, input, left, right string }{
		{"normal", "one two three four", "one two", "three four"},
		{"italic", "<i>one two three four</i>", "<i>one two</i>", "<i>three four</i>"},
		{"bold", "<b>one two three four</b>", "<b>one two</b>", "<b>three four</b>"},
		{"underline", "<u>one two three four</u>", "<u>one two</u>", "<u>three four</u>"},
		{"nested", "<b><i>one two three four</i></b>", "<b><i>one two</i></b>", "<b><i>three four</i></b>"},
		{"attributes", "<font color=\"light blue\">one two three four</font>", "<font color=\"light blue\">one two</font>", "<font color=\"light blue\">three four</font>"},
		{"VTT class", "<c.blue>one two three four</c>", "<c.blue>one two</c>", "<c.blue>three four</c>"},
		{"multiline", "one two\nthree four", "one two", "three four"},
		{"CRLF", "one two\r\nthree four", "one two", "three four"},
		{"Unicode", "Bună țară 👩‍💻 frumoasă", "Bună țară", "👩‍💻 frumoasă"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b, err := Split(tc.input)
			if err != nil || a != tc.left || b != tc.right || Suspicious(a) || Suspicious(b) {
				t.Fatalf("%q / %q: %v", a, b, err)
			}
		})
	}
}

func TestSplitRefusesUnsafeFormatting(t *testing.T) {
	for _, s := range []string{"", "one", "<i></i>", "<i>one two", "<b><i>one two</b></i>", `{\i1}one two`, "<ruby>字<rt>zi</rt></ruby> two", "one <00:01.000>two", "one<br>two"} {
		if _, _, err := Split(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestSafeWrap(t *testing.T) {
	for _, tc := range []struct{ name, input, expected string }{
		{"normal", "one two three four", "one two\nthree four"},
		{"italic", "<i>one two three four</i>", "<i>one two\nthree four</i>"},
		{"bold", "<b>one two three four</b>", "<b>one two\nthree four</b>"},
		{"nested", "<b><i>one two three four</i></b>", "<b><i>one two\nthree four</i></b>"},
		{"attributes", "<font color=\"light blue\">one two three four</font>", "<font color=\"light blue\">one two\nthree four</font>"},
		{"multiline", "one two\nthree four", "one two\nthree four"},
		{"CRLF", "one two\r\nthree four five", "one two\r\nthree four\r\nfive"},
		{"empty", "", ""},
		{"Unicode", "Bună țară 🙂 lume", "Bună țară 🙂\nlume"},
		{"ASS", "{\\i1}one two three four{\\i0}", "{\\i1}one two\nthree four{\\i0}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			width := 10
			if tc.name == "Unicode" {
				width = 11
			}
			s, err := Wrap(tc.input, width, 4)
			if err != nil || s != tc.expected {
				t.Fatalf("%q: %v", s, err)
			}
		})
	}
}

func TestWrapLimits(t *testing.T) {
	for _, s := range []string{"<i>abcdefghijk</i>", "one two three four five six", "<i>bad markup"} {
		if _, err := Wrap(s, 10, 1); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	// Attribute bytes remain untouched, including multiple spaces.
	s, err := Wrap(`<font color="light  blue">one two three</font>`, 10, 3)
	if err != nil || !strings.Contains(s, `color="light  blue"`) {
		t.Fatal(s, err)
	}
}
