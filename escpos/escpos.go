package escpos

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
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
func textReplace(data string) string {
	for k, v := range textReplaceMap {
		data = strings.Replace(data, k, v, -1)
	}
	return data
}

type Escpos struct {
	// destination
	dst io.Writer

	// font metrics
	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	prevByte      string
	column        uint8
	maxColumn     uint8
	charHeight    uint8
	lineSpacing   uint8
	barcodeHeight uint8

	printDensity   uint8
	printBreakTime uint8
	// state toggles GS[char]
	reverse, smooth uint8

	resumeTime     uint64
	dotPrintTime   uint64
	dotFeedTime    uint64
	maxChunkHeight uint8

	Verbose bool
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

	e.prevByte = "\n"
	e.column = 0
	e.maxColumn = 32
	e.charHeight = 24
	e.lineSpacing = 6
	e.barcodeHeight = 50
	e.printDensity = 10

	// #if PRINTER_FIRMWARE >= 264
	//  // Configure tab stops on recent printers
	// Set tab stops...
	e.Write("\x1BD")
	e.WriteBytes([]byte{4, 8, 12, 16})  // ...every 4 columns,
	e.WriteBytes([]byte{20, 24, 28, 0}) // 0 marks end-of-list.
	// #endif
}

// create Escpos printer
func New(dst io.Writer) (e *Escpos) {
	e = &Escpos{dst: dst}
	e.printDensity = 10
	e.printBreakTime = 2
	e.reset()
	return
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

func (e *Escpos) WriteBytes(data []byte) {
	e.timeoutWait()
	if e.Verbose {
		fmt.Println(data)
	}
	e.dst.Write(data)
	// e.timeoutSet(2 * BYTE_TIME);
}

// write raw bytes to printer
func (e *Escpos) WriteRaw(data []byte) (n int, err error) {
	if len(data) > 0 {
		e.timeoutWait()
		if e.Verbose {
			fmt.Printf("Writing %d bytes\n", len(data))
			fmt.Println(data)
		}
		e.dst.Write(data)
		// e.timeoutSet(BYTE_TIME);
	} else {
		if e.Verbose {
			fmt.Printf("Wrote NO bytes\n")
		}
	}
	return 0, nil
}

// write a string to the printer
func (e *Escpos) Write(data string) (int, error) {
	// if e.Verbose {
	// 	fmt.Printf("func Write()\n")
	// }
	return e.WriteRaw([]byte(data))
}

func (e *Escpos) timeoutSet(x uint64) {
	// if(!dtrEnabled) resumeTime = micros() + x;
	e.resumeTime = x
}

func (e *Escpos) timeoutWait() {
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
	// #if PRINTER_FIRMWARE >= 264
	//   delay(50);
	//   writeBytes(ASCII_ESC, '8', 0, 0); // Sleep off (important!)
	// #else
	//   // Datasheet recommends a 50 mS delay before issuing further commands,
	//   // but in practice this alone isn't sufficient (e.g. text size/style
	//   // commands may still be misinterpreted on wake).  A slightly longer
	//   // delay, interspersed with NUL chars (no-ops) seems to help.
	//   for(uint8_t i=0; i<10; i++) {
	//     writeBytes(0);
	//     timeoutSet(10000L);
	//   }
	// #endif
}

// func (e *Escpos) Begin(heatTime uint8) {
func (e *Escpos) Begin() {
	// The printer can't start receiving data immediately upon power up --
	// it needs a moment to cold boot and initialize.  Allow at least 1/2
	// sec of uptime before printer can receive data.
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
	e.Write("\x12#\x4A")

	// Enable DTR pin if requested
	// if(dtrPin < 255) {
	//   pinMode(dtrPin, INPUT_PULLUP);
	//   writeBytes(ASCII_GS, 'a', (1 << 5));
	// e.Write("\x1Da\x20")
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

// init/reset printer settings
func (e *Escpos) Init() {
	e.reset()
	e.Write("\x1B@")
}

// end output
func (e *Escpos) End() {
	e.Write("\xFA")
}

// send cut
func (e *Escpos) Cut() {
	e.Write("\x1DVA0")
}

// send cash
func (e *Escpos) Cash() {
	e.Write("\x1B\x70\x00\x0A\xFF")
}

// send linefeed
func (e *Escpos) Linefeed() {
	e.Write("\n")
}

// send N formfeeds
func (e *Escpos) FormfeedN(n int) {
	e.Write(fmt.Sprintf("\x1Bd%c", 1))
}

// send formfeed
func (e *Escpos) Formfeed() {
	e.FormfeedN(1)
}

// set font
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

func (e *Escpos) SendFontSize() {
	e.Write(fmt.Sprintf("\x1D!%c", ((e.width-1)<<4)|(e.height-1)))
}

// set font size
func (e *Escpos) SetFontSize(width, height uint8) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		e.width = width
		e.height = height
		e.SendFontSize()
	} else {
		log.Fatal(fmt.Sprintf("Invalid font size passed: %d x %d", width, height))
	}
}

