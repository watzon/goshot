// Package goshot composes beautiful screenshots of code and terminal
// output: content is rendered to an image, wrapped in window chrome, and
// placed on a backdrop.
//
//	img, err := goshot.New().
//		WithContent(code.New(source).WithTheme("dracula")).
//		WithChrome(chrome.Mac().WithTitle("main.go").Dark()).
//		WithBackground(background.Solid(color.RGBA{30, 30, 46, 255})).
//		Image()
package goshot

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
)

// Content produces the innermost image — highlighted code, terminal
// output, or anything else.
type Content interface {
	Render() (image.Image, error)
}

// Layer wraps an image in additional imagery. Window chrome and
// backgrounds are layers.
type Layer interface {
	Render(image.Image) (image.Image, error)
}

// Canvas is a render pipeline: content, then chrome, then background.
// Every part is optional, but at least one must be set.
type Canvas struct {
	content    Content
	chrome     Layer
	background Layer
}

// New creates an empty canvas.
func New() *Canvas { return &Canvas{} }

// WithContent sets the content to render.
func (c *Canvas) WithContent(content Content) *Canvas { c.content = content; return c }

// WithChrome wraps the content in window decorations.
func (c *Canvas) WithChrome(l Layer) *Canvas { c.chrome = l; return c }

// WithBackground places the result on a backdrop.
func (c *Canvas) WithBackground(l Layer) *Canvas { c.background = l; return c }

// Image renders the full pipeline.
func (c *Canvas) Image() (image.Image, error) {
	if c.content == nil && c.chrome == nil && c.background == nil {
		return nil, fmt.Errorf("goshot: canvas is empty")
	}

	var img image.Image
	var err error
	if c.content != nil {
		if img, err = c.content.Render(); err != nil {
			return nil, err
		}
	}
	for _, layer := range []Layer{c.chrome, c.background} {
		if layer == nil {
			continue
		}
		if img, err = layer.Render(img); err != nil {
			return nil, err
		}
	}
	return img, nil
}

// Save renders the canvas and writes it to path, picking the format from
// the file extension (.png, .jpg/.jpeg, or .bmp).
func (c *Canvas) Save(path string) error {
	img, err := c.Image()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Encode(f, img, strings.TrimPrefix(filepath.Ext(path), "."))
}

// Encode writes an image in the named format (png, jpg/jpeg, or bmp).
func Encode(w io.Writer, img image.Image, format string) error {
	switch strings.ToLower(format) {
	case "png", "":
		return png.Encode(w, img)
	case "jpg", "jpeg":
		return jpeg.Encode(w, img, nil)
	case "bmp":
		return bmp.Encode(w, img)
	}
	return fmt.Errorf("goshot: unsupported format %q", format)
}
