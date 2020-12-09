package dbgpxy

import (
	"fmt"
	"strings"
)

// CommandArgs represent command and arguments sent by an IDE
type CommandArgs interface {
	GetCommand() string
	GetArguments() map[string]string
}

func newCommandArgs(command string, arguments map[string]string) CommandArgs {
	return &commandArgsImpl{
		command:   command,
		arguments: arguments,
	}
}

type commandArgsImpl struct {
	command   string
	arguments map[string]string
}

func (c *commandArgsImpl) GetCommand() string {
	return c.command
}

func (c *commandArgsImpl) GetArguments() map[string]string {
	return c.arguments
}

func getCommandArgs(line string) (CommandArgs, error) {

	if line[len(line)-1:] != "\000" {
		return nil, fmt.Errorf("command should terminated by null")
	}

	cmdArgs := strings.SplitN(strings.TrimSuffix(line, "\000"), " ", 2)
	if len(cmdArgs) < 2 {
		return nil, fmt.Errorf("command incomplete")
	}

	command := cmdArgs[0]
	arguments := make(map[string]string)

	parts := strings.Split(cmdArgs[1], " ")
	total := len(parts)

	for i := 0; i < total; i++ {
		part := parts[i]
		if part[0] != '-' {
			continue
		}
		if i+1 >= total {
			break
		}
		i++
		arguments[part[1:]] = parts[i]
	}

	return newCommandArgs(command, arguments), nil
}
