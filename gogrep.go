// Command-line tool to process files in specified directories.
// It formats file paths and contents, optionally filters by substrings and extensions,
// and performs specified actions (print, copy, or both) on the output generated
// in the specified formats (tree, list, contents, or combinations).
//
// Usage:
//
//	gogrep [flags]
//
// Flags:
//
//	--dir strings        Directories to search (comma-separated, default ["."])
//	--dir-depth int      Maximum directory depth to search (default -1, meaning infinite)
//	--ext strings        File extensions to include (comma-separated, default [])
//	--substring strings  Substrings to filter files by (comma-separated, default [])
//	--action strings     Actions to perform: print, copy (comma-separated, default print,copy)
//	--format strings     Output formats: tree, list, contents (comma-separated, default tree,contents)
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
//	gogrep --substring=store --action=print --format=list                                               # Print the list of files with "store" in the path
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

// TreeNode represents a node in the directory tree, with a flag to distinguish directories from files.
type TreeNode struct {
	IsDir    bool
	Children map[string]*TreeNode
}

// Insert adds a path into the tree structure, respecting whether it’s a file or directory.
func Insert(node *TreeNode, parts []string, isDir bool) {
	if len(parts) == 0 {
		return
	}
	part := parts[0]
	if _, ok := node.Children[part]; !ok {
		// Intermediate parts are directories; last part uses isDir
		node.Children[part] = &TreeNode{
			IsDir:    len(parts) > 1 || isDir,
			Children: make(map[string]*TreeNode),
		}
	}
	if len(parts) > 1 {
		Insert(node.Children[part], parts[1:], isDir)
	} else {
		node.Children[part].IsDir = isDir
	}
}

// Print generates a hierarchical string representation of the tree.
func Print(node *TreeNode, indent string) string {
	var keys []string
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		child := node.Children[key]
		if child.IsDir {
			b.WriteString(indent + key + "/\n")
			b.WriteString(Print(child, indent+"  "))
		} else {
			b.WriteString(indent + key + "\n")
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
	FormatTree     Format = iota // Format to display the directory tree
	FormatList                   // Format to display the list of filenames
	FormatContents               // Format to display the contents of the files
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
func parseFormat(formatString string) (Format, error) {
	switch formatString {
	case "tree":
		return FormatTree, nil
	case "list":
		return FormatList, nil
	case "contents":
		return FormatContents, nil
	default:
		return 0, fmt.Errorf("invalid format: %s", formatString)
	}
}

// expandTilde replaces ~ with the user's home directory in the given path.
// If the path does not start with ~, it is returned as is.
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

// areExtMatches returns true if the filename has any of the specified extensions.
// If exts is empty, it matches all extensions.
// The comparison is case-insensitive and requires an exact match.
func areExtMatches(filename string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	filenameExt := filepath.Ext(filename)
	if filenameExt == "" {
		return false
	}
	// Remove the leading dot from filenameExt
	filenameExt = strings.TrimPrefix(filenameExt, ".")
	for _, ext := range exts {
		// Remove the leading dot from ext, if present
		ext = strings.TrimPrefix(ext, ".")
		if strings.EqualFold(filenameExt, ext) {
			return true
		}
	}
	return false
}

// anySubstringMatches returns true if any of the substrings match the path or content.
// If substrings is empty, it matches all paths and contents.
// The comparison is case-insensitive.
func anySubstringMatches(substrings []string, path, content string) bool {
	if len(substrings) == 0 {
		return true
	}
	for _, sub := range substrings {
		if strings.Contains(strings.ToLower(path), strings.ToLower(sub)) || strings.Contains(content, sub) {
			return true
		}
	}
	return false
}

// copyToClipboard copies a string to the clipboard using the pbcopy command.
// Note: This function is only supported on macOS.
func copyToClipboard(str []byte) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader(str)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}

// getTildePath returns the current working directory with the user's home directory replaced by a tilde.
func getTildePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %w", err)
	}
	return strings.Replace(cwd, home, "~", 1), nil
}

