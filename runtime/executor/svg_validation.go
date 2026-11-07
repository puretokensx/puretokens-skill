package main

import (
	"bytes"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

const (
	maxSVGBytes         = 8 << 20
	maxSVGDepth         = 128
	maxSVGTokens        = 200000
	maxSVGAttributes    = 64
	maxSVGAttributeSize = 64 << 10
	maxSVGStyleSize     = 256 << 10
	maxSVGExpandedNodes = 200000
	svgNamespace        = "http://www.w3.org/2000/svg"
	xlinkNamespace      = "http://www.w3.org/1999/xlink"
	xmlNamespace        = "http://www.w3.org/XML/1998/namespace"
)

var svgElements = svgNameSet(`svg g defs title desc style path rect circle ellipse line polyline polygon
text tspan textPath use symbol linearGradient radialGradient stop pattern clipPath mask marker
filter feBlend feColorMatrix feComponentTransfer feComposite feConvolveMatrix feDiffuseLighting
feDisplacementMap feDistantLight feDropShadow feFlood feFuncA feFuncB feFuncG feFuncR feGaussianBlur
feMerge feMergeNode feMorphology feOffset fePointLight feSpecularLighting feSpotLight feTile feTurbulence`)

var svgAttributes = svgNameSet(`id class style href version width height viewBox preserveAspectRatio
x y x1 y1 x2 y2 dx dy cx cy r rx ry d points pathLength transform
fill fill-opacity fill-rule stroke stroke-width stroke-opacity stroke-linecap stroke-linejoin
stroke-miterlimit stroke-dasharray stroke-dashoffset opacity color color-interpolation
color-interpolation-filters color-rendering shape-rendering text-rendering image-rendering
clip-path clip-rule mask filter marker-start marker-mid marker-end vector-effect paint-order
display visibility overflow
font-family font-size font-style font-weight font-stretch font-variant font-size-adjust
text-anchor text-decoration textLength lengthAdjust letter-spacing word-spacing
dominant-baseline alignment-baseline baseline-shift writing-mode direction unicode-bidi
rotate startOffset method spacing
gradientUnits gradientTransform spreadMethod offset stop-color stop-opacity fx fy fr
patternUnits patternContentUnits patternTransform clipPathUnits maskUnits maskContentUnits
markerUnits markerWidth markerHeight refX refY orient
filterUnits primitiveUnits in in2 result mode type values operator k1 k2 k3 k4
order kernelMatrix divisor bias targetX targetY edgeMode kernelUnitLength preserveAlpha
surfaceScale diffuseConstant specularConstant specularExponent limitingConeAngle
scale xChannelSelector yChannelSelector azimuth elevation pointsAtX pointsAtY pointsAtZ z
stdDeviation flood-color flood-opacity lighting-color radius baseFrequency numOctaves seed
stitchTiles tableValues slope intercept amplitude exponent`)

var svgCSSProperties = svgNameSet(`fill fill-opacity fill-rule stroke stroke-width stroke-opacity
stroke-linecap stroke-linejoin stroke-miterlimit stroke-dasharray stroke-dashoffset opacity color
color-interpolation color-interpolation-filters color-rendering shape-rendering text-rendering
image-rendering clip-path clip-rule mask filter marker-start marker-mid marker-end vector-effect
paint-order display visibility overflow stop-color stop-opacity flood-color flood-opacity lighting-color
font font-family font-size font-style font-weight font-stretch font-variant font-size-adjust
text-anchor text-decoration letter-spacing word-spacing dominant-baseline alignment-baseline
baseline-shift writing-mode direction unicode-bidi transform transform-origin`)

var svgFunctions = svgNameSet(`rgb rgba hsl hsla matrix translate translatex translatey scale scalex
scaley rotate skewx skewy`)

var (
	svgIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:-]{0,127}$`)
	svgFragment   = regexp.MustCompile(`^#[A-Za-z_][A-Za-z0-9_.:-]{0,127}$`)
	svgLocalURL   = regexp.MustCompile(`(?i)url\(\s*(?:"(#[A-Za-z_][A-Za-z0-9_.:-]{0,127})"|'(#[A-Za-z_][A-Za-z0-9_.:-]{0,127})'|(#[A-Za-z_][A-Za-z0-9_.:-]{0,127}))\s*\)`)
	svgFunction   = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_-]*)\s*\(`)
	svgSelector   = regexp.MustCompile(`^[A-Za-z0-9_#.* ,>+~\t\r\n-]+$`)
	svgXMLHeader  = regexp.MustCompile(`^version\s*=\s*(?:"1\.0"|'1\.0')(?:\s+encoding\s*=\s*(?:"(?i:UTF-8)"|'(?i:UTF-8)'))?(?:\s+standalone\s*=\s*(?:"(?:yes|no)"|'(?:yes|no)'))?\s*$`)
)

type svgNode struct {
	edges []int
	refs  []string
}

func svgNameSet(names string) map[string]bool {
	result := make(map[string]bool)
	for _, name := range strings.Fields(names) {
		result[name] = true
	}
	return result
}

// SVG remains original UTF-8 bytes. Validation permits an inert drawing subset,
// never scripts, animation, embedded documents, external resources or entities.
func validSVG(data []byte) bool {
	if len(data) == 0 || len(data) > maxSVGBytes {
		return false
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var nodes []svgNode
	var stack []int
	ids := make(map[string]int)
	tokens, styleNode := 0, -1
	var style strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false
		}
		tokens++
		if tokens > maxSVGTokens {
			return false
		}
		switch element := token.(type) {
		case xml.Directive:
			return false
		case xml.ProcInst:
			if tokens != 1 || element.Target != "xml" || !svgXMLHeader.Match(element.Inst) {
				return false
			}
		case xml.StartElement:
			if styleNode >= 0 || len(stack) >= maxSVGDepth || element.Name.Space != svgNamespace || !svgElements[element.Name.Local] || len(element.Attr) > maxSVGAttributes {
				return false
			}
			if len(stack) == 0 && (len(nodes) > 0 || element.Name.Local != "svg") {
				return false
			}
			index := len(nodes)
			nodes = append(nodes, svgNode{})
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				nodes[parent].edges = append(nodes[parent].edges, index)
			}
			stack = append(stack, index)
			attributes := make(map[xml.Name]bool)
			for _, attribute := range element.Attr {
				if attributes[attribute.Name] || len(attribute.Value) > maxSVGAttributeSize {
					return false
				}
				attributes[attribute.Name] = true
				refs, ok := svgAttribute(attribute)
				if !ok {
					return false
				}
				nodes[index].refs = append(nodes[index].refs, refs...)
				if attribute.Name.Space == "" && attribute.Name.Local == "id" {
					if _, duplicate := ids[attribute.Value]; duplicate || !svgIdentifier.MatchString(attribute.Value) {
						return false
					}
					ids[attribute.Value] = index
				}
			}
			if element.Name.Local == "style" {
				styleNode = index
				style.Reset()
			}
		case xml.EndElement:
			if len(stack) == 0 {
				return false
			}
			if styleNode == stack[len(stack)-1] {
				refs, ok := svgStyleSheet(style.String())
				if !ok {
					return false
				}
				nodes[styleNode].refs = append(nodes[styleNode].refs, refs...)
				styleNode = -1
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 && len(bytes.TrimSpace(element)) > 0 {
				return false
			}
			if styleNode >= 0 {
				if style.Len()+len(element) > maxSVGStyleSize {
					return false
				}
				style.Write(element)
			}
		}
	}
	return len(nodes) > 0 && len(stack) == 0 && validSVGReferences(nodes, ids)
}

