// Command-line tool to process files in specified directories.
// It formats file paths and contents, optionally filters by substrings and extensions,
// and performs specified actions (print, copy, or both) on the output generated
// in the specified formats (tree, filenames, contents, or combinations).
//
// Usage:
//
//	gogrep [flags]
//
// Flags:
//
//	--dir stringSlice        Directories to search (comma-separated, default ["."])
//	--dir-depth int          Maximum directory depth to search (default -1, meaning infinite)
//	--ext stringSlice        File extensions to include (comma-separated, default [])
//	--substring stringSlice  Substrings to filter files by (comma-separated, default [])
//	--action stringSlice     Actions to perform: print, copy (comma-separated, default print,copy)
//	--format stringSlice     Output formats: tree, filenames, contents (comma-separated, default tree,contents)
//
// If no directories are provided, it searches the current directory.
// If no extensions are provided, all files are processed.
// If no substrings are provided, all files (filtered by extensions if provided) are included.
// The --action flag specifies the actions to perform on the output (e.g., print, copy, print,copy).
// The --format flag specifies the output formats to generate and concatenate (e.g., tree, contents, tree,contents).
//
// Examples:
//
//	gogrep                                                                                              # Process all files in the current directory and print+copy the contents
//	gogrep --substring=store --action=print --format=filenames                                          # Print the list of filenames containing "store"
//	gogrep --dir=app --ext=.js --action=copy --format=contents                                          # Copy the contents of .js files in app/ to clipboard
//	gogrep --dir=foo,bar --substring=bar,baz --ext=.ts,.tsx --action=print,copy --format=tree,contents  # Print and copy the tree and contents of .ts/.tsx files with "bar" or "baz"
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"foo.bar/lib/logutils"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// Tree represents a directory hierarchy for the --format=tree option.
type Tree map[string]Tree

// Insert adds a file path into the tree structure.
func (t Tree) Insert(path string) {
	parts := strings.Split(path, "/")
	current := t
	for i, part := range parts {
		if i == len(parts)-1 {
			// File (leaf node)
			current[part] = make(Tree)
		} else {
			// Directory
			if _, ok := current[part]; !ok {
				current[part] = make(Tree)
			}
			current = current[part]
		}
	}
}

// Print generates a hierarchical string representation of the tree.
func (t Tree) Print(indent string) string {
	var keys []string
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		if len(t[key]) == 0 {
			// File
			b.WriteString(indent + key + "\n")
		} else {
			// Directory
			b.WriteString(indent + key + "/\n")
			b.WriteString(t[key].Print(indent + "  "))
		}
	}
	return b.String()
}

// Action represents the possible actions that can be performed on the output.
type Action int

const (
	ActionPrint Action = iota // Action to print the output to the console
	ActionCopy                // Action to copy the output to the clipboard
)

// Format represents the possible output formats.
type Format int

const (
	FormatTree      Format = iota // Format to display the directory tree
	FormatFilenames               // Format to list the filenames
	FormatContents                // Format to display the contents of the files
)

// Command-line flags
var (
	dirs       []string
	dirDepth   int
	exts       []string
	substrings []string
	actions    []string
	formats    []string
)

// Styles for the help message
var (
	styleBoldBrightWhite = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	styleBoldGreen       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleBoldRed         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

	styleBlue  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleFaint = lipgloss.NewStyle().Faint(true)
)

var threeOrMoreNewlinesRegex = regexp.MustCompile(`\n{3,}`)

// parseAction converts a single action string to an Action enum.
// It returns an error if the string does not correspond to a valid Action.
func parseAction(actionString string) (Action, error) {
	switch actionString {
	case "print":
		return ActionPrint, nil
	case "copy":
		return ActionCopy, nil
	default:
		return 0, fmt.Errorf("invalid action: %s", actionString)
	}
}

// parseFormat converts a single format string to a Format enum.
// It returns an error if the string does not correspond to a valid Format.
func parseFormat(formatString string) (Format, error) {
	switch formatString {
	case "tree":
		return FormatTree, nil
	case "filenames":
		return FormatFilenames, nil
	case "contents":
		return FormatContents, nil
	default:
		return 0, fmt.Errorf("invalid format: %s", formatString)
	}
}