// help returns the help message for the root command.
func help() (string, error) {
	cwd, err := getTildePath()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	var b strings.Builder
	b.WriteString(styleBoldGreen.Render(`gogrep`) + ` greps files in specified directories ` + styleFaint.Render(`(`+cwd+`)`) + "\n\n")
	b.WriteString(styleBoldBrightWhite.Render(`Usage: gogrep [flags]`) + "\n\n")
	b.WriteString(styleBoldBrightWhite.Render(`Flags:`) + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--dir`) + `        Directories to search (comma-separated, default [.])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--dir-depth`) + `  Maximum directory depth to search (default -1, meaning infinite)` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--ext`) + `        File extensions to include (comma-separated, default [])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--substring`) + `  Substrings to filter by (comma-separated, default [])` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--action`) + `     Actions to perform: print, copy (comma-separated, default print,copy)` + "\n")
	b.WriteString(`  ` + styleCyan.Render(`--format`) + `     Output formats: tree, list, contents (comma-separated, default tree,contents)` + "\n\n")
	b.WriteString(styleBoldBrightWhite.Render(`Examples:`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep`) + `                                                                                              ` + styleFaint.Render(`Process all files in the current directory and print+copy the contents`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --substring=store --action=print --format=list`) + `                                               ` + styleFaint.Render(`Print the list of files with "store" in the path`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir=app --ext=.js --action=copy --format=contents`) + `                                          ` + styleFaint.Render(`Copy the contents of .js files in app/ to clipboard`) + "\n")
	b.WriteString(`  ` + styleBlue.Render(`gogrep --dir=foo,bar --substring=bar,baz --ext=.ts,.tsx --action=print,copy --format=tree,contents`) + `  ` + styleFaint.Render(`Print and copy the tree and contents of .ts/.tsx files with "bar" or "baz"`))
	return b.String(), nil
}