func svgAttribute(attribute xml.Attr) ([]string, bool) {
	name, value := attribute.Name.Local, attribute.Value
	if attribute.Name.Space == "xmlns" || attribute.Name.Space == "" && name == "xmlns" {
		if attribute.Name.Space == "xmlns" && name == "xml" {
			return nil, value == xmlNamespace
		}
		if name == "xmlns" && attribute.Name.Space == "xmlns" || value == xmlNamespace {
			return nil, false
		}
		return nil, value == svgNamespace || value == xlinkNamespace
	}
	if attribute.Name.Space == xmlNamespace {
		return nil, name == "space" && (value == "default" || value == "preserve") || name == "lang" && svgPlainValue(value)
	}
	if attribute.Name.Space != "" && (attribute.Name.Space != xlinkNamespace || name != "href") || !svgAttributes[name] {
		return nil, false
	}
	if name == "href" {
		if !svgFragment.MatchString(value) {
			return nil, false
		}
		return []string{value[1:]}, true
	}
	if name == "style" {
		return svgStyleDeclarations(value)
	}
	return svgStaticValue(value)
}

func svgPlainValue(value string) bool {
	value = strings.ToLower(value)
	for _, denied := range []string{"url", "@", "\\", "/*", "*/", "javascript:", "data:", "http:", "https:", "file:", "//", "expression", "-moz-binding", "behavior", "<", ">", "{", "}"} {
		if strings.Contains(value, denied) {
			return false
		}
	}
	for _, character := range value {
		if character < 32 && character != '\n' && character != '\r' && character != '\t' || character == 127 {
			return false
		}
	}
	return true
}

