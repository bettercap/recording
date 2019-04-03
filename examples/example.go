package main

import (
	"fmt"

	"github.com/bettercap/recording"
)

const fileName = "/home/evilsocket/bettercap-session.record"

func main() {
	// load a recording from a file and print progress while loading it
	arch, err := recording.Load(fileName, func(perc float64, done int, total int) {
		fmt.Printf("loaded %d/%d frames ( %.2f ) ...\n", done, total, perc)
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("loaded session started at %s and stopped at %s (%s):\n\n", arch.Session.StartedAt(), arch.Session.StoppedAt(), arch.Session.Duration())

	// keep iterating every session frame until they're over
	// the same is doable with the `arch.Events` field
	for arch.Session.Over() == false {
		raw := arch.Session.Next()
		// parse the raw JSON into a map or whatever to access its attributes
		fmt.Printf("%s\n", raw)
	}
}
