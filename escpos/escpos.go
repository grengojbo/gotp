package escpos

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/grengojbo/gotp/models"
	"github.com/tarm/serial"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

const (
	// BAUDRATE Though most of these printers are factory configured for 19200 baud
	// operation, a few rare specimens instead work at 9600.  If so, change
	// this constant.  This will NOT make printing slower!  The physical
	// print and feed mechanisms are the bottleneck, not the port speed.
	BAUDRATE = 19200

	// BYTETIME Number of microseconds to issue one byte to the printer.  11 bits
	// (not 8) to accommodate idle, start and stop bits.  Idle time might
	// be unnecessary, but erring on side of caution here.
	BYTETIME = (((11 * 1000000) + (BAUDRATE / 2)) / BAUDRATE)

	// ASCIILF -> \n
	ASCIILF = byte(10)
)

// text replacement map
var textReplaceMap = map[string]string{
	// horizontal tab
	"&#9;":  "\x09",
	"&#x9;": "\x09",

	// linefeed
	"&#10;": "\n",
	"&#xA;": "\n",

	// xml stuff
	"&apos;": "'",
	"&quot;": `"`,
	"&gt;":   ">",
	"&lt;":   "<",

	// ampersand must be last to avoid double decoding
	"&amp;": "&",
}

// replace text from the above map
func (e *Escpos) textReplace(data string) string {
	for k, v := range textReplaceMap {
		data = strings.Replace(data, k, v, -1)
	}
	return data
}

// Escpos - library for the Adafruit Thermal Printer:
// https://www.adafruit.com/product/597
type Escpos struct {
	enc *encoding.Encoder
	// destination
	// dst io.Writer
	// config *serial.Config
	Serial *serial.Port

	// font metrics
	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	prevByte      byte
	column        uint8
	maxColumn     uint8
	charHeight    int64
	lineSpacing   int64
	barcodeHeight uint8

	printDensity   uint8
	printBreakTime uint8
	// state toggles GS[char]
	reverse, smooth uint8

	resumeTime     int64
	dotPrintTime   int64
	dotFeedTime    int64
	maxChunkHeight uint8

	Verbose  bool
	Debug    bool
	Firmware int
	err      error
}

// reset toggles
func (e *Escpos) reset() {
	if e.Verbose {
		fmt.Printf("func reset()\n")
	}
	// x1B -> ESC byte{27}
	e.Write("\x1B@")

	e.width = 1
	e.height = 1

	e.underline = 0
	e.emphasize = 0
	e.upsidedown = 0
	e.rotate = 0

	e.reverse = 0
	e.smooth = 0

	e.prevByte = ASCIILF
	e.column = 0
	e.maxColumn = 32
	e.charHeight = 24
	e.lineSpacing = 6
	e.barcodeHeight = 50
	e.printDensity = 10

	//  // Configure tab stops on recent printers
	// Set tab stops...
	if e.Firmware >= 264 {
		e.Write("\x1BD")
		e.WriteBytes([]byte{4, 8, 12, 16})  // ...every 4 columns,
		e.WriteBytes([]byte{20, 24, 28, 0}) // 0 marks end-of-list.
	}
}

// New - create Escpos printer
func New(debug bool, port string, baud int) (e *Escpos) {
	e = &Escpos{Debug: debug}
	e.enc = charmap.CodePage437.NewEncoder()
	e.Firmware = 268
	if !e.Debug {
		config := &serial.Config{Name: port, Baud: baud}
		s, err := serial.OpenPort(config)
		if err != nil {
			e.err = err
		} else {
			e.Serial = s
		}
	}

	e.printDensity = 10
	e.printBreakTime = 2
	e.timeoutSet(500000)
	e.reset()
	return
}

// IsOk - check error
func (e *Escpos) IsOk() bool {
	if e.err != nil {
		return false
	}
	return true
}
func (e *Escpos) SetDefault() {
	if e.Verbose {
		fmt.Println("TODO: SetDefault()")
	}
	// online();
	// justify('L');
	// inverseOff();
	// doubleHeightOff();
	// setLineHeight(30);
	// boldOff();
	// underlineOff();
	// setBarcodeHeight(50);
	// setSize('s');
	// setCharset();
	// setCodePage();
}

