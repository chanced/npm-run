package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/fatih/color"
)

func main() {
	cmd := command()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	fmt.Println(cmd)
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

func command() *exec.Cmd {
	var c Command
	if len(os.Args) > 1 {
		c = parse(os.Args)
	} else {
		c = prompt()
	}
	fmt.Printf("%+v", c)

	return exec.Command("npm", c.args()...)
}

type Command struct {
	Script     string `survey:"script"`
	Arguments  []string
	Workspaces []string `survey:"workspaces"`
	// Args from prompt
	Args string `survey:"args"`
}

func (c Command) args() []string {
	result := []string{"run", c.Script}
	for _, w := range c.Workspaces {
		w = strings.Trim(w, " ")
		if w != "" {
			result = append(result, "--workspace="+w)
		}
	}
	if c.Args != "" {
		var quote *rune
		args := strings.FieldsFunc(c.Args, func(r rune) bool {
			switch r {
			case '"', '`', '\'':
				if quote == nil {
					quote = &r
				} else {
					quote = nil
				}
			}
			return quote == nil && r == ' '
		})
		if quote != nil {
			color.Red("Unmatched quote:", *quote)
			os.Exit(1)
		}
		for i, arg := range args {
			args[i] = strings.TrimFunc(arg, func(r rune) bool {
				return r == '"' || r == '\''
			})
		}
		c.Arguments = append(c.Arguments, args...)
	}

	if len(c.Arguments) > 0 {
		result = append(result, "--")
		result = append(result, c.Arguments...)
	}
	return result
}

type packageJSON struct {
	Name       string            `json:"name"`
	Scripts    map[string]string `json:"scripts"`
	Workspaces []string          `json:"workspaces"`
}

// returns arguments, workspaces
func parse(args []string) Command {
	cmd := Command{
		Workspaces: []string{},
	}
	args = args[1:]
	foundNonFlag := false
	cp := make([]string, 0, len(args))
	nextIsWorkspace := false
	for _, a := range args {
		a = strings.Trim(a, " ")
		if a == "" {
			continue
		}
		if !foundNonFlag && strings.HasPrefix(a, "-w") {
			if strings.HasPrefix(a, "-w=") {
				cmd.Workspaces = append(cmd.Workspaces, strings.Split(strings.Replace(a, "-w=", "", 1), ",")...)
				nextIsWorkspace = false
			} else {
				nextIsWorkspace = true
			}
			continue
		}
		if strings.HasPrefix(a, "--workspace") {
			if strings.HasPrefix(a, "--workspace=") {
				cmd.Workspaces = append(cmd.Workspaces, strings.Split(strings.Replace(a, "--workspace=", "", 1), ",")...)
				nextIsWorkspace = false
			} else {
				nextIsWorkspace = true
			}
			continue
		}
		if nextIsWorkspace {
			cmd.Workspaces = append(cmd.Workspaces, a)
			nextIsWorkspace = false
			continue
		}
		foundNonFlag = true
		if cmd.Script == "" {
			cmd.Script = a
			continue
		}
		cp = append(cp, a)
	}
	cmd.Arguments = cp
	return cmd
}

func openPackage(p string) packageJSON {
	pkgpath := path.Join(p, "package.json")
	data, err := os.ReadFile(pkgpath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			color.Red("package.json not found: %s", pkgpath)
		} else {
			color.Red("error reading package.json")
		}
		os.Exit(1)
	}
	var pkg packageJSON
	err = json.Unmarshal(data, &pkg)
	if err != nil {
		color.Red("error parsing package.json: %s", p)
		os.Exit(1)
	}
	return pkg
}

func loadWorkspacePackages(pkg packageJSON) map[string]packageJSON {
	if pkg.Workspaces == nil {
		return map[string]packageJSON{}
	}
	result := map[string]packageJSON{}
	for _, pattern := range pkg.Workspaces {
		doublestar.GlobWalk(os.DirFS(cwd()), pattern, func(path string, d fs.DirEntry) error {
			if !d.IsDir() {
				return nil
			}
			wp := openPackage(path)
			if _, ok := result[wp.Name]; ok {
				color.Red("duplicate package name:", wp.Name)
				os.Exit(1)
			}
			result[wp.Name] = wp
			return nil
		})
	}
	return result
}

func cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		color.Red("error getting current directory")
		fmt.Println(err)
		os.Exit(1)
	}
	return cwd
}

func keys(m map[string]string) []string {
	res := make([]string, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	return res
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	s[len(s)-1] = ""
	s = s[:len(s)-1]
	return s
}

func find(s []string, v string) int {
	for i, val := range s {
		if val == v {
			return i
		}
	}
	return -1
}

func intersectScripts(workspaces []packageJSON) []string {
	if len(workspaces) == 0 {
		return []string{}
	}
	if len(workspaces) == 1 {
		return keys(workspaces[0].Scripts)
	}
	t := make(map[string]struct{}, len(workspaces[0].Scripts))
	for k := range workspaces[0].Scripts {
		t[k] = struct{}{}
	}

	for _, pkg := range workspaces[1:] {
		for k := range t {
			if _, ok := pkg.Scripts[k]; !ok {
				delete(t, k)
			}
		}
	}
	res := make([]string, 0, len(t))
	for k := range t {
		res = append(res, k)
	}
	return res
}

func workspaceNames(ws map[string]packageJSON) []string {
	res := make([]string, 0, len(ws))
	for k := range ws {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func prompt() Command {
	var a Command

	scripts := []string{}
	pkg := openPackage(cwd())
	ws := loadWorkspacePackages(pkg)

	if len(ws) > 0 {
		prompt := &survey.MultiSelect{
			Message: "Workspaces:",
			Options: workspaceNames(ws),
		}

		err := survey.AskOne(prompt, &a.Workspaces)
		if err != nil {
			color.Red("error getting workspace selection")
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if len(a.Workspaces) > 0 {
		wss := []packageJSON{}
		for _, k := range a.Workspaces {
			wss = append(wss, ws[k])
		}
		scripts = intersectScripts(wss)
	} else {
		for k := range pkg.Scripts {
			scripts = append(scripts, k)
		}
	}
	if len(scripts) == 0 {
		color.Red("no common scripts found")
		os.Exit(1)
	}
	sort.Strings(scripts)

	qs := []*survey.Question{}

	qs = append(qs, &survey.Question{
		Name: "script",
		Prompt: &survey.Select{
			Message: "Script:",
			Options: scripts,
		},
		Validate: survey.Required,
	})
	qs = append(qs, &survey.Question{
		Name: "args",
		Prompt: &survey.Input{
			Message: "Arguments:",
		},
	})

	err := survey.Ask(qs, &a)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
	return a
}

// TODO: there doesn't seem to be a way to get the list of workspaces from
// npm in order to prompt, the globs would need to be expanded and each
// package.json file would need to be read for the name.

// this strategy also does not take into account that each workspace would
// presumably have different scripts. This would require a different
// approach. Workspaces would need to be selected first which I'm not crazy
// about for my own workflow.

// given that this is just a simple utility for me and I doubt I'll use the
// prompts much, if at all this is more effort than I'm willing to put in at
// the moment, plus it'd slow things down
