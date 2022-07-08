// Loader adapted from Egon's https://github.com/egonelbre/expgio.
package async

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"gioui.org/layout"
)

// Tag pointer used to identify a unique resource.
type Tag interface{}

// LoadFunc function that performs the blocking load.
type LoadFunc func(ctx context.Context) interface{}

// Resource is an async entity that can be in various states and potentially
// contain a value.
type Resource struct {
	// State reports current state for this resource.
	State State
	// Value for the resource. Nil if not ready.
	Value interface{}
}

// State that an async Resource can be in.
type State byte

const (
	Queued State = iota
	Loading
	Loaded
)

// Loader is an asynchronously loaded resource.
// Start and poll a resource with Schedule method.
// Track frames with Frame method to detect stale data.
// Respond to updates in event loop by selecting on Updated channel.
type Loader struct {
	// Scheduler provides scheduling behaviour. Defaults to a sized worker pool.
	// The caller can provide a scheduler that implements the best strategy for
	// the their usecase.
	Scheduler Scheduler
	// MaxLoaded specifies the maximum number of resources to load before
	// de-allocating old resources.
	MaxLoaded int
	// active frame being layed out.
	// Access must be synchronized with atomics.
	active int64
	// finished frames that have been layed out.
	// Access must be synchronized with atomics.
	finished int64
	// update chan reports that a resource's status has changed.
	// Useful for invalidating the window.
	updated chan struct{}
	// init allows Loader to have a useful zero value by lazily allocating on
	// first use.
	init sync.Once
	// loader contains the queue and lookup map.
	loader
}

// Scheduler schedules work according to some strategy.
// Implementations can implement the best way to distribute work for a given
// application.
//
// TODO(jfm): context cancellation.
type Scheduler interface {
	// Schedule a piece of work. This method is allowed to block.
	Schedule(func())
}

// FixedWorkerPool implements a simple fixed-size worker pool that lets go
// runtime schedule work atop some number of goroutines.
//
// This pool will minimize goroutine latency at the cost of maintaining the
// configured number of goroutines throughout the lifetime of the pool.
type FixedWorkerPool struct {
	// Workers specifies the number of concurrent workers in this pool.
	Workers int
	// queue of work. Unbuffered so it will block if worker pull is at capacity.
	queue chan func()
	// once time initialization.
	sync.Once
}

// Schedule work to be executed by the available workers. This is a blocking
// call if all workers are busy.
func (p *FixedWorkerPool) Schedule(work func()) {
	p.Once.Do(func() {
		p.queue = make(chan func())
		if p.Workers <= 0 {
			p.Workers = runtime.NumCPU()
		}
		for ii := 0; ii < p.Workers; ii++ {
			go func() {
				for w := range p.queue {
					if w != nil {
						w()
					}
				}
			}()
		}
	})
	p.queue <- work
}

// DynamicWorkerPool implements a simple dynamic-sized worker pool that spins up
// a new worker per unit of work, until the maximum number of workers has been
// reached.
//
// This pool will minimize idle memory as goroutines will die off once complete,
// but will incur the latency cost, such that it is, of spinning up goroutines
// on-the-fly.
//
// Additionally, ordering of work is inconsistent with highly dynamic layouts.
type DynamicWorkerPool struct {
	// Workers specifies the maximum allowed number of concurrent workers in
	// this pool. Defaults to NumCPU.
	Workers int64
	// count is a semaphore queue that limits the number of workers at any
	// given time. The size of the buffer for the channel provides the limit.
	count chan struct{}
	// queue of work. Unbuffered so it will block if worker pool is at capacity.
	queue chan func()
	// once time initialization.
	sync.Once
}

// Schedule work to be executed by the available workers. This is a blocking
// call if all workers are busy.
//
// Workers are limited by a buffer of semaphores.
// Each worker holds a semaphore for the duration of it's life and returns it
// before exiting.
func (p *DynamicWorkerPool) Schedule(work func()) {
	p.Once.Do(func() {
		if p.Workers <= 0 {
			p.Workers = int64(runtime.NumCPU())
		}
		p.queue = make(chan func())
		p.count = make(chan struct{}, p.Workers)
		for ii := 0; ii < int(p.Workers); ii++ {
			p.count <- struct{}{}
		}
		go func() {
			for w := range p.queue {
				w := w
				if w != nil {
					sem := <-p.count
					go func() {
						w()
						p.count <- sem
					}()
				}
			}
		}()
	})
	p.queue <- work
}

