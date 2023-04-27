// Code generated by 'yaegi extract image'. DO NOT EDIT.

//go:build go1.18 && !go1.19 && stdimage
// +build go1.18,!go1.19,stdimage

package stdlib

import (
	"image"
	"image/color"
	"reflect"
)

func init() {
	Symbols["image/image"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Black":                  reflect.ValueOf(&image.Black).Elem(),
		"Decode":                 reflect.ValueOf(image.Decode),
		"DecodeConfig":           reflect.ValueOf(image.DecodeConfig),
		"ErrFormat":              reflect.ValueOf(&image.ErrFormat).Elem(),
		"NewAlpha":               reflect.ValueOf(image.NewAlpha),
		"NewAlpha16":             reflect.ValueOf(image.NewAlpha16),
		"NewCMYK":                reflect.ValueOf(image.NewCMYK),
		"NewGray":                reflect.ValueOf(image.NewGray),
		"NewGray16":              reflect.ValueOf(image.NewGray16),
		"NewNRGBA":               reflect.ValueOf(image.NewNRGBA),
		"NewNRGBA64":             reflect.ValueOf(image.NewNRGBA64),
		"NewNYCbCrA":             reflect.ValueOf(image.NewNYCbCrA),
		"NewPaletted":            reflect.ValueOf(image.NewPaletted),
		"NewRGBA":                reflect.ValueOf(image.NewRGBA),
		"NewRGBA64":              reflect.ValueOf(image.NewRGBA64),
		"NewUniform":             reflect.ValueOf(image.NewUniform),
		"NewYCbCr":               reflect.ValueOf(image.NewYCbCr),
		"Opaque":                 reflect.ValueOf(&image.Opaque).Elem(),
		"Pt":                     reflect.ValueOf(image.Pt),
		"Rect":                   reflect.ValueOf(image.Rect),
		"RegisterFormat":         reflect.ValueOf(image.RegisterFormat),
		"Transparent":            reflect.ValueOf(&image.Transparent).Elem(),
		"White":                  reflect.ValueOf(&image.White).Elem(),
		"YCbCrSubsampleRatio410": reflect.ValueOf(image.YCbCrSubsampleRatio410),
		"YCbCrSubsampleRatio411": reflect.ValueOf(image.YCbCrSubsampleRatio411),
		"YCbCrSubsampleRatio420": reflect.ValueOf(image.YCbCrSubsampleRatio420),
		"YCbCrSubsampleRatio422": reflect.ValueOf(image.YCbCrSubsampleRatio422),
		"YCbCrSubsampleRatio440": reflect.ValueOf(image.YCbCrSubsampleRatio440),
		"YCbCrSubsampleRatio444": reflect.ValueOf(image.YCbCrSubsampleRatio444),
		"ZP":                     reflect.ValueOf(&image.ZP).Elem(),
		"ZR":                     reflect.ValueOf(&image.ZR).Elem(),

		// type definitions
		"Alpha":               reflect.ValueOf((*image.Alpha)(nil)),
		"Alpha16":             reflect.ValueOf((*image.Alpha16)(nil)),
		"CMYK":                reflect.ValueOf((*image.CMYK)(nil)),
		"Config":              reflect.ValueOf((*image.Config)(nil)),
		"Gray":                reflect.ValueOf((*image.Gray)(nil)),
		"Gray16":              reflect.ValueOf((*image.Gray16)(nil)),
		"Image":               reflect.ValueOf((*image.Image)(nil)),
		"NRGBA":               reflect.ValueOf((*image.NRGBA)(nil)),
		"NRGBA64":             reflect.ValueOf((*image.NRGBA64)(nil)),
		"NYCbCrA":             reflect.ValueOf((*image.NYCbCrA)(nil)),
		"Paletted":            reflect.ValueOf((*image.Paletted)(nil)),
		"PalettedImage":       reflect.ValueOf((*image.PalettedImage)(nil)),
		"Point":               reflect.ValueOf((*image.Point)(nil)),
		"RGBA":                reflect.ValueOf((*image.RGBA)(nil)),
		"RGBA64":              reflect.ValueOf((*image.RGBA64)(nil)),
		"RGBA64Image":         reflect.ValueOf((*image.RGBA64Image)(nil)),
		"Rectangle":           reflect.ValueOf((*image.Rectangle)(nil)),
		"Uniform":             reflect.ValueOf((*image.Uniform)(nil)),
		"YCbCr":               reflect.ValueOf((*image.YCbCr)(nil)),
		"YCbCrSubsampleRatio": reflect.ValueOf((*image.YCbCrSubsampleRatio)(nil)),

		// interface wrapper definitions
		"_Image":         reflect.ValueOf((*_image_Image)(nil)),
		"_PalettedImage": reflect.ValueOf((*_image_PalettedImage)(nil)),
		"_RGBA64Image":   reflect.ValueOf((*_image_RGBA64Image)(nil)),
	}
}

// _image_Image is an interface wrapper for Image type
type _image_Image struct {
	IValue      interface{}
	WAt         func(x int, y int) color.Color
	WBounds     func() image.Rectangle
	WColorModel func() color.Model
}

func (W _image_Image) At(x int, y int) color.Color {
	return W.WAt(x, y)
}
func (W _image_Image) Bounds() image.Rectangle {
	return W.WBounds()
}
func (W _image_Image) ColorModel() color.Model {
	return W.WColorModel()
}

// _image_PalettedImage is an interface wrapper for PalettedImage type
type _image_PalettedImage struct {
	IValue        interface{}
	WAt           func(x int, y int) color.Color
	WBounds       func() image.Rectangle
	WColorIndexAt func(x int, y int) uint8
	WColorModel   func() color.Model
}

func (W _image_PalettedImage) At(x int, y int) color.Color {
	return W.WAt(x, y)
}
func (W _image_PalettedImage) Bounds() image.Rectangle {
	return W.WBounds()
}
func (W _image_PalettedImage) ColorIndexAt(x int, y int) uint8 {
	return W.WColorIndexAt(x, y)
}
func (W _image_PalettedImage) ColorModel() color.Model {
	return W.WColorModel()
}

// _image_RGBA64Image is an interface wrapper for RGBA64Image type
type _image_RGBA64Image struct {
	IValue      interface{}
	WAt         func(x int, y int) color.Color
	WBounds     func() image.Rectangle
	WColorModel func() color.Model
	WRGBA64At   func(x int, y int) color.RGBA64
}

func (W _image_RGBA64Image) At(x int, y int) color.Color {
	return W.WAt(x, y)
}
func (W _image_RGBA64Image) Bounds() image.Rectangle {
	return W.WBounds()
}
func (W _image_RGBA64Image) ColorModel() color.Model {
	return W.WColorModel()
}
func (W _image_RGBA64Image) RGBA64At(x int, y int) color.RGBA64 {
	return W.WRGBA64At(x, y)
}