// Root command definition
var rootCmd = &cobra.Command{
	Use:   "gogrep",
	Short: "Process files in specified directories",
	Long: `A command-line tool to process files in specified directories.
It formats file paths and contents, optionally filters by substrings and extensions,
and performs specified actions on the output generated in the specified formats.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Print the help message if no arguments are provided
		if len(os.Args) == 1 {
			help, _ := help()
			fmt.Println(help)
			os.Exit(0)
		}

		// Parse the actions
		var parsedActions []Action
		for _, actionStr := range actions {
			action, _ := parseAction(actionStr)
			parsedActions = append(parsedActions, action)
		}

		// Parse the formats
		var parsedFormats []Format
		for _, formatStr := range formats {
			format, _ := parseFormat(formatStr)
			parsedFormats = append(parsedFormats, format)
		}

		// Collect files with depth control and extension filter
		type Entry struct {
			Path  string
			IsDir bool
			Depth int
		}
		entriesByRoot := make(map[string][]Entry)
		for _, dir := range dirs {
			entriesByRoot[dir] = []Entry{}
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				var depth int
				if relPath == "." {
					depth = 0
				} else {
					depth = strings.Count(relPath, string(os.PathSeparator)) + 1
				}
				if !info.IsDir() && (dirDepth == -1 || depth <= dirDepth) && areExtMatches(info.Name(), exts) {
					entriesByRoot[dir] = append(entriesByRoot[dir], Entry{Path: path, IsDir: false, Depth: depth})
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}
		}

		// Ensure there are files to process
		if len(entriesByRoot) == 0 {
			fmt.Println("No files found.")
			return nil
		}

		// Confirm before processing a large number of files (50+)
		totalFiles := 0
		for _, entries := range entriesByRoot {
			totalFiles += len(entries)
		}
		if totalFiles > 50 {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println(styleBoldRed.Render(fmt.Sprintf("WARNING: Processing %s files. Proceed? [y/N] ", humanize.Comma(int64(totalFiles)))))
			response, _ := reader.ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(response), "y") {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Process the files
		var outputs []string
		for _, format := range parsedFormats {
			var output string
			switch format {
			case FormatContents:
				var b strings.Builder
				for _, entries := range entriesByRoot {
					for _, entry := range entries {
						content, err := os.ReadFile(entry.Path)
						if err != nil {
							slog.Error("failed to read file", slog.String("path", entry.Path), slog.String("error", err.Error()))
							continue
						}
						contentStr := string(content)
						if len(substrings) == 0 || anySubstringMatches(substrings, entry.Path, contentStr) {
							b.WriteString("# " + entry.Path + "\n")
							b.WriteString(contentStr + "\n\n")
						}
					}
				}
				output = b.String()

			case FormatList:
				var filteredFiles []string
				for _, entries := range entriesByRoot {
					for _, entry := range entries {
						if len(substrings) == 0 || anySubstringMatches(substrings, entry.Path, "") {
							filteredFiles = append(filteredFiles, entry.Path)
						}
					}
				}
				sort.Strings(filteredFiles)
				output = strings.Join(filteredFiles, "\n")

			case FormatTree:
				var b strings.Builder
				for root, entries := range entriesByRoot {
					rootNode := &TreeNode{IsDir: true, Children: make(map[string]*TreeNode)}
					hasEntries := false
					for _, entry := range entries {
						if len(substrings) == 0 || anySubstringMatches(substrings, entry.Path, "") {
							relPath, err := filepath.Rel(root, entry.Path)
							if err != nil {
								return fmt.Errorf("failed to get relative path: %w", err)
							}
							parts := strings.Split(relPath, string(os.PathSeparator))
							Insert(rootNode, parts, entry.IsDir)
							hasEntries = true
						}
					}
					if hasEntries {
						b.WriteString(root + "/\n")
						b.WriteString(Print(rootNode, "  "))
					}
				}
				output = b.String()

			default:
				slog.Error("internal error")
				continue
			}
			output = threeOrMoreNewlinesRegex.ReplaceAllString(output, "\n\n")
			output = strings.TrimSpace(output)
			outputs = append(outputs, output)
		}
		combinedOutput := strings.Join(outputs, "\n\n")

		// Perform the specified actions
		for _, action := range parsedActions {
			switch action {
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

// PreRunE validates the command-line flags before the main command executes.
func PreRunE(cmd *cobra.Command, args []string) error {
	// Expand the flag --dir (replace ~ with the user's home directory)
	var expandedDirs []string
	for _, dir := range dirs {
		expanded, err := expandTilde(dir)
		if err != nil {
			return err
		}
		expandedDirs = append(expandedDirs, expanded)
	}
	dirs = expandedDirs

	// Validate the flag --dir
	var invalidDirs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			invalidDirs = append(invalidDirs, dir)
		}
	}
	if len(invalidDirs) > 0 {
		return fmt.Errorf("directories are invalid: %s", strings.Join(invalidDirs, ", "))
	}

	// Validate the flag --dir-depth
	if dirDepth < -1 {
		return fmt.Errorf("directory depth is invalid: %d", dirDepth)
	}

	// Validate the flag --action
	var invalidActions []string
	for _, action := range actions {
		if _, err := parseAction(action); err != nil {
			invalidActions = append(invalidActions, action)
		}
	}
	if len(invalidActions) > 0 {
		return fmt.Errorf("actions are invalid: %s", strings.Join(invalidActions, ", "))
	}

	// Validate the flag --format
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
	logutils.Configure(logutils.Configuration{IsJSONEnabled: false})

	rootCmd.Flags().StringSliceVar(&dirs, "dir", []string{"."}, "Directories to search (comma-separated, default [.])")
	rootCmd.Flags().IntVar(&dirDepth, "dir-depth", -1, "Maximum directory depth to search (default -1, meaning infinite)")
	rootCmd.Flags().StringSliceVar(&exts, "ext", []string{}, "File extensions to include (comma-separated, default [])")
	rootCmd.Flags().StringSliceVar(&substrings, "substring", []string{}, "Substrings to filter files by (comma-separated, default [])")
	rootCmd.Flags().StringSliceVar(&actions, "action", []string{"print", "copy"}, "Actions to perform: print, copy (comma-separated, default print,copy)")
	rootCmd.Flags().StringSliceVar(&formats, "format", []string{"tree", "contents"}, "Output formats: tree, list, contents (comma-separated, default tree,contents)")

	rootCmd.PreRunE = PreRunE

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		help, _ := help()
		fmt.Println(help)
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
