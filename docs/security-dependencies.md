# Dependency security baseline

This project treats dependency security as a release concern. The checks below are
intended to be reproducible locally and in CI without exposing credentials.

## Go modules

- `go vet ./...`, `go test -race ./...`, and `govulncheck ./...` run in CI.
- `golang.org/x/crypto` is required for bcrypt password hashing. The project does
  not use `golang.org/x/crypto/openpgp`.
- The Go vulnerability database currently reports `GO-2026-5932` for the
  unmaintained `openpgp` package. It has no fixed upstream version, but it is not
  reachable from this project because there is no OpenPGP import. The
  `scripts/check-no-openpgp.sh` check prevents an accidental import from turning
  this module-only advisory into an application vulnerability.
- If OpenPGP becomes a product requirement, stop and select a maintained,
  reviewed implementation instead of importing the legacy package.

## JavaScript dependencies

The two frontends are intentionally handled in phases:

- Direct runtime dependencies with compatible security updates are kept current
  in `package.json` and both lockfiles.
- CI blocks on fixable high and critical vulnerabilities in the production
  dependency graph with:

  ```sh
  npm audit --omit=dev --audit-level=high
  ```

- The admin console remains a Vue 2 application for this phase. Its full
  development-tree audit can still report advisories in the archived Vue CLI 5,
  Vue 2 compiler, and related build tooling. The blog's SVG sprite build chain
  has the same kind of development-only transitive exposure. These are tracked
  exceptions until the Vue 2/build-chain migration is implemented; `npm audit
  fix --force` is not an accepted remediation.
- Any new fixable high/critical issue in production dependencies must be fixed
  before release, even when the full development-tree audit contains an existing
  exception.

## Container images

- Base and infrastructure images are pinned by tag and immutable digest in the
  Dockerfile and Compose manifests.
- CI builds the application image and scans it. Fixable `HIGH` and `CRITICAL`
  findings in that image fail CI; unfixed findings remain visible for
  follow-up.
- CI also scans every public service image referenced by the Compose manifests.
  Those results are report-only because the remediation is an upstream image
  release: the updated pinned digests reduce the old-image exposure, but still
  contain vendor-image findings (including package updates and vulnerabilities
  in bundled service binaries) until upstream publishes fully patched builds.
  The workflow writes a warning to the job summary so a patched vendor digest
  can be adopted without hiding the finding or weakening the application-image
  gate.
- Updating these manifests does not operate the deployment. Existing containers
  and their data volumes must not be stopped, recreated, or migrated as part of
  a source change.
- Renovate is enabled for Go modules, npm lockfiles, Dockerfiles, and Compose
  images, with digest pinning and manual merge approval for image updates.

## Local checks

From the repository root:

```sh
bash scripts/check-no-openpgp.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
go test -race ./...

cd web/blog && npm ci --no-audit --no-fund && npm audit --omit=dev --audit-level=high
cd ../admin && npm ci --no-audit --no-fund && npm audit --omit=dev --audit-level=high
```

Container scans are performed by the CI runner with Trivy; they do not require
the local production containers to be running.
