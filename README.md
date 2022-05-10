# run

A simple program that runs local npm scripts.

[![asciicast](https://asciinema.org/a/Ei8b8iZBkj7qhYkPNMtC7WwFj.svg)](https://asciinema.org/a/Ei8b8iZBkj7qhYkPNMtC7WwFj)

## Install

```
go install github.com/chanced/run@latest
```

## Features

### Run a known script

Run npm scripts directly with arguments:

```bash
run [script-name] [args...]
```

#### Workspaces

Specify any number of workspaces but they must come before the script name.

```bash
run -w my-workspace my-script --arg value
```

### Discover scripts

Execute `run` to enter prompt mode which offers a guided selection of scripts.

#### Workspaces

If the root `package.json` contains `workspaces`, the globs/paths are expanded
and then loaded concurently. Once all `package.json` have been loaded and
parsed, the workspaces are offered as the first prompt.

If a single workspace is selected, the scripts available are presented as the
next prompt.

If multiple workspaces are selected, only the scripts common between the
selections.

## License

MIT
