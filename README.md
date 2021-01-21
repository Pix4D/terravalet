# Terravalet

A tool to help with some [Terraform](https://www.terraform.io/) operations.

For the time being it generates state migration scripts that work also for Terraform workspaces. The idea of migrations comes from the excellent [tfmigrate](https://github.com/minamijoyo/tfmigrate).

## Status

This is BETA code, although we already use it in production. The instructions provide also ways to manually recover, but be careful and ensure you UNDERSTAND what you are doing if you use it for something important!

You have been warned.

The API can change in breaking ways until we reach v1.0.0

## Usage

There are two modes of operation:
- [Rename resources](#rename-resources-within-the-same-state) within the same state, with optional fuzzy match.
- [Move resources](#-move-resources-from-one-state-to-another) from one state to another.

### Rename resources within the same state

#### Collect information

```
$ cd $ROOT_MODULE_DIR
$ terraform plan -no-color 2>&1 | tee plan-01.txt
```

#### Exact match, success

Take as input the Terraform plan `plan-01.txt` and generate UP and DOWN migration scripts:

```
$ terravalet rename\
    -plan plan-01.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
```

NOTE: It us up to the user to ensure that the migration number is correct with respect to what is already present in the migration directory.

#### Exact match, failure

Depending on _how_ the elements have been renamed in the Terraform configuration, it is possible that the exact match will fail:

```
$ terravalet rename\
    -plan plan-01.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
match_exact:
unmatched create:
  aws_route53_record.private["foo"]
unmatched destroy:
  aws_route53_record.foo_private
```

#### Fuzzy match

**WARNING** Fuzzy match can make mistakes. It is up to you to validate that the migration makes sense

If exact match failed, it is possible to enable [q-gram distance](https://github.com/dexyk/stringosim) fuzzy matching with the `-fuzzy-match` flag:

```
$ terravalet rename-fuzzy-match \
    -plan plan-01.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
WARNING fuzzy match enabled. Double-check the following matches:
 9 aws_route53_record.foo_private -> aws_route53_record.private["foo"]
```

## Move resources from one state to another

Not yet implemented.

## Install

### Install from binary package

1. Download the archive for your platform from the [releases page](https://github.com/Pix4D/terravalet/releases).
2. Unarchive and copy the `terravalet` executable somewhere in your `$PATH`. I like to use `$HOME/bin/`.

### Install from source

1. Install [Go](https://golang.org/).
2. Install [task](https://taskfile.dev/).
3. Run `task`
   ```
   $ task
   ```
4. Copy the executable `bin/terravalet` to a directory in your `$PATH`. I like to use `$HOME/bin/`.

## Making a release

### Setup

1. Install [github-release](https://github.com/github-release/github-release).
2. Install [gopass](https://github.com/gopasspw/gopass) or equivalent.
3. Configure a GitHub token:
   3.1 Go to [Personal Access tokens](https://github.com/settings/tokens)
   3.2 Click on "Generate new token"
   3.3 Select only the `repo` scope
4. Store the token securely with a tool like `gopass`. The name `GITHUB_TOKEN` is expected by `github-release`
   ```
   $ gopass insert gh/terravalet/GITHUB_TOKEN
   ```

### Each time

1. Update [CHANGELOG](CHANGELOG.md)
2. Update this README and/or additional documentation.
3. Commit and push.
4. Begin the release process with
   ```
   $ env RELEASE_TAG=v0.1.0 gopass env gh/terravalet task release
   ```
5. Finish the release process by following the instructions printed by `task` above.
6. To recover from a half-baked release, see the hints in the [Taskfile](Taskfile.yml).

## License

This code is released under the MIT license, see file [LICENSE](LICENSE).
