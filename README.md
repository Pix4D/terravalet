# Terravalet

A tool to help with some [Terraform](https://www.terraform.io/) operations.

For the time being it generates state migration scripts that work also for Terraform workspaces.

The idea of migrations comes from [tfmigrate](https://github.com/minamijoyo/tfmigrate). Then this blog [post](https://medium.com/@lynnlin827/moving-terraform-resources-states-from-one-remote-state-to-another-c76f8b76a996)  made me realize that `terraform state mv` had a bug and how to workaround it.

**DISCLAIMER Manipulating Terraform state is inherently dangerous. It is your responsibility to be careful and ensure you UNDERSTAND what you are doing**.

## Status

This is BETA code, although we already use it in production.

The project follows [semantic versioning](https://semver.org/). In particular, we are currently at major version 0: anything MAY change at any time. The public API SHOULD NOT be considered stable.

## Overall approach and migration scripts

The overall approach is for Terravalet to generate migration scripts, not to perform any changes directly. This for two reasons:

1. Safety. The operator can review the generated migration scripts for correctness.
2. Gitops-style. The migration scripts are meant to be stored in git in the same branch (and thus same PR) that performs the Terraform changes and can optionally be hooked to an automatic deployment system.

Terravalet takes as input the output of `terraform plan` per each involved root module and generates one UP and one DOWN migration script.

### Remote and local state

At least until Terraform 0.14, `terraform state mv` has a bug: if a remote backend for the state is configured (which will always be the case for prod), it will remove entries from the remote state but it will not add entries to it. It will fail silently and leave an empty backup file, so you loose your state.

For this reason Terravalet operates on local state and leaves to the operator to perform `terraform state pull` and `terraform state push`.

### Terraform workspaces

Be careful when using Terraform workspaces, since they are invisible and persistent global state :-(. Remember to always explicitly run `terraform workspace select` before anything else.

## Usage

There are two modes of operation:
- [Rename resources](#rename-resources-within-the-same-state) within the same state, with optional fuzzy match.
- [Move resources](#-move-resources-from-one-state-to-another) from one state to another.

they will be explained in the following sections.

You can also look at the tests and in particular at the files below testdata/ for a rough idea.

## Rename resources within the same state

Only one Terraform root module (and thus only one state) is involved. This actually covers two different use cases:

1. Renaming resources within the same root module.
2. Moving resources to/from a non-root Terraform module (this will actually _rename_ the resources, since they will get or loose the `module.` prefix).

### Collect information and remote state

```
$ cd $ROOT_MODULE_DIR
$ terraform workspace select $WS
$ terraform plan -no-color 2>&1 | tee plan.txt

$ terraform state pull > local.tfstate
$ cp local.tfstate local.tfstate.BACK
```

The backup is needed to recover in case of errors. It must be done now.

### Generate migration scripts: exact match, success

Take as input the Terraform plan `plan.txt` (explicit) and the local state `local.tfstate` (implicit) and generate UP and DOWN migration scripts:

```
$ terravalet rename \
    -plan plan.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
```

### Generate migration scripts: exact match, failure

Depending on _how_ the elements have been renamed in the Terraform configuration, it is possible that the exact match will fail:

```
$ terravalet rename \
    -plan plan.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
match_exact:
unmatched create:
  aws_route53_record.private["foo"]
unmatched destroy:
  aws_route53_record.foo_private
```

In this case, you can attempt fuzzy matching.

### Generate migration scripts: fuzzy match

**WARNING** Fuzzy match can make mistakes. It is up to you to validate that the migration makes sense.

If the exact match failed, it is possible to enable [q-gram distance](https://github.com/dexyk/stringosim) fuzzy matching with the `-fuzzy-match` flag:

```
$ terravalet rename-fuzzy-match \
    -plan plan.txt -up 001_TITLE.up.sh -down 001_TITLE.down.sh
WARNING fuzzy match enabled. Double-check the following matches:
 9 aws_route53_record.foo_private -> aws_route53_record.private["foo"]
```

### Run the migration script

1. Review the contents of `001_TITLE.up.sh`.
2. Run it: `sh ./001_TITLE.up.sh`

### Push the migrated state

1. `terraform state push local.tfstate`. In case of error, DO NOT FORCE the push unless you understand very well what you are doing.

### Recovery in case of error

Push the `local.tfstate.BACK`.

## Move resources from one state to another

Two Terraform root modules (and thus two states) are involved. The names of the resources stay the same, but we move them from the `$SRC_ROOT` root module to the `$DST_ROOT` root module.

### Collect information and remote state

Source root:

```
$ cd $SRC_ROOT
$ terraform workspace select $WS
$ terraform plan -no-color 2>&1 | tee src-plan.txt

$ terraform state pull > local.tfstate
$ cp local.tfstate local.tfstate.BACK
```

Destination root:

```
$ cd $DST_ROOT
$ terraform workspace select $WS
$ terraform plan -no-color 2>&1 | tee dst-plan.txt

$ terraform state pull > local.tfstate
$ cp local.tfstate local.tfstate.BACK
```

The backups are needed to recover in case of errors. They must be done now.

### Generate migration scripts

Take as input the two Terraform plans `src-plan.txt`, `dst-plan.txt`, the two local state files in the corresponding directories and generate UP and DOWN migration scripts.

Assuming the following directory layout, where `repo` is the top-level directory and `src` and `dst` are the two Terraform root modules:

```
repo/
├── src/
├── dst/
```

the generated migration scripts will be easier to understand and portable from one operator to another if you run terravalet from the `repo` directory and use relative paths:

```
$ cd repo
$ terravalet move \
    -src-plan  src/src-plan.txt  -dst-plan  dst/dst-plan.txt \
    -src-state src/local.tfstate -dst-state dst/local.tfstate \
    -up 001_TITLE.up.sh -down 001_TITLE.down.sh
```

### Run the migration script

1. Review the contents of `001_TITLE.up.sh`.
2. Run it: `sh ./001_TITLE.up.sh`

### Push the migrated states

In case of error, DO NOT FORCE the push unless you understand very well what you are doing.

```
$ cd src
$ terraform state push local.tfstate
```

and

```
$ cd dst
$ terraform state push local.tfstate
```

### Recovery in case of error

Push the two backups `src/local.tfstate.BACK` and `dst/local.tfstate.BACK`.

## Install

### Install from binary package

1. Download the archive for your platform from the [releases page](https://github.com/Pix4D/terravalet/releases).
2. Unarchive and copy the `terravalet` executable somewhere in your `$PATH`.

### Install from source

1. Install [Go](https://golang.org/).
2. Install [task](https://taskfile.dev/).
3. Run `task`
   ```
   $ task
   ```
4. Copy the executable `bin/terravalet` to a directory in your `$PATH`.

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
