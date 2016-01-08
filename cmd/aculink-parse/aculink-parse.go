package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/comstud/go-aculink/aculink"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <string>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
	}

	arg := flag.Args()[0]

	var data *aculink.Data

	if arg[0] == '{' {
		var err error
		data, err = aculink.DataFromJSON(arg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		data, _ = aculink.NewData(arg)
	}
	fmt.Printf("Got data: %s\n", data.JSONString())

}