// send underline
func (e *Escpos) SendUnderline() {
	e.Write(fmt.Sprintf("\x1B-%c", e.underline))
}

// send emphasize / doublestrike
func (e *Escpos) SendEmphasize() {
	e.Write(fmt.Sprintf("\x1BG%c", e.emphasize))
}

// send upsidedown
func (e *Escpos) SendUpsidedown() {
	e.Write(fmt.Sprintf("\x1B{%c", e.upsidedown))
}

// send rotate
func (e *Escpos) SendRotate() {
	e.Write(fmt.Sprintf("\x1BR%c", e.rotate))
}

// send reverse
func (e *Escpos) SendReverse() {
	e.Write(fmt.Sprintf("\x1DB%c", e.reverse))
}

// send smooth
func (e *Escpos) SendSmooth() {
	e.Write(fmt.Sprintf("\x1Db%c", e.smooth))
}

// send move x
func (e *Escpos) SendMoveX(x uint16) {
	e.Write(string([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)}))
}

// send move y
func (e *Escpos) SendMoveY(y uint16) {
	e.Write(string([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)}))
}

// set underline
func (e *Escpos) SetUnderline(v uint8) {
	e.underline = v
	e.SendUnderline()
}

// set emphasize
func (e *Escpos) SetEmphasize(u uint8) {
	e.emphasize = u
	e.SendEmphasize()
}

// set upsidedown
func (e *Escpos) SetUpsidedown(v uint8) {
	e.upsidedown = v
	e.SendUpsidedown()
}

// set rotate
func (e *Escpos) SetRotate(v uint8) {
	e.rotate = v
	e.SendRotate()
}

// set reverse
func (e *Escpos) SetReverse(v uint8) {
	e.reverse = v
	e.SendReverse()
}

// set smooth
func (e *Escpos) SetSmooth(v uint8) {
	e.smooth = v
	e.SendSmooth()
}

// pulse (open the drawer)
func (e *Escpos) Pulse() {
	// with t=2 -- meaning 2*2msec
	e.Write("\x1Bp\x02")
}

// set alignment
func (e *Escpos) SetAlign(align string) {
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	default:
		log.Fatal(fmt.Sprintf("Invalid alignment: %s", align))
	}
	e.Write(fmt.Sprintf("\x1Ba%c", a))
}

// set language -- ESC R
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

// do a block of text
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
	if dw, ok := params["dw"]; ok && (dw == "true" || dw == "1") {
		e.SetFontSize(2, e.height)
	}

	// do dh (double font height)
	if dh, ok := params["dh"]; ok && (dh == "true" || dh == "1") {
		e.SetFontSize(e.width, 2)
	}

	// do font width
	if width, ok := params["width"]; ok {
		if i, err := strconv.Atoi(width); err == nil {
			e.SetFontSize(uint8(i), e.height)
		} else {
			log.Fatal(fmt.Sprintf("Invalid font width: %s", width))
		}
	}

	// do font height
	if height, ok := params["height"]; ok {
		if i, err := strconv.Atoi(height); err == nil {
			e.SetFontSize(e.width, uint8(i))
		} else {
			log.Fatal(fmt.Sprintf("Invalid font height: %s", height))
		}
	}

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
	data = textReplace(data)
	if len(data) > 0 {
		e.Write(data)
	}
}

// feed the printer
func (e *Escpos) Feed(params map[string]string) {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			e.FormfeedN(i)
		} else {
			log.Fatal(fmt.Sprintf("Invalid line number %d", l))
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			e.SendMoveY(uint16(i))
		} else {
			log.Fatal(fmt.Sprintf("Invalid unit number %d", u))
		}
	}

	// send linefeed
	e.Linefeed()

	// reset variables
	e.reset()

	// reset printer
	e.SendEmphasize()
	e.SendRotate()
	e.SendSmooth()
	e.SendReverse()
	e.SendUnderline()
	e.SendUpsidedown()
	e.SendFontSize()
	e.SendUnderline()
}

// feed and cut based on parameters
func (e *Escpos) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		e.Formfeed()
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

// write an image
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

// write a "node" to the printer
func (e *Escpos) WriteNode(name string, params map[string]string, data string) {
	cstr := ""
	if data != "" {
		str := data[:]
		if len(data) > 40 {
			str = fmt.Sprintf("%s ...", data[0:40])
		}
		cstr = fmt.Sprintf(" => '%s'", str)
	}
	log.Printf("Write: %s => %+v%s\n", name, params, cstr)

	switch name {
	case "text":
		e.Text(params, data)
	case "feed":
		e.Feed(params)
	case "cut":
		e.FeedAndCut(params)
	case "pulse":
		e.Pulse()
	case "image":
		e.Image(params, data)
	}
}