// func (e *Escpos) WriteString(src ...string) {
// 	data := []byte{}
// 	for _, e := range src {
// 		r := []byte(e)
// 		data = append(data, r)
// 	}
// }

// WriteBytes - write byte
func (e *Escpos) WriteBytes(data []byte) {
	e.timeoutWait()
	if e.Verbose {
		fmt.Println(data)
	}
	if !e.Debug {
		// e.dst.Write(data)
		_, err := e.Serial.Write(data)
		if err != nil {
			e.err = err
		}
	}
	e.timeoutSet(int64(len(data)) * BYTETIME)
}

// WriteRaw - write raw bytes to printer
func (e *Escpos) WriteRaw(data []byte) (n int, err error) {
	if len(data) > 0 {
		e.timeoutWait()
		if e.Verbose {
			fmt.Printf("Writing %d bytes\n", len(data))
			fmt.Println(data)
		}
		if !e.Debug {
			// e.dst.Write(data)
			n, err = e.Serial.Write(data)
		}
		e.timeoutSet(int64(len(data)) * BYTETIME)
		// OR
		// e.timeoutSet(BYTETIME)
	} else {
		if e.Verbose {
			fmt.Printf("Wrote NO bytes\n")
		}
	}
	return n, err
}

// Write - write a string to the printer
func (e *Escpos) Write(data string) (int, error) {
	// if e.Verbose {
	// 	fmt.Printf("func Write()\n")
	// }
	return e.WriteRaw([]byte(data))
}

func (e *Escpos) timeoutSet(x int64) {
	// if(!dtrEnabled) resumeTime = micros() + x;
	e.resumeTime = x
}

func (e *Escpos) timeoutWait() {
	time.Sleep(time.Microsecond * time.Duration(e.resumeTime))
	// if(dtrEnabled) {
	//    while(digitalRead(dtrPin) == HIGH);
	//  } else {
	//    while((long)(micros() - resumeTime) < 0L); // (syntax is rollover-proof)
	//  }
}

// Wake the printer from a low-energy state.
func (e *Escpos) wake() {

	if e.Verbose {
		fmt.Printf("func wake()\n")
	}
	e.timeoutSet(0)           // Reset timeout counter
	e.WriteBytes([]byte{255}) // Wake
	if e.Firmware >= 264 {
		//   delay(50);
		time.Sleep(time.Millisecond * 50)
		//   writeBytes(ASCII_ESC, '8', 0, 0); // Sleep off (important!)
		e.WriteBytes([]byte{27, 56, 0, 0})
	} else {
		//   // Datasheet recommends a 50 mS delay before issuing further commands,
		//   // but in practice this alone isn't sufficient (e.g. text size/style
		//   // commands may still be misinterpreted on wake).  A slightly longer
		//   // delay, interspersed with NUL chars (no-ops) seems to help.
		//   for(uint8_t i=0; i<10; i++) {
		//     writeBytes(0);
		//     timeoutSet(10000L);
		e.WriteBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		e.timeoutSet(10000)
	}
}

