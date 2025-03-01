<!-- https://grok.com/chat/81e884f7-4ee3-4283-9a5d-f2e6c11bf9d0 -->

# `grokker` - A Command-Line Tool for Grokking Files

`grokker` is a command-line tool intended to be used in conjunction with AI models like Grok 3 to make it easier to give eyes to the directory structure and files you are working with. It is akin to `grep` but offers a streamlined set of flags to make it easier to just get at the folders and files you are looking for.

Use `grokker` to save you time and energy wrestling with convoluted Unix commands like `find . -type f -name "*.js" | grep "store" | xargs -I {} bash -c 'echo "# {}"; cat {}'`.

For example:

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
- **Scan all files and folders in the current directory with file names and contents that match substrings `foo`, `bar`**:
  ```bash
  grokker --substring=foo,bar
  ```

One of the neat features of `grokker` is the `format` and `action` flags.

- The `format` flag allows you to specify the output format of the files and folders you are scanning.

  The available formats are:

  - `tree`: A directory tree of the files and folders.
  - `list`: A list of file paths.
  - `contents`: The contents of the files.

  Formats can also be used in combination, for example:

  - Show the tree and file contents:
    ```bash
    grokker --format=tree,contents
    ```

- The `action` flag allows you to specify what you want to do with the output.

  The available actions are:

  - `print`: Print the output to the console.
  - `copy`: Copy the output to the clipboard. **Note**: At present this depends on `pbcopy` which is only available on macOS.

  Actions can also be used in combination, for example:

  - Print and copy the output:
    ```bash
    grokker --action=print,copy
    ```

## Install Grokker

**Note**: `grokker` is written in Go and assumes you have Go installed on your system. If you do not already have Go installed, you can download it from the [official website](https://golang.org/dl/).

To install `grokker`, use the following command:

```bash
go install github.com/zaydek/grokker@latest
```

Once installed, you should be able to invoke `grokker` from anywhere even without calling `source` or other shell commands.

## Flow with Grokker

I built `grokker` for myself to help with the following use case: You are working on a complicated code base and do not trust VS Code Copilot or Cursor with significant refactors. Instead, you want to use a frontier model such as Grok 3 that at the time of this writing is not available in the API (surely to change shortly). Use `grokker` to grep for the files you are looking for in a structured way. I have found this to be a very powerful way to use Grok 3 without wasting energy or effort. Finally, copy Grok's output as an input for VS Code's Copilot or Cursor OR simply ask Grok to give you the entire file for anything it has changed and paste files one at a time, depending on what you're interested in doing.

### Flags

- **`--dir=[string,...string]`**
  Specifies the directories to search. Multiple directories can be provided as a comma-separated list such as `--dir=path/to/dir1,path/to/dir2`.

  - **Default**: `--dir=.` (current directory)
  - **Note**: Syntax expansion is supported for:
    - `~` (home directory)
    - `./` (current directory)
    - `../` (parent directory)

- **`--dir-depth=int`**
  Sets the maximum recursion depth for directories. If you specify `1`, `grokker` will only search the top-level directory. You should generally not need to manually set this unless you have an arbitrarily deep directory structure.

  - **Default**: `--dir-depth=-1` (unlimited depth)

- **`--ext=[string,...string]`**
  Specifies the file extensions to include. Extensions must include the leading dot (e.g., `.ts`, `.tsx`). Multiple extensions can be provided as a comma-separated list such as `--ext=.ts,.tsx`.

  - **Default**: `--ext=[]` (include all files, does not filter by extension)

- **`--substring=[string,...string]`**
  Specifies substrings to filter file names or contents by. Multiple substrings can be provided as a comma-separated list such as `--substring=foo,bar,"hello world"`.

  - **Default**: `[]` (all files)
  - **Note**: Substring matching is case-sensitive.
  - **Note**: Substrings may be unquoted. If the substring uses special characters, use double quotes or single quotes (recommended). For example, `--substring="hello world"` and `--substring='hello world'`.

- **`--action=[action,...action]`**
  Specifies the actions to perform on the output. Multiple actions can be provided as a comma-separated list such as `--action=print,copy`.

  - **Valid actions**: `print`, `copy`
    - **`print`**: Prints the output to the console.
    - **`copy`**: Copies the output to the clipboard.
  - **Default**: `"print,copy"`

- **`--format=[format,...format]`**
  Specifies the output formats to generate. Multiple formats can be provided as a comma-separated list, and they will be concatenated in the output such as `--format=tree,contents`.
  - **Valid formats**: `tree`, `list`, `contents`
    - **`tree`**: Generates a hierarchical directory tree. Use `tree` when you want to visualize the directory structure.
    - **`list`**: Generates a flat list of file paths. Use `list` when you want to list files akin to `ls -1`.
    - **`contents`**: Generates the contents of the files.
  - **Default**: `"tree,contents"`
  - **Note**: `tree` prints file paths hierarchically but the output is not identical to the `tree` command. For example:
    - `tree`:
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
