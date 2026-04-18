package main

import (
	"math/rand"

	"github.com/NimbleMarkets/ntcharts/sparkline"
)

func getVisualizer() string {
	sl := sparkline.New(132, 5)
	var s []float64

	for i := 0; i < 132; i++ {
		s = append(s, (rand.Float64() * 4.0))
	}
	sl.PushAll(s)
	sl.Draw()

	return sl.View()
}