// Begin - The printer can't start receiving data immediately upon power up --
// it needs a moment to cold boot and initialize.  Allow at least 1/2
// sec of uptime before printer can receive data.
// func (e *Escpos) Begin(heatTime uint8) {
func (e *Escpos) Begin() {
	e.timeoutSet(500000)
	e.wake()
	e.reset()

	if e.Verbose {
		fmt.Printf("func Begin()\n")
	}
	// ESC 7 n1 n2 n3 Setting Control Parameter Command
	// n1 = "max heating dots" 0-255 -- max number of thermal print head
	//      elements that will fire simultaneously.  Units = 8 dots (minus 1).
	//      Printer default is 7 (64 dots, or 1/6 of 384-dot width), this code
	//      sets it to 11 (96 dots, or 1/4 of width).
	// n2 = "heating time" 3-255 -- duration that heating dots are fired.
	//      Units = 10 us.  Printer default is 80 (800 us), this code sets it
	//      to value passed (default 120, or 1.2 ms -- a little longer than
	//      the default because we've increased the max heating dots).
	// n3 = "heating interval" 0-255 -- recovery time between groups of
	//      heating dots on line; possibly a function of power supply.
	//      Units = 10 us.  Printer default is 2 (20 us), this code sets it
	//      to 40 (throttled back due to 2A supply).
	// More heating dots = more peak current, but faster printing speed.
	// More heating time = darker print, but slower printing speed and
	// possibly paper 'stiction'.  More heating interval = clearer print,
	// but slower printing speed.

	// writeBytes(ASCII_ESC, '7');   // Esc 7 (print settings)
	e.Write("\x1B7")
	e.WriteBytes([]byte{11, 80, 40})
	// OR
	// e.WriteBytes([]byte{7, 80, 2})

	// writeBytes(11, heatTime, 40); // Heating dots, heat time, heat interval

	// Print density description from manual:
	// DC2 # n Set printing density
	// D4..D0 of n is used to set the printing density.  Density is
	// 50% + 5% * n(D4-D0) printing density.
	// D7..D5 of n is used to set the printing break time.  Break time
	// is n(D7-D5)*250us.
	// (Unsure of the default value for either -- not documented)

	e.printDensity = 10  // 100% (? can go higher, text is darker but fuzzy)
	e.printBreakTime = 2 // 500 uS

	// writeBytes(ASCII_DC2, '#', (printBreakTime << 5) | printDensity);
	// fmt.Println((e.printBreakTime << 5) | e.printDensity)
	// e.Write(fmt.Sprintf("\x12#%v", (e.printBreakTime<<5)|e.printDensity))
	e.Write(fmt.Sprintf("\x12#%c", (e.printBreakTime<<5)|e.printDensity))

	// Enable DTR pin if requested
	// if(dtrPin < 255) {
	//   pinMode(dtrPin, INPUT_PULLUP);
	//   writeBytes(ASCII_GS, 'a', (1 << 5));
	// e.Write(fmt.Sprintf("\x1Da%c", (1 << 5)))
	//   dtrEnabled = true;
	// }

	e.dotPrintTime = 30000 // See comments near top of file for
	e.dotFeedTime = 2100   // an explanation of these values.
	e.maxChunkHeight = 255
}

// void Adafruit_Thermal::test(){
//   println(F("Hello World!"));
//   feed(2);
// }

// TestPage - print test page
func (e *Escpos) TestPage() {
	if e.Verbose {
		fmt.Printf("func TestPage()\n")
	}
	// writeBytes(ASCII_DC2, 'T');
	e.Write("\x12T")
	// timeoutSet(
	//   e.dotPrintTime * 24 * 26 +      // 26 lines w/text (ea. 24 dots high)
	//   e.dotFeedTime * (6 * 26 + 30)); // 26 text lines (feed 6 dots) + blank line
	e.timeoutSet(e.dotPrintTime*24*26 + e.dotFeedTime*(6*26+30))
}

// SetAlign - set alignment
// align (left, center, right)
func (e *Escpos) SetAlign(align string) (err error) {
	if e.Verbose {
		fmt.Printf("func SetAlign()\n")
	}
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	case "L":
		a = 0
	case "C":
		a = 1
	case "R":
		a = 2
	default:
		err = fmt.Errorf("Invalid alignment: %s", align)
	}
	e.Write(fmt.Sprintf("\x1Ba%c", a))
	return err
}

// WriteText - The underlying method for all high-level printing (e.g. println()).
// The inherited Print class handles the rest!
func (e *Escpos) WriteText(data string) (err error) {
	if e.Verbose {
		fmt.Printf("func SetAlign()\n")
	}
	data = e.textReplace(data)
	rawData, err := e.enc.String(data)
	if err != nil {
		return fmt.Errorf("Couldn't encode to charset (%s)", err)
	}
	if len(rawData) > 0 {
		// b := byte{19}
		for _, c := range []byte(rawData) {
			if c != 0x13 {
				e.timeoutWait()
				if !e.Debug {
					_, err := e.Serial.Write([]byte{c})
					if err != nil {
						e.err = err
					}
				} else {
					// fmt.Printf("%c", c)
					fmt.Printf("%d ", c)
				}
				d := int64(BYTETIME)
				if c == ASCIILF || e.column == e.maxColumn {
					e.timeoutSet(int64(BYTETIME) + ((e.charHeight + e.lineSpacing) * e.dotFeedTime))
					e.timeoutWait()
					e.column = 0
					c = ASCIILF
					if !e.Debug {
						_, err := e.Serial.Write([]byte{c})
						if err != nil {
							e.err = err
						}
					} else {
						fmt.Println("")
					}
					d += ((e.charHeight * e.dotPrintTime) + (e.lineSpacing * e.dotFeedTime))
				} else {
					e.column++
				}
				e.timeoutSet(int64(d))
				e.prevByte = c
			}
		}
	} else {
		return fmt.Errorf("len data = 0 :)")
	}
	return nil
}

