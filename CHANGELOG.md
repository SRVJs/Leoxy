## [Unreleased]

### Changed
- upgrade Go to 1.27.3 and refresh dependencies
- bump version to 1.9.1

### Fixed
- correct Docker Compose build context and Dockerfile path

## [3.8.2] - 2026-10-26

### Fixed
- fix prefix handling for root and non-root paths
- correct path trimming in proxy handlers
- update version references

# Changelog

## [3.8.0] - 2026-10-23

### Added
- build and publish Docker images through the reusable CD workflow
- limit Docker publishing to pushes on main and use lowercase image names
- create GitHub releases from version.json on main pushes

### Fixed
- match configured literal access paths exactly
