package main

import (
	"github.com/common-nighthawk/go-figure"
)

func printHello() {
	myFigure := figure.NewFigure("Hello World", "", true)
	myFigure.Print()
}