// Flush - set \f
func (e *Escpos) Flush() {
	e.Write("\f")
}

// These commands work only on printers w/recent firmware ------------------

// SetCharset - Alters some chars in ASCII 0x23-0x7E range; see datasheet
func (e *Escpos) SetCharset(val uint8) {
	if e.Verbose {
		fmt.Printf("func SetCharset()\n")
	}
	if val > 15 {
		val = 15
	}
	e.Write(fmt.Sprintf("\x1BR%c", val))
}

// SetCodePage - Selects alt symbols for 'upper' ASCII values 0x80-0xFF
func (e *Escpos) SetCodePage(code string) {
	if e.Verbose {
		fmt.Printf("func SetCodePage()\n")
	}
	var n byte
	switch code {
	case "PC437": // USA: Standard Europe
		e.enc = charmap.CodePage437.NewEncoder()
		n = 0
	case "PC850": // Western Europe
		e.enc = charmap.CodePage850.NewEncoder()
		n = 2
	case "CP1251": // Cyrillic
		e.enc = charmap.Windows1251.NewEncoder()
		n = 6
	default:
		n = 47
	}
	e.Write(fmt.Sprintf("\x1Bt%c", n))
}

func (e *Escpos) tab() {
	if e.Verbose {
		fmt.Printf("func tab()\n")
	}
	e.Write("\t")
	e.column = (e.column + 4)
}

// LinePrint - print line -------
func (e *Escpos) LinePrint() {
	for i := 0; i < int(e.maxColumn); i++ {
		e.Write("-")
	}
}

// Feed - send N feeds
func (e *Escpos) Feed(n int) {
	if e.Firmware >= 264 {
		e.Write(fmt.Sprintf("\x1Bd%c", n))
		e.timeoutSet(e.dotFeedTime * e.charHeight)
		e.prevByte = ASCIILF
		e.column = 0
	} else {
		for i := 0; i < n; i++ {
			e.WriteBytes([]byte{10})
		}
	}
}

// Linefeed -  send linefeed
func (e *Escpos) Linefeed() {
	if e.Verbose {
		fmt.Printf("func Linefeed()\n")
	}
	e.Feed(1)
	// byte 110
	// e.Write("\n")
	// в python version \n 10
	// e.WriteBytes([]byte{10})
}

// FormFeed -  send formfeed
func (e *Escpos) FormFeed() {
	e.Feed(1)
}

// SetBold - bold mode true/false
func (e *Escpos) SetBold(state bool) {
	if state {
		e.WriteBytes([]byte{27, 32, 1})
		e.WriteBytes([]byte{27, 69, 1})
	} else {
		e.WriteBytes([]byte{27, 32, 0})
		e.WriteBytes([]byte{27, 69, 0})
	}
}

// SetSmall - set small font true/false
func (e *Escpos) SetSmall(state bool) {
	if state {
		e.WriteBytes([]byte{27, 33, 1})
	} else {
		e.WriteBytes([]byte{27, 33, 0})
	}
}

// SetFontSize - set font size
// large/medium/normal
func (e *Escpos) SetFontSize(name string) {
	if name == "large" || name == "L" {
		e.charHeight = 48
		e.maxColumn = 16
		e.WriteBytes([]byte{29, 33, 17, 10})
	} else if name == "medium" || name == "M" {
		e.charHeight = 48
		e.maxColumn = 32
		e.WriteBytes([]byte{29, 33, 1, 10})
	} else {
		e.charHeight = 24
		e.maxColumn = 32
		e.WriteBytes([]byte{29, 33, 0, 10})
	}
}

// DoubleHeight - set double height
func (e *Escpos) DoubleHeight(state bool) {
	if state {
		e.Write(fmt.Sprintf("\x1B!%c", (1 << 4)))
	} else {
		e.Write(fmt.Sprintf("\x1B!%c", 0))
	}
}

