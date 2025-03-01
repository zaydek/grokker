<!-- https://grok.com/chat/81e884f7-4ee3-4283-9a5d-f2e6c11bf9d0 -->

# `grokker` - A Command-Line Tool for File Processing and AI Prompting

`grokker` is a versatile command-line tool designed to process files in specified directories, making it easier to generate structured inputs for AI models like Grok 3. It allows users to filter files based on extensions and substrings, and then output the results in various formats such as a directory tree, a list of file paths, or the contents of the files. The output can be printed to the console or copied to the clipboard, facilitating seamless integration into workflows involving AI prompting.

---

## Why `grokker` Exists

`grokker` was created to simplify the process of preparing file-based inputs for AI prompting, particularly with tools like Grok 3. When working with AI models on tasks involving codebases or large sets of files, gathering and structuring the necessary data can be time-consuming and error-prone. `grokker` automates this process by providing a flexible way to filter files, visualize directory structures, and extract contents, all tailored to the needs of AI-driven workflows.

---

## Installation and Global Usage

`grokker` is designed to be a global CLI tool that can be invoked from any project directory. To install it, use the following command:

```bash
go install github.com/yourusername/grokker@latest
```

Once installed, you can use `grokker` from any directory, including those with `~` (home directory) and `./` (current directory) paths. This makes it easy to integrate into your workflow, whether you're working on a single project or managing multiple repositories.

### Example Use Case: Semantic Refactoring with Grok 3

One powerful way to use `grokker` is for broad, semantically related refactoring tasks. For instance, if you need to refactor code related to concepts like "store" or "e-commerce," you can use `grokker` to pull out the relevant files and their directory structure as plaintext. Here's how:

1. **Run `grokker` with Substring Filters**:
   ```bash
   grokker --substring=store,e-commerce --format=tree,contents --action=copy
   ```
   This command filters files containing "store" or "e-commerce" in their paths or contents, generates the directory tree and file contents, and copies the output to the clipboard.

2. **Input to Grok 3**:
   Paste the copied output into Grok 3 as part of your prompt. The structured data (tree and contents) provides Grok 3 with the context it needs to understand the project structure and make high-quality suggestions.

3. **Receive and Apply Changes**:
   Grok 3 can return refactored code or suggestions based on the input. You can then directly paste these changes back into your editor (e.g., VS Code or Cursor). For more targeted changes, ask Grok 3 to provide entire files for any modified content, allowing you to replace files one at a time.

This workflow leverages `grokker` to streamline the process of gathering and structuring data for AI-driven refactoring, making it faster and more efficient to implement broad changes across your codebase.

---

## Problems Solved by `grokker`

- **File Selection and Filtering**: Quickly select files based on their extensions or substrings in their paths or contents.
- **Directory Structure Visualization**: Generate a clear, hierarchical view of the directory tree to provide context for AI models or human users.
- **Content Extraction**: Efficiently extract and format the contents of multiple files for use in prompts or other applications.
- **Output Flexibility**: Choose whether to print the output to the console, copy it to the clipboard, or both, streamlining integration into various workflows.

---

## Usage

The basic syntax of `grokker` is:

```bash
grokker [flags]
```

### Flags

- **`--dir strings`**
  Specifies the directories to search. Multiple directories can be provided as a comma-separated list. Use `~` to represent the home directory.
  - **Default**: `"."` (current directory)

- **`--dir-depth int`**
  Sets the maximum directory depth to search. A value of `-1` means infinite depth.
  - **Default**: `-1`

- **`--ext strings`**
  Specifies the file extensions to include. Extensions must include the leading dot (e.g., `.ts`, `.tsx`). Multiple extensions can be provided as a comma-separated list. If not specified, all files are included.
  - **Default**: `[]` (all files)

- **`--substring strings`**
  Specifies substrings to filter files by. Files are included if their paths or contents contain any of the specified substrings (case-insensitive for paths). Multiple substrings can be provided as a comma-separated list. If not specified, all files (after extension filtering) are included.
  - **Default**: `[]` (all files)

- **`--action strings`**
  Specifies the actions to perform on the output. Possible values are `print` (print to console) and `copy` (copy to clipboard). Multiple actions can be provided as a comma-separated list.
  - **Default**: `"print,copy"`

