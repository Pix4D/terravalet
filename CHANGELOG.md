# Terravalet Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Fixes
### Breaking changes
### Changes
### New

## [v0.6.2] - (Unreleased)

### Changes

- Command-line parsing: replace flaggy with [go-arg](https://github.com/alexflint/go-arg).

### Fixes

- Fixes a test flake due to the use of unsorted set (i.e {abcde} -> {abdcde} or {abdecde} -> {abdcde})

## [v0.6.1] - (2021-08-24)

### Changes

- Bump go to latest v1.17

### Fixes

- Fix breaking too early the loop giving a not fully compiled script.

## [v0.6.0] - (2021-08-11)

### New

- Subcommand `import` (new functionality):
  ```
  import - Import resources generated out-of-band of Terraform

  Flags:
      -res-defs   Path to resources definitions
      -src-plan   Path to the SRC terraform plan in json format
      -up         Path to the resources import script to generate (import.up.sh).
      -down       Path to the resources remove script to generate (import.down.sh).
  ```

## [v0.5.0] - (2021-07-23)

### Fixes

- Improved Fuzzy matching selection algorithm to iteratively consume the best matching create/destroy combination.

## [v0.4.0] - (2021-01-25)

### Fixes

### Breaking changes

- Due to the introduction of subcommands, the CLI API has changed; now it must be invoked by specifying a subcommand. See section New for details.

### Changes

### New

- Introduction of subcommands.
- Subcommand `rename` (existing functionality):
  ```
  rename - Rename resources in the same tf root environment

    Flags:
         -plan          Path to the terraform plan.
         -fuzzy-match   Enable q-gram distance fuzzy matching.
         -up            Path to the up migration script to generate
         -down          Path to the down migration script to generate
  ```
- Subcommand `move` (new functionality):
  ```
  move - Move resources from one root environment to another

    Flags:
         -src-plan    Path to the SRC terraform plan
         -dst-plan    Path to the DST terraform plan
         -src-state   Path to the SRC local state to modify
         -dst-state   Path to the DST local state to modify
         -up          Path to the up migration script to generate
         -down        Path to the down migration script to generate
  ```

## [v0.3.0] - (2020-12-11)

### New

- Fuzzy matching. See README for more information.

## [v0.2.0] - (2020-11-27)

### Breaking changes

- The CLI API has changed; now it must be invoked as
  ```
  $ terravalet -plan=PLAN -up=UP.sh -down=DOWN.sh
  ```

### Changes

- Migration script: do not print any more the count `>>> 1/N`, because each time N changed, this was causing N spurious diffs, hiding the real elements that changed. The `terravalet_output_format` is now 2.
- Migration script: do not take a lock; it is useless as long as the operations are strictly on a local state file. This speeds up the runtime.

### New

- Generate also the DOWN migration script.
- Extensive tests.

## [v0.1.0] - (2020-11-20)

### New

- For the time being, this repository is kept private. Will be open-sourced later.
- First release, with scripted release support.
- Basic functionalities, generate the UP migration script only.
- CLI API:
  ```
  $ terravalet -plan=PLAN > UP.sh
  ```
- flag `-version` reports the git commit.


[v0.1.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.1.0
[v0.2.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.2.0
[v0.3.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.3.0
[v0.4.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.4.0
[v0.5.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.5.0
[v0.6.0]: https://github.com/Pix4D/terravalet/releases/tag/v0.6.0
