package repair

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/analyzer"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func TestCasePreservesMarkup(t *testing.T) {
	for _, tc := range []struct{ tool, in, want string }{
		{"upper", `<v Alice><c.Green><font color="#aaBBcc">Hello &amp; bye</font></c></v>`, `<v Alice><c.Green><font color="#aaBBcc">HELLO &amp; BYE</font></c></v>`},
		{"lower", `{\pos(10,20)\t(0,100,\fscx120)\i1}HELLO\NTHERE\hFRIEND`, `{\pos(10,20)\t(0,100,\fscx120)\i1}hello\Nthere\hfriend`},
		{"sentence", `<c.dot>hELLO <b>WORLD</b>! <lang en-US>gOOD</lang> DAY. {\fnArial\i1}nEXT`, `<c.dot>Hello <b>world</b>! <lang en-US>Good</lang> day. {\fnArial\i1}Next`},
		{"html", `Hello<br>World`, "Hello\nWorld"},
		{"quotes", `<font color="red">"Hello"</font>`, `<font color="red">“Hello”</font>`},
	} {
		d := model.Document{Entries: []model.Entry{{Text: tc.in}}}
		if err := Text(&d, Operation{Tool: tc.tool}); err != nil {
			t.Fatal(err)
		}
		if got := d.Entries[0].Text; got != tc.want {
			t.Errorf("%s => %q; want %q", tc.tool, got, tc.want)
		}
	}
	if got := analyzer.Plain("Hello<br>World"); got != "Hello\nWorld" {
		t.Fatal(got)
	}
}
