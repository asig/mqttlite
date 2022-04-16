/*
 * Copyright (c) 2022 Andreas Signer <asigner@gmail.com>
 *
 * This file is part of mqttlite.
 *
 * mqttlite is free software: you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * mqttlite is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with mqttlite.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type resource struct {
	name string
	content []byte
}

var(
	flagOut = flag.String("out", "", "Destination file")
	flagPrefix = flag.String("prefix", "", "Prefix to be removed from filenames")
	flagPkg = flag.String("pkg", "", "Package to be used.")

	out *os.File
	resources []resource
)

func handleFile(f string) error {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}
	resources = append(resources, resource{f, data})
	return nil
}


func toByteArray(bytes []byte) []string {
	var lines []string
	linelen := 18
	start := 0
	for start < len(bytes) {
		var s []string
		for i := 0; i < linelen; i++ {
			if start + i < len(bytes) {
				s = append(s, fmt.Sprintf("0x%02x", bytes[start+i]))
			}
		}
		lines = append(lines, strings.Join(s, ", "))
		start = start + linelen
	}
	return lines
}

func makeName(n string) string {
	if strings.HasPrefix(n, *flagPrefix) {
		n = n[len(*flagPrefix):]
	}
	return n
}

func main() {
	flag.Parse()

	if *flagOut == "" {
		flag.Usage()
		return
	}

	for _, f := range flag.Args() {
		err := handleFile(f)
		if err != nil {
			fmt.Printf("Error while processing file %q: %s\n", f, err)
		}
	}

	var err error
	out, err = os.Create(*flagOut)
	if err != nil {
		fmt.Printf("Can't create output file %q: %s", *flagOut, err)
		return
	}

	fmt.Fprintf(out, "/*\n")
	fmt.Fprintf(out, "/* Automatically generated on %s\n", time.Now())
	fmt.Fprintf(out, "/* DO NOT EDIT\n")
	fmt.Fprintf(out, " */\n")
	fmt.Fprintf(out, "package %s\n", *flagPkg)
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "import \"strings\"\n")
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "var (\n")
	for idx, r := range resources {
		fmt.Fprintf(out, "\tdata_%03d = {\n", idx)
		for _, l := range toByteArray(r.content) {
			fmt.Fprintf(out, "\t\t%s,\n", l)
		}
		fmt.Fprintf(out, "\t},\n")
	}
	fmt.Fprintf(out, ")\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "func init() {\n")
	for idx, r := range resources {
		fmt.Fprintf(out, "\tfs.Add(%q, data_%03d)\n", makeName(r.name), idx)
	}
	fmt.Fprintf(out, "}\n")
}

