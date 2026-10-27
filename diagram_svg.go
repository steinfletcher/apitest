package apitest

import (
	"fmt"
	htmlTemplate "html/template"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// layout of the sequence diagram, in pixels. Text is monospace, so widths are estimated from character counts.
const (
	diagramMarginX    = 16.0
	diagramMarginY    = 12.0
	participantH      = 34.0
	participantPadX   = 16.0
	participantMinW   = 72.0
	participantCharW  = 7.8 // 13px monospace
	participantMinGap = 48.0
	labelCharW        = 7.2 // 12px monospace
	labelPadX         = 10.0
	labelGap          = 24.0 // space either side of a label between two lifelines
	rowH              = 46.0
	firstRowOffset    = 36.0
	selfLoopW         = 30.0
	selfLoopH         = 16.0
	arrowHeadLen      = 9.0
	arrowHeadHalf     = 4.5
	maxDiagramLabel   = 60
)

type diagramLayout struct {
	index  map[string]int
	x      []float64
	boxW   []float64
	width  float64
	height float64
}

// renderDiagram draws the participants and events as an inline SVG sequence diagram. Each arrow links to its
// entry in the event log, so the diagram is navigable without JavaScript.
func renderDiagram(participants []string, events []reportEvent) htmlTemplate.HTML {
	l := layoutDiagram(participants, events)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" class="seq" viewBox="0 0 %s %s" width="%s" height="%s" role="img" aria-label="Sequence diagram of %d events between %d participants">`,
		num(l.width), num(l.height), num(l.width), num(l.height), len(events), len(participants))

	bottomY := l.height - diagramMarginY - participantH
	for i, name := range participants {
		x := l.x[i]
		w := l.boxW[i]
		fmt.Fprintf(&b, `<g class="participant" data-participant="%s">`, esc(name))
		fmt.Fprintf(&b, `<line class="lifeline" x1="%s" y1="%s" x2="%s" y2="%s"/>`, num(x), num(diagramMarginY+participantH), num(x), num(bottomY))
		for _, y := range []float64{diagramMarginY, bottomY} {
			fmt.Fprintf(&b, `<rect class="box" x="%s" y="%s" width="%s" height="%s" rx="9"/>`, num(x-w/2), num(y), num(w), num(participantH))
			fmt.Fprintf(&b, `<text class="name" x="%s" y="%s" text-anchor="middle" dominant-baseline="central">%s</text>`, num(x), num(y+participantH/2), esc(displayName(name)))
		}
		b.WriteString(`</g>`)
	}

	for i, e := range events {
		y := diagramMarginY + participantH + firstRowOffset + float64(i)*rowH
		from, to := l.x[l.index[e.Source]], l.x[l.index[e.Target]]
		label := truncate(e.Label, maxDiagramLabel)
		classes := "msg " + e.Kind
		if e.StatusClass != "" {
			classes += " s-" + e.StatusClass
		}
		if e.Unanswered {
			classes += " unanswered"
		}

		fmt.Fprintf(&b, `<a href="#event-%d" class="%s" data-event="%d" data-source="%s" data-target="%s">`, e.Number, classes, e.Number, esc(e.Source), esc(e.Target))
		fmt.Fprintf(&b, `<title>%d. %s → %s: %s</title>`, e.Number, esc(displayName(e.Source)), esc(displayName(e.Target)), esc(e.Label))

		labelW := labelWidth(e.Number, label)
		if e.Source == e.Target {
			// a message a participant sends to itself is drawn as a loop to the right of its lifeline
			fmt.Fprintf(&b, `<path class="hit" d="M%s %sh%sv%sh-%s"/>`, num(from), num(y), num(selfLoopW), num(selfLoopH), num(selfLoopW))
			fmt.Fprintf(&b, `<path class="line" d="M%s %sh%sv%sh-%s"/>`, num(from), num(y), num(selfLoopW), num(selfLoopH), num(selfLoopW))
			writeArrowHead(&b, e.Kind, from, y+selfLoopH, -1)
			lx := from + selfLoopW + 8
			fmt.Fprintf(&b, `<rect class="label-bg" x="%s" y="%s" width="%s" height="20" rx="6"/>`, num(lx), num(y-2), num(labelW))
			writeLabel(&b, e.Number, label, lx+labelPadX, y+8, "start")
		} else {
			dir := 1.0
			if to < from {
				dir = -1
			}
			fmt.Fprintf(&b, `<line class="hit" x1="%s" y1="%s" x2="%s" y2="%s"/>`, num(from), num(y), num(to), num(y))
			fmt.Fprintf(&b, `<line class="line" x1="%s" y1="%s" x2="%s" y2="%s"/>`, num(from), num(y), num(to-dir*arrowHeadLen*0.6), num(y))
			writeArrowHead(&b, e.Kind, to, y, dir)
			if e.Unanswered {
				fmt.Fprintf(&b, `<g class="no-reply"><circle cx="%s" cy="%s" r="7"/><path d="M%s %sl6 6M%s %sl-6 6"/></g>`,
					num(to+dir*12), num(y), num(to+dir*12-3), num(y-3), num(to+dir*12+3), num(y-3))
			}
			mid := (from + to) / 2
			fmt.Fprintf(&b, `<rect class="label-bg" x="%s" y="%s" width="%s" height="20" rx="6"/>`, num(mid-labelW/2), num(y-26), num(labelW))
			writeLabel(&b, e.Number, label, mid, y-16, "middle")
		}
		b.WriteString(`</a>`)
	}

	b.WriteString(`</svg>`)
	return htmlTemplate.HTML(b.String())
}

