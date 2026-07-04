package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/watzon/goshot/code"
	"github.com/watzon/goshot/fonts"
	"github.com/watzon/goshot/term"
)

func themesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "themes",
		Short: "List syntax and terminal themes",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("Syntax themes:")
			printColumns(code.Themes())
			fmt.Println("\nTerminal themes (exec):")
			printColumns(term.Themes())
		},
	}
}

func fontsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fonts",
		Short: "List available font families",
		Run: func(cmd *cobra.Command, _ []string) {
			printColumns(fonts.List())
		},
	}
}

func languagesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "languages",
		Short: "List supported languages",
		Run: func(cmd *cobra.Command, _ []string) {
			printColumns(code.Languages(true))
		},
	}
}

func versionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the goshot version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("goshot", version)
		},
	}
}

func printColumns(items []string) {
	sort.Strings(items)
	for _, item := range items {
		fmt.Println(" ", item)
	}
}
