# `grokker` - A Command-Line Tool for Grokking Files

`grokker` is a command-line tool intended to be used in conjunction with AI models like Grok 3 to make it easier to give eyes to the directory structure and files you’re working with. It’s akin to `grep` but offers a streamlined set of flags to just get at the folders and files you’re looking for.

Instead of wrestling with convoluted Unix commands that might not even work—like `find . -type f -name "*.js" | grep "store" | xargs -I {} bash -c 'echo "# {}"; cat {}'`—you can simply use `grokker`. Here’s how:

- **Scan all files and folders in the current directory**:
  ```bash
  grokker
  ```
- **Scan all files and folders in the current directory up to one level deep**:
  ```bash
  grokker --dir-depth=1
  ```
- **Scan all files and folders in the current directory with file extensions `.ts`, `.tsx`**:
  ```bash
  grokker --ext=.ts,.tsx
  ```
- **Scan all files and folders in the current directory with file names or contents matching substrings `foo`, `bar`**:
  ```bash
  grokker --substring=foo,bar
  ```

One of the neat features of `grokker` (not `grok`, oops!) is the `format` and `action` flags.

- The `format` flag lets you pick how the output looks:
  - `tree`: A directory tree of the files and folders.
  - `list`: A plain list of file paths.
  - `contents`: The actual contents of the files.
  - Combine them if you want, like this:
    ```bash
    grokker --format=tree,contents
    ```
    That’ll show the tree *and* file contents.

- The `action` flag decides what happens with the output:
  - `print`: Prints it to the console.
  - `copy`: Copies it to the clipboard. (**Note**: This uses `pbcopy`, so it’s macOS-only for now.)
  - Use both together if you’re feeling fancy:
    ```bash
    grokker --action=print,copy
    ```

## Install Grokker

**Note**: `grokker` is written in Go, so you’ll need Go installed. If you don’t have it, grab it from the [official website](https://golang.org/dl/).

To install `grokker`, run:

```bash
go install github.com/zaydek/grokker
```

Once it’s in, you can call `grokker` from anywhere—no need for `source` or other shell tricks. Check it with `grokker --help` to make sure it’s there.

## Flow with Grokker

I built `grokker` for myself to tackle this: You’re deep in a messy codebase and don’t trust VS Code Copilot or Cursor for big refactors. Instead, you want a heavy hitter like Grok 3, which (at the time of writing) isn’t available via API—though that’s bound to change soon. With `grokker`, you can quickly grab the files you need in a structured way. I’ve found this super powerful for feeding context to Grok 3 without burning time or energy. Then, take Grok’s output and paste it into Copilot or Cursor, or just ask Grok for the updated tree and files—it’s up to you.

### Flags

- **`--dir=[string,...string]`**
  Tells `grokker` which directories to search. Throw in multiple ones with commas, like `--dir=path/to/dir1,path/to/dir2`.
  - **Default**: `--dir=.` (current directory)
  - **Note**: It handles shortcuts like `~` (home), `./` (here), and `../` (up one).

- **`--dir-depth=int`**
  Sets how deep `grokker` digs into subdirectories. Use `1` to stay shallow, or leave it at `-1` for everything. You probably won’t mess with this unless your folders are a rabbit hole.
  - **Default**: `--dir-depth=-1` (unlimited depth)

- **`--ext=[string,...string]`**
  Picks files by their extensions. Include the dot (e.g., `.ts`, `.tsx`). List multiple with commas, like `--ext=.ts,.tsx`.
  - **Default**: `--ext=[]` (grabs all files, no filtering)

- **`--substring=[string,...string]`**
  Filters files by substrings in their names *or* contents. Use commas for multiple, like `--substring=foo,bar,"hello world"`.
  - **Default**: `[]` (no filtering, all files)
  - **Note**: It’s case-sensitive.
  - **Note**: If your substring has spaces or weird characters, quote it—e.g., `--substring="hello world"` or `--substring='foo.bar'`.

- **`--action=[action,...action]`**
  Decides what to do with the output. Mix and match with commas, like `--action=print,copy`.
  - **Valid actions**:
    - **`print`**: Dumps it to the console.
    - **`copy`**: Copies it to the clipboard.
  - **Default**: `"print,copy"`

- **`--format=[format,...format]`**
  Controls how the output looks. Combine them with commas, like `--format=tree,contents`.
  - **Valid formats**:
    - **`tree`**: Shows a hierarchical tree—great for seeing the structure.
    - **`list`**: Gives a flat list of paths, like `ls -1`.
    - **`contents`**: Spits out the file contents.
  - **Default**: `"tree,contents"`
  - **Note**: The `tree` output isn’t exactly like the `tree` command. Compare:
    - `tree` command:
      ```
      .
      ├── app
      │   └── store.js
      └── lib
          └── storeUtils.js
      ```
    - `grokker`:
      ```
      ./
        app/
          store.js
        lib/
          storeUtils.js
      ```

## Examples

- **Process all files in the current directory and print+copy the contents**:
  ```bash
  grokker --dir=.
  ```

- **Print the list of files with "store" in the path**:
  ```bash
  grokker --substring=store --action=print --format=list
  ```

- **Copy the contents of `.js` files in `app/` to clipboard**:
  ```bash
  grokker --dir=app --ext=.js --action=copy --format=contents
  ```

- **Print and copy the tree and contents of `.ts`/`.tsx` files with "bar" or "baz"**:
  ```bash
  grokker --dir=foo,bar --substring=bar,baz --ext=.ts,.tsx --action=print,copy --format=tree,contents
  ```

## License

This project is licensed under the MIT License. See the [LICENSE.md](LICENSE.md) file for details.
