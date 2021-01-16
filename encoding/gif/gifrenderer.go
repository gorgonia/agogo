package gif

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"math"
	"strings"

	"github.com/golang/freetype/truetype"
	"github.com/gorgonia/agogo/game"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/math/fixed"
)

var tt font.Face
var regular *truetype.Font

const (
	dpi             = 144.0
	fontsize        = 12.0
	lineheight      = 1.2
	dummyLongString = `Epoch 100000, Game Number: 10000`
)

func init() {
	var err error
	if regular, err = truetype.Parse(gomono.TTF); err != nil {
		panic(err)
	}

	tt = truetype.NewFace(regular, &truetype.Options{
		Size:    fontsize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

var globPalette = color.Palette{
	color.Gray{0},
	color.Gray{253},
}

// Encoder is a structure that encodes a game state according to the agogo.OutputEncoder interface
type Encoder struct {
	H, W int
	font.Drawer

	out *gif.GIF
	io.Writer
	face font.Face

	maxH, maxW  int // maxHeight and maxWidth
	padH, padW  int // padding so everything don't start at the topleft
	fontsize    float64
	initialized bool
}

// NewGifEncoder with height and width
func NewGifEncoder(h, w int) *Encoder {
	return &Encoder{
		H:    -1,
		W:    -1,
		maxH: h,
		maxW: w,
		padH: 10,
		padW: 10,

		Drawer: font.Drawer{
			Src: image.Black,
		},
		out: &gif.GIF{LoopCount: -1},
	}
}

// Encode a game
func (enc *Encoder) Encode(ms game.MetaState) error {
	g := ms.State()
	gameNum := ms.GameNumber()
	gameName := ms.Name()
	epoch := ms.Epoch()
	repr := fmt.Sprintf("%s", g)

	if !enc.initialized {
		// lazy init of specifications
		enc.face = truetype.NewFace(regular, &truetype.Options{
			Size:    fontsize,
			DPI:     dpi,
			Hinting: font.HintingFull,
		})
		enc.Drawer.Src = image.Black
		enc.Drawer.Face = enc.face

		// first calculate how long the max length will be
		splits := strings.Split(repr, "\n")
		oneline := splits[0]
		maxW := maxInt(font.MeasureString(enc.Face, oneline).Ceil(), font.MeasureString(enc.Face, dummyLongString).Ceil())
		dy := int(math.Ceil(fontsize * lineheight * dpi / 72))
		w := maxW + 2*enc.padW
		h := (len(splits)+3)*dy + 2*enc.padH // + 3 is for the 3 extra lines: game name, state, and winner

		w = minInt(w, enc.maxW)
		h = minInt(h, enc.maxH)

		if w == enc.maxW {
			enc.padW = 0
		}
		if h == enc.maxH {
			enc.padH = 0
		}

		enc.H = h
		enc.W = w
		enc.initialized = true
	}

	x := 0
	y := 0

	bg := image.White
	im := image.NewPaletted(image.Rect(0, 0, enc.W, enc.H), globPalette)
	draw.Draw(im, im.Bounds(), bg, image.ZP, draw.Src)
	dy := int(math.Ceil(fontsize * lineheight * dpi / 72))
	enc.Dot = fixed.Point26_6{
		X: fixed.I(x + enc.padW),
		Y: fixed.I(y + enc.padH),
	}
	y += dy
	text := strings.Split(repr, "\n")
	enc.Dst = im
	for _, s := range text {
		enc.Dot = fixed.P(0+enc.padW, y)
		enc.DrawString(s)
		y += dy
	}
	enc.Dot = fixed.P(0+enc.padW, y)
	enc.DrawString(gameName)
	y += dy

	enc.Dot = fixed.P(0+enc.padW, y)
	enc.DrawString(fmt.Sprintf("Epoch %d, Game Number: %d ", epoch, gameNum))
	y += dy

	var delay int
	if ok, winner := g.Ended(); ok {
		delay = 300
		enc.Dot = fixed.P(0+enc.padW, y)
		enc.DrawString(fmt.Sprintf("Winner: %s", winner))
	}
	enc.out.Image = append(enc.out.Image, im)
	enc.out.Delay = append(enc.out.Delay, delay)
	return nil
}

// Flush writes the gif into the writer
func (enc *Encoder) Flush() error { return gif.EncodeAll(enc.Writer, enc.out) }
