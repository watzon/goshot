// Package cli implements the goshot command line interface.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// options holds every CLI setting. Defaults may be overridden by the
// config file, then by flags.
type options struct {
	// input/output
	input         string
	output        string
	toClipboard   bool
	fromClipboard bool
	toStdout      bool

	// appearance
	chromeStyle  string
	lightMode    bool
	theme        string
	language     string
	font         string
	lineHeight   float64
	background   string
	bgImage      string
	bgImageFit   string
	bgBlur       float64
	bgBlurType   string
	cornerRadius float64
	noWindow     bool
	windowTitle  string
	windowRadius float64
	autoTitle    bool

	// gradient
	gradientType      string
	gradientStops     []string
	gradientAngle     float64
	gradientCenterX   float64
	gradientCenterY   float64
	gradientIntensity float64

	// shadow
	shadowBlur    float64
	shadowColor   string
	shadowSpread  float64
	shadowOffsetX float64
	shadowOffsetY float64

	// layout
	padHoriz      int
	padVert       int
	codePad       [4]int // top, bottom, left, right
	numberPad     int
	minWidth      int
	maxWidth      int
	tabWidth      int
	noLineNumbers bool
	lineRanges    []string
	highlights    []string

	// redaction
	redact         bool
	redactStyle    string
	redactBlur     float64
	redactPatterns []string
	redactAreas    []string

	// terminal (exec)
	args           []string
	cols           int
	rows           int
	autoSize       bool
	cellPad        [4]int // left, right, top, bottom
	cellSpacing    int
	showPrompt     bool
	promptTemplate string
}

var opts options

// Execute runs the CLI.
func Execute(version string) error {
	root := &cobra.Command{
		Use:   "goshot [file] [flags]",
		Short: "Create beautiful screenshots of code and terminal output",
		Long: "Goshot renders source code and terminal output as images with\n" +
			"syntax highlighting, window chrome, and rich backgrounds.",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := readSource(args)
			if err != nil {
				return err
			}
			return renderCode(source)
		},
	}

	flags := root.PersistentFlags()
	groups := map[*pflag.FlagSet]string{
		outputFlags():     "output",
		appearanceFlags(): "appearance",
		gradientFlags():   "gradient",
		shadowFlags():     "shadow",
		layoutFlags():     "layout",
		redactionFlags():  "redaction",
	}
	for fs := range groups {
		flags.AddFlagSet(fs)
	}

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		return applyConfigFile(cmd.Flags())
	}

	root.AddCommand(execCommand(), themesCommand(), fontsCommand(), languagesCommand(), versionCommand(version))
	groupUsage(root, groups)
	return root.Execute()
}

func readSource(args []string) (string, error) {
	switch {
	case opts.fromClipboard:
		return clipboard.ReadAll()
	case len(args) > 0:
		opts.input = args[0]
		data, err := os.ReadFile(args[0])
		return string(data), err
	default:
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	}
}

// detectLanguage picks the language from the flag or the file extension.
func detectLanguage() string {
	if opts.language != "" {
		return opts.language
	}
	if opts.input != "" {
		return strings.TrimPrefix(filepath.Ext(opts.input), ".")
	}
	return ""
}

// --- flag groups -------------------------------------------------------

func outputFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("output", pflag.ContinueOnError)
	fs.StringVarP(&opts.output, "output", "o", "output.png", "Output image path (templated, extension picks the format)")
	fs.BoolVarP(&opts.toClipboard, "to-clipboard", "c", false, "Copy the image to the clipboard")
	fs.BoolVar(&opts.fromClipboard, "from-clipboard", false, "Read input from the clipboard")
	fs.BoolVarP(&opts.toStdout, "to-stdout", "s", false, "Write the PNG to stdout")
	return fs
}

func appearanceFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("appearance", pflag.ContinueOnError)
	fs.StringVarP(&opts.chromeStyle, "chrome", "C", "mac", "Window chrome (mac, windows, gnome, breeze, blank)")
	fs.BoolVarP(&opts.lightMode, "light-mode", "L", false, "Use the chrome's light palette")
	fs.StringVarP(&opts.theme, "theme", "t", "ayu-dark", "Syntax / terminal theme name")
	fs.StringVar(&opts.language, "language", "", "Language override (default: detect from extension)")
	fs.StringVarP(&opts.font, "font", "f", "JetBrainsMonoNerdFont", "Font list with optional size (e.g. 'Hack; FiraCode=15')")
	fs.Float64Var(&opts.lineHeight, "line-height", 1.0, "Line height multiplier")
	fs.StringVarP(&opts.background, "background", "b", "#ABB8C3", "Background color (hex or 'transparent')")
	fs.StringVar(&opts.bgImage, "background-image", "", "Background image path")
	fs.StringVar(&opts.bgImageFit, "background-image-fit", "cover", "Image fit (fit, fill, cover, stretch, tile)")
	fs.Float64Var(&opts.bgBlur, "background-blur", 0, "Background blur radius")
	fs.StringVar(&opts.bgBlurType, "background-blur-type", "gaussian", "Blur type (gaussian, pixelated)")
	fs.Float64Var(&opts.cornerRadius, "corner-radius", 10, "Corner radius of the whole image")
	fs.BoolVar(&opts.noWindow, "no-window-controls", false, "Hide the window title bar and controls")
	fs.StringVar(&opts.windowTitle, "window-title", "", "Window title text")
	fs.Float64Var(&opts.windowRadius, "window-corner-radius", 10, "Corner radius of the window")
	fs.StringSliceVar(&opts.lineRanges, "line-range", nil, "Line ranges to render (e.g. 5..10)")
	fs.StringSliceVar(&opts.highlights, "highlight-lines", nil, "Line ranges to highlight")
	fs.BoolVar(&opts.noLineNumbers, "no-line-numbers", false, "Hide line numbers")
	return fs
}

func gradientFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("gradient", pflag.ContinueOnError)
	fs.StringVar(&opts.gradientType, "gradient-type", "", "Gradient type (linear, radial, angular, diamond, spiral, square, star)")
	fs.StringSliceVar(&opts.gradientStops, "gradient-stops", []string{"#232323;0", "#383838;100"}, "Gradient stops ('#color;position')")
	fs.Float64Var(&opts.gradientAngle, "gradient-angle", 45, "Gradient angle in degrees")
	fs.Float64Var(&opts.gradientCenterX, "gradient-center-x", 0.5, "Gradient center X (0-1)")
	fs.Float64Var(&opts.gradientCenterY, "gradient-center-y", 0.5, "Gradient center Y (0-1)")
	fs.Float64Var(&opts.gradientIntensity, "gradient-intensity", 5, "Spiral tightness / star point count")
	return fs
}

func shadowFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("shadow", pflag.ContinueOnError)
	fs.Float64Var(&opts.shadowBlur, "shadow-blur", 0, "Shadow blur radius (0 disables the shadow)")
	fs.StringVar(&opts.shadowColor, "shadow-color", "#00000033", "Shadow color")
	fs.Float64Var(&opts.shadowSpread, "shadow-spread", 0, "Shadow spread radius")
	fs.Float64Var(&opts.shadowOffsetX, "shadow-offset-x", 0, "Shadow X offset")
	fs.Float64Var(&opts.shadowOffsetY, "shadow-offset-y", 0, "Shadow Y offset")
	return fs
}

func layoutFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("layout", pflag.ContinueOnError)
	fs.IntVar(&opts.padHoriz, "pad-horiz", 100, "Horizontal background padding")
	fs.IntVar(&opts.padVert, "pad-vert", 80, "Vertical background padding")
	fs.IntVar(&opts.codePad[0], "code-pad-top", 10, "Code top padding")
	fs.IntVar(&opts.codePad[1], "code-pad-bottom", 10, "Code bottom padding")
	fs.IntVar(&opts.codePad[2], "code-pad-left", 10, "Code left padding")
	fs.IntVar(&opts.codePad[3], "code-pad-right", 10, "Code right padding")
	fs.IntVar(&opts.numberPad, "line-number-pad", 10, "Gap between line numbers and code")
	fs.IntVar(&opts.minWidth, "min-width", 0, "Minimum image width")
	fs.IntVar(&opts.maxWidth, "max-width", 0, "Maximum image width (wraps long lines)")
	fs.IntVar(&opts.tabWidth, "tab-width", 4, "Tab width in spaces")
	return fs
}

func redactionFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("redaction", pflag.ContinueOnError)
	fs.BoolVar(&opts.redact, "redact", false, "Redact sensitive values (keys, tokens, passwords)")
	fs.StringVar(&opts.redactStyle, "redact-style", "block", "Redaction style (block, blur)")
	fs.Float64Var(&opts.redactBlur, "redact-blur", 5, "Blur radius for redacted areas")
	fs.StringSliceVar(&opts.redactPatterns, "redact-pattern", nil, "Extra redaction regex (first capture group is hidden)")
	fs.StringSliceVar(&opts.redactAreas, "redact-area", nil, "Manual redaction area 'x,y,width,height'")
	return fs
}

// groupUsage prints flags grouped by their flag set, in a stable order.
func groupUsage(cmd *cobra.Command, groups map[*pflag.FlagSet]string) {
	const bold, dim, reset = "\x1b[1m", "\x1b[2m", "\x1b[0m"
	order := []string{"output", "appearance", "gradient", "shadow", "layout", "redaction", "terminal"}

	cmd.SetUsageFunc(func(c *cobra.Command) error {
		w := c.OutOrStderr()
		fmt.Fprintf(w, "%sUsage:%s\n  %s\n\n", bold, reset, c.UseLine())

		byName := map[string]*pflag.FlagSet{}
		for fs, name := range groups {
			byName[name] = fs
		}
		for _, name := range order {
			fs, ok := byName[name]
			if !ok {
				continue
			}
			fmt.Fprintf(w, "%s%s%s flags:%s\n", bold, strings.ToUpper(name[:1]), name[1:], reset)
			fs.VisitAll(func(f *pflag.Flag) {
				label := "    --" + f.Name
				if f.Shorthand != "" {
					label = "-" + f.Shorthand + ", --" + f.Name
				}
				usage := f.Usage
				if def := f.DefValue; def != "" && def != "false" && def != "[]" && def != "0" {
					usage += fmt.Sprintf(" %s(default %s)%s", dim, def, reset)
				}
				fmt.Fprintf(w, "  %-28s %s\n", label, usage)
			})
			fmt.Fprintln(w)
		}

		if c.HasAvailableSubCommands() {
			fmt.Fprintf(w, "%sCommands:%s\n", bold, reset)
			for _, sub := range c.Commands() {
				if !sub.Hidden {
					fmt.Fprintf(w, "  %-14s %s\n", sub.Name(), sub.Short)
				}
			}
		}
		return nil
	})
}
