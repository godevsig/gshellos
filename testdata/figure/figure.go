package main

import (
	"github.com/common-nighthawk/go-figure"
)

func main() {
	myFigure := figure.NewFigure("123c", "", true)
	myFigure.Print()
	//printHello()
}

//output:
//  _   ____    _____
// / | |___ \  |___ /    ___
// | |   __) |   |_ \   / __|
// | |  / __/   ___) | | (__
// |_| |_____| |____/   \___|
