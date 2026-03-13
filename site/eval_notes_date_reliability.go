package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	evalGitHubOwner = "deepakjois"
	evalGitHubRepo  = "debugjois.dev"
	evalGitHubRef   = "main"

	gitHubClientTimeout = 15 * time.Second
)

type EvalNotesDateReliabilityCmd struct {
	FolderID    string `name:"folder-id" env:"GOOGLE_DRIVE_FOLDER_ID" required:"true" help:"ID of the Google Drive folder to inspect"`
	Creds       string `env:"GOOGLE_DRIVE_CREDENTIALS_FILE" required:"true" help:"Path to the service account key JSON file"`
	GitHubToken string `name:"github-token" required:"true" help:"GitHub token used to query commit history on main"`
	SourceDir   string `default:"content/daily-notes" help:"Local directory containing daily notes"`
	Since       string `help:"Only evaluate notes on or after this YYYY-MM-DD date"`
}

type evalGitHubClient struct {
	client *github.Client
}

type noteEvaluation struct {
	Name               string
	LocalMD5           string
	DriveMD5           string
	DriveModifiedTime  *time.Time
	GitHubCommitTime   *time.Time
	MD5Match           bool
	NewerSide          string
	TimestampDelta     time.Duration
	HasTimestampDelta  bool
	MissingOnDrive     bool
	GitHubLookupFailed bool
	ErrorMessage       string
}

type evaluationSummary struct {
	TotalNotes            int
	ComparedSuccessfully  int
	TimestampCompared     int
	MD5Matches            int
	MD5Mismatches         int
	DriveNewerAll         int
	GitHubNewerAll        int
	SameTimestampAll      int
	DriveNewerMismatches  int
	GitHubNewerMismatches int
	SameTimestampMismatch int
	UnknownNewerMismatch  int
	OverOneHour           int
	OverOneDay            int
	OverOneWeek           int
	OverThirtyDays        int
	TotalAbsDelta         time.Duration
	MaxAbsDelta           time.Duration
	MaxAbsDeltaFile       string
	MissingOnDrive        int
	MissingOnGitHub       int
	HardErrors            int
}

type timestampDeltaRecord struct {
	Name  string
	Delta time.Duration
}

func (cmd *EvalNotesDateReliabilityCmd) Run() error {
	ctx := context.Background()

	entries, err := os.ReadDir(cmd.SourceDir)
	if err != nil {
		return fmt.Errorf("read source directory: %w", err)
	}

	noteEntries, err := selectAllNoteEntries(entries, cmd.Since)
	if err != nil {
		return err
	}
	if len(noteEntries) == 0 {
		return fmt.Errorf("no daily note files found in %s", cmd.SourceDir)
	}

	driveService, err := cmd.newDriveService(ctx)
	if err != nil {
		return err
	}

	driveChildren, err := listDriveChildren(driveService, cmd.FolderID)
	if err != nil {
		return fmt.Errorf("list drive folder contents: %w", err)
	}

	githubClient := newEvalGitHubClient(cmd.GitHubToken)
	summary := evaluationSummary{TotalNotes: len(noteEntries)}
	deltaRecords := make([]timestampDeltaRecord, 0, len(noteEntries))

	fmt.Printf("START  total_notes=%d\n", len(noteEntries))

	for i, entry := range noteEntries {
		fmt.Printf("PROGRESS %d/%d %s\n", i+1, len(noteEntries), entry.Name())
		evaluation := cmd.evaluateNote(ctx, githubClient, driveChildren, entry)
		updateEvaluationSummary(&summary, evaluation)
		if evaluation.HasTimestampDelta {
			deltaRecords = append(deltaRecords, timestampDeltaRecord{Name: evaluation.Name, Delta: evaluation.TimestampDelta})
		}
		printEvaluation(evaluation)
	}

	printEvaluationSummary(summary, deltaRecords)

	return nil
}

func (cmd *EvalNotesDateReliabilityCmd) newDriveService(ctx context.Context) (*drive.Service, error) {
	creds, err := os.ReadFile(cmd.Creds)
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}

	driveService, err := drive.NewService(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, creds))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	return driveService, nil
}

func newEvalGitHubClient(token string) *evalGitHubClient {
	client := github.NewClient(&http.Client{Timeout: gitHubClientTimeout}).WithAuthToken(strings.TrimSpace(token))
	return &evalGitHubClient{client: client}
}

