package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"
	chromaFormatters "github.com/alecthomas/chroma/v2/formatters"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"

	t "github.com/LeperGnome/bt/internal/tree"
)

type previewGenFunc = func(node *t.Node, dim Dimentions, style Stylesheet) string

const (
	tgpChunkSize = 2048
	// tgpClearMedia = "\033_Ga=d\033\\" // TODO: TGP support
)

func GeneratePreview(node *t.Node, dim Dimentions, style Stylesheet, glowEnabled bool, chromaEnabled bool, chromaStyle string) string {
	parts := strings.Split(node.Info.Name(), ".")
	ext := strings.ToLower(parts[len(parts)-1])

	// Image extensions handled by half-block renderer
	switch ext {
	case "png", "jpg", "jpeg", "gif":
		return genHalfBlockPreview(node, dim, style)
	}

	// Read file content once for text-based renderers
	buf := make([]byte, previewTextBytesLimit)
	n, err := node.ReadContent(buf, previewTextBytesLimit)
	if err != nil {
		return ""
	}
	content := buf[:n]

	if !utf8.Valid(content) {
		contentStyle := style.PlainTextPreview.MaxWidth(dim.Width - 1)
		return contentStyle.Render(binaryContentPlaceholder)
	}

	contentStr := string(content)

	// Glow (glamour) rendering for markdown/text files
	if glowEnabled {
		switch ext {
		case "md", "txt", "markdown":
			rendered, err := glamourRender(contentStr, dim.Width)
			if err == nil {
				return rendered
			}
			// fall through to plain text on error
		}
	}

	// Chroma syntax highlighting for code files
	if chromaEnabled {
		rendered, err := chromaRender(contentStr, node.Path, chromaStyle)
		if err == nil {
			return rendered
		}
		// fall through to plain text on error
	}

	// Plain text fallback
	contentStyle := style.PlainTextPreview.MaxWidth(dim.Width - 1)
	contentLines := strings.Split(contentStr, "\n")
	return contentStyle.Render(strings.Join(contentLines, "\n"))
}

