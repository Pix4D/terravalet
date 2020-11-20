# Terravalet

A tool to help with certain [Terraform](https://www.terraform.io/) operations.

For the time being it generates state migration scripts that work also for Terraform workspaces. The idea of migrations comes from the excellent [tfmigrate](https://github.com/minamijoyo/tfmigrate).

# Status

This is BETA code, although we already use it in production. The instructions provide also ways to manually recover, but be careful and ensure you UNDERSTAND what you are doing if you use it for something important!

You have been warned.

The API can change in breaking ways until we reach v1.0.0

# Usage

Collect informations for Terravalet:

```
$ cd $ROOT_MODULE_DIR
$ terraform plan -no-color 2>&1 | tee ~/work/plan-01.txt
```

## Current

Generate a script with local state.
Running this script will NOT change the remote source state.

```
$ terravalet ~/work/plan-01.txt ../migrations/001_$TITLE.up.sh
```

## Upcoming

Take as input the Terraform plan `~/work/plan-01.txt` and generate two migration scripts with prefix `../migrations/001_TITLE`:
- `../migrations/001_TITLE.up.sh`
- `../migrations/001_TITLE.down.sh`

```
$ terravalet -scripts=../migrations/001_TITLE -plan=~/work/plan-01.txt
```

NOTE: It us up to the user to ensure that the migration number is correct with respect to what is already present in the migration directory.

# Install

## Install from source

1. Install [Go](https://golang.org/).
2. Install [task](https://taskfile.dev/).
3. Run `task`
   ```
   $ task
   ```
4. Copy the executable `bin/terravalet` to a directory in your `$PATH`. I like to use `$HOME/bin/`.

# License

This code is released under the MIT license, see file [LICENSE](LICENSE).