func (cmd *EvalNotesDateReliabilityCmd) evaluateNote(ctx context.Context, githubClient *evalGitHubClient, driveChildren *driveChildren, entry os.DirEntry) noteEvaluation {
	name := entry.Name()
	localPath := filepath.Join(cmd.SourceDir, name)
	evaluation := noteEvaluation{Name: name}

	localMD5, err := calculateMD5(localPath)
	if err != nil {
		evaluation.ErrorMessage = fmt.Sprintf("local md5: %v", err)
		return evaluation
	}
	evaluation.LocalMD5 = localMD5

	driveFile := driveChildren.files[name]
	if driveFile == nil {
		evaluation.MissingOnDrive = true
		return evaluation
	}
	if driveFile.MimeType == driveFolderMimeType {
		evaluation.ErrorMessage = "drive entry is a folder"
		return evaluation
	}

	evaluation.DriveMD5 = driveFile.MD5Checksum
	if driveFile.ModifiedTime != "" {
		modifiedTime, err := time.Parse(time.RFC3339, driveFile.ModifiedTime)
		if err != nil {
			evaluation.ErrorMessage = fmt.Sprintf("parse drive modified time: %v", err)
			return evaluation
		}
		evaluation.DriveModifiedTime = &modifiedTime
	}

	commitTime, err := githubClient.latestCommitTimeForPath(ctx, filepath.ToSlash(filepath.Join("site", cmd.SourceDir, name)))
	if err != nil {
		evaluation.GitHubLookupFailed = true
		evaluation.ErrorMessage = fmt.Sprintf("github commit lookup: %v", err)
		return evaluation
	}
	if commitTime == nil {
		evaluation.GitHubLookupFailed = true
		return evaluation
	}
	evaluation.GitHubCommitTime = commitTime

	evaluation.MD5Match = evaluation.LocalMD5 == evaluation.DriveMD5
	evaluation.NewerSide = determineNewerSide(evaluation.GitHubCommitTime, evaluation.DriveModifiedTime)
	if evaluation.GitHubCommitTime != nil && evaluation.DriveModifiedTime != nil {
		evaluation.TimestampDelta = evaluation.GitHubCommitTime.Sub(*evaluation.DriveModifiedTime)
		evaluation.HasTimestampDelta = true
	}

	return evaluation
}

