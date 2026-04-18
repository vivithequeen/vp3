package main

import (
	"github.com/NimbleMarkets/ntcharts/sparkline"
)

func getVisualizer() string {
	sl := sparkline.New(10, 5)
	sl.PushAll([]float64{7.81, 3.82, 8.39, 2.06, 4.19, 4.34, 6.83, 2.51, 9.21, 1.3})
	sl.Draw()

	return sl.View()
}