func (e *Escpos) setBarcodeHeight(val uint8) {
	if val < 1 {
		val = 1
	} else if val > 255 {
		val = 255
	}
	e.barcodeHeight = val
	e.Write(fmt.Sprintf("\x1D\x68%c", val))
}

// BarcodeChr - 1:Abovebarcode 2:Below 3:Both 0:Not printed
func (e *Escpos) BarcodeChr(val uint8) {
	e.Write(fmt.Sprintf("\x1D\x68%c", val))
	// 		self.write(chr(29)) # Leave
	// 		self.write(chr(72)) # Leave
	// 		self.write(msg)     # Print barcode # 1:Abovebarcode 2:Below 3:Both 0:Not printed
}

// BarCode print barcode
func (e *Escpos) BarCode(code string, data string) {
	if e.Verbose {
		fmt.Printf("func BarCode()\n")
	}
	var a uint8
	switch code {
	case "UPC_A":
		a = 0
	case "UPC_E":
		a = 1
	case "UPCA":
		a = 0
	case "UPCE":
		a = 1
	case "EAN13":
		a = 2
	case "EAN8":
		a = 3
	case "CODE39":
		a = 4
	case "I25":
		a = 5
	case "CODEBAR":
		a = 6
	case "CODE93":
		a = 7
	case "CODE128":
		a = 8
	case "CODE11":
		a = 9
	case "MSI":
		a = 10
	default:
		a = 4
	}
	// Print label below barcode
	e.WriteBytes([]byte{29, 72, 2})
	// Barcode width
	e.WriteBytes([]byte{29, 119, 3})
	//  Barcode type
	e.Write(fmt.Sprintf("\x1D\x6B%c", a))
	e.timeoutWait()
	e.timeoutSet((int64(e.barcodeHeight) + 40) * e.dotPrintTime)
	e.Write(data)
	// super(Adafruit_Thermal, self).write(text)
	e.prevByte = ASCIILF
	e.Feed(2)
}

// WriteNode write a "node" to the printer
func (e *Escpos) WriteNode(data []models.Printer, set *models.BarCodeOption) {
	for _, row := range data {
		// if i%20 == 0 {
		// 	time.Sleep(1000 * time.Millisecond)
		// }
		if row.Line && len(row.Text) == 0 {
			e.LinePrint()
		} else if row.Image {
			if e.Debug {
				fmt.Println("TODO: add print image")
			}
		} else if row.BarCode {
			e.SetAlign(row.Align)
			// if len(row.Size) > 0 {
			// 	if msg, err :=  strconv.Atoi(row.Size); err == nil {
			e.BarcodeChr(set.Chr)
			e.setBarcodeHeight(set.Height)
			e.BarCode(set.Code, row.Text)
			// 		e.
			// 	}
			// }
		} else if row.QrCode {
			if e.Debug {
				fmt.Println("TODO: add print QR code")
			}
		} else {
			if row.Style == "bold" {
				e.SetBold(true)
			} else if row.Style == "small" {
				e.SetSmall(true)
			}
			if row.Size != "normal" {
				e.SetFontSize(row.Size)
			}
			e.SetAlign(row.Align)
			e.WriteText(row.Text)

			e.timeoutWait()
			e.Linefeed()
			e.timeoutWait()
			e.SetAlign("left")
			if row.Style == "bold" {
				e.SetBold(false)
			} else if row.Style == "small" {
				e.SetSmall(false)
			}
			if row.Size != "normal" {
				e.SetFontSize("normal")
			}
			if row.Line {
				e.LinePrint()
			}
			if e.Debug {
				fmt.Println(">>>>>>>>>>>>>>>>>>>>", row.Text)
			}
		}
	}
}

// -------------- TODO --------------

// func (e *Escpos) SetCharSpacing(val uint8) {
// 	if e.Verbose {
// 		fmt.Printf("func SetCharSpacing()\n")
// 	}
// 	e.Write(fmt.Sprintf("\x1B %c", val))
// }

// Init - init/reset printer settings
func (e *Escpos) Init() {
	e.reset()
	e.Write("\x1B@")
}

// End output
func (e *Escpos) End() {
	e.Write("\xFA")
}

// Cut - send cut
func (e *Escpos) Cut() {
	e.Write("\x1DVA0")
}