// expandTilde replaces ~ with the user's home directory in the given path.
// It returns an error if the home directory cannot be determined.
func expandTilde(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user's home directory: %w", err)
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

// Root command definition
var rootCmd = &cobra.Command{
	Use:   "gogrep",
	Short: "Process files in specified directories",
	Long: `A command-line tool to process files in specified directories.
It formats file paths and contents, optionally filters by substrings and extensions,
and performs specified actions on the output generated in the specified formats.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Print help message if no arguments are provided
		if len(os.Args) == 1 {
			help, _ := help()
			fmt.Println(help)
			os.Exit(0)
		}

		// Parse the actions
		var parsedActions []Action
		for _, actionStr := range actions {
			act, _ := parseAction(actionStr) // No error check needed, validated in PreRunE
			parsedActions = append(parsedActions, act)
		}

		// Parse the formats
		var parsedFormats []Format
		for _, formatStr := range formats {
			fmt, _ := parseFormat(formatStr) // No error check needed, validated in PreRunE
			parsedFormats = append(parsedFormats, fmt)
		}

		// Collect files grouped by root directory
		filesByRoot := make(map[string][]string)
		for _, dir := range dirs {
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Check depth for directories if dirDepth is specified
				if info.IsDir() && dirDepth != -1 {
					relPath, err := filepath.Rel(dir, path)
					if err != nil {
						return err
					}
					var depth int
					if relPath == "." {
						depth = 0 // Root directory itself
					} else {
						depth = strings.Count(relPath, string(os.PathSeparator)) + 1 // Depth relative to root
					}
					if depth > dirDepth {
						return filepath.SkipDir // Skip directories beyond max depth
					}
				}
				// Process files if they match extensions
				if !info.IsDir() && isValidExt(info.Name(), exts) {
					filesByRoot[dir] = append(filesByRoot[dir], path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}
		}

		// Confirm before processing a large number of files (50+)
		totalFiles := 0
		for _, files := range filesByRoot {
			totalFiles += len(files)
		}
		if totalFiles > 50 {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println(styleBoldRed.Render(fmt.Sprintf("WARNING: Processing %s files. Proceed? [y/N] ", humanize.Comma(int64(totalFiles)))))
			response, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		// Process files and generate output
		var outputs []string
		for _, format := range parsedFormats {
			var output string
			switch format {
			case FormatContents:
				var b strings.Builder
				for _, paths := range filesByRoot {
					for _, path := range paths {
						content, err := os.ReadFile(path)
						if err != nil {
							slog.Error("failed to read file", slog.String("path", path), slog.String("error", err.Error()))
							continue
						}
						contentStr := string(content)
						if len(substrings) == 0 || anySubstringMatches(substrings, path, contentStr) {
							b.WriteString("# " + path + "\n")
							b.WriteString(contentStr + "\n\n")
						}
					}
				}
				output = b.String()

			case FormatFilenames:
				var filteredFiles []string
				for _, paths := range filesByRoot {
					for _, path := range paths {
						if len(substrings) == 0 || anySubstringMatches(substrings, path, "") {
							filteredFiles = append(filteredFiles, path)
						}
					}
				}
				sort.Strings(filteredFiles)
				output = strings.Join(filteredFiles, "\n")

			case FormatTree:
				var b strings.Builder
				for root := range filesByRoot {
					tree := make(Tree)
					hasFiles := false
					for _, path := range filesByRoot[root] {
						if len(substrings) == 0 || anySubstringMatches(substrings, path, "") {
							relPath, err := filepath.Rel(root, path)
							if err != nil {
								return fmt.Errorf("failed to get relative path: %w", err)
							}
							tree.Insert(relPath)
							hasFiles = true
						}
					}
					if hasFiles {
						b.WriteString(root + "/\n")
						b.WriteString(tree.Print("  "))
					}
				}
				output = b.String()

			default:
				slog.Error("internal error")
				continue
			}
			// Normalize output by replacing three or more newlines with two newlines
			output = threeOrMoreNewlinesRegex.ReplaceAllString(output, "\n\n")
			output = strings.TrimSpace(output)
			outputs = append(outputs, output)
		}
		combinedOutput := strings.Join(outputs, "\n\n")

		// Perform the specified actions on the output
		for _, act := range parsedActions {
			switch act {
			case ActionPrint:
				fmt.Println(combinedOutput)
			case ActionCopy:
				copyToClipboard([]byte(combinedOutput))
			default:
				slog.Error("internal error")
			}
		}
		return nil
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
	b.WriteString(styleBoldBrightWhite.Render(`Usage: gogrep [flags]`) + "\n\n")
	b.WriteString(styleBoldBrightWhite.Render(`Flags:`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--dir`) + `        Directories to search (comma-separated, default [.])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--dir-depth`) + `  Maximum directory depth to search (default -1, meaning infinite)` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--ext`) + `        File extensions to include (comma-separated, default [])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--substring`) + `  Substrings to filter by (comma-separated, default [])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--action`) + `     Actions to perform: print, copy (comma-separated, default print,copy)` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--format`) + `     Output formats: tree, filenames, contents (comma-separated, default tree,contents)` + "\n\n")
	b.WriteString(styleBoldBrightWhite.Render(`Examples:`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep`) + `                                                                                              ` + styleFaint.Render(`Process all files in the current directory and print+copy the contents`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --substring=store --action=print --format=filenames`) + `                                          ` + styleFaint.Render(`Print the list of filenames containing "store"`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir=app --ext=.js --action=copy --format=contents`) + `                                          ` + styleFaint.Render(`Copy the contents of .js files in app/ to clipboard`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir=foo,bar --substring=bar,baz --ext=.ts,.tsx --action=print,copy --format=tree,contents`) + `  ` + styleFaint.Render(`Print and copy the tree and contents of .ts/.tsx files with "bar" or "baz"`))
	return b.String(), nil
}

