package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/codegangsta/cli"
	"github.com/grengojbo/gotp/escpos"
	"github.com/grengojbo/gotp/models"
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
	cmdFile,
}

var cmdTest = cli.Command{
	Name:   "test",
	Usage:  "Print Test Page",
	Action: runTest,
}

var cmdFile = cli.Command{
	Name:   "file",
	Usage:  "Print from file",
	Action: runFile,
}

var cmdText = cli.Command{
	Name:   "text",
	Usage:  "Print text",
	Action: runText,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "align, a",
			Usage: "text align (L,C,R)",
			Value: "left",
		},
	},
}

func runTest(c *cli.Context) {
	if c.GlobalBool("verbose") {
		fmt.Println("Print test page")
	}
	p := escpos.New(c.GlobalBool("debug"), "/dev/ttyAMA0", 19200)
	p.Verbose = c.GlobalBool("verbose")

	p.Begin()
	p.SetCodePage(c.GlobalString("encode"))
	p.TestPage()

	if c.GlobalBool("verbose") {
		fmt.Println("Finish :)")
	}
}

func runFile(c *cli.Context) {
	if c.GlobalBool("verbose") {
		fmt.Println("Print from file")
	}
	if c.Args().Present() {
	} else {
		fmt.Println("Is not file path")
	}
	res, err := models.LoadPrintModel(c.Args().First())
	if err != nil {
		fmt.Println(err)
	} else {
		p := escpos.New(c.GlobalBool("debug"), "/dev/ttyAMA0", 19200)
		p.Verbose = c.GlobalBool("verbose")

		p.Begin()
		p.SetCodePage(c.GlobalString("encode"))

		if len(res.Header) > 0 {
			p.WriteNode(res.Header, &res.BarCode)
			p.Feed(1)
		}
		if len(res.Lines) > 0 {
			p.WriteNode(res.Lines, &res.BarCode)
			// p.Feed(2)
		}
		if len(res.Footer) > 0 {
			p.WriteNode(res.Footer, &res.BarCode)
			p.Feed(3)
		}
	}

	if c.GlobalBool("verbose") {
		fmt.Println("Finish :)")
	}
}

func runText(c *cli.Context) {
	if c.GlobalBool("verbose") {
		fmt.Println("Print text")
	}
	if c.Args().Present() {
		p := escpos.New(c.GlobalBool("debug"), "/dev/ttyAMA0", 19200)
		p.Verbose = c.GlobalBool("verbose")

		if c.GlobalBool("verbose") {
			fmt.Println("---------------------------------")
			fmt.Println(c.Args())
			fmt.Println("---------------------------------")
		}
		p.Begin()
		p.SetCodePage(c.GlobalString("encode"))
		p.SetAlign(c.String("align"))
		for _, src := range c.Args() {
			// p.Write(src)
			if err := p.WriteText(src); err != nil {
				fmt.Println(err)
			}
			p.Linefeed()
		}
		p.Feed(2)
	} else {
		fmt.Println("Is not argument :)")
	}

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
		cli.StringFlag{
			Name:  "encode",
			Usage: "Setting Code page",
			Value: "PC437",
		},
	}

	app.Run(os.Args)
}
