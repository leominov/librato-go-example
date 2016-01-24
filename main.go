package main

import (
	"os"
	"runtime"
	"time"

	"github.com/mihasya/go-metrics-librato"
	"github.com/rcrowley/go-metrics"
)

type App struct {
	LibratoEmail  string
	LibratoToken  string
	LibratoSource string
}

func ReportMemstatsMetrics() {
	memStats := &runtime.MemStats{}
	lastSampleTime := time.Now()
	var lastPauseNs uint64
	var lastNumGC uint64
	var runCount int

	sleep := 10 * time.Second

	for {
		runCount += 1
		runtime.ReadMemStats(memStats)

		now := time.Now()

		metrics.GetOrRegisterGauge("golang.run.count", metrics.DefaultRegistry).Update(int64(runCount))
		metrics.GetOrRegisterGauge("golang.goroutines", metrics.DefaultRegistry).Update(int64(runtime.NumGoroutine()))
		metrics.GetOrRegisterGauge("golang.memory.allocated", metrics.DefaultRegistry).Update(int64(memStats.Alloc))
		metrics.GetOrRegisterGauge("golang.memory.mallocs", metrics.DefaultRegistry).Update(int64(memStats.Mallocs))
		metrics.GetOrRegisterGauge("golang.memory.frees", metrics.DefaultRegistry).Update(int64(memStats.Frees))
		metrics.GetOrRegisterGauge("golang.memory.gc.total_pause", metrics.DefaultRegistry).Update(int64(memStats.PauseTotalNs))
		metrics.GetOrRegisterGauge("golang.memory.gc.heap", metrics.DefaultRegistry).Update(int64(memStats.HeapAlloc))
		metrics.GetOrRegisterGauge("golang.memory.gc.stack", metrics.DefaultRegistry).Update(int64(memStats.StackInuse))

		if lastPauseNs > 0 {
			pauseSinceLastSample := memStats.PauseTotalNs - lastPauseNs
			metrics.GetOrRegisterGauge("golang.memory.gc.pause_per_second", metrics.DefaultRegistry).Update(int64(float64(pauseSinceLastSample) / sleep.Seconds()))
		}
		lastPauseNs = memStats.PauseTotalNs

		countGC := int(uint64(memStats.NumGC) - lastNumGC)
		if lastNumGC > 0 {
			diff := float64(countGC)
			diffTime := now.Sub(lastSampleTime).Seconds()
			metrics.GetOrRegisterGauge("golang.memory.gc.gc_per_second", metrics.DefaultRegistry).Update(int64(diff / diffTime))
		}

		if countGC > 0 {
			if countGC > 256 {
				countGC = 256
			}

			for i := 0; i < countGC; i++ {
				idx := int((memStats.NumGC-uint32(i))+255) % 256
				pause := time.Duration(memStats.PauseNs[idx])
				metrics.GetOrRegisterTimer("golang.memory.gc.pause", metrics.DefaultRegistry).Update(pause)
			}
		}

		lastNumGC = uint64(memStats.NumGC)
		lastSampleTime = now

		time.Sleep(sleep)
	}
}

func main() {
	app := App{
		LibratoEmail:  os.Getenv("LIBRATO_EMAIL"),
		LibratoToken:  os.Getenv("LIBRATO_TOKEN"),
		LibratoSource: os.Getenv("ID"),
	}

	go librato.Librato(metrics.DefaultRegistry, time.Minute,
		app.LibratoEmail, app.LibratoToken, app.LibratoSource,
		[]float64{0.50, 0.75, 0.90, 0.95, 0.99, 0.999, 1.0}, time.Millisecond)

	ReportMemstatsMetrics()
}