// PreRunE validates the command-line flags before the main command executes.
func PreRunE(cmd *cobra.Command, args []string) error {
	// Expand tilde in directories
	var expandedDirs []string
	for _, dir := range dirs {
		expanded, err := expandTilde(dir)
		if err != nil {
			return err
		}
		expandedDirs = append(expandedDirs, expanded)
	}
	dirs = expandedDirs // Update dirs with expanded paths

	// Validate the directories
	var invalidDirs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			invalidDirs = append(invalidDirs, dir)
		}
	}
	if len(invalidDirs) > 0 {
		return fmt.Errorf("directories are invalid: %s", strings.Join(invalidDirs, ", "))
	}

	// Validate the extensions
	var invalidExts []string
	for _, ext := range exts {
		if !strings.HasPrefix(ext, ".") {
			invalidExts = append(invalidExts, ext)
		}
	}
	if len(invalidExts) > 0 {
		return fmt.Errorf("extensions are invalid: %s", strings.Join(invalidExts, ", "))
	}

	// Validate the actions
	var invalidActions []string
	for _, action := range actions {
		if _, err := parseAction(action); err != nil {
			invalidActions = append(invalidActions, action)
		}
	}
	if len(invalidActions) > 0 {
		return fmt.Errorf("actions are invalid: %s", strings.Join(invalidActions, ", "))
	}

	// Validate the formats
	var invalidFormats []string
	for _, format := range formats {
		if _, err := parseFormat(format); err != nil {
			invalidFormats = append(invalidFormats, format)
		}
	}
	if len(invalidFormats) > 0 {
		return fmt.Errorf("formats are invalid: %s", strings.Join(invalidFormats, ", "))
	}
	return nil
}

func main() {
	// Configure logging
	logutils.Configure(logutils.Configuration{IsJSONEnabled: false})

	// Define the root command flags
	rootCmd.Flags().StringSliceVar(&dirs, "dir", []string{"."}, "Directories to search (comma-separated, default [.])")
	rootCmd.Flags().IntVar(&dirDepth, "dir-depth", -1, "Maximum directory depth to search (default -1, meaning infinite)")
	rootCmd.Flags().StringSliceVar(&exts, "ext", []string{}, "File extensions to include (comma-separated, default [])")
	rootCmd.Flags().StringSliceVar(&substrings, "substring", []string{}, "Substrings to filter files by (comma-separated, default [])")
	rootCmd.Flags().StringSliceVar(&actions, "action", []string{"print", "copy"}, "Actions to perform: print, copy (comma-separated, default print,copy)")
	rootCmd.Flags().StringSliceVar(&formats, "format", []string{"tree", "contents"}, "Output formats: tree, filenames, contents (comma-separated, default tree,contents)")

	rootCmd.PreRunE = PreRunE

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
