# Ranger v0.1

### Run commands in nested directory structures.

`ranger` lets you define a directory structure using a path-like string, using placeholders to indicate variables, e.g. `./data/@department/@expenses`. Ranger will visit the subdirectories, associate appropriate values to `@department` and `@expenses` and run a command replacing variable names with values. You can define filters on variables or transform them in the command.

## Build

To build, just run `go build ranger.go`. Then move the `ranger` executable somewhere in your `PATH`.

## Usage

```
ranger [-root path] [-structure path] \
       [-filter filter] [-filter filter] \
       [-unique] [-debug] [-log] [-echo] \
       command

-help
  Show help.
-debug
  Output debug info.
-echo
  Only show commands, do not execute them.
-filter structure
  Filters acting on variables defined in structure.
  A filter is given as variable_name:glob_pattern,
  e.g. -filter @filename:*.txt. (default map[]).
-log
  Show commands while executing.
-root structure
  Path relative to which structure is evaluated.
-structure string
  Path-like string describing the directory structure being visited.
  Variables can be defined by prepending @ to a name, 
  e.g. /some/directory/@subdir/@filename.
-unique command
  Only call command once for a unique combination of arguments.
```

## Example

Starting from `/home/joe/data`, visit every entry in `*/*`, associating the name of the subdirectory to `@department` and the files therein to `@expenses`. Only visit departments ending with `ENG` and expense files starting with `2016` and having a `.tab` extension. For each file in each subdirectory matching the criteria, replace tabs with commas (using `sed -e 's/\t/, /g`) and save the result in `/home/joe/out/`, changing the extension from `.tab` to `.csv`:

```
ranger -root /home/joe/data \
       -structure @department/@expenses \
       -filter @department:*ENG \
       -filter @expenses:2016*.tab \
       sed -e 's/\t/, /g' @@expenses > \
       /home/joe/out/@{expenses/tab/csv}
```


## Limitations

* Currently `ranger` collects all entries and then runs all commands. It should do that lazily instead.
* Variable substitution `@{foo/txt/csv}` currently doesn't accept regex's.
* Commands are executed serially. They could be run in parallel with a defined maximum number of processes.

##License

MIT license http://www.opensource.org/licenses/mit-license.php/

Copyright (C) 2014-2015 Luca Antiga http://lantiga.github.io

