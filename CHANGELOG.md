# Terravalet Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased [v0.4.0] - (2021-XX-XX)
### Fixes

### Breaking changes

- The CLI API has changed; now it must be invoked by specifying the subcommand:
  ```
  $ terravalet rename -plan=PLAN -up=UP.sh -down=DOWN.sh
  ```

### Changes
### New

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
