package recording

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/evilsocket/islazy/fs"
)

type LoadProgress func(perc float64, done int, total int)

type Archive struct {
	sync.Mutex

	Session *Record `json:"session"`
	Events  *Record `json:"events"`

	fileName   string       `json:"-"`
	done       int          `json:"-"`
	total      int          `json:"-"`
	progress   float64      `json:"-"`
	onProgress LoadProgress `json:"-"`
}

func New(fileName string) *Archive {
	r := &Archive{
		fileName: fileName,
	}

	r.Session = NewRecord(r.onPartialProgress)
	r.Events = NewRecord(r.onPartialProgress)

	return r
}

func (r *Archive) onPartialProgress(done int) {
	r.done += done
	r.progress = float64(r.done) / float64(r.total) * 100.0

	if r.onProgress != nil {
		r.onProgress(r.progress, r.done, r.total)
	}
}

func Load(fileName string, cb LoadProgress) (*Archive, error) {
	if !fs.Exists(fileName) {
		return nil, fmt.Errorf("%s does not exist", fileName)
	}

	compressed, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error while reading %s: %s", fileName, err)
	}

	decompress, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("error while reading gzip file %s: %s", fileName, err)
	}
	defer decompress.Close()

	raw, err := ioutil.ReadAll(decompress)
	if err != nil {
		return nil, fmt.Errorf("error while decompressing %s: %s", fileName, err)
	}

	rec := &Archive{}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err = decoder.Decode(rec); err != nil {
		return nil, fmt.Errorf("error while parsing %s: %s", fileName, err)
	}

	rec.Session.OnProgress(rec.onPartialProgress)
	rec.Events.OnProgress(rec.onPartialProgress)

	rec.fileName = fileName
	rec.total = rec.Session.Frames() + rec.Events.Frames()
	rec.progress = 0.0
	rec.done = 0
	rec.onProgress = cb

	// reset state and precompute frames
	if err = rec.Session.Compile(); err != nil {
		return nil, err
	} else if err = rec.Events.Compile(); err != nil {
		return nil, err
	}

	return rec, nil
}

func (r *Archive) NewState(session []byte, events []byte) error {
	if err := r.Session.AddState(session); err != nil {
		return err
	} else if err := r.Events.AddState(events); err != nil {
		return err
	}
	return r.Flush()
}

func (r *Archive) save() error {
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	if err := encoder.Encode(r); err != nil {
		return err
	}

	data := buf.Bytes()

	compressed := new(bytes.Buffer)
	compress := gzip.NewWriter(compressed)

	if _, err := compress.Write(data); err != nil {
		return err
	} else if err = compress.Flush(); err != nil {
		return err
	} else if err = compress.Close(); err != nil {
		return err
	}

	return ioutil.WriteFile(r.fileName, compressed.Bytes(), os.ModePerm)
}

func (r *Archive) Flush() error {
	r.Lock()
	defer r.Unlock()
	return r.save()
}
