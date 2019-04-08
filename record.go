package recording

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"
)

type progressCallback func(done int)

type Record struct {
	sync.Mutex

	Head    []byte  `json:"data"`
	Patches []patch `json:"states"`

	current    []byte
	numPatches int
	index      int
	frames     []frame
	progress   progressCallback
}

func NewRecord(progress progressCallback) *Record {
	return &Record{
		Head:       nil,
		current:    nil,
		Patches:    make([]patch, 0),
		numPatches: 0,
		index:      0,
		frames:     nil,
		progress:   progress,
	}
}

func (e *Record) addPatch(p patch) {
	e.Patches = append(e.Patches, p)
	e.numPatches++
	e.index = 0
}

func (e *Record) AddState(state []byte) error {
	e.Lock()
	defer e.Unlock()

	// set reference state
	if e.Head == nil {
		e.Head = state
	} else if err, p := doDiff(e.current, state); err != nil {
		return err
	} else {
		e.addPatch(p)
	}

	e.current = state
	return nil
}

func (e *Record) Reset() {
	e.Lock()
	defer e.Unlock()
	e.current = e.Head
	e.numPatches = len(e.Patches)
	e.index = 0
}

func (e *Record) addFrameAt(idx int, f frame) {
	e.current = f
	e.frames[idx] = e.current
	e.progress(1)
}

func (e *Record) Compile() error {
	e.Lock()
	defer e.Unlock()

	defer func() {
		// after generating the frames, free as much memory as possible
		runtime.GC()
	}()

	// reset the state
	e.current = e.Head
	e.numPatches = len(e.Patches)
	e.index = 0
	e.frames = make([]frame, e.numPatches+1)

	// first is the master frame
	e.frames[0] = frame(e.Head)
	// precompute frames so they can be accessed by index
	for i := 0; i < e.numPatches; i++ {
		if err, newFrame := doPatch(e.current, e.Patches[i]); err != nil {
			return err
		} else {
			e.addFrameAt(i+1, newFrame)
		}
	}

	e.progress(1)

	return nil
}

func (e *Record) OnProgress(cb progressCallback) {
	e.Lock()
	defer e.Unlock()
	e.numPatches = len(e.Patches)
	e.progress = cb
}

func (e *Record) Frames() int {
	e.Lock()
	defer e.Unlock()
	// master + sub states
	return e.numPatches + 1
}

func (e *Record) Index() int {
	e.Lock()
	defer e.Unlock()
	return e.index + 1
}

func (e *Record) SetFrom(from int) {
	e.Lock()
	defer e.Unlock()
	e.index = from
}

func (e *Record) Over() bool {
	e.Lock()
	defer e.Unlock()
	return e.index > e.numPatches
}

func (e *Record) Next() []byte {
	e.Lock()
	defer e.Unlock()
	cur := e.index
	e.index++
	return e.frames[cur]
}

func (e *Record) TimeOf(idx int) time.Time {
	e.Lock()
	defer e.Unlock()

	buf := e.frames[idx]
	frame := make(map[string]interface{})

	if err := json.Unmarshal(buf, &frame); err != nil {
		panic(err)
		return time.Time{}
	} else if tm, err := time.Parse(time.RFC3339, frame["polled_at"].(string)); err != nil {
		panic(err)
		return time.Time{}
	} else {
		return tm
	}
}

func (e *Record) StartedAt() time.Time {
	return e.TimeOf(0)
}

func (e *Record) StoppedAt() time.Time {
	return e.TimeOf(e.numPatches)
}

func (e *Record) Duration() time.Duration {
	return e.StoppedAt().Sub(e.StartedAt())
}
