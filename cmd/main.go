package main

import (
	"fmt"
	"log"

	"maschine_chords_converter/internal/converter"
)

const debug = false // if true, the local folder "./sets" is used, for development purposes

func main() {
	c := converter.New()
	c.SetDebug(debug)

	if err := c.Run(); err != nil {
		log.Fatal(err.Error())
	}

	if debug {
		fmt.Println("processing complete...")
		return
	}

	fmt.Println("processing complete. Press Enter to exit...")
	_, _ = fmt.Scanln()
}
