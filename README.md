# jpkg
A very simple git package manager.

## Background
Package managers are complicated even though you just want to install only 1 simple plugin.
Especially, in neovim/vim, there are tons of package managers only for an editor.
I don't want to migrate the package manager every time when I want to revise the nvim config, and don't want to learn how to use the new libraries anymore.

I'm also tired to use TMP/oh-my-zsh/Vundle or something because these are essentially the same (git clone) but hard to use without documents, so I implemented this very simple package manager.

jpkg is also useful across any softwares that require git like package managers.

- [zsh example](https://github.com/jyane/dotfiles/blob/a71b60bc9e6c7a06bd5940a8267be1168c788071/install.sh#L38)
- [tmux example](https://github.com/jyane/dotfiles/blob/a71b60bc9e6c7a06bd5940a8267be1168c788071/install.sh#L39)

## Usage

Prepare a manifest file which is written in protocol buffer with text format. The proto definition is available in `proto/jpkg.proto`.
Only `url` is the required field.

```
repositories: [
  {
    url: "https://github.com/jyane/dotfiles.git"
    hash: "594b1f068266bbb7655c4163377f67ae9610dd4d"
  },
  {
    url: "https://github.com/jyane/tzcon.git"
  }
]
```

Running `jpkg --mode=install` will clone the git repository and checkout to HEAD or specified hash.
The command will automatically create a lock file.
If you have the lock file, the tool will see the lock file to install the packages.

## Examples
Neovim sees `${CONFIG}/start/` directory when it searches for plugins, so if you want to install the plugins, use `jpkg --mode=install --base-dir=start/`.

Running `jpkg` will print the usage.
