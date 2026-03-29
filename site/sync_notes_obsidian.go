package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveFolderMimeType = "application/vnd.google-apps.folder"

type SyncNotesObsidianCmd struct {
	SharedDriveName string `env:"OBSIDIAN_SHARED_DRIVE" default:"obsidian" help:"Name of the Google Drive shared drive"`
	VaultFolder     string `env:"OBSIDIAN_VAULT_FOLDER" default:"PersonalKnowledgeWiki" help:"Name of the vault folder within the shared drive"`
}

func (sn *SyncNotesObsidianCmd) Run() error {
	ctx := context.Background()

	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err != nil {
		return fmt.Errorf("google credentials not available; configure Application Default Credentials and retry: %w", err)
	}

	srv, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}

	driveList, err := srv.Drives.List().Q(fmt.Sprintf(`name = "%s"`, sn.SharedDriveName)).Do()
	if err != nil {
		return fmt.Errorf("list shared drives: %w", err)
	}
	if len(driveList.Drives) == 0 {
		return fmt.Errorf("shared drive %q not found", sn.SharedDriveName)
	}
	driveID := driveList.Drives[0].Id

	vaultFolderID, err := findDriveFolder(srv, driveID, driveID, sn.VaultFolder)
	if err != nil {
		return fmt.Errorf("find vault folder: %w", err)
	}

	dailyFolderID, err := findDriveFolder(srv, driveID, vaultFolderID, "daily")
	if err != nil {
		return fmt.Errorf("find daily folder: %w", err)
	}

	if err := syncDriveFolder(srv, driveID, dailyFolderID, "content/daily-notes"); err != nil {
		return fmt.Errorf("sync from Drive: %w", err)
	}

	return nil
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

type driveFileInfo struct {
	ID          string
	Name        string
	MimeType    string
	MD5Checksum string
}

func listDriveFiles(srv *drive.Service, driveID, folderID string) (map[string]*driveFileInfo, error) {
	files := make(map[string]*driveFileInfo)
	var pageToken string
	for {
		query := srv.Files.List().
			Q(fmt.Sprintf(`"%s" in parents and trashed = false`, folderID)).
			Corpora("drive").
			DriveId(driveID).
			IncludeItemsFromAllDrives(true).
			SupportsAllDrives(true).
			Fields("nextPageToken, files(id, name, mimeType, md5Checksum)")

		if pageToken != "" {
			query = query.PageToken(pageToken)
		}

		result, err := query.Do()
		if err != nil {
			return nil, fmt.Errorf("list Drive folder %s: %w", folderID, err)
		}

		for _, f := range result.Files {
			files[f.Name] = &driveFileInfo{
				ID:          f.Id,
				Name:        f.Name,
				MimeType:    f.MimeType,
				MD5Checksum: f.Md5Checksum,
			}
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}
	return files, nil
}

func syncDriveFolder(srv *drive.Service, driveID, driveFolderID, localDir string) error {
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("create local directory %s: %w", localDir, err)
	}

	remoteFiles, err := listDriveFiles(srv, driveID, driveFolderID)
	if err != nil {
		return err
	}

	remoteFolders := make(map[string]*driveFileInfo)
	remoteRegular := make(map[string]*driveFileInfo)
	for name, f := range remoteFiles {
		if f.MimeType == driveFolderMimeType {
			remoteFolders[name] = f
		} else {
			remoteRegular[name] = f
		}
	}

	// Download new or changed files
	for name, rf := range remoteRegular {
		localPath := filepath.Join(localDir, name)
		if needsDownload(localPath, rf.MD5Checksum) {
			fmt.Printf("Downloading %s\n", filepath.Join(localDir, name))
			if err := downloadDriveFile(srv, rf.ID, localPath); err != nil {
				return fmt.Errorf("download %s: %w", name, err)
			}
		}
	}

	// Delete local files/dirs not present on Drive
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("read local dir %s: %w", localDir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if _, exists := remoteFolders[name]; !exists {
				fmt.Printf("Removing directory %s\n", filepath.Join(localDir, name))
				if err := os.RemoveAll(filepath.Join(localDir, name)); err != nil {
					return fmt.Errorf("remove directory %s: %w", name, err)
				}
			}
		} else {
			if _, exists := remoteRegular[name]; !exists {
				fmt.Printf("Removing %s\n", filepath.Join(localDir, name))
				if err := os.Remove(filepath.Join(localDir, name)); err != nil {
					return fmt.Errorf("remove file %s: %w", name, err)
				}
			}
		}
	}

	// Recurse into subfolders
	for name, folder := range remoteFolders {
		if err := syncDriveFolder(srv, driveID, folder.ID, filepath.Join(localDir, name)); err != nil {
			return fmt.Errorf("sync subfolder %s: %w", name, err)
		}
	}

	return nil
}

func needsDownload(localPath, remoteMD5 string) bool {
	localMD5, err := calculateMD5(localPath)
	if err != nil {
		return true
	}
	return localMD5 != remoteMD5
}

func downloadDriveFile(srv *drive.Service, fileID, localPath string) error {
	resp, err := srv.Files.Get(fileID).SupportsAllDrives(true).Download()
	if err != nil {
		return fmt.Errorf("download from Drive: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
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
