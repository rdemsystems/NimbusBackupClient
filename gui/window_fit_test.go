//go:build !service

package main

import "testing"

func TestFitWindowSize(t *testing.T) {
	cases := []struct {
		name         string
		w, h, sw, sh int
		wantW, wantH int
		wantChanged  bool
	}{
		{"fits a full HD screen", 1200, 840, 1920, 1080, 1200, 840, false},
		{"shrinks on a 1366x768 laptop", 1200, 840, 1366, 768, 1200, 691, true},
		{"shrinks both on a small screen", 1200, 840, 1024, 600, 921, 540, true},
		{"never below the minimum size", 1200, 840, 400, 300, 480, 360, true},
		{"unknown screen size", 1200, 840, 0, 0, 1200, 840, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, h, changed := fitWindowSize(c.w, c.h, c.sw, c.sh)
			if w != c.wantW || h != c.wantH || changed != c.wantChanged {
				t.Fatalf("fitWindowSize(%d, %d, %d, %d) = %d, %d, %v; want %d, %d, %v",
					c.w, c.h, c.sw, c.sh, w, h, changed, c.wantW, c.wantH, c.wantChanged)
			}
		})
	}
}