// loader wraps up state that needs to be synchronized together.
type loader struct {
	// mu is the primary mutex used to synchronize.
	mu sync.Mutex
	// refresh sleeps the loop, ensuring we only try to process the queue when
	// something has actually changed.
	refresh sync.Cond
	// lookup is a map of async resources mapped to a unique tag similar to
	// gio router api. Tag value must be a hashable type.
	lookup map[Tag]*resource
	// queue of resources to process in sequence.
	queue []*resource
}

// Updated returns a channel that reports whether loader has been updated.
// Integrate this into gio event loop to, for example, invalidate the window.
//
// 	case <-loader.Updated():
//		w.Invalidate()
//
func (l *Loader) Updated() <-chan struct{} {
	l.init.Do(l.initialize)
	return l.updated
}

// Frame wraps a widget and tracks frame updates.
//
// Typically you should wrap your entire UI so that each frame is counted.
// However, it is sufficient to wrap only the widget that expects to use the
// loader during it's layout.
//
// A frame is currently updating if activeFrame < finishedFrame.
func (l *Loader) Frame(gtx layout.Context, w layout.Widget) layout.Dimensions {
	atomic.AddInt64(&l.active, 1)
	dim := w(gtx)
	atomic.StoreInt64(&l.finished, atomic.LoadInt64(&l.active))
	l.refresh.Signal()
	return dim
}

// DefaultMaxLoaded is used when no max is specified.
const DefaultMaxLoaded = 10

// Schedule a resource to be loaded asynchronously, returning a resource that
// will hold the loaded value at some point.
//
// Schedule should be called per frame and the state of the resource checked
// accordingly.
//
// The first call will queue up the load, subsequent calls will poll for the
// status.
func (l *Loader) Schedule(tag Tag, load LoadFunc) Resource {
	l.init.Do(l.initialize)
	return l.loader.establish(tag, load, atomic.LoadInt64(&l.active))
}

func (l *Loader) initialize() {
	if l.MaxLoaded == 0 {
		l.MaxLoaded = DefaultMaxLoaded
	}
	l.updated = make(chan struct{}, 1)
	l.loader.lookup = make(map[Tag]*resource)
	l.loader.refresh.L = &l.loader.mu
	if l.Scheduler == nil {
		l.Scheduler = &FixedWorkerPool{Workers: l.MaxLoaded}
	}
	// TODO(jfm): expose context in the public api so that loads can be
	// cancelled by it.
	// Egon's example ran this at the top of the event loop. By placing it
	// here we achieve useful zero-value but since the context is not
	// exposed, it's useless.
	go l.run(context.Background())
}

// LoaderStats tracks some stats about the loader.
type LoaderStats struct {
	Lookup int
	Queued int
}

// Stats reports runtime data about this loader.
func (l *loader) Stats() LoaderStats {
	l.mu.Lock()
	defer l.mu.Unlock()
	return LoaderStats{
		Lookup: len(l.lookup),
		Queued: len(l.queue),
	}
}

// update signals to the outside world that some resource has experienced a
// state change.
//
// This is particularly useful for invaliding the window, forcing a re-layout
// immediately.
func (l *Loader) update() {
	select {
	case l.updated <- struct{}{}:
	default:
	}
}

