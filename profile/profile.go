// Package profile unifies the profiling api between Gio profiler and pkg/profile.
package profile

import (
	"log"

	"gioui.org/layout"
	"gioui.org/x/profiling"
	"github.com/pkg/profile"
)

// Profiler unifies the profiling api between Gio profiler and pkg/profile.
type Profiler struct {
	Starter  func(p *profile.Profile)
	Stopper  func()
	Recorder func(gtx layout.Context)
}

// Start profiling.
func (pfn *Profiler) Start() {
	if pfn.Starter != nil {
		pfn.Stopper = profile.Start(pfn.Starter).Stop
	}
}

// Stop profiling.
func (pfn *Profiler) Stop() {
	if pfn.Stopper != nil {
		pfn.Stopper()
	}
}

// Record GUI stats per frame.
func (pfn Profiler) Record(gtx layout.Context) {
	if pfn.Recorder != nil {
		pfn.Recorder(gtx)
	}
}

// Opt specifies the various profiling options.
type Opt string

const (
	None      Opt = "none"
	CPU       Opt = "cpu"
	Memory    Opt = "mem"
	Block     Opt = "block"
	Goroutine Opt = "goroutine"
	Mutex     Opt = "mutex"
	Trace     Opt = "trace"
	Gio       Opt = "gio"
)

// NewProfiler creates a profiler based on the selected option.
func (p Opt) NewProfiler() Profiler {
	switch p {
	case "", None:
		return Profiler{}
	case CPU:
		return Profiler{Starter: profile.CPUProfile}
	case Memory:
		return Profiler{Starter: profile.MemProfile}
	case Block:
		return Profiler{Starter: profile.BlockProfile}
	case Goroutine:
		return Profiler{Starter: profile.GoroutineProfile}
	case Mutex:
		return Profiler{Starter: profile.MutexProfile}
	case Trace:
		return Profiler{Starter: profile.TraceProfile}
	case Gio:
		var (
			recorder *profiling.CSVTimingRecorder
			err      error
		)
		return Profiler{
			Starter: func(*profile.Profile) {
				recorder, err = profiling.NewRecorder(nil)
				if err != nil {
					log.Printf("starting profiler: %v", err)
				}
			},
			Stopper: func() {
				if recorder == nil {
					return
				}
				if err := recorder.Stop(); err != nil {
					log.Printf("stopping profiler: %v", err)
				}
			},
			Recorder: func(gtx layout.Context) {
				if recorder == nil {
					return
				}
				recorder.Profile(gtx)
			},
		}
	}
	return Profiler{}
}
