package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/xpty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

func execCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] -- command [args...]",
		Short: "Run a command and screenshot its output",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.args = args
			captured, err := capture(cmd.Context(), args)
			if err != nil {
				return err
			}
			return renderTerm(args, captured)
		},
	}

	fs := pflag.NewFlagSet("terminal", pflag.ContinueOnError)
	fs.IntVarP(&opts.cols, "width", "w", 120, "Terminal width in cells")
	fs.IntVarP(&opts.rows, "height", "H", 40, "Terminal height in cells")
	fs.BoolVarP(&opts.autoSize, "auto-size", "A", false, "Grow the canvas to fit the output")
	fs.IntVar(&opts.cellPad[0], "pad-left", 1, "Left padding in cells")
	fs.IntVar(&opts.cellPad[1], "pad-right", 1, "Right padding in cells")
	fs.IntVar(&opts.cellPad[2], "pad-top", 1, "Top padding in cells")
	fs.IntVar(&opts.cellPad[3], "pad-bottom", 1, "Bottom padding in cells")
	fs.IntVar(&opts.cellSpacing, "cell-spacing", 0, "Extra spacing between cells in pixels")
	fs.BoolVarP(&opts.showPrompt, "show-prompt", "p", false, "Render a prompt line above the output")
	fs.StringVarP(&opts.promptTemplate, "prompt-template", "P", "\x1b[1;35m❯ \x1b[0;32m[command]\x1b[0m", "Prompt template ([command] is replaced)")
	fs.BoolVar(&opts.autoTitle, "auto-title", false, "Use the command as the window title")
	cmd.Flags().AddFlagSet(fs)
	return cmd
}

// capture runs a command inside a pseudo-terminal and returns everything
// it printed, ANSI escapes included.
func capture(ctx context.Context, args []string) ([]byte, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}
	pty, err := xpty.NewPty(width, height)
	if err != nil {
		return nil, err
	}
	defer pty.Close()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := pty.Start(cmd); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	go io.Copy(&out, pty)

	if err := xpty.WaitProcess(ctx, cmd); err != nil {
		return stderr.Bytes(), fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return out.Bytes(), nil
}
