package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const driveFolderMimeType = "application/vnd.google-apps.folder"

var dailyNoteFilenamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)

type SyncNotesToGdriveCmd struct {
	FolderID  string `name:"folder-id" env:"GOOGLE_DRIVE_FOLDER_ID" required:"true" help:"ID of the Google Drive folder to sync into"`
	Creds     string `env:"GOOGLE_DRIVE_CREDENTIALS_FILE" required:"true" help:"Path to the service account key JSON file"`
	SourceDir string `default:"content/daily-notes" help:"Local directory to sync to Google Drive"`
	All       bool   `default:"false" help:"Sync all note files instead of only the 30 most recent notes"`
	DryRun    bool   `help:"Show planned actions without uploading or creating folders"`
	Debug     bool   `help:"Print remote folder inspection details while syncing"`
}

type driveFile struct {
	ID           string
	Name         string
	MimeType     string
	MD5Checksum  string
	ModifiedTime string
}

func (cmd *SyncNotesToGdriveCmd) Run() error {
	ctx := context.Background()

	if _, err := os.Stat(cmd.SourceDir); err != nil {
		return fmt.Errorf("stat source directory: %w", err)
	}

	creds, err := os.ReadFile(cmd.Creds)
	if err != nil {
		return fmt.Errorf("read credentials file: %w", err)
	}

	drv, err := drive.NewService(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, creds))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}

	if err := cmd.syncNotes(ctx, drv); err != nil {
		return fmt.Errorf("failed to sync notes: %w", err)
	}

	return nil
}

func (cmd *SyncNotesToGdriveCmd) syncNotes(ctx context.Context, drv *drive.Service) error {
	entries, err := os.ReadDir(cmd.SourceDir)
	if err != nil {
		return fmt.Errorf("read local directory %s: %w", cmd.SourceDir, err)
	}

	entries = cmd.selectEntries(entries)

	remoteChildren, err := listDriveChildren(drv, cmd.FolderID)
	if err != nil {
		return fmt.Errorf("list remote folder %s: %w", cmd.FolderID, err)
	}

	if cmd.Debug {
		cmd.printRemoteChildren(".", cmd.FolderID, remoteChildren)
	}

	for _, entry := range entries {
		name := entry.Name()
		localPath := filepath.Join(cmd.SourceDir, name)
		remoteFile := remoteChildren.files[name]
		if err := cmd.syncFile(ctx, drv, localPath, name, cmd.FolderID, remoteFile); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *SyncNotesToGdriveCmd) syncFile(ctx context.Context, drv *drive.Service, localPath, relativePath, remoteFolderID string, remoteFile *driveFile) error {
	localMD5, err := calculateMD5(localPath)
	if err != nil {
		return fmt.Errorf("calculate md5 for %s: %w", relativePath, err)
	}

	if remoteFile == nil {
		fmt.Printf("CREATE %s\n", relativePath)
		if cmd.DryRun {
			return nil
		}
		return createDriveFile(ctx, drv, remoteFolderID, localPath)
	}

	if remoteFile.MimeType == driveFolderMimeType {
		return fmt.Errorf("remote path %s exists and is a folder", relativePath)
	}

	if localMD5 == remoteFile.MD5Checksum {
		if cmd.DryRun {
			fmt.Printf("SKIP   %s\n", relativePath)
		}
		return nil
	}

	fmt.Printf("UPDATE %s\n", relativePath)
	if cmd.DryRun {
		return nil
	}

	return updateDriveFile(ctx, drv, remoteFile.ID, localPath)
}

type driveChildren struct {
	folders map[string]*driveFile
	files   map[string]*driveFile
}

func listDriveChildren(drv *drive.Service, folderID string) (*driveChildren, error) {
	children := &driveChildren{
		folders: make(map[string]*driveFile),
		files:   make(map[string]*driveFile),
	}

	var pageToken string
	for {
		query := drv.Files.List().
			Q(fmt.Sprintf("'%s' in parents and trashed = false", folderID)).
			Fields("nextPageToken, files(id, name, mimeType, md5Checksum, modifiedTime)")

		if pageToken != "" {
			query = query.PageToken(pageToken)
		}

		result, err := query.Do()
		if err != nil {
			return nil, err
		}

		for _, file := range result.Files {
			entry := &driveFile{
				ID:           file.Id,
				Name:         file.Name,
				MimeType:     file.MimeType,
				MD5Checksum:  file.Md5Checksum,
				ModifiedTime: file.ModifiedTime,
			}

			if file.MimeType == driveFolderMimeType {
				if _, exists := children.folders[file.Name]; !exists {
					children.folders[file.Name] = entry
				}
				continue
			}

			if _, exists := children.files[file.Name]; !exists {
				children.files[file.Name] = entry
			}
		}

		if result.NextPageToken == "" {
			break
		}

		pageToken = result.NextPageToken
	}

	return children, nil
}

func (cmd *SyncNotesToGdriveCmd) printRemoteChildren(relativePath, remoteFolderID string, children *driveChildren) {
	pathLabel := relativePath
	if pathLabel == "." || pathLabel == "" {
		pathLabel = "/"
	}

	folderNames := sortedDriveNames(children.folders)
	fileNames := sortedDriveNames(children.files)

	fmt.Printf("DEBUG  remote %s (id=%s): %d folders, %d files\n", pathLabel, remoteFolderID, len(folderNames), len(fileNames))
	for _, name := range folderNames {
		fmt.Printf("DEBUG    folder %s\n", joinRelativePath(pathLabel, name))
	}
	for _, name := range fileNames {
		fmt.Printf("DEBUG    file   %s\n", joinRelativePath(pathLabel, name))
	}
}

func (cmd *SyncNotesToGdriveCmd) selectEntries(entries []os.DirEntry) []os.DirEntry {
	files := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() && dailyNoteFilenamePattern.MatchString(entry.Name()) {
			files = append(files, entry)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})

	if !cmd.All && len(files) > 30 {
		files = files[:30]
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files
}

func sortedDriveNames(entries map[string]*driveFile) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func joinRelativePath(base, name string) string {
	if base == "." || base == "" || base == "/" {
		return filepath.ToSlash(name)
	}
	return filepath.ToSlash(filepath.Join(base, name))
}

func createDriveFile(ctx context.Context, drv *drive.Service, parentID, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	driveFile := &drive.File{
		Name:    filepath.Base(localPath),
		Parents: []string{parentID},
	}

	_, err = drv.Files.Create(driveFile).
		Media(file, googleapi.ContentType(googleDriveContentType(localPath))).
		Fields("id").
		Context(ctx).
		Do()
	return err
}

func updateDriveFile(ctx context.Context, drv *drive.Service, fileID, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = drv.Files.Update(fileID, nil).
		Media(file, googleapi.ContentType(googleDriveContentType(localPath))).
		Fields("id").
		Context(ctx).
		Do()
	return err
}

func googleDriveContentType(path string) string {
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

func calculateMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
