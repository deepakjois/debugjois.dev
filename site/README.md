Daily Log: https://debugjois.dev/daily

After checking out repo do the following

```
$ cd site

$ go build -o debugjois-site .

$ ./debugjois-site --help
Usage: debugjois-site <command> [flags]

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  build [flags]
    Build the static site

  sync-notes-obsidian --obsidian-vault=STRING [flags]
    Sync daily notes from Obsidian vault

  upload [flags]
    Upload files to S3 bucket

  build-newsletter [flags]
    Build weekly newsletter from daily notes

```
