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
	args := append([]string{"run"}, loadArgs()...)
	cmd := exec.Command("npm", args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func loadArgs() []string {
	args := os.Args
	if len(args) < 2 {
		args = promptArgs()
	} else {
		args = args[1:]
	}
	if len(args) > 1 {
		return append([]string{args[0], "--"}, args[1:]...)
	}
	return []string{args[0]}
}

func promptArgs() []string {
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
	a := struct {
		Script    string `survey:"script"`
		Arguments string `survey:"arguments"`
	}{}
	err = survey.Ask(qs, &a)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
	args := strings.Split(strings.TrimSpace(a.Arguments), " ")
	return append([]string{a.Script}, args...)
}
