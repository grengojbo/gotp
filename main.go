package main

import (
	// "github.com/bamarni/printer"
	"bufio"
	"fmt"
	"os"

	"github.com/grengojbo/gotp/escpos"
)

func main() {
	fmt.Println("Start print...")
	// printer := printer.NewPrinter(os.Stdout)
	// printer.Print(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	p := escpos.New(w)
	p.Verbose = true
	p.Begin()
	// p.SetSmooth(1)
	// p.SetFontSize(2, 3)
	// p.SetFont("A")
	p.Write("test ")
	p.Linefeed()
	// p.Cut()
	// p.End()

	w.Flush()
}
