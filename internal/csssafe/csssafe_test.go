package csssafe

import "testing"

func TestColor(t *testing.T) {
	for _, c := range []string{
		"#c00", "#cc0000", "#cc0000ff", "rgb(255,0,0)",
		"rgba(255, 0, 0, .5)", "rgb(100%, 0%, 50%)", "rgba(0,0,0,100%)",
		"red", "transparent",
	} {
		if got := Color(c); got != c {
			t.Fatalf("Color(%q) = %q", c, got)
		}
	}
	for _, c := range []string{
		"red;background:url(javascript:1)", "#c00;body{}", "url(x)", "expression(x)", "нечто",
		"rgb(,)", "rgba(1)", "rgb(1,2)", "rgb(1,2,3,4)", "rgba(1,2,3)",
		"rgb(256,0,0)", "rgb(0,101%,0)", "rgba(0,0,0,1.01)",
		"rgba(0,0,0,101%)", "rgb(1..2,0,0)",
	} {
		if got := Color(c); got != "" {
			t.Fatalf("Color(%q) = %q, want empty", c, got)
		}
	}
}

func TestLength(t *testing.T) {
	for _, v := range []string{"0", "10px", "12.5pt", "100%", "auto"} {
		if got := Length(v); got != v {
			t.Fatalf("Length(%q) = %q", v, got)
		}
	}
	for _, v := range []string{`10px;color:red`, `url(x)`, `calc(100%)`, `auto;background:red`} {
		if got := Length(v); got != "" {
			t.Fatalf("Length(%q) = %q, want empty", v, got)
		}
	}
}

func TestFontFamily(t *testing.T) {
	if got := FontFamily(`Arial";color:red`); got != "Arialcolor:red" {
		t.Fatalf("FontFamily stripped to %q", got)
	}
}

func TestTextAlign(t *testing.T) {
	if got := TextAlign(" Center "); got != "center" {
		t.Fatalf("TextAlign = %q", got)
	}
	if got := TextAlign("left;position:absolute"); got != "" {
		t.Fatalf("TextAlign injected = %q", got)
	}
}
