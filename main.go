package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/codegangsta/cli"
	"github.com/grengojbo/gotp/escpos"
)

var (
	// Version - current version
	Version   = "0.1.0"
	BuildTime = "2015-09-20 UTC"
	GitHash   = "c00"
)

// Commands - list command
var Commands = []cli.Command{
	cmdTest,
	cmdText,
}

var cmdTest = cli.Command{
	Name:   "test",
	Usage:  "Print Test Page",
	Action: runTest,
}

var cmdText = cli.Command{
	Name:   "text",
	Usage:  "Print text",
	Action: runText,
}

func runTest(c *cli.Context) {
	if c.GlobalBool("verbose") {
		fmt.Println("Print test page")
	}
	p := escpos.New(c.GlobalBool("debug"), "/dev/ttyAMA0", 19200)
	p.Verbose = c.GlobalBool("verbose")

	p.Begin()
	p.TestPage()

	if c.GlobalBool("verbose") {
		fmt.Println("Finish :)")
	}
}

func runText(c *cli.Context) {
	if c.GlobalBool("verbose") {
		fmt.Println("Print text")
	}
	p := escpos.New(c.GlobalBool("debug"), "/dev/ttyAMA0", 19200)
	p.Verbose = c.GlobalBool("verbose")

	if c.GlobalBool("verbose") {
		fmt.Println("---------------------------------")
		fmt.Println(c.Args())
		fmt.Println("---------------------------------")
	}
	p.Begin()
	p.SetAlign("R")
	for _, src := range c.Args() {
		// p.Write(src)
		p.WriteText(src)
		p.Linefeed()
	}
	p.Linefeed()

	if c.GlobalBool("verbose") {
		fmt.Println("Finish :)")
	}
}

func main() {
	runtime.GOMAXPROCS(1)

	app := cli.NewApp()
	app.Name = "print-pos"
	app.Version = Version
	app.Usage = "Mini Thermal Printer cli print"
	app.Author = "Oleg Dolya"
	app.Email = "oleg.dolya@gmail.com"
	app.EnableBashCompletion = true
	app.Commands = Commands
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Verbose mode",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Debug mode",
		},
	}

	app.Run(os.Args)
}
