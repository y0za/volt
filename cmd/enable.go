package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/vim-volt/volt/logger"
	"github.com/vim-volt/volt/pathutil"
)

type enableFlagsType struct {
	helped bool
}

var enableFlags enableFlagsType

func init() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Print(`
Usage
  volt enable {repository} [{repository2} ...]

Quick example
  $ volt enable tyru/caw.vim # will enable tyru/caw.vim plugin in current profile

Description
  This is shortcut of:
  volt profile add {current profile} {repository} [{repository2} ...]` + "\n\n")
		//fmt.Println("Options")
		//fs.PrintDefaults()
		fmt.Println()
		enableFlags.helped = true
	}

	cmdFlagSet["enable"] = fs
}

type enableCmd struct{}

func Enable(args []string) int {
	cmd := enableCmd{}

	reposPathList, err := cmd.parseArgs(args)
	if err == ErrShowedHelp {
		return 0
	}
	if err != nil {
		logger.Error("Failed to parse args: " + err.Error())
		return 10
	}

	profCmd := profileCmd{}
	err = profCmd.doAdd(append(
		[]string{"-current"},
		reposPathList...,
	))
	if err != nil {
		logger.Error(err.Error())
		return 11
	}

	return 0
}

func (*enableCmd) parseArgs(args []string) ([]string, error) {
	fs := cmdFlagSet["enable"]
	fs.Parse(args)
	if enableFlags.helped {
		return nil, ErrShowedHelp
	}

	if len(fs.Args()) == 0 {
		fs.Usage()
		return nil, errors.New("repository was not given")
	}

	// Normalize repos path
	var reposPathList []string
	for _, arg := range fs.Args() {
		reposPath, err := pathutil.NormalizeRepos(arg)
		if err != nil {
			return nil, err
		}
		reposPathList = append(reposPathList, reposPath)
	}

	return reposPathList, nil
}
