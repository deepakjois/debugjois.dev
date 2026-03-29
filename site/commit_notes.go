package main

import (
	"fmt"
	"time"

	"github.com/bitfield/script"
)

type CommitNotesCmd struct {
	SkipCI bool `help:"Append [skip ci] to the commit message"`
}

func (c *CommitNotesCmd) Run() error {
	status, err := script.Exec("git status -s content/daily-notes/").String()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if status == "" {
		fmt.Println("No changes to commit.")
		return nil
	}

	if _, err := script.Exec("git add content/daily-notes/").Stdout(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	msg := fmt.Sprintf("Obsidian Gdrive Sync %s", time.Now().Format("2006-01-02 15:04:05"))
	if c.SkipCI {
		msg += " [skip ci]"
	}
	if _, err := script.Exec(fmt.Sprintf("git commit -m '%s'", msg)).Stdout(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	fmt.Println("Changes committed successfully.")
	return nil
}