// Cash - send cash
func (e *Escpos) Cash() {
	e.Write("\x1B\x70\x00\x0A\xFF")
}

// SetFont - set font
func (e *Escpos) SetFont(font string) {
	f := 0

	switch font {
	case "A":
		f = 0
	case "B":
		f = 1
	case "C":
		f = 2
	default:
		log.Fatal(fmt.Sprintf("Invalid font: '%s', defaulting to 'A'", font))
		f = 0
	}

	e.Write(fmt.Sprintf("\x1BM%c", f))
}

// SendFontSize -
func (e *Escpos) SendFontSize() {
	e.Write(fmt.Sprintf("\x1D!%c", ((e.width-1)<<4)|(e.height-1)))
}

// set font size
// func (e *Escpos) SetFontSize(width, height uint8) {
// 	if width > 0 && height > 0 && width <= 8 && height <= 8 {
// 		e.width = width
// 		e.height = height
// 		e.SendFontSize()
// 	} else {
// 		log.Fatal(fmt.Sprintf("Invalid font size passed: %d x %d", width, height))
// 	}
// }

// SendUnderline - send underline
func (e *Escpos) SendUnderline() {
	e.Write(fmt.Sprintf("\x1B-%c", e.underline))
}

// SendEmphasize - send emphasize / doublestrike
func (e *Escpos) SendEmphasize() {
	e.Write(fmt.Sprintf("\x1BG%c", e.emphasize))
}

// SendUpsidedown - send upsidedown
func (e *Escpos) SendUpsidedown() {
	e.Write(fmt.Sprintf("\x1B{%c", e.upsidedown))
}

// SendRotate - send rotate
func (e *Escpos) SendRotate() {
	e.Write(fmt.Sprintf("\x1BR%c", e.rotate))
}

// SendReverse - send reverse
func (e *Escpos) SendReverse() {
	e.Write(fmt.Sprintf("\x1DB%c", e.reverse))
}

// SendSmooth - send smooth
func (e *Escpos) SendSmooth() {
	e.Write(fmt.Sprintf("\x1Db%c", e.smooth))
}

// SendMoveX - send move x
func (e *Escpos) SendMoveX(x uint16) {
	e.Write(string([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)}))
}

// SendMoveY - send move y
func (e *Escpos) SendMoveY(y uint16) {
	e.Write(string([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)}))
}

// SetUnderline - set underline
func (e *Escpos) SetUnderline(v uint8) {
	e.underline = v
	e.SendUnderline()
}

// SetEmphasize - set emphasize
func (e *Escpos) SetEmphasize(u uint8) {
	e.emphasize = u
	e.SendEmphasize()
}

// SetUpsidedown - set upsidedown
func (e *Escpos) SetUpsidedown(v uint8) {
	e.upsidedown = v
	e.SendUpsidedown()
}

// SetRotate - set rotate
func (e *Escpos) SetRotate(v uint8) {
	e.rotate = v
	e.SendRotate()
}

// SetReverse - set reverse
func (e *Escpos) SetReverse(v uint8) {
	e.reverse = v
	e.SendReverse()
}

// SetSmooth - set smooth
func (e *Escpos) SetSmooth(v uint8) {
	e.smooth = v
	e.SendSmooth()
}

// Pulse - pulse (open the drawer)
func (e *Escpos) Pulse() {
	// with t=2 -- meaning 2*2msec
	e.Write("\x1Bp\x02")
}

// SetLang - set language -- ESC R
func (e *Escpos) SetLang(lang string) {
	l := 0

	switch lang {
	case "en":
		l = 0
	case "fr":
		l = 1
	case "de":
		l = 2
	case "uk":
		l = 3
	case "da":
		l = 4
	case "sv":
		l = 5
	case "it":
		l = 6
	case "es":
		l = 7
	case "ja":
		l = 8
	case "no":
		l = 9
	default:
		log.Fatal(fmt.Sprintf("Invalid language: %s", lang))
	}
	e.Write(fmt.Sprintf("\x1BR%c", l))
}

