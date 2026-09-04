# dotlink-lint

Most dotfiles repos end up with a shell script that does `ln -sf` a dozen
times, and it silently overwrites whatever was at the target before. This is
a small checker for that: you describe the symlinks you want in a plain text
manifest, and it tells you what's already correct, what would be created,
and what's actually going to clobber something, without touching the
filesystem.

## Manifest format

One entry per line: `source -> target`. `source` is resolved relative to
`--root` (your dotfiles checkout). `target` must be absolute or start with
`~`. Blank lines and lines starting with `#` are ignored.

```
# ~/dotfiles/links.manifest
zsh/zshrc -> ~/.zshrc
git/gitconfig -> ~/.gitconfig
nvim -> ~/.config/nvim
tmux/tmux.conf -> ~/.tmux.conf
```

## Usage

```
$ dotlink-lint --root ~/dotfiles ~/dotfiles/links.manifest
   1  ok              zsh/zshrc -> ~/.zshrc
   2  pending         git/gitconfig -> ~/.gitconfig
   3  blocked         nvim -> ~/.config/nvim  (/home/me/.config/nvim already exists and is not a symlink)
   4  missing_source  tmux/tmux.conf -> ~/.tmux.conf  (/home/me/dotfiles/tmux/tmux.conf: no such file or directory)

4 entries: 1 ok, 1 pending, 1 blocked, 0 conflict, 1 missing source, 1 invalid
```

Add `--json` to get the same data as a structured array instead, for use in
a setup script or CI check:

```
$ dotlink-lint --json --root ~/dotfiles ~/dotfiles/links.manifest
[
  {
    "line": 1,
    "status": "ok",
    "source": "zsh/zshrc",
    "target": "~/.zshrc"
  },
  {
    "line": 3,
    "status": "blocked",
    "source": "nvim",
    "target": "~/.config/nvim",
    "detail": "/home/me/.config/nvim already exists and is not a symlink"
  }
]
```

Add `--apply` to actually create the links for every `pending` entry. It
never touches `ok`, `blocked`, `conflict`, `missing_source`, or `invalid`
entries - those are left exactly as reported, so a blocked file is never
silently overwritten. Missing parent directories under the target are
created as needed. After applying, the same report is printed again so you
can see the final state, including anything that failed to link:

```
$ dotlink-lint --apply --root ~/dotfiles ~/dotfiles/links.manifest
   1  ok              zsh/zshrc -> ~/.zshrc
   2  ok              git/gitconfig -> ~/.gitconfig
   3  blocked         nvim -> ~/.config/nvim  (/home/me/.config/nvim already exists and is not a symlink)
   4  missing_source  tmux/tmux.conf -> ~/.tmux.conf  (/home/me/dotfiles/tmux/tmux.conf: no such file or directory)

4 entries: 2 ok, 0 pending, 1 blocked, 0 conflict, 1 missing source, 0 invalid
```

## Statuses

- `ok` - target is already a symlink pointing at source.
- `pending` - target doesn't exist yet; creating the link is safe.
- `blocked` - target exists and is a real file or directory, not a symlink.
- `conflict` - target is a symlink to somewhere else, or two entries claim
  the same target.
- `missing_source` - source doesn't exist under `--root`.
- `invalid` - the manifest line itself couldn't be parsed.

The exit code is non-zero if any entry is `blocked`, `conflict`,
`missing_source`, or `invalid`, so it's usable as a CI gate on a dotfiles
repo.

## Build

```
go build -o dotlink-lint .
```

No dependencies beyond the standard library.
