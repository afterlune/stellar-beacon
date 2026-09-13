# Dependency security baseline

This project treats dependency security as a release concern. The checks below are
intended to be reproducible locally and in CI without exposing credentials.

## Go modules

- `go vet ./...`, `go test -race ./...`, and `govulncheck ./...` run in CI.
- `golang.org/x/crypto` is required for bcrypt password hashing. The project does
  not use `golang.org/x/crypto/openpgp`. Keep it on a patched release (currently
  `v0.56.0` or later) to address the SSH package advisories.
- The Go vulnerability database currently reports `GO-2026-5932` for the
  unmaintained `openpgp` package. It has no fixed upstream version, but it is not
  reachable from this project because there is no OpenPGP import. The
  `scripts/checks/check-no-openpgp.sh` check prevents an accidental import from turning
  this module-only advisory into an application vulnerability.
- If OpenPGP becomes a product requirement, stop and select a maintained,
  reviewed implementation instead of importing the legacy package.

## JavaScript dependencies

The frontends are audited as a single dependency tree, including development and
build tools:

- CI blocks on any npm advisory in the workspace with:

  ```sh
  npm audit --registry=https://registry.npmjs.org
  ```

- Both frontends use Vite; the blog no longer includes the Vue CLI/Webpack SVG
  sprite build chain. The blog's typecheck and production build run in CI.
- `echarts` is kept at `v6.1.0` or later to include the XSS fix. Do not use
  `npm audit fix --force`; review and explicitly update affected direct
  dependencies instead.

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
bash scripts/checks/check-no-openpgp.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
go test -race ./...

cd web && npm ci --no-audit --no-fund
npm audit --registry=https://registry.npmjs.org
npm run typecheck --workspace=@stellar-beacon/blog
npm run build:blog
npm run build:admin
```

Container scans are performed by the CI runner with Trivy; they do not require
the local production containers to be running.
