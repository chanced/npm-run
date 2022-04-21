package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
)

func main() {
	args := parse_args()
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

func parse_args() []string {
	if len(os.Args) < 2 {
		fmt.Printf("usage:\nrun %s %s\n", color.CyanString("<script>"), color.MagentaString("<arguments>"))
		p, err := os.ReadFile("package.json")
		var pkg packageJSON
		if err == nil {
			err = json.Unmarshal(p, &pkg)
			if err == nil {
				fmt.Println("\navailable scripts:")
				for k := range pkg.Scripts {
					fmt.Printf("\t%s\n", k)
				}
			}
		}

		os.Exit(0)
	}
	if len(os.Args) > 2 {
		return append([]string{"run", os.Args[1], "--"}, os.Args[2:]...)
	}
	return []string{"run", os.Args[1]}
}