// Text - do a block of text
func (e *Escpos) Text(params map[string]string, data string) {

	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// set lang
	if lang, ok := params["lang"]; ok {
		e.SetLang(lang)
	}

	// set smooth
	if smooth, ok := params["smooth"]; ok && (smooth == "true" || smooth == "1") {
		e.SetSmooth(1)
	}

	// set emphasize
	if em, ok := params["em"]; ok && (em == "true" || em == "1") {
		e.SetEmphasize(1)
	}

	// set underline
	if ul, ok := params["ul"]; ok && (ul == "true" || ul == "1") {
		e.SetUnderline(1)
	}

	// set reverse
	if reverse, ok := params["reverse"]; ok && (reverse == "true" || reverse == "1") {
		e.SetReverse(1)
	}

	// set rotate
	if rotate, ok := params["rotate"]; ok && (rotate == "true" || rotate == "1") {
		e.SetRotate(1)
	}

	// set font
	if font, ok := params["font"]; ok {
		e.SetFont(strings.ToUpper(font[5:6]))
	}

	// do dw (double font width)
	// if dw, ok := params["dw"]; ok && (dw == "true" || dw == "1") {
	// 	e.SetFontSize(2, e.height)
	// }

	// // do dh (double font height)
	// if dh, ok := params["dh"]; ok && (dh == "true" || dh == "1") {
	// 	e.SetFontSize(e.width, 2)
	// }

	// // do font width
	// if width, ok := params["width"]; ok {
	// 	if i, err := strconv.Atoi(width); err == nil {
	// 		e.SetFontSize(uint8(i), e.height)
	// 	} else {
	// 		log.Fatal(fmt.Sprintf("Invalid font width: %s", width))
	// 	}
	// }

	// // do font height
	// if height, ok := params["height"]; ok {
	// 	if i, err := strconv.Atoi(height); err == nil {
	// 		e.SetFontSize(e.width, uint8(i))
	// 	} else {
	// 		log.Fatal(fmt.Sprintf("Invalid font height: %s", height))
	// 	}
	// }

	// do y positioning
	if x, ok := params["x"]; ok {
		if i, err := strconv.Atoi(x); err == nil {
			e.SendMoveX(uint16(i))
		} else {
			log.Fatal("Invalid x param %d", x)
		}
	}

	// do y positioning
	if y, ok := params["y"]; ok {
		if i, err := strconv.Atoi(y); err == nil {
			e.SendMoveY(uint16(i))
		} else {
			log.Fatal("Invalid y param %d", y)
		}
	}

	// do text replace, then write data
	data = e.textReplace(data)
	if len(data) > 0 {
		e.Write(data)
	}
}

// FeedAndCut - feed and cut based on parameters
func (e *Escpos) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		e.FormFeed()
	}

	e.Cut()
}

// used to send graphics headers
func (e *Escpos) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	e.Write("\x1b(L")
	e.WriteRaw([]byte{byte(l % 256), byte(l / 256), m, fn})
	e.WriteRaw(data)
}

// Image - write an image
func (e *Escpos) Image(params map[string]string, data string) {
	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// get width
	wstr, ok := params["width"]
	if !ok {
		log.Fatal("No width specified on image")
	}

	// get height
	hstr, ok := params["height"]
	if !ok {
		log.Fatal("No height specified on image")
	}

	// convert width
	width, err := strconv.Atoi(wstr)
	if err != nil {
		log.Fatal("Invalid image width %s", wstr)
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		log.Fatal("Invalid image height %s", hstr)
	}

	// decode data frome b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Image len:%d w: %d h: %d\n", len(dec), width, height)

	// $imgHeader = self::dataHeader(array($img -> getWidth(), $img -> getHeight()), true);
	// $tone = '0';
	// $colors = '1';
	// $xm = (($size & self::IMG_DOUBLE_WIDTH) == self::IMG_DOUBLE_WIDTH) ? chr(2) : chr(1);
	// $ym = (($size & self::IMG_DOUBLE_HEIGHT) == self::IMG_DOUBLE_HEIGHT) ? chr(2) : chr(1);
	//
	// $header = $tone . $xm . $ym . $colors . $imgHeader;
	// $this -> graphicsSendData('0', 'p', $header . $img -> toRasterFormat());
	// $this -> graphicsSendData('0', '2');

	header := []byte{
		byte('0'), 0x01, 0x01, byte('1'),
	}

	a := append(header, dec...)

	e.gSend(byte('0'), byte('p'), a)
	e.gSend(byte('0'), byte('2'), []byte{})

}
