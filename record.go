package recording

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kr/binarydist"
)

type patch []byte
type frame []byte

type progressCallback func(done int)

type Record struct {
	sync.Mutex

	Data      []byte  `json:"data"`
	Cur       []byte  `json:"-"`
	States    []patch `json:"states"`
	NumStates int     `json:"-"`
	CurState  int     `json:"-"`

	frames   []frame
	progress progressCallback
}

func NewRecord(progress progressCallback) *Record {
	return &Record{
		Data:      nil,
		Cur:       nil,
		States:    make([]patch, 0),
		NumStates: 0,
		CurState:  0,
		frames:    nil,
		progress:  progress,
	}
}

func (e *Record) AddState(state []byte) error {
	e.Lock()
	defer e.Unlock()

	// set reference state
	if e.Data == nil {
		e.Data = state
	} else {
		// create a patch
		oldReader := bytes.NewReader(e.Cur)
		newReader := bytes.NewReader(state)
		writer := new(bytes.Buffer)

		if err := binarydist.Diff(oldReader, newReader, writer); err != nil {
			return err
		}

		e.States = append(e.States, patch(writer.Bytes()))
		e.NumStates++
		e.CurState = 0
	}
	e.Cur = state

	return nil
}

func (e *Record) Reset() {
	e.Lock()
	defer e.Unlock()
	e.Cur = e.Data
	e.NumStates = len(e.States)
	e.CurState = 0
}

func (e *Record) Compile() error {
	e.Lock()
	defer e.Unlock()

	// reset the state
	e.Cur = e.Data
	e.NumStates = len(e.States)
	e.CurState = 0
	e.frames = make([]frame, e.NumStates+1)

	// first is the master frame
	e.frames[0] = frame(e.Data)
	// precompute frames so they can be accessed by index
	for i := 0; i < e.NumStates; i++ {
		patch := e.States[i]
		oldReader := bytes.NewReader(e.Cur)
		patchReader := bytes.NewReader(patch)
		newWriter := new(bytes.Buffer)

		if err := binarydist.Patch(oldReader, newWriter, patchReader); err != nil {
			return err
		}

		e.Cur = newWriter.Bytes()
		e.frames[i+1] = e.Cur

		e.progress(1)
	}

	e.progress(1)

	return nil
}

func (e *Record) Frames() int {
	e.Lock()
	defer e.Unlock()
	// master + sub states
	return e.NumStates + 1
}

func (e *Record) CurFrame() int {
	e.Lock()
	defer e.Unlock()
	return e.CurState + 1
}

func (e *Record) SetFrom(from int) {
	e.Lock()
	defer e.Unlock()
	e.CurState = from
}

func (e *Record) Over() bool {
	e.Lock()
	defer e.Unlock()
	return e.CurState > e.NumStates
}

func (e *Record) Next() []byte {
	e.Lock()
	defer e.Unlock()
	cur := e.CurState
	e.CurState++
	return e.frames[cur]
}

func (e *Record) TimeOf(idx int) time.Time {
	e.Lock()
	defer e.Unlock()

	buf := e.frames[idx]
	frame := make(map[string]interface{})

	if err := json.Unmarshal(buf, &frame); err != nil {
		fmt.Printf("%v\n", err)
		return time.Time{}
	} else if tm, err := time.Parse(time.RFC3339, frame["polled_at"].(string)); err != nil {
		fmt.Printf("%v\n", err)
		return time.Time{}
	} else {
		return tm
	}
}

func (e *Record) StartedAt() time.Time {
	return e.TimeOf(0)
}

func (e *Record) StoppedAt() time.Time {
	return e.TimeOf(e.NumStates)
}

func (e *Record) Duration() time.Duration {
	return e.StoppedAt().Sub(e.StartedAt())
}
