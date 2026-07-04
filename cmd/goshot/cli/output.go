package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/atotto/clipboard"
	"github.com/watzon/goshot"
)

// output renders the canvas and delivers it to every requested target.
func output(canvas *goshot.Canvas, err error) error {
	if err != nil {
		return err
	}
	img, err := canvas.Image()
	if err != nil {
		return err
	}

	delivered := false
	if opts.toClipboard {
		var buf bytes.Buffer
		if err := goshot.Encode(&buf, img, "png"); err != nil {
			return err
		}
		if err := clipboard.WriteAll(buf.String()); err != nil {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
		fmt.Fprintln(os.Stderr, "copied image to clipboard")
		delivered = true
	}
	if opts.toStdout {
		if err := goshot.Encode(os.Stdout, img, "png"); err != nil {
			return err
		}
		delivered = true
	}
	if opts.output != "" && (!delivered || flagChanged("output")) {
		path, err := expandFilename(opts.output)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		format := strings.TrimPrefix(filepath.Ext(path), ".")
		if err := goshot.Encode(f, img, format); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "wrote", path)
	}
	return nil
}

// templateData is what filename and prompt templates can reference.
type templateData struct {
	User     string
	Host     string
	Path     string
	Command  string
	Filename string
	FileBase string
	FileExt  string
}

func newTemplateData(command string) templateData {
	d := templateData{Command: command, Filename: "stdin", FileBase: "goshot"}
	if u, err := user.Current(); err == nil {
		d.User = u.Username
	}
	d.Host, _ = os.Hostname()
	d.Path, _ = os.Getwd()
	switch {
	case opts.input != "":
		d.Filename = filepath.Base(opts.input)
		d.FileExt = filepath.Ext(opts.input)
		d.FileBase = strings.TrimSuffix(d.Filename, d.FileExt)
	case command != "":
		d.Filename = sanitize(command)
	case opts.fromClipboard:
		d.Filename = "clipboard"
	}
	return d
}

// expandFilename runs the output path through text/template and expands
// ~ and environment variables.
func expandFilename(path string) (string, error) {
	t, err := template.New("filename").Funcs(template.FuncMap{
		"formatDate": func(layout string) string { return time.Now().Format(layout) },
	}).Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid output template: %w", err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, newTemplateData(strings.Join(opts.args, " "))); err != nil {
		return "", fmt.Errorf("output template: %w", err)
	}
	out := os.ExpandEnv(buf.String())
	if strings.HasPrefix(out, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			out = filepath.Join(home, out[2:])
		}
	}
	if filepath.Ext(out) == "" {
		out += ".png"
	}
	return out, nil
}

// promptFunc renders the prompt template with [command] and template
// variables substituted.
func promptFunc(tmpl string) func(command string) string {
	return func(command string) string {
		tmpl := strings.ReplaceAll(tmpl, "[command]", command)
		t, err := template.New("prompt").Parse(tmpl)
		if err != nil {
			return tmpl
		}
		var buf strings.Builder
		if err := t.Execute(&buf, newTemplateData(command)); err != nil {
			return tmpl
		}
		return buf.String()
	}
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/ :\"<>|?*`, r) {
			return '_'
		}
		return r
	}, s)
}