- **`--format strings`**
  Specifies the output formats to generate. Possible values are `tree` (directory tree), `list` (list of file paths), and `contents` (file contents). Multiple formats can be provided as a comma-separated list, and they will be concatenated in the output.
  - **Default**: `"tree,contents"`

### Notes

- If no flags are provided, `grokker` processes all files in the current directory, generates the directory tree and file contents, and both prints and copies the output.
- For large operations (more than 50 files), `grokker` prompts for confirmation to prevent accidental heavy processing.

---

## Examples

Below are several examples that demonstrate how to use `grokker` in different scenarios, including the use of the `tree` format to showcase its utility.

### 1. Basic Usage

Running `grokker` without any flags processes all files in the current directory, generates the directory tree and file contents, and both prints and copies the output.

```bash
grokker
```

**Sample Output** (assuming a simple project structure):

```
./
  main.go
  utils.go

# ./main.go
package main
import "fmt"
func main() {
    fmt.Println("Hello, world!")
}

# ./utils.go
package main
func util() string {
    return "Utility function"
}
```

### 2. Filtering by Substrings

To print the list of files that contain the substring "store" in their paths:

```bash
grokker --substring=store --action=print --format=list
```

**Sample Output**:

```
app/store.js
lib/storeUtils.js
```

### 3. Filtering by Extensions

To copy the contents of all `.js` files in the `app` directory to the clipboard:

```bash
grokker --dir=app --ext=.js --action=copy --format=contents
```

**Sample Output** (copied to clipboard):

```
# app/store.js
function createStore() {
    return {};
}
```

### 4. Combining Filters and Formats

To print and copy the directory tree and contents of `.ts` and `.tsx` files in the `foo` and `bar` directories that contain "bar" or "baz" in their paths or contents:

```bash
grokker --dir=foo,bar --substring=bar,baz --ext=.ts,.tsx --action=print,copy --format=tree,contents
```

**Sample Output** (printed and copied):

```
foo/
  barComponent.tsx
  bazHelper.ts
bar/
  barUtils.ts

# foo/barComponent.tsx
import React from 'react';
export const BarComponent = () => <div>Bar</div>;

# foo/bazHelper.ts
export function baz() {
    return "Baz";
}

# bar/barUtils.ts
export function barUtil() {
    return "Bar Utility";
}
```

### 5. Visualizing Directory Structure with `tree`

To visualize the directory structure of the current directory, including only `.md` files:

```bash
grokker --ext=.md --format=tree --action=print
```

**Sample Output**:

```
./
  README.md
  docs/
    guide.md
    reference.md
```

This example demonstrates how `grokker` can help users understand the structure of their project, which is particularly useful when preparing context for AI models.

### 6. Using Multiple Directories and Depth Control

To print the tree of `.go` files in the `src` and `pkg` directories, limiting the search to a depth of 1:

```bash
grokker --dir=src,pkg --ext=.go --dir-depth=1 --format=tree --action=print
```

**Sample Output**:

```
src/
  main.go
pkg/
  utils.go
```

---

## Advanced Usage

`grokker` supports more advanced scenarios to enhance its utility:

- **Combining Multiple Formats**: Generate both the directory tree and the list of files for a quick overview.

  ```bash
  grokker --format=tree,list --action=print
  ```

  **Sample Output**:

  ```
  ./
    app/
      store.js
    lib/
      storeUtils.js

  app/store.js
  lib/storeUtils.js
  ```

- **Handling Large Numbers of Files**: When processing more than 50 files, `grokker` prompts for confirmation. For example:

  ```bash
  grokker --dir=large_project --ext=.py
  ```

  **Console Interaction**:

  ```
  WARNING: Processing 120 files. Proceed? [y/N]
  ```

- **Integration with Other Tools**: Pipe the output to other command-line tools for further processing.

  ```bash
  grokker --format=list --action=print | grep "test"
  ```

---

## Conclusion

`grokker` is a powerful and flexible tool that simplifies the process of gathering and formatting file data, especially for AI prompting with tools like Grok 3. Its ability to filter files by extensions and substrings, generate structured outputs like directory trees, and handle the results with customizable actions makes it an invaluable asset for developers and AI practitioners. Whether you're preparing data for code analysis, documentation generation, or debugging, `grokker` streamlines your workflow and enhances productivity.

Try `grokker` in your next project and see how it can transform the way you work with files and AI!