func (c *evalGitHubClient) latestCommitTimeForPath(ctx context.Context, path string) (*time.Time, error) {
	commits, _, err := c.client.Repositories.ListCommits(ctx, evalGitHubOwner, evalGitHubRepo, &github.CommitsListOptions{
		SHA:  evalGitHubRef,
		Path: path,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, nil
	}

	commit := commits[0]
	if commit == nil || commit.Commit == nil || commit.Commit.Committer == nil || commit.Commit.Committer.Date == nil {
		return nil, fmt.Errorf("missing committer timestamp")
	}

	commitTime := commit.Commit.Committer.GetDate().UTC()
	return &commitTime, nil
}

func determineNewerSide(githubTime, driveTime *time.Time) string {
	if githubTime == nil || driveTime == nil {
		return "unknown"
	}

	githubUTC := githubTime.UTC()
	driveUTC := driveTime.UTC()

	switch {
	case githubUTC.After(driveUTC):
		return "github"
	case driveUTC.After(githubUTC):
		return "drive"
	default:
		return "same"
	}
}

func updateEvaluationSummary(summary *evaluationSummary, evaluation noteEvaluation) {
	if evaluation.MissingOnDrive {
		summary.MissingOnDrive++
		return
	}

	if evaluation.GitHubLookupFailed {
		summary.MissingOnGitHub++
		if evaluation.ErrorMessage != "" {
			summary.HardErrors++
		}
		return
	}

	if evaluation.ErrorMessage != "" {
		summary.HardErrors++
		return
	}

	summary.ComparedSuccessfully++
	if evaluation.HasTimestampDelta {
		absDelta := absDuration(evaluation.TimestampDelta)
		summary.TimestampCompared++
		summary.TotalAbsDelta += absDelta
		switch evaluation.NewerSide {
		case "drive":
			summary.DriveNewerAll++
		case "github":
			summary.GitHubNewerAll++
		case "same":
			summary.SameTimestampAll++
		}
		if absDelta > summary.MaxAbsDelta {
			summary.MaxAbsDelta = absDelta
			summary.MaxAbsDeltaFile = evaluation.Name
		}
		if absDelta > time.Hour {
			summary.OverOneHour++
		}
		if absDelta > 24*time.Hour {
			summary.OverOneDay++
		}
		if absDelta > 7*24*time.Hour {
			summary.OverOneWeek++
		}
		if absDelta > 30*24*time.Hour {
			summary.OverThirtyDays++
		}
	}

	if evaluation.MD5Match {
		summary.MD5Matches++
		return
	}

	summary.MD5Mismatches++
	switch evaluation.NewerSide {
	case "drive":
		summary.DriveNewerMismatches++
	case "github":
		summary.GitHubNewerMismatches++
	case "same":
		summary.SameTimestampMismatch++
	default:
		summary.UnknownNewerMismatch++
	}
}

func printEvaluation(evaluation noteEvaluation) {
	switch {
	case evaluation.ErrorMessage != "":
		fmt.Printf("ERROR  %s | %s\n", evaluation.Name, evaluation.ErrorMessage)
	case evaluation.MissingOnDrive:
		fmt.Printf("MISSING %s | drive=missing local_md5=%s\n", evaluation.Name, evaluation.LocalMD5)
	case evaluation.GitHubLookupFailed:
		fmt.Printf("GITHUB %s | commit=missing drive_modified=%s\n", evaluation.Name, formatTimePointer(evaluation.DriveModifiedTime))
	case evaluation.MD5Match:
		return
	default:
		fmt.Printf("DIFF   %s | newer=%s delta=%s github=%s drive=%s local_md5=%s drive_md5=%s\n", evaluation.Name, evaluation.NewerSide, formatDuration(evaluation.TimestampDelta), formatTimePointer(evaluation.GitHubCommitTime), formatTimePointer(evaluation.DriveModifiedTime), evaluation.LocalMD5, evaluation.DriveMD5)
	}
}

func printEvaluationSummary(summary evaluationSummary, deltaRecords []timestampDeltaRecord) {
	fmt.Println()
	fmt.Println("Summary")
	fmt.Printf("total_notes=%d\n", summary.TotalNotes)
	fmt.Printf("compared_successfully=%d\n", summary.ComparedSuccessfully)
	fmt.Printf("timestamp_compared=%d\n", summary.TimestampCompared)
	fmt.Printf("md5_matches=%d\n", summary.MD5Matches)
	fmt.Printf("md5_mismatches=%d\n", summary.MD5Mismatches)
	fmt.Printf("github_newer_all=%d\n", summary.GitHubNewerAll)
	fmt.Printf("drive_newer_all=%d\n", summary.DriveNewerAll)
	fmt.Printf("same_timestamp_all=%d\n", summary.SameTimestampAll)
	fmt.Printf("drive_newer_mismatches=%d\n", summary.DriveNewerMismatches)
	fmt.Printf("github_newer_mismatches=%d\n", summary.GitHubNewerMismatches)
	fmt.Printf("same_timestamp_mismatches=%d\n", summary.SameTimestampMismatch)
	fmt.Printf("unknown_newer_mismatches=%d\n", summary.UnknownNewerMismatch)
	fmt.Printf("over_1h=%d\n", summary.OverOneHour)
	fmt.Printf("over_1d=%d\n", summary.OverOneDay)
	fmt.Printf("over_1w=%d\n", summary.OverOneWeek)
	fmt.Printf("over_30d=%d\n", summary.OverThirtyDays)
	if summary.TimestampCompared > 0 {
		fmt.Printf("avg_abs_delta=%s\n", formatDuration(summary.TotalAbsDelta/time.Duration(summary.TimestampCompared)))
		fmt.Printf("max_abs_delta=%s (%s)\n", formatDuration(summary.MaxAbsDelta), summary.MaxAbsDeltaFile)
	}
	fmt.Printf("missing_on_drive=%d\n", summary.MissingOnDrive)
	fmt.Printf("missing_on_github=%d\n", summary.MissingOnGitHub)
	fmt.Printf("hard_errors=%d\n", summary.HardErrors)

	printTopTimestampDeltas(deltaRecords)
}

func formatTimePointer(t *time.Time) string {
	if t == nil {
		return "missing"
	}
	return t.UTC().Format(time.RFC3339)
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func printTopTimestampDeltas(deltaRecords []timestampDeltaRecord) {
	if len(deltaRecords) == 0 {
		return
	}

	sort.Slice(deltaRecords, func(i, j int) bool {
		return absDuration(deltaRecords[i].Delta) > absDuration(deltaRecords[j].Delta)
	})

	limit := 10
	if len(deltaRecords) < limit {
		limit = len(deltaRecords)
	}

	fmt.Println("top_abs_deltas:")
	for _, record := range deltaRecords[:limit] {
		fmt.Printf("  %s delta=%s\n", record.Name, formatDuration(record.Delta))
	}
}

func selectAllNoteEntries(entries []os.DirEntry, since string) ([]os.DirEntry, error) {
	var sinceDate string
	if strings.TrimSpace(since) != "" {
		parsedSince, err := time.Parse("2006-01-02", since)
		if err != nil {
			return nil, fmt.Errorf("parse --since: %w", err)
		}
		sinceDate = parsedSince.Format("2006-01-02")
	}

	files := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !dailyNoteFilenamePattern.MatchString(entry.Name()) {
			continue
		}

		if sinceDate != "" && strings.TrimSuffix(entry.Name(), ".md") < sinceDate {
			continue
		}

		files = append(files, entry)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files, nil
}