// layoutDiagram positions the participants so that every label fits between the lifelines it connects
func layoutDiagram(participants []string, events []reportEvent) diagramLayout {
	l := diagramLayout{index: map[string]int{}, x: make([]float64, len(participants)), boxW: make([]float64, len(participants))}
	for i, name := range participants {
		l.index[name] = i
		l.boxW[i] = max(participantMinW, float64(utf8.RuneCountInString(displayName(name)))*participantCharW+2*participantPadX)
		if i == 0 {
			l.x[i] = diagramMarginX + l.boxW[i]/2
		} else {
			l.x[i] = l.x[i-1] + l.boxW[i-1]/2 + participantMinGap + l.boxW[i]/2
		}
	}

	type span struct {
		lo, hi int
		need   float64
	}
	var spans []span
	rightOverhang := 0.0
	for _, e := range events {
		a, b := l.index[e.Source], l.index[e.Target]
		w := labelWidth(e.Number, truncate(e.Label, maxDiagramLabel))
		if a == b {
			need := selfLoopW + 8 + w + labelGap/2
			if a == len(participants)-1 {
				rightOverhang = max(rightOverhang, need)
			} else {
				spans = append(spans, span{lo: a, hi: a + 1, need: need + l.boxW[a+1]/2})
			}
			continue
		}
		lo, hi := min(a, b), max(a, b)
		spans = append(spans, span{lo: lo, hi: hi, need: w + 2*labelGap})
	}
	// widen the narrowest spans first, pushing the participants to their right along
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].hi-spans[i].lo < spans[j].hi-spans[j].lo })
	for _, s := range spans {
		if gap := l.x[s.hi] - l.x[s.lo]; gap < s.need {
			for k := s.hi; k < len(l.x); k++ {
				l.x[k] += s.need - gap
			}
		}
	}

	last := len(participants) - 1
	l.width = l.x[last] + max(l.boxW[last]/2, rightOverhang) + diagramMarginX
	l.height = 2*diagramMarginY + 2*participantH + firstRowOffset + float64(len(events))*rowH
	return l
}

func writeArrowHead(b *strings.Builder, kind string, x, y, dir float64) {
	baseX := x - dir*arrowHeadLen
	if kind == "request" {
		fmt.Fprintf(b, `<path class="head" d="M%s %sL%s %sL%s %sZ"/>`, num(x), num(y), num(baseX), num(y-arrowHeadHalf), num(baseX), num(y+arrowHeadHalf))
		return
	}
	fmt.Fprintf(b, `<path class="head open" d="M%s %sL%s %sL%s %s"/>`, num(baseX), num(y-arrowHeadHalf), num(x), num(y), num(baseX), num(y+arrowHeadHalf))
}

func writeLabel(b *strings.Builder, n int, label string, x, y float64, anchor string) {
	fmt.Fprintf(b, `<text class="label" x="%s" y="%s" text-anchor="%s" dominant-baseline="central"><tspan class="num">%d</tspan> %s</text>`,
		num(x), num(y), anchor, n, esc(label))
}

func labelWidth(n int, label string) float64 {
	return float64(len(strconv.Itoa(n))+1+utf8.RuneCountInString(label))*labelCharW + 2*labelPadX
}

func displayName(name string) string {
	if name == "" {
		return "?"
	}
	return name
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func esc(s string) string {
	return htmlTemplate.HTMLEscapeString(s)
}

func num(f float64) string {
	return strconv.FormatFloat(math.Round(f*10)/10, 'f', -1, 64)
}