// run the persistent processing goroutine that performs the blocking operations.
func (l *Loader) run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		l.refresh.Signal()
	}()

	loader := &l.loader

	loader.mu.Lock()
	defer loader.mu.Unlock()

	firstIteration := true
	for {
		if !firstIteration {
			// Wait to be woken up by a change. Three conditions which provoke this:
			// 1. a new frame layout
			// 2. scheduling a _new_ resource
			// 3. context cancellation
			// Each iteration synchronizes access to the map and queue.
			loader.refresh.Wait()
			if ctx.Err() != nil {
				return
			}
		}
		firstIteration = false
		loader.purge(atomic.LoadInt64(&l.finished), l.MaxLoaded)
		for r := loader.next(); r != nil; r = loader.next() {
			r := r
			if l.isOld(r) {
				loader.remove(r)
				continue
			}
			loader.mu.Unlock()
			l.update()
			l.Scheduler.Schedule(func() {
				r.Load(ctx, func(_ State) {
					l.update()
				})
			})
			loader.mu.Lock()
		}
	}
}

// isOld reports whether the resource is old.
// Old is defined as being last used in a frame prior to the current frame.
// For example, a resource is "old" if the last frame it was used was frame 3
// and the current frame is frame 10.
func (l *Loader) isOld(r *resource) bool {
	return atomic.LoadInt64(&r.frame) < atomic.LoadInt64(&l.finished)
}

// establish a resource for the given tag and load function.
// If the resource does not already exist it is first allocated.
// A copy of the state and a reference to the value are returned.
func (l *loader) establish(tag Tag, load LoadFunc, activeFrame int64) Resource {
	l.mu.Lock()
	r, ok := l.lookup[tag]
	if !ok {
		r = &resource{
			tag:   tag,
			load:  load,
			state: Queued,
			value: nil,
		}
		l.lookup[tag] = r
		l.queue = append(l.queue, r)
		l.refresh.Signal()
	}
	l.mu.Unlock()
	// Set the resource frame to that of the currently active frame.
	// This "freshens" the resource, indicating that it has recently been
	// accessed.
	atomic.StoreInt64(&r.frame, activeFrame)
	state, value := r.Get()
	return Resource{
		State: state,
		Value: value,
	}
}

// next selects the next resource off the queue.
// Only call this when lock has been acquired.
func (l *loader) next() *resource {
	if len(l.queue) == 0 {
		return nil
	}
	r := l.queue[0]
	l.queue = l.queue[1:]
	return r
}

// purge removes stale data such that it gets garbage collected.
//
// A resource is purged if it is old and the maximum number of resources has
// been exhausted.
//
// Only call this when lock has been acquired.
func (l *loader) purge(activeFrame int64, max int) {
	for _, r := range l.lookup {
		if len(l.lookup) < max {
			break
		}
		if isOld := atomic.LoadInt64(&r.frame) < activeFrame; isOld {
			l.remove(r)
		}
	}
}

// remove the resource from the local storage and let it be garbage collected.
func (l *loader) remove(r *resource) {
	delete(l.lookup, r.tag)
}

// resource records data about a loading value.
// state and value are synchronized by the mutex, tag and load are set once
// during allocation, and frame is synchronized via atomic operations.
type resource struct {
	sync.Mutex
	// frame wherein this data is valid.
	// Access must be synchronized with atomics.
	frame int64
	// state of the resource for this frame.
	// Access is synchronized by mutex, use Get method.
	state State
	// value for the resource, if acquired.
	// Access is synchronized by mutex, use Get method.
	value interface{}
	// tag of the resource.
	// Used to uniquely identify the resource stored in a map.
	// Must be a hashable value.
	// Unsynchronized field, do not modify.
	tag Tag
	// load function for the resource.
	// Used to perform the blocking load of the value, like a network call or
	// disk operation.
	// Unsynchronized field, do not modify.
	load LoadFunc
}

// Load the value for the resource using the configured closure.
// State changes occur during load sequence, invoking onChange callback per
// state change.
func (r *resource) Load(ctx context.Context, onChange func(State)) {
	r.Set(Loading, nil)
	onChange(r.state)
	v := r.load(ctx)
	r.Set(Loaded, v)
	onChange(r.state)
}

// Get the state and value for the resource.
func (r *resource) Get() (State, interface{}) {
	r.Lock()
	defer r.Unlock()
	return r.state, r.value
}

// Set the state and value for the resource.
func (r *resource) Set(s State, v interface{}) {
	r.Lock()
	r.state = s
	r.value = v
	r.Unlock()
}
