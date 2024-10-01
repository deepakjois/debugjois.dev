package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type SyncNotesGdriveCmd struct {
	FolderId string `required:"true" help:"ID of the Google Drive folder containing the notes"`
	Creds    string `required:"true" help:"Path to the service account key JSON file"`
}

func (cmd *SyncNotesGdriveCmd) Run() error {
	ctx := context.Background()

	drv, err := drive.NewService(ctx, option.WithCredentialsFile(cmd.Creds))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}

	if err := cmd.syncFolder(ctx, drv, cmd.FolderId, "content/daily-notes/"); err != nil {
		return fmt.Errorf("failed to sync folder: %w", err)
	}

	return nil
}

func (cmd *SyncNotesGdriveCmd) syncFolder(ctx context.Context, drv *drive.Service, folderID, localPath string) error {
	var pageToken string
	for {
		query := drv.Files.List().
			Q(fmt.Sprintf("'%s' in parents and trashed = false", folderID)).
			Fields("nextPageToken, files(id, name, mimeType, md5Checksum)")

		if pageToken != "" {
			query = query.PageToken(pageToken)
		}

		result, err := query.Do()
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		for _, f := range result.Files {
			localPath := filepath.Join(localPath, f.Name)

			if f.MimeType == "application/vnd.google-apps.folder" {
				if err := os.MkdirAll(localPath, 0755); err != nil {
					return fmt.Errorf("failed to create local folder: %w", err)
				}
				if err := cmd.syncFolder(ctx, drv, f.Id, localPath); err != nil {
					return err
				}
			} else {
				if needsSync(localPath, f.Md5Checksum) {
					if err := downloadFile(drv, f.Id, localPath); err != nil {
						return fmt.Errorf("failed to download file %s: %w", f.Name, err)
					}
					fmt.Println(strings.TrimPrefix(localPath, "content/daily-notes/"))
				}
			}
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return nil
}

func needsSync(path, remoteMD5 string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true
	}

	localMD5, err := calculateMD5(path)
	if err != nil {
		return true
	}

	return localMD5 != remoteMD5
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

func downloadFile(srv *drive.Service, fileID, path string) error {
	resp, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
