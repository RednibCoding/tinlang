# VS Code language support for the Tin Scripting Language
Simple syntax highlighting for vs code

## Package the extension
In the root of the `tin-lang` directory run:
```
vsce package
```
This will create a `.vsix` file

If you don't have **vsce** installed (which is needed to build a `.vsix`-file) run the following command:
```
npm install -g vsce
```
>Note: you only need `vsce` if you want to build the plugin yourself, otherwise proceed with [Install the plugin](#install-the-plugin)

## Install the plugin
To install the plugin from the `.vsix` file open the vscode command palette (ctrl + shift + P) and search for Install from VSIX. Then select the `tin-lang-x.x.x.vsix`-file mentioned above.
