# vk-sender

## Command Guide

### Run with exported .env (one‑liner)

Exports all variables from `.env` into the current shell and runs the service.

```
export $(cat .env | xargs) && go run ./cmd/vk-sender
```

### Run with `source` (safer for complex values)

Loads `.env` preserving quotes and special characters, then runs the service.

```
set -a && source .env && set +a && go run ./cmd/vk-sender
```

### Fetch/clean module deps

Resolves dependencies and prunes unused ones.

```
go mod tidy
```

### Verbose build (diagnostics)

Builds the binary with verbose and command tracing. Removes old binary after build to keep the tree clean.

```
go build -v -x ./cmd/vk-sender && rm -f vk-sender
```

### Docker build (Buildx)

Builds the image with detailed progress logs and without cache.

```
docker buildx build --no-cache --progress=plain .
```

### Create and push tag

Cuts a release tag and pushes it to remote.

```
git tag v0.0.1
git push origin v0.0.1
```

### Manage tags

List all tags, delete a tag locally and remotely, verify deletion.

```
git tag -l
git tag -d vX.Y.Z
git push --delete origin vX.Y.Z
git ls-remote --tags origin | grep 'refs/tags/vX.Y.Z$'
```