func glamourRender(content string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("light"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	rendered, err := r.Render(content)
	if err != nil {
		return "", err
	}
	return rendered, nil
}

func chromaRender(content, path, chromaStyle string) (string, error) {
	lexer := chromaLexers.Match(path)
	if lexer == nil {
		return "", fmt.Errorf("no lexer found for %s", path)
	}

	style := chromaStyles.Get(chromaStyle)
	if style == nil {
		style = chromaStyles.Fallback
	}

	formatter := chromaFormatters.Get("terminal256")
	if formatter == nil {
		return "", fmt.Errorf("terminal256 formatter not available")
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func genHalfBlockPreview(node *t.Node, dim Dimentions, style Stylesheet) string {
	f, err := os.Open(node.Path)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	preview := imageHalfBlockRepr(f, dim.Height, dim.Width)
	return preview
}

func imageHalfBlockRepr(r io.Reader, heightSymbols, widthSymbols int) string {
	img, _, err := image.Decode(r)
	if err != nil {
		return err.Error()
	}

	// Fitting image into space we have.

	heightHb := heightSymbols - 1 // For bottom resolution output.
	heightHb *= 2                 // Due to most fonts w/h ratio.
	widthHb := widthSymbols

	res := []string{}

	pxWidth := img.Bounds().Max.X
	pxHeight := img.Bounds().Max.Y

	sectorWidthRatio := int(math.Ceil(float64(pxWidth) / float64(widthHb)))
	sectorHeightRatio := int(math.Ceil(float64(pxHeight) / float64(heightHb)))

	sectorSizePx := max(sectorWidthRatio, sectorHeightRatio)

	widthSectors := pxWidth / sectorSizePx
	heightSectors := pxHeight / sectorSizePx

	if heightSectors&1 == 1 {
		heightSectors -= 1
	}

	pxCnt := uint32(sectorSizePx * sectorSizePx)

	// Loop through sectors.
	for sy := 0; sy < heightSectors; sy += 2 {
		line := []string{}
		for sx := range widthSectors {
			// Loop through pixels within sector.

			// Caclucating mean RGB for top part of symbol (background for half-block).
			sumsTop := [3]uint32{0, 0, 0}
			for y := sy * sectorSizePx; y < (sy+1)*sectorSizePx; y++ {
				for x := sx * (sectorSizePx); x < (sx+1)*sectorSizePx; x++ {
					c := img.At(x, y)
					r, g, b, _ := c.RGBA()
					sumsTop[0] += r
					sumsTop[1] += g
					sumsTop[2] += b
				}
			}
			meansTop := [3]uint32{sumsTop[0] / pxCnt, sumsTop[1] / pxCnt, sumsTop[2] / pxCnt}

			// Caclucating mean RGB for bottom part of symbol (foreground for half-block).
			sumsBottom := [3]uint32{0, 0, 0}
			for y := (sy + 1) * sectorSizePx; y < (sy+2)*sectorSizePx; y++ {
				for x := sx * (sectorSizePx); x < (sx+1)*sectorSizePx; x++ {
					c := img.At(x, y)
					r, g, b, _ := c.RGBA()
					sumsBottom[0] += r
					sumsBottom[1] += g
					sumsBottom[2] += b
				}
			}
			meansBottom := [3]uint32{sumsBottom[0] / pxCnt, sumsBottom[1] / pxCnt, sumsBottom[2] / pxCnt}

			line = append(line, fmt.Sprintf(
				"\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm▄\x1b[0m",
				meansTop[0]>>8,
				meansTop[1]>>8,
				meansTop[2]>>8,

				meansBottom[0]>>8,
				meansBottom[1]>>8,
				meansBottom[2]>>8,
			))
		}
		res = append(res, strings.Join(line, ""))
	}
	res = append(res, fmt.Sprintf("%dx%d", pxWidth, pxHeight))
	return strings.Join(res, "\n")
}

func genPlainTextPreview(node *t.Node, dim Dimentions, style Stylesheet) string {
	buf := make([]byte, previewTextBytesLimit)
	n, err := node.ReadContent(buf, previewTextBytesLimit)
	if err != nil {
		return ""
	}
	content := buf[:n]

	contentStyle := style.PlainTextPreview.MaxWidth(dim.Width - 1) // -1 for border...

	var contentLines []string
	if !utf8.Valid(content) {
		contentLines = []string{binaryContentPlaceholder}
	} else {
		contentLines = strings.Split(string(content), "\n")
	}

	return contentStyle.Render(strings.Join(contentLines, "\n"))
}

// TODO: TGP support
func genTPGPreviewPNG(node *t.Node, dim Dimentions, _ Stylesheet) string {
	preview, err := tgpDirectChunks(node.Path, dim.Height, dim.Width)
	if err != nil {
		return err.Error()
	}
	return preview
}

// TODO: TGP support
func tgpDirectChunks(path string, height, width int) (string, error) {
	res := []string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	data64 := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(data64, data)

	chunk := 0

	for {
		if chunk == 0 && (chunk+1)*tgpChunkSize < len(data64) {
			chunkData := fmt.Sprintf(
				"\033_Ga=T,C=1,f=100,r=%d,c=%d,m=1;%s\033\\",
				height-1, width-1,
				string(data64[chunk*tgpChunkSize:(chunk+1)*tgpChunkSize]),
			)
			res = append(res, chunkData)
		} else if (chunk+1)*tgpChunkSize < len(data64) {
			chunkData := fmt.Sprintf(
				"\033_Gm=1;%s\033\\",
				string(data64[chunk*tgpChunkSize:(chunk+1)*tgpChunkSize]),
			)
			res = append(res, chunkData)
		} else if chunk == 0 {
			chunkData := fmt.Sprintf(
				"\033_Ga=T,C=1,f=100;%s\033\\",
				string(data64),
			)
			res = append(res, chunkData)
			break
		} else {
			chunkData := fmt.Sprintf(
				"\033_Gm=0;%s\033\\",
				string(data64[chunk*tgpChunkSize:]),
			)
			res = append(res, chunkData)
			break
		}
		chunk++
	}

	return strings.Join(res, ""), nil
}
