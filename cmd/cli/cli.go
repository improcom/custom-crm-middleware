package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

type run func(...string) int

type command struct {
	name string
	args string
	info string
	run
	subs map[string]command
}

var commands = make(map[string]command)

func Register(cmd command) {
	commands[cmd.name] = cmd
}

func Command(name, args, info string, run run) command {
	return command{name: name, args: args, info: info, run: run}
}

func Group(name, info string, cmds ...command) command {
	subs := make(map[string]command)
	for _, cmd := range cmds {
		subs[cmd.name] = cmd
	}
	return command{name: name, info: info, subs: subs}
}

func Exec(args ...string) int {
	ctx := ""
	rest := args
	if len(args) > 0 {
		ctx = filepath.Base(args[0])
		rest = args[1:]
	}
	return exec(commands, ctx, rest...)
}

func exec(cmds map[string]command, path string, args ...string) int {
	if len(args) > 0 {
		if cmd, ok := cmds[args[0]]; ok {
			if cmd.subs != nil {
				return exec(cmd.subs, path+" "+cmd.name, args[1:]...)
			}
			if cmd.run != nil {
				if cmd.args != "" && len(args[1:]) == 0 {
					fmt.Fprintf(os.Stderr, "Missing arguments\nUsage: %s %s\n", path+" "+cmd.name, cmd.args)
					return 1
				}
				return cmd.run(args[1:]...)
			}
		}
	}

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Usage:", path, "<command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range cmds {
		hint := cmd.args
		if hint == "" && cmd.subs != nil {
			hint = "<command>"
		}
		fmt.Fprintln(w, "\t", cmd.name, hint, "\t\t", cmd.info)
	}
	fmt.Fprintln(w)
	w.Flush()
	return 1
}
