# How to Release

Releases are automated via GoReleaser and GitHub Actions. A push of a semver tag triggers the workflow, which builds binaries for all platforms and creates a GitHub Release.

## Steps

1. Make sure all changes are committed and pushed to `main`.

2. Tag the release:
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

3. The [release workflow](.github/workflows/release.yml) will automatically:
   - Build binaries for Linux, macOS, and Windows (amd64 + arm64)
   - Create a GitHub Release with archives, checksums, and a changelog

4. Verify at https://github.com/stefanclaw/stefanclaw/releases/latest

## Dry run

Test the release process locally without publishing:

```bash
make release-dry-run
```

This produces archives in `dist/` so you can inspect the output.

## Version format

Use [semantic versioning](https://semver.org/): `vMAJOR.MINOR.PATCH` (e.g. `v0.1.0`, `v1.0.0`).