func svgStaticValue(value string) ([]string, bool) {
	var refs []string
	value = svgLocalURL.ReplaceAllStringFunc(value, func(local string) string {
		groups := svgLocalURL.FindStringSubmatch(local)
		for _, group := range groups[1:] {
			if group != "" {
				refs = append(refs, group[1:])
				break
			}
		}
		return ""
	})
	if !svgPlainValue(value) {
		return nil, false
	}
	for _, function := range svgFunction.FindAllStringSubmatch(value, -1) {
		if !svgFunctions[strings.ToLower(function[1])] {
			return nil, false
		}
	}
	return refs, true
}

func svgStyleDeclarations(value string) ([]string, bool) {
	var refs []string
	for _, declaration := range strings.Split(value, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		name, content, found := strings.Cut(declaration, ":")
		if !found || !svgCSSProperties[strings.ToLower(strings.TrimSpace(name))] || strings.TrimSpace(content) == "" {
			return nil, false
		}
		local, ok := svgStaticValue(content)
		if !ok {
			return nil, false
		}
		refs = append(refs, local...)
	}
	return refs, true
}

func svgStyleSheet(value string) ([]string, bool) {
	var refs []string
	for strings.TrimSpace(value) != "" {
		selector, rest, found := strings.Cut(value, "{")
		if !found || strings.TrimSpace(selector) == "" || !svgSelector.MatchString(selector) {
			return nil, false
		}
		declarations, rest, found := strings.Cut(rest, "}")
		if !found {
			return nil, false
		}
		local, ok := svgStyleDeclarations(declarations)
		if !ok {
			return nil, false
		}
		refs = append(refs, local...)
		value = rest
	}
	return refs, true
}

// Containment and local references share one bounded DAG. This also rejects
// recursive use/gradient chains and exponential local expansion.
func validSVGReferences(nodes []svgNode, ids map[string]int) bool {
	for index := range nodes {
		for _, id := range nodes[index].refs {
			target, exists := ids[id]
			if !exists {
				return false
			}
			nodes[index].edges = append(nodes[index].edges, target)
		}
	}
	states, expanded, heights := make([]uint8, len(nodes)), make([]int, len(nodes)), make([]int, len(nodes))
	var visit func(int, int) bool
	visit = func(index, depth int) bool {
		if depth > maxSVGDepth || states[index] == 1 {
			return false
		}
		if states[index] == 2 {
			return heights[index] <= maxSVGDepth-depth+1
		}
		states[index], expanded[index], heights[index] = 1, 1, 1
		for _, target := range nodes[index].edges {
			if !visit(target, depth+1) || expanded[target] > maxSVGExpandedNodes-expanded[index] {
				return false
			}
			expanded[index] += expanded[target]
			if heights[target]+1 > heights[index] {
				heights[index] = heights[target] + 1
			}
		}
		states[index] = 2
		return true
	}
	return visit(0, 1)
}
