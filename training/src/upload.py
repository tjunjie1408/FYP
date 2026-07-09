"""Upload model artifact + metrics to MinIO (S3).

Contract: --s3-path is "<bucket>/<prefix>" (e.g. models/training/mnist-demo),
exactly the `s3-path` workflow parameter the operator computes in
operator/internal/controller/workflow_builder.go (S3Path). The resulting
objects back the InferenceService STORAGE_URI s3://<bucket>/<prefix>/.
"""

import argparse
import os

import boto3


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--s3-path", required=True, help="bucket/prefix, e.g. models/training/mnist-demo")
    p.add_argument("--files", nargs="+", default=["/workspace/model.pt", "/workspace/metrics.json"])
    args = p.parse_args()

    endpoint = os.environ.get("S3_ENDPOINT", "http://minio.minio.svc.cluster.local:9000")
    bucket, _, prefix = args.s3_path.partition("/")
    if not bucket or not prefix:
        raise SystemExit(f"--s3-path must be bucket/prefix, got: {args.s3_path}")

    s3 = boto3.client(
        "s3",
        endpoint_url=endpoint,
        # AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY come from the
        # minio-credentials Secret via envFrom in the WorkflowTemplate.
        region_name=os.environ.get("AWS_REGION", "us-east-1"),
    )

    for path in args.files:
        key = f"{prefix}/{os.path.basename(path)}"
        s3.upload_file(path, bucket, key)
        print(f"uploaded s3://{bucket}/{key}", flush=True)


if __name__ == "__main__":
    main()
