// Package domain holds side-effect-free AI Core helpers shared by modules.
package domain

import "time"

// BusinessLocation returns the configured location or UTC. Retrieval and state
// code need a concrete *time.Location; a nil location would silently fall back to
// UTC at each call site, so it is normalized once here.
func BusinessLocation(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}
