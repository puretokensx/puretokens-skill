package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const staticSVGFixture = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="120" height="80" viewBox="0 0 120 80"><title>Static &amp; original</title><defs><linearGradient id="paint"><stop offset="0" stop-color="#fff"/><stop offset="1" stop-color="#123456"/></linearGradient><path id="curve" d="M0 10h100"/><clipPath id="crop"><rect width="120" height="80"/></clipPath></defs><style>.shape { fill: url('#paint'); stroke: rgb(12, 30, 40); stroke-width: 2 }</style><g clip-path="url(#crop)"><rect class="shape" width="120" height="80"/><circle cx="10" cy="10" r="5"/><ellipse cx="30" cy="10" rx="10" ry="5"/><line x1="0" y1="0" x2="10" y2="10"/><polyline points="0,0 4,5 8,0"/><polygon points="0,0 2,4 4,0"/><text xml:space="preserve" style="font-family: 'Noto Sans'; font-size: 12px"><tspan>Hello &amp; world</tspan><textPath xlink:href="#curve">Local text</textPath></text><use href="#curve" transform="translate(0 30)"/></g></svg>`

func svgFileAccepts(t *testing.T, data []byte) bool {
	t.Helper()
	path := filepath.Join(t.TempDir(), "original.svg")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	accepted := validMediaFile(path, "image/svg+xml")
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, data) {
		t.Fatal("validation changed original SVG bytes")
	}
	return accepted
}

func TestSVGStaticOutputPreservesOriginalBytes(t *testing.T) {
	for _, payload := range []string{
		staticSVGFixture,
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + staticSVGFixture,
		"\xef\xbb\xbf" + staticSVGFixture,
		`<s:svg xmlns:s="http://www.w3.org/2000/svg"><s:text>Read https://example.test and &lt;script&gt; as text.</s:text></s:svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><!-- kept --><style><![CDATA[path { fill: #aabbcc; opacity: .5 }]]></style><path d="M0 0L10 10Z"/></svg>`,
	} {
		if !svgFileAccepts(t, []byte(payload)) {
			t.Error("valid static SVG rejected")
		}
	}
}

func TestSVGRejectsActiveContentAndExternalReferences(t *testing.T) {
	for _, content := range []string{
		`<script>alert(1)</script>`,
		`<foreignObject><div xmlns="http://www.w3.org/1999/xhtml">html</div></foreignObject>`,
		`<a href="javascript:alert(1)"><text>link</text></a>`,
		`<image href="data:image/png;base64,abcd"/>`,
		`<animate attributeName="href" values="javascript:alert(1)"/>`,
		`<set attributeName="onload" to="alert(1)"/>`,
		`<rect onload="alert(1)"/>`,
		`<rect ONLOAD="alert(1)"/>`,
		`<rect xml:base="https://example.test/"/>`,
		`<use href="https://example.test/a.svg#shape"/>`,
		`<use href="//example.test/a.svg#shape"/>`,
		`<use href="data:image/svg+xml,test"/>`,
		`<use href="file:///etc/passwd"/>`,
		`<use href="&#x6a;avascript:alert(1)"/>`,
		`<use xmlns:xlink="http://www.w3.org/1999/xlink" xlink:href="../image.svg#shape"/>`,
		`<rect fill="url(https://example.test/paint)"/>`,
		`<rect fill="url(&#x2f;&#x2f;example.test/paint)"/>`,
		`<rect style="fill: url(data:image/png,test)"/>`,
		`<rect style="width: expression (alert(1))"/>`,
		`<rect style="behavior: url(#paint)"/>`,
		`<rect style="-moz-binding: url(#paint)"/>`,
		`<rect style="--remote: 'https://example.test/a'; fill: var(--remote)"/>`,
		`<style>@import 'https://example.test/a.css';</style>`,
		`<style>@font-face { font-family: external; src: url(https://example.test/a.woff) }</style>`,
		`<style>.x {fill:ur<![CDATA[l]]>(//example.test/x)}</style>`,
		`<style>@im<!--split-->port 'x.css';</style>`,
		`<style>.x { fill: u\72l(https://example.test/x) }</style>`,
		`<style>.x { fill: u/**/rl(https://example.test/x) }</style>`,
		`<style>.x { fill: url('#paint\")') }</style>`,
		`<style><g/></style>`,
		`<rect xmlns="http://www.w3.org/1999/xhtml"/>`,
		`<rect xmlns:x="https://example.test/ns" x:href="#paint"/>`,
		`<rect xmlns:xlink="http://www.w3.org/1999/xlink" xlink:onload="alert(1)"/>`,
		`<?xml-stylesheet href="https://example.test/a.css"?>`,
		`<?xml version="1.0"?>`,
	} {
		t.Run(content, func(t *testing.T) {
			payload := `<svg xmlns="http://www.w3.org/2000/svg">` + content + `</svg>`
			if svgFileAccepts(t, []byte(payload)) {
				t.Fatal("active or external SVG content accepted")
			}
		})
	}
}

