// Command-line tool to process files in specified directories.
// It formats file paths and contents, optionally filters by substrings and extensions,
// and either prints to console, copies to clipboard, or both.
//
// Usage:
//
//	gogrep [flags]
//
// Flags:
//
//	--dir stringSlice        Directories to search (comma-separated, default ["."])
//	--ext stringSlice        File extensions to include (comma-separated, default [])
//	--substring stringSlice  Substrings to filter files by (comma-separated, default [])
//	--action string          Action to perform: print, copy, or both (default "both")
//
// If no directories are provided, it searches the current directory.
// If no extensions are provided, all files are processed.
// If no substrings are provided, all files (filtered by extensions if provided) are included.
// The --action flag controls whether to print to console, copy to clipboard, or both.
//
// Examples:
//
//	gogrep                                       # Process all files in the current directory and print+copy
//	gogrep --substring="store" --action=print    # Print files containing "store" in the current directory
//	gogrep --dir="app" --ext=".js" --action=copy # Copy .js files in app/ to clipboard
//	gogrep --dir="foo,bar" --substring="bar,baz,qux" --ext=".ts,.tsx" --action=both # Process .ts and .tsx files in foo/ and bar/
//	                                                                                # containing "bar", "baz", or "qux" and print+copy
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"foo.bar/lib/logutils"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type Action string

const (
	ActionPrint Action = "print" // Print to the console
	ActionCopy  Action = "copy"  // Copy to the clipboard
	ActionBoth  Action = "both"  // Print to the console and copy to the clipboard
)

// Command-line flags
var (
	epoch = time.Now()

	dirs       []string // Directories to search
	exts       []string // File extensions to include
	substrings []string // Substrings to filter by
	action     string   // Action: print, copy, or both
)

// Styles for help message
var (
	// Bold styles
	styleBoldGreen = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleBoldRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleBoldWhite = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	// Regular styles
	styleBlue  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleFaint = lipgloss.NewStyle().Faint(true)
)

var threeOrMoreNewlinesRegex = regexp.MustCompile(`\n{3,}`)

