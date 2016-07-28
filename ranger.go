package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type stringmap map[string][]string

func (s *stringmap) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringmap) Set(value string) error {
	kv := strings.Split(value, ":")
	(*s)[kv[0]] = strings.Split(kv[1], ",")
	return nil
}

type Ranger struct {
	structure string
	filters   stringmap
	root      string
	unique    bool
	command   []string
}

func getCleanStructure(r Ranger) (string, bool) {
	cleanStructure := filepath.Clean(r.structure)
	isAbs := filepath.IsAbs(cleanStructure)

	if isAbs == false {
		cleanStructure = filepath.Join(r.root, cleanStructure)
		if filepath.IsAbs(r.root) {
			isAbs = true
		}
	}

	return cleanStructure, isAbs
}

func makeGlob(splitStructure []string, isAbs bool, filters map[string]string) string {

	globSlice := make([]string, 0)
	for _, el := range splitStructure {
		if el == "" {
			continue
		}
		//re := regexp.MustCompile("^@[^{}]*")
		//re := regexp.MustCompile("@{[^{}]*}")

		if strings.HasPrefix(el, "@") == false {
			globSlice = append(globSlice, el)
			continue
		}
		filter, ok := filters[el]
		if ok == true {
			globSlice = append(globSlice, filter)
		} else {
			globSlice = append(globSlice, "*")
		}
	}

	glob := filepath.Join(globSlice...)
	if isAbs {
		glob = "/" + glob
	}

	glob = filepath.FromSlash(glob)
	return glob
}

func makeGlobs(r Ranger) []string {

	cleanStructure, isAbs := getCleanStructure(r)
	splitStructure := strings.Split(filepath.ToSlash(cleanStructure), "/")

	totalFilters := 1
	for _, v := range r.filters {
		totalFilters *= len(v)
	}

	unrolledFilters := make([]map[string]string, totalFilters)
	for i := 0; i < totalFilters; i++ {
		unrolledFilters[i] = make(map[string]string)
	}

	for k, v := range r.filters {
		ncopies := totalFilters / len(v)
		for i := 0; i < totalFilters; i++ {
			unrolledFilters[i][k] = v[i/ncopies]
		}
	}

	globs := make([]string, 0)
	for _, filters := range unrolledFilters {
		glob := makeGlob(splitStructure, isAbs, filters)
		globs = append(globs, glob)
	}

	return globs
}

func runGlobs(globs []string) []string {

	allRes := make([]string, 0)
	for _, glob := range globs {
		res, _ := filepath.Glob(glob)
		for _, el := range res {
			allRes = append(allRes, el)
		}
	}

	return allRes
}

func makeCommands(r Ranger, globRes []string) [][]string {

	cleanStructure, _ := getCleanStructure(r)
	splitStructure := strings.Split(filepath.ToSlash(cleanStructure), "/")

	commands := make([][]string, len(globRes))

	for i, globPath := range globRes {
		splitPath := strings.Split(filepath.ToSlash(globPath), "/")

		vars := make(map[string]string)
		for i, el := range splitPath {
			if len(splitStructure) <= i {
				break
			}

			if strings.HasPrefix(splitStructure[i], "@") == false {
				continue
			}

			vars[splitStructure[i]] = el
		}

		command := make([]string, 0)
		for _, el := range r.command {
			arg := el
			outArg := arg

			for k, v := range vars {
				absSlice := make([]string, 0)
				for i, p := range splitStructure {
					absSlice = append(absSlice, splitPath[i])
					if p == k {
						break
					}
				}
				absPath := filepath.Join(absSlice...)
				if filepath.IsAbs(globPath) {
					absPath = "/" + absPath
				}
				if r.root != "" {
					absPath, _ = filepath.Rel(r.root, absPath)
				}

				re := regexp.MustCompile("@" + k)
				outArg = re.ReplaceAllString(outArg, absPath)

				re = regexp.MustCompile("@@{" + k[1:] + "}")
				outArg = re.ReplaceAllString(outArg, absPath)

				re = regexp.MustCompile("@{" + k[1:] + "/.*/.*}")
				matches := re.FindAllString(outArg, -1)

				for _, match := range matches {
					splitMatch := strings.Split(match[2:len(match)-1], "/")
					if len(splitMatch) != 3 {
						continue
					}
					subRe := regexp.MustCompile(splitMatch[1])
					subsArg := subRe.ReplaceAllString(v, splitMatch[2])
					subRe = regexp.MustCompile(match)
					outArg = subRe.ReplaceAllString(outArg, subsArg)
				}

				re = regexp.MustCompile("@{" + k[1:] + "}")
				outArg = re.ReplaceAllString(outArg, v)

				re = regexp.MustCompile(k)
				outArg = re.ReplaceAllString(outArg, v)
			}

			command = append(command, outArg)
		}

		commands[i] = command
	}

	if r.unique {
		commandMap := make(map[string][]string)

		for _, command := range commands {
			commandMap[strings.Join(command, " ")] = command
		}

		commands = make([][]string, len(commandMap))
		i := 0
		for _, command := range commandMap {
			commands[i] = command
			i++
		}
	}

	return commands
}

func main() {

	// TODO COMMANDS:
	// take, take-nth, take-rand
	// count (count is like unique + echo, but it also shows the numerosity)
	// tabulate

	// TODO FEATURES:
	// parallelization
	// laziness (for this and the previous, use channels and have goroutines consume)

	root := flag.String("root", "", "Path relative to which `structure` is evaluated.")
	unique := flag.Bool("unique", false, "Only call `command` once for a unique combination of arguments.")
	structure := flag.String("structure", "", "Path-like string describing the directory structure being visited. Variables can be defined by prepending @ to a name, e.g. /some/directory/@subdir/@filename.")

	debug := flag.Bool("debug", false, "Output debug info.")
	echo := flag.Bool("echo", false, "Only show commands, do not execute them.")
	log := flag.Bool("log", false, "Show commands while executing.")

	filters := make(stringmap)
	flag.Var(&filters, "filter", "Filters acting on variables defined in `structure`. A filter is given as variable_name:glob_pattern, e.g. -filter @filename:*.txt.")

	flag.Usage = func() {
		fmt.Printf("Program: ranger 0.1\n")
		fmt.Printf("Author:  Luca Antiga, Orobix\n")
		fmt.Printf("Usage:   ranger [-root path] [-structure path] [-filter filter] [-filter filter] [-unique] [-debug] [-log] [-echo] command\n")
		fmt.Printf("Example: ranger -root /home/joe/data -structure @department/@expenses -filter @department:*ENG -filter @expenses:2016*.tab sed -e 's/\\t/, /g' < @@expenses > out/@{expenses/tab/csv}\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	ranger := Ranger{
		structure: *structure,
		filters:   filters,
		root:      *root,
		unique:    *unique,
		command:   flag.Args()}

	if len(flag.Args()) == 0 {
		flag.Usage()
		return
	}

	if *debug {
		fmt.Println("Ranger:", ranger)
	}

	globs := makeGlobs(ranger)
	if *debug {
		fmt.Println("Globs:", globs)
	}

	globRes := runGlobs(globs)
	if *debug {
		fmt.Println("GlobRes:", globRes)
	}

	commands := makeCommands(ranger, globRes)
	if *debug {
		fmt.Println("Commands:", commands)
	}

	for _, command := range commands {
		if *debug {
			fmt.Println("Running:", strings.Join(command, " "))
		}
		if *echo {
			fmt.Println(strings.Join(command, " "))
			continue
		}
		if *log {
			fmt.Println(strings.Join(command, " "))
		}
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}
