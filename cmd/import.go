package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vim-volt/go-volt/copyutil"
	"github.com/vim-volt/go-volt/lockjson"
	"github.com/vim-volt/go-volt/pathutil"
	"github.com/vim-volt/go-volt/transaction"
)

type importCmd struct{}

func Import(args []string) int {
	cmd := importCmd{}

	from, reposPath, err := cmd.parseArgs(args)
	if err != nil {
		fmt.Println("[ERROR] Failed to parse args: " + err.Error())
		return 10
	}

	err = cmd.doImport(from, reposPath)
	if err != nil {
		fmt.Println("[ERROR] Failed to import: " + err.Error())
		return 11
	}

	return 0
}

func (*importCmd) parseArgs(args []string) (string, string, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Println(`
Usage
  (1) volt import {repository}
  (2) volt import {from} {repository}

Description
  1st form:
    Import local {repository} to lock.json
  2nd form:
    Import local {from} repository as {repository} to lock.json

Options`)
		fs.PrintDefaults()
		fmt.Println()
	}
	fs.Parse(args)

	fsArgs := fs.Args()
	switch len(fsArgs) {
	case 1:
		reposPath, err := pathutil.NormalizeRepos(fsArgs[0])
		return pathutil.FullReposPathOf(reposPath), reposPath, err
	case 2:
		reposPath, err := pathutil.NormalizeImportedRepos(fsArgs[1])
		return fsArgs[0], reposPath, err
	default:
		fs.Usage()
		return "", "", errors.New("invalid arguments")
	}
}

func (cmd *importCmd) doImport(from, reposPath string) error {
	// Check from and destination (full path of repos path) path
	if _, err := os.Stat(from); os.IsNotExist(err) {
		return errors.New("no such a directory: " + from)
	}

	dst := pathutil.FullReposPathOf(reposPath)
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		return errors.New("the repository already exists: " + reposPath)
	}

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("could not read lock.json: " + err.Error())
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return errors.New("failed to begin transaction: " + err.Error())
	}
	defer transaction.Remove()
	lockJSON.TrxID++

	fmt.Printf("[INFO] Importing '%s' as '%s' ...\n", from, reposPath)

	// Copy directory from to dst
	err = copyutil.CopyDir(from, dst)
	if err != nil {
		return err
	}

	// Find matching profile
	profile, err := lockJSON.Profiles.FindByName(lockJSON.ActiveProfile)
	if err != nil {
		return err
	}

	// Add repos to lockJSON
	reposType, err := cmd.detectReposType(dst)
	lockJSON.Repos = append(lockJSON.Repos, lockjson.Repos{
		Type:  reposType,
		TrxID: lockJSON.TrxID,
		Path:  reposPath,
	})

	// Add repos to profiles[]/repos_path
	if !profile.ReposPath.Contains(reposPath) {
		// Add repos to 'profiles[]/repos_path'
		profile.ReposPath = append(profile.ReposPath, reposPath)
	}

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return errors.New("could not write to lock.json: " + err.Error())
	}

	// Rebuild start dir
	err = (&rebuildCmd{}).doRebuild()
	if err != nil {
		return errors.New("could not rebuild " + pathutil.VimVoltDir() + ": " + err.Error())
	}

	return nil
}

func (*importCmd) detectReposType(fullpath string) (lockjson.ReposType, error) {
	if _, err := os.Stat(fullpath); os.IsNotExist(err) {
		return "", errors.New("no such a directory: " + fullpath)
	}
	if _, err := os.Stat(filepath.Join(fullpath, ".git")); !os.IsNotExist(err) {
		return lockjson.ReposGitType, nil
	}
	return lockjson.ReposStaticType, nil
}