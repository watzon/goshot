// Automatic redaction of secrets, in block and blur styles.
package main

import (
	"image/color"
	"log"
	"strings"

	"github.com/watzon/goshot"
	"github.com/watzon/goshot/background"
	"github.com/watzon/goshot/chrome"
	"github.com/watzon/goshot/code"
)

// The fake key is assembled at runtime so secret scanners don't flag
// this file; it still matches the redaction patterns when rendered.
var source = `config := map[string]string{
	"api_key":  "` + "sk_live_" + strings.Repeat("x4", 16) + `",
	"password": "hunter2hunter2",
	"database": "postgres://admin:s3cr3t@db.internal:5432/prod",
}
`

func main() {
	for _, style := range []code.RedactStyle{code.RedactBlock, code.RedactBlur} {
		err := goshot.New().
			WithContent(code.New(source).
				WithLanguage("go").
				WithTheme("github-dark").
				WithRedaction(code.NewRedaction().WithStyle(style).WithBlurRadius(4))).
			WithChrome(chrome.Windows().WithTitle("config.go").Dark()).
			WithBackground(background.Solid(color.RGBA{13, 17, 23, 255}).WithPadding(50).WithCornerRadius(8)).
			Save("redaction_" + string(style) + ".png")
		if err != nil {
			log.Fatal(err)
		}
	}
}
