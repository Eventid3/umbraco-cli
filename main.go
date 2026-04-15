package main

import (
	_ "embed"

	"github.com/Eventid3/umbraco-cli/cmd"
)

//go:embed skill/SKILL.md
var skillContent string

func main() {
	cmd.SetSkillContent(skillContent)
	cmd.Execute()
}
