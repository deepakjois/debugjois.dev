package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	driveFolderMimeType = "application/vnd.google-apps.folder"
	sharedDriveName     = "obsidian"
	vaultFolderName     = "PersonalKnowledgeWiki"
	dailyFolderName     = "daily"
)

var (
	driveOnce     sync.Once
	driveSrv      *drive.Service
	driveID       string
	dailyFolderID string
	driveInitErr  error
)

func getDriveClient(ctx context.Context) (*drive.Service, string, string, error) {
	driveOnce.Do(func() {
		driveSrv, driveID, dailyFolderID, driveInitErr = initDriveService(ctx)
	})
	return driveSrv, driveID, dailyFolderID, driveInitErr
}

func initDriveService(ctx context.Context) (*drive.Service, string, string, error) {
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err != nil {
		return nil, "", "", fmt.Errorf("google credentials not available: %w", err)
	}

	srv, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, "", "", fmt.Errorf("create Drive service: %w", err)
	}

	driveList, err := srv.Drives.List().Q(fmt.Sprintf(`name = "%s"`, sharedDriveName)).Do()
	if err != nil {
		return nil, "", "", fmt.Errorf("list shared drives: %w", err)
	}
	if len(driveList.Drives) == 0 {
		return nil, "", "", fmt.Errorf("shared drive %q not found", sharedDriveName)
	}
	id := driveList.Drives[0].Id

	vaultFolderID, err := findDriveFolder(srv, id, id, vaultFolderName)
	if err != nil {
		return nil, "", "", fmt.Errorf("find vault folder: %w", err)
	}

	dailyID, err := findDriveFolder(srv, id, vaultFolderID, dailyFolderName)
	if err != nil {
		return nil, "", "", fmt.Errorf("find daily folder: %w", err)
	}

	return srv, id, dailyID, nil
}

func findDriveFolder(srv *drive.Service, driveID, parentID, name string) (string, error) {
	q := fmt.Sprintf(`"%s" in parents and name = "%s" and mimeType = "%s" and trashed = false`, parentID, name, driveFolderMimeType)
	result, err := srv.Files.List().
		Q(q).
		Corpora("drive").
		DriveId(driveID).
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Fields("files(id)").
		Do()
	if err != nil {
		return "", fmt.Errorf("query Drive for folder %q: %w", name, err)
	}
	if len(result.Files) == 0 {
		return "", fmt.Errorf("folder %q not found under parent %s", name, parentID)
	}
	return result.Files[0].Id, nil
}

func loadDailyNoteContentFromDrive(ctx context.Context, date string) (string, error) {
	srv, did, folderID, err := getDriveClient(ctx)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s.md", date)
	q := fmt.Sprintf(`"%s" in parents and name = "%s" and trashed = false`, folderID, filename)
	result, err := srv.Files.List().
		Q(q).
		Corpora("drive").
		DriveId(did).
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Fields("files(id)").
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("search Drive for %s: %w", filename, err)
	}

	if len(result.Files) == 0 {
		return fmt.Sprintf("### %s\n", date), nil
	}

	resp, err := srv.Files.Get(result.Files[0].Id).
		SupportsAllDrives(true).
		Context(ctx).
		Download()
	if err != nil {
		return "", fmt.Errorf("download %s from Drive: %w", filename, err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s content: %w", filename, err)
	}

	return string(content), nil
}

func saveDailyNoteContentToDrive(ctx context.Context, title, contents, _ string) error {
	srv, did, folderID, err := getDriveClient(ctx)
	if err != nil {
		return err
	}

	q := fmt.Sprintf(`"%s" in parents and name = "%s" and trashed = false`, folderID, title)
	result, err := srv.Files.List().
		Q(q).
		Corpora("drive").
		DriveId(did).
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Fields("files(id)").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("search Drive for %s: %w", title, err)
	}

	reader := bytes.NewReader([]byte(contents))

	if len(result.Files) > 0 {
		_, err = srv.Files.Update(result.Files[0].Id, &drive.File{}).
			SupportsAllDrives(true).
			Media(reader).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("update %s on Drive: %w", title, err)
		}
		return nil
	}

	_, err = srv.Files.Create(&drive.File{
		Name:    title,
		Parents: []string{folderID},
	}).
		SupportsAllDrives(true).
		Media(reader).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("create %s on Drive: %w", title, err)
	}

	return nil
}
