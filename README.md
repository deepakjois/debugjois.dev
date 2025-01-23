Website: https://debugjois.dev

Daily Log: https://debugjois.dev/daily

```
$ ./debugjois.dev --help
Usage: debugjois.dev <command> [flags]

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  build [flags]
    Build the static site

  sync-notes-obsidian --obsidian-vault=STRING [flags]
    Sync daily notes from Obsidian vault

  sync-notes-gdrive --folder-id=STRING --creds=STRING [flags]
    Sync daily notes from Google Drive

  upload [flags]
    Upload files to S3 bucket

  build-newsletter [flags]
    Build weekly newsletter from daily notes

  index [flags]
    Index daily notes

  search <query> [flags]
    Search indexed daily notes
```