// Root command definition
var rootCmd = &cobra.Command{
	Use:   "gogrep",
	Short: "Process files in specified directories",
	Long: `A command-line tool to process files in specified directories.
It formats file paths and contents, optionally filters by substrings and extensions,
and either prints to console, copies to clipboard, or both.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Print help message if no arguments are provided
		if len(os.Args) == 1 {
			help, err := help()
			if err != nil {
				err := fmt.Errorf("failed to get help message: %w", err)
				slog.Error(err.Error())
				os.Exit(1)
			}
			fmt.Println(help)
			os.Exit(0)
		}

		// Get the files to process
		var filesToProcess []string
		for _, dir := range dirs {
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isValidExt(info.Name(), exts) {
					filesToProcess = append(filesToProcess, path)
				}
				return nil
			})
			if err != nil {
				err := fmt.Errorf("failed to walk directory: %w", err)
				slog.Error(err.Error(), slog.String("dir", dir))
				os.Exit(1)
			}
		}

		// Confirm before processing many files (50+)
		if len(filesToProcess) > 50 {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println(styleBoldRed.Render(fmt.Sprintf("WARNING: Processing %d files. Proceed? [y/N] ", len(filesToProcess))))
			response, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Operation cancelled.") // TODO
				os.Exit(0)
			}
		}

		// Process the files
		var b strings.Builder
		for _, path := range filesToProcess {
			content, err := os.ReadFile(path)
			if err != nil {
				err := fmt.Errorf("failed to read file: %w", err)
				slog.Error(err.Error(), slog.String("path", path))
				continue
			}
			contentStr := string(content)
			if len(substrings) == 0 || anySubstringMatches(substrings, path, contentStr) {
				b.WriteString("# " + path + "\n")
				b.Write(content)
				b.WriteString("\n\n")
			}
		}
		output := b.String()

		// Remove three or more consecutive newlines
		output = threeOrMoreNewlinesRegex.ReplaceAllString(output, "\n\n")
		output = strings.TrimSpace(output)

		// Perform the action
		switch Action(action) {
		case ActionPrint:
			fmt.Println(output)
			//// fmt.Println(styleFaint.Render("(" + time.Since(epoch).Round(time.Millisecond).String() + ")"))
		case ActionCopy:
			copyToClipboard([]byte(output))
			fmt.Println(styleBoldGreen.Render("Copied to clipboard!") + " " + styleFaint.Render("("+time.Since(epoch).Round(time.Millisecond).String()+")"))
		case ActionBoth:
			fmt.Println(output)
			copyToClipboard([]byte(output))
			//// fmt.Println(styleFaint.Render("(" + time.Since(epoch).Round(time.Millisecond).String() + ")"))
		default:
			slog.Error("internal error")
		}
	},
}

// isValidExt returns true if the filename has one of the specified extensions.
// If no extensions are provided, it always returns true.
func isValidExt(filename string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	for _, ext := range exts {
		// Lowercase all strings for case-insensitive comparison
		if strings.HasSuffix(strings.ToLower(filename), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// anySubstringMatches returns true if any of the substrings are found in the path or content.
func anySubstringMatches(substrings []string, path, content string) bool {
	for _, sub := range substrings {
		// Lowercase all strings for case-insensitive comparison
		if strings.Contains(strings.ToLower(path), strings.ToLower(sub)) || strings.Contains(strings.ToLower(content), strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// copyToClipboard copies a string to the clipboard using the pbcopy command.
// It returns an error if the command fails.
func copyToClipboard(str []byte) error {
	// Run the pbcopy command
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader(str)
	if err := cmd.Run(); err != nil {
		err := fmt.Errorf("failed to copy to clipboard: %w", err)
		return err
	}
	return nil
}

// getTildePath returns the current working directory with the user's home directory replaced by a tilde.
// It returns an error if the user's home directory cannot be determined.
func getTildePath() (string, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		err := fmt.Errorf("failed to get current working directory: %w", err)
		return "", err
	}

	// Get the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		err := fmt.Errorf("failed to get user's home directory: %w", err)
		return "", err
	}

	// Replace the home directory with a tilde
	return strings.Replace(cwd, home, "~", 1), nil
}

// help returns the help message for the root command.
func help() (string, error) {
	// Get current working directory
	cwd, err := getTildePath()
	if err != nil {
		err := fmt.Errorf("failed to get current working directory: %w", err)
		return "", err
	}

	// Build the help message
	var b strings.Builder
	b.WriteString(styleBoldGreen.Render(`gogrep`) + ` greps files in specified directories ` + styleFaint.Render(`(`+cwd+`)`) + "\n\n")
	b.WriteString(styleBoldWhite.Render(`Usage: gogrep [flags]`) + "\n\n")
	b.WriteString(styleBoldWhite.Render(`Flags:`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--dir`) + `        Directories to search (comma-separated)       ` + styleFaint.Render(`default ["."]`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--ext`) + `        File extensions to include (comma-separated)  ` + styleFaint.Render(`default []`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--substring`) + `  Substrings to filter by (comma-separated)     ` + styleFaint.Render(`default []`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--action`) + `     Action to perform (print, copy, or both)      ` + styleFaint.Render(`default "both"`) + "\n\n")
	b.WriteString(styleBoldWhite.Render(`Examples:`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep`) + `                                                                       ` + styleFaint.Render(`Process all files in the current directory and print+copy`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --substring="store" --action=print`) + `                                    ` + styleFaint.Render(`Print files containing "store" in the current directory`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir="app" --ext=".js" --action=copy`) + `                                 ` + styleFaint.Render(`Copy .js files in app/ to clipboard`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir="foo,bar" --substring="bar,baz" --ext=".ts,.tsx" --action=both`) + `  ` + styleFaint.Render(`Process .ts/.tsx files with "bar" or "baz"`))
	return b.String(), nil
}

func main() {
	// Configure logging
	logutils.Configure(logutils.Configuration{IsJSONEnabled: false})

	// Define the root command
	rootCmd.Flags().StringSliceVar(&dirs, "dir", []string{"."}, "Directories to search (comma-separated, default [.])")
	rootCmd.Flags().StringSliceVar(&exts, "ext", []string{}, "File extensions to include (comma-separated, default [])")
	rootCmd.Flags().StringSliceVar(&substrings, "substring", []string{}, "Substrings to filter files by (comma-separated, default [])")
	rootCmd.Flags().StringVar(&action, "action", "both", "Action to perform: print, copy, or both (default both)")

	// Validate the flags
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Ensure the directories exist
		var invalidDirs []string
		for _, dir := range dirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				invalidDirs = append(invalidDirs, dir)
			}
		}
		if len(invalidDirs) > 0 {
			var b strings.Builder
			b.WriteString("one or more directories do not exist: ")
			for i, dir := range invalidDirs {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(dir)
			}
			err := errors.New(b.String())
			//// slog.Error(err.Error())
			//// os.Exit(1)
			return err
		}

		// Ensure the extensions are valid
		var invalidExts []string
		for _, ext := range exts {
			if !strings.HasPrefix(ext, ".") {
				invalidExts = append(invalidExts, ext)
			}
		}
		if len(invalidExts) > 0 {
			var b strings.Builder
			b.WriteString("one or more extensions must start with a period: ")
			for i, ext := range invalidExts {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(ext)
			}
			err := errors.New(b.String())
			//// slog.Error(err.Error())
			//// os.Exit(1)
			return err
		}

		// Ensure the action is valid
		if Action(action) != ActionPrint && Action(action) != ActionCopy && Action(action) != ActionBoth {
			err := errors.New("action must be print, copy, or both")
			//// slog.Error(err.Error(), slog.String("action", action))
			//// os.Exit(1)
			return err
		}
		return nil
	}

	// Set up the help message
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		help, _ := help()
		fmt.Println(help)
	})

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
