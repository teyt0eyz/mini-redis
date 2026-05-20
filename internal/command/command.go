package command

import "strings"

type Command struct {
	Name string
	Args []string
}

func Parse(raw string) Command {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return Command{}
	}
	return Command{
		Name: strings.ToUpper(parts[0]),
		Args: parts[1:],
	}
}
