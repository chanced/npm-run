package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

func main() {
	cmd := command()
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
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
	return exec.Command("npm", c.args()...)
}

type Command struct {
	Script     string   `survey:"script"`
	Arguments  string   `survey:"arguments"`
	Workspaces []string `survey:"workspaces"`
}

func (c Command) args() []string {
	args := []string{"run", c.Script}
	for _, w := range c.Workspaces {
		w = strings.Trim(w, " ")
		if w != "" {
			args = append(args, "--workspace="+w)
		}
	}
	opts := strings.Trim(c.Arguments, " ")
	if len(opts) > 0 {
		args = append(args, "--")
		args = append(args, c.Arguments)
	}
	return args
}

type packageJSON struct {
	Scripts    map[string]string `json:"scripts"`
	Workspaces *[]string         `json:"workspaces,omitempty"`
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
	cmd.Arguments = strings.Join(cp, " ")
	return cmd
}

func prompt() Command {
	p, err := os.ReadFile("package.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			color.Red("package.json not found")
		} else {
			color.Red("error reading package.json")
		}
		os.Exit(1)
	}
	var pkg packageJSON
	err = json.Unmarshal(p, &pkg)
	if err != nil {
		color.Red("error parsing package.json")
		os.Exit(1)
	}
	scripts := []string{}
	for k := range pkg.Scripts {
		scripts = append(scripts, k)
	}
	sort.Strings(scripts)
	qs := []*survey.Question{
		{
			Name: "script",
			Prompt: &survey.Select{
				Message: "Script:",
				Options: scripts,
			},
			Validate: survey.Required,
		},
		{
			Name: "arguments",
			Prompt: &survey.Input{
				Message: "Arguments:",
			},
		},
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

	// if pkg.Workspaces != nil && len(*pkg.Workspaces) > 0 {
	// 	qs = append(qs, &survey.Question{
	// 		Name: "workspaces",
	// 		Prompt: &survey.MultiSelect{
	// 			Message: "Workspaces:",
	// 			Options: *pkg.Workspaces,
	// 		},
	// 	})
	// }
	var a Command
	err = survey.Ask(qs, &a)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
	return a
}
