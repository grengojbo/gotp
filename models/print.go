package models

import (
	"fmt"
	"os"

	"github.com/antonholmquist/jason"
)

// Printer - params line
type Printer struct {
	Line    bool   `json:"line"`
	Align   string `json:"align"`
	Style   string `json:"style"`
	Size    string `json:"size"`
	Text    string `json:"text"`
	Image   bool   `json:"image"`
	BarCode bool   `json:"barCode"`
	QrCode  bool   `json:"qrCode"`
}

// PrinterLine - print collection
type PrinterLine struct {
	Header  []Printer     `json:"header"`
	Lines   []Printer     `json:"lines"`
	Footer  []Printer     `json:"footer"`
	BarCode BarCodeOption `json:"barCode"`
}

// BarCodeOption - print option for bar code
type BarCodeOption struct {
	Height uint8  `json:"height"`
	Chr    uint8  `json:"chr"`
	Code   string `json:"code"`
}

// LoadPrintModel - lading model
func LoadPrintModel(file string) (res PrinterLine, err error) {
	f, err := os.Open(file)
	if err != nil {
		return res, fmt.Errorf("Load file: %s", err.Error())
	}
	v, _ := jason.NewObjectFromReader(f)
	header, _ := v.GetObjectArray("header")
	lines, _ := v.GetObjectArray("lines")
	footer, _ := v.GetObjectArray("footer")
	b, _ := v.GetObject("barCode")
	height, _ := b.GetInt64("height")
	chr, _ := b.GetInt64("chr")
	code, _ := b.GetString("code")
	res.BarCode.Height = uint8(height)
	res.BarCode.Chr = uint8(chr)
	res.BarCode.Code = code

	for _, row := range header {
		line, _ := row.GetBoolean("line")
		image, _ := row.GetBoolean("image")
		barCode, _ := row.GetBoolean("barCode")
		qrCode, _ := row.GetBoolean("qrCode")
		align, _ := row.GetString("align")
		style, _ := row.GetString("style")
		size, _ := row.GetString("size")
		text, _ := row.GetString("text")
		r := Printer{
			Line:    line,
			Image:   image,
			BarCode: barCode,
			QrCode:  qrCode,
			Align:   align,
			Style:   style,
			Size:    size,
			Text:    text,
		}
		res.Header = append(res.Header, r)
	}
	for _, row := range lines {
		// fmt.Println(row)
		line, _ := row.GetBoolean("line")
		image, _ := row.GetBoolean("image")
		barCode, _ := row.GetBoolean("barCode")
		qrCode, _ := row.GetBoolean("qrCode")
		align, _ := row.GetString("align")
		style, _ := row.GetString("style")
		size, _ := row.GetString("size")
		text, _ := row.GetString("text")
		r := Printer{
			Line:    line,
			Image:   image,
			BarCode: barCode,
			QrCode:  qrCode,
			Align:   align,
			Style:   style,
			Size:    size,
			Text:    text,
		}
		res.Lines = append(res.Lines, r)
	}
	for _, row := range footer {
		line, _ := row.GetBoolean("line")
		image, _ := row.GetBoolean("image")
		barCode, _ := row.GetBoolean("barCode")
		qrCode, _ := row.GetBoolean("qrCode")
		align, _ := row.GetString("align")
		style, _ := row.GetString("style")
		size, _ := row.GetString("size")
		text, _ := row.GetString("text")
		r := Printer{
			Line:    line,
			Image:   image,
			BarCode: barCode,
			QrCode:  qrCode,
			Align:   align,
			Style:   style,
			Size:    size,
			Text:    text,
		}
		res.Footer = append(res.Footer, r)
	}
	return res, err
}