func TestSVGRejectsMalformedOrUnboundedDocuments(t *testing.T) {
	for _, payload := range []string{
		"", `<svg/>`, `<html/>`, staticSVGFixture + staticSVGFixture, staticSVGFixture + "tail",
		`<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg xmlns="http://www.w3.org/2000/svg">&x;</svg>`,
		`<!DOCTYPE svg><svg xmlns="http://www.w3.org/2000/svg"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><text>&unknown;</text></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><g></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" id="a" id="b"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><path id="a"/><path id="a"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><use href="#missing"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><g id="loop"><use href="#loop"/></g></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><defs><linearGradient id="a" href="#b"/><linearGradient id="b" href="#a"/></defs></svg>`,
		`<?xml version="1.0"?><?xml version="1.0"?>` + staticSVGFixture,
		`<?xml anything="true"?>` + staticSVGFixture,
		` <?xml version="1.0"?>` + staticSVGFixture,
		`<?xml version="1.0" encoding="ISO-8859-1"?>` + staticSVGFixture,
		`<svg xmlns="http://www.w3.org/2000/svg"><text>` + "\xff" + `</text></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg">` + strings.Repeat("<g>", 128) + strings.Repeat("</g>", 128) + `</svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg">` + strings.Repeat("<path/>", 100000) + `</svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><text>` + strings.Repeat("x", 8<<20) + `</text></svg>`,
	} {
		if svgFileAccepts(t, []byte(payload)) {
			t.Errorf("malformed or unbounded SVG accepted: %.160s", payload)
		}
	}
}

func TestSVGOutputRequiresExactReviewedModelAndExplicitFormat(t *testing.T) {
	for _, tc := range []struct {
		model, kind, format, mime string
		allowed                   bool
	}{
		{"gpt-image-2.6", "image", "svg", "image/svg+xml", true},
		{"gpt-image-2.6-flare", "image", "svg", "image/svg+xml", true},
		{"gpt-image-2.6-sunburst", "image", "svg", "image/svg+xml", true},
		{"gpt-image-2.6", "image", "", "image/svg+xml", false},
		{"gpt-image-2.6", "image", "png", "image/svg+xml", false},
		{"gpt-image-2.6", "image", "svg", "image/png", false},
		{"", "image", "svg", "image/svg+xml", false},
		{"gpt-image-2", "image", "svg", "image/svg+xml", false},
		{"gpt-image-2.6-other", "image", "svg", "image/svg+xml", false},
		{"gpt-image-2.6", "video", "svg", "image/svg+xml", false},
		{"gpt-image-2.6", "audio_task", "svg", "image/svg+xml", false},
	} {
		request := taskRequest{Kind: tc.kind, Model: tc.model, Parameters: map[string]any{"output_format": tc.format}}
		if got := taskImageFormatMatches(request, tc.mime); got != tc.allowed {
			t.Errorf("model=%q kind=%q format=%q MIME=%q: allowed=%v, want %v", tc.model, tc.kind, tc.format, tc.mime, got, tc.allowed)
		}
	}
}

func TestSVGDoesNotBroadenRasterReferenceValidation(t *testing.T) {
	if validMediaPrefix([]byte(staticSVGFixture), "image/svg+xml") {
		t.Fatal("raster prefix validator accepted SVG")
	}
}

func TestSVGRejectsReservedNamespaceRebinding(t *testing.T) {
	for _, declaration := range []string{
		`xmlns:xml="http://www.w3.org/2000/svg"`,
		`xmlns:xmlns="http://www.w3.org/2000/svg"`,
		`xmlns:alias="http://www.w3.org/XML/1998/namespace"`,
	} {
		payload := `<svg xmlns="http://www.w3.org/2000/svg" ` + declaration + `><path d="M0 0L1 1"/></svg>`
		if svgFileAccepts(t, []byte(payload)) {
			t.Errorf("reserved XML namespace binding accepted: %s", declaration)
		}
	}
}

func TestSVGRejectsDeepAndExponentialLocalExpansion(t *testing.T) {
	for _, exponential := range []bool{false, true} {
		var content strings.Builder
		content.WriteString(`<svg xmlns="http://www.w3.org/2000/svg"><defs><g id="n0"><path d="M0 0L1 1"/></g>`)
		count := 128
		if exponential {
			count = 20
		}
		for index := 1; index <= count; index++ {
			fmt.Fprintf(&content, `<g id="n%d"><use href="#n%d"/>`, index, index-1)
			if exponential {
				fmt.Fprintf(&content, `<use href="#n%d"/>`, index-1)
			}
			content.WriteString("</g>")
		}
		fmt.Fprintf(&content, `</defs><use href="#n%d"/></svg>`, count)
		if svgFileAccepts(t, []byte(content.String())) {
			t.Errorf("unbounded local expansion accepted (exponential=%v)", exponential)
		}
	}
}

func TestSVGBoundsAttributesAndCSS(t *testing.T) {
	for _, content := range []string{
		`<path d="` + strings.Repeat("M0 0", 20000) + `"/>`,
		`<style>` + strings.Repeat(" ", 262145) + `</style>`,
	} {
		payload := `<svg xmlns="http://www.w3.org/2000/svg">` + content + `</svg>`
		if svgFileAccepts(t, []byte(payload)) {
			t.Fatal("oversized SVG attribute or stylesheet accepted")
		}
	}
}
