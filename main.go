package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/adityasaky/go-metapackage/metapackage"
)

func main() {
	goMod := flag.Bool("gomod", false, "Set to true to enable Go Mod support")
	flag.Parse()
	arguments := flag.Args()

	if len(arguments) > 1 {
		fmt.Println("Error: too many packages")
		os.Exit(1)
	}

	if *goMod {
		os.Setenv("GO111MODULE", "on")
	} else {
		os.Setenv("GO111MODULE", "off")
	}

	allFunctions, err := metapackage.FindAllFunctions(arguments[0])
	if err != nil {
		fmt.Println("Error: FindAllFunctions", err)
	}

	sources := metapackage.GenerateSources(allFunctions)

	// TODO: fix output file handling
	err = metapackage.GeneratePackage(sources, "buildme.go")
	if err != nil {
		fmt.Println("Error: GeneratePackage", err)
	}
}
