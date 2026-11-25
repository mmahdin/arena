package trace

import (
	"context"
	"encoding/xml"
	"strings"
)

// EstimateSections describes disjoint top-level rendered XML sections where
// those exist. They are an alternative breakdown, never added to block totals.
// Parsing failures do not affect inference or the top-level block estimate.
func EstimateSections(ctx context.Context, source, text string) {
	if !Enabled(ctx) {
		return
	}
	decoder := xml.NewDecoder(strings.NewReader(text))
	depth := 0
	start := int64(0)
	name := ""
	for {
		before := decoder.InputOffset()
		token, err := decoder.Token()
		if err != nil {
			return
		}
		switch v := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				start = before
				name = v.Name.Local
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				end := decoder.InputOffset()
				EstimateText(ctx, source+"/section/"+name, text[start:end])
			}
		}
	}
}
