# ADR-0006: Pin the MinIO image; deploy via plain manifests

**Status:** Accepted

## Context
Community MinIO stopped publishing free Docker images / went source-only in
late 2025. The full web console was trimmed from later community builds.

## Decision
Pin `quay.io/minio/minio:RELEASE.2025-04-22T22-12-26Z` (last full-console
image) and `quay.io/minio/mc:RELEASE.2025-04-16T18-13-26Z` for the bucket-init
job. Deploy MinIO via plain manifests (Deployment + Service + PVC + Secret),
not a helm chart, to avoid chart churn.

## Rationale
- A pinned, known-good image is reproducible and avoids surprise console/feature
  regressions.
- Plain manifests are easy to read and debug for a single-replica demo store.

## Consequences
- The tag must be verified to still pull before first bootstrap.
- If the image is ever pulled from the registry, S3-compatible drop-ins
  (SeaweedFS, Garage) are the documented fallback; the platform only depends on
  the S3 API surface, not MinIO specifically.
