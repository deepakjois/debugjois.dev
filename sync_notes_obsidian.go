package main

import (
	"fmt"
	"time"

	"github.com/bitfield/script"
)

type SyncNotesObsidianCmd struct {
	ObsidianVault string `env:"OBSIDIAN_VAULT" required:"true" help:"Path to Obsidian vault containing the notes"`
	NoGit         bool   `help:"Skip git operations and only run rsync"`
}

func (sn *SyncNotesObsidianCmd) Run() error {
	// Check if ObsidianVault is set
	if sn.ObsidianVault == "" {
		return fmt.Errorf("vault is not set")
	}

	if !sn.NoGit {
		// Check if git repo is clean
		if status, err := script.Exec("git status -s").String(); err != nil {
			return fmt.Errorf("failed to check git status: %w", err)
		} else if status != "" {
			return fmt.Errorf("git repository is not clean. Please commit or stash changes")
		}

		// Update repo
		if _, err := script.Exec("git pull").String(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %w", err)
		}
	}

	// rsync contents
	rsyncCmd := fmt.Sprintf("rsync -az --ignore-times --checksum --delete --out-format='%%n' '%s/daily/' content/daily-notes/", sn.ObsidianVault)
	if _, err := script.Exec(rsyncCmd).Stdout(); err != nil {
		return fmt.Errorf("failed to execute rsync command: %w", err)
	}

	if !sn.NoGit {
		// Check if there are changes
		status, err := script.Exec("git status -s").String()
		if err != nil {
			return fmt.Errorf("failed to check git status after rsync: %w", err)
		}
		if status == "" {
			fmt.Println("No changes to commit.")
			return nil
		}

		// Commit changes with date and time
		currentDatetime := time.Now().Format("2006-01-02 15:04:05")
		if _, err := script.Exec("git add content/daily-notes/").Stdout(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
		msg := fmt.Sprintf("Obsidian Sync %s", currentDatetime)
		if _, err := script.Exec(fmt.Sprintf("git commit -m '%s'", msg)).Stdout(); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
		fmt.Println("Changes committed successfully.")
		if _, err := script.Exec("git push").Stdout(); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}
