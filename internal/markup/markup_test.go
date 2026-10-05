package markup

import "testing"

func TestValidSubtitleMarkup(t *testing.T) {
	for _, s := range []string{
		`<c.green.bold>Hello</c>`, `<v.narrator Alice Smith>Hello</v>`, `<v Alice>Hello`,
		`<lang en-US><b.title>Hello</b></lang>`, `<ruby>漢<rt>kan</rt></ruby>`, `<ruby>漢<rt>kan</ruby>`,
		`<font color="red" title="a > b">Hello</font>`, `<b class="X">Hello</b>`,
		`Hello<br>World<br/>Again<BR />End`, `<00:01.000>Hello`, `{\pos(10,20)\i1}Hello`,
	} {
		if Suspicious(s) {
			t.Errorf("valid markup warned: %q", s)
		}
		if got := Clean(s); got != s {
			t.Errorf("valid markup modified: %q => %q", s, got)
		}
	}
}

func TestConservativeCleanup(t *testing.T) {
	for _, s := range []string{`<custom data-x="Y">Hello</custom>`, `<unfinished Hello`, `Hello</other>`} {
		if !Suspicious(s) {
			t.Errorf("unknown/malformed not warned: %q", s)
		}
		if got := Clean(s); got != s {
			t.Errorf("unknown syntax damaged: %q", got)
		}
	}
	if got := Clean("<b>Hello<br>World"); got != "Hello<br>World" {
		t.Fatal(got)
	}
	if got := StripHTML(`Hello<br>World<BR />Again<br/>End`); got != "Hello\nWorld\nAgain\nEnd" {
		t.Fatal(got)
	}
}
