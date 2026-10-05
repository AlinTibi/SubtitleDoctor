package repair

import "testing"

func TestSafeTags(t *testing.T) {
	for _, tc := range []struct{ in, out string }{{"<b>Hello</b>", "<b>Hello</b>"}, {"<b>Hello", "Hello"}, {"Hello</i>", "Hello"}, {"<script>Hello</script>", "<script>Hello</script>"}, {"< >Hello", "< >Hello"}, {"<b><i>Hello</i>", "<i>Hello</i>"}} {
		if got := CleanHTML(tc.in); got != tc.out {
			t.Fatalf("%q -> %q; want %q", tc.in, got, tc.out)
		}
	}
}
