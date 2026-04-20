package main

import (
	"math"
	"math/cmplx"
	"sync"

	"github.com/10d9e/gofft"
	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/faiface/beep"
)

const (
	fftSize   = 1024
	visBars   = 132
	visHeight = 5
)

var (
	fftPlanner = gofft.NewPlanner()
	fftPlan    = fftPlanner.PlanForward(fftSize)

	sampleBufMu sync.Mutex
	sampleBuf   [fftSize]float64
	sampleBufW  int

	hannWindow = buildHannWindow(fftSize)
)

func buildHannWindow(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

// tapStreamer wraps a streamer and captures mono-mixed samples into a ring
// buffer so the visualizer can run FFT on whatever is actually being played.
type tapStreamer struct {
	wrapped beep.Streamer
}

func (t *tapStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := t.wrapped.Stream(samples)
	if n > 0 {
		sampleBufMu.Lock()
		w := sampleBufW
		for i := 0; i < n; i++ {
			sampleBuf[w] = (samples[i][0] + samples[i][1]) * 0.5
			w++
			if w == fftSize {
				w = 0
			}
		}
		sampleBufW = w
		sampleBufMu.Unlock()
	}
	return n, ok
}

func (t *tapStreamer) Err() error { return t.wrapped.Err() }

func snapshotSamples() []complex128 {
	out := make([]complex128, fftSize)
	sampleBufMu.Lock()
	w := sampleBufW
	for i := 0; i < fftSize; i++ {
		out[i] = complex(sampleBuf[(w+i)%fftSize]*hannWindow[i], 0)
	}
	sampleBufMu.Unlock()
	return out
}

func getVisualizer() string {
	buf := snapshotSamples()
	fftPlan.Process(buf)

	halfN := fftSize / 2
	mags := make([]float64, halfN)
	for i := 0; i < halfN; i++ {
		mags[i] = cmplx.Abs(buf[i])
	}

	// Bucket FFT bins into visBars using a log-frequency scale so bass
	// doesn't collapse into a single bar.
	bars := make([]float64, visBars)
	logMin := math.Log(1.0)
	logMax := math.Log(float64(halfN))
	for i := 0; i < visBars; i++ {
		lo := int(math.Exp(logMin + (logMax-logMin)*float64(i)/float64(visBars)))
		hi := int(math.Exp(logMin + (logMax-logMin)*float64(i+1)/float64(visBars)))
		if hi <= lo {
			hi = lo + 1
		}
		if hi > halfN {
			hi = halfN
		}
		var peak float64
		for j := lo; j < hi; j++ {
			if mags[j] > peak {
				peak = mags[j]
			}
		}
		bars[i] = math.Log1p(peak)
	}

	sl := sparkline.New(visBars, visHeight)
	sl.PushAll(bars)
	sl.Draw()
	return sl.View()
}
