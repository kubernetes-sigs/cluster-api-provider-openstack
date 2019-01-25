// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
)

const (
	header = `package %s

/*
This file is auto-generated DO NOT TOUCH!
*/

const (
`
)

func main() {

	fileName := flag.String("input-filename", "", "Input filename to put into a variable")
	varName := flag.String("variable-name", "", "Name of the variable to use")
	outName := flag.String("output-filename", "", "for example ubuntu-template.go")
	packageName := flag.String("package-name", "userdata", "Go package name to use, default userdata")
	flag.Parse()

	if *fileName == "" {
		log.Fatal("Please provide an input-filename")
	}
	if *varName == "" {
		log.Fatal("Please provide a variable-name")
	}
	if *outName == "" {
		log.Fatal("Please provide an output-filename")
	}

	data, err := os.Open(*fileName)
	if err != nil {
		log.Fatalf("unable to read data: %v", err)
	}

	buf := bytes.NewBuffer([]byte(fmt.Sprintf(header, *packageName)))
	fmt.Fprintf(buf, "%s = `", *varName)

	_, err = io.Copy(buf, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(buf, "\n`")
	fmt.Fprintln(buf, ")")

	res, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	out, err := os.Create(*outName)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = out.Write(res)
	if err != nil {
		log.Fatal(err)
	}
}
