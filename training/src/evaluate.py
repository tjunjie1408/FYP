"""Evaluate the trained checkpoint against the test split.

Writes /workspace/metrics.json and exits NON-ZERO if accuracy is below
--threshold. That failure is deliberate API surface: it is how the demo
triggers the operator's retry path (e.g. submit a TrainingJob with
epochs "0" and watch backoffLimit get consumed).
"""

import argparse
import json
import sys

import torch
from torch.utils.data import DataLoader
from torchvision import datasets, transforms

from train import ARCHS


def load_test(dataset: str, root: str):
    if dataset == "mnist":
        tf = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.1307,), (0.3081,))]
        )
        return datasets.MNIST(root, train=False, download=True, transform=tf)
    if dataset == "cifar10":
        tf = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.49, 0.48, 0.45), (0.25, 0.24, 0.26)),
            ]
        )
        return datasets.CIFAR10(root, train=False, download=True, transform=tf)
    raise SystemExit(f"unknown dataset: {dataset}")


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--dataset", default="mnist")
    p.add_argument("--model-path", default="/workspace/model.pt")
    p.add_argument("--metrics-out", default="/workspace/metrics.json")
    p.add_argument("--data-root", default="/workspace/data")
    # 0.90 is comfortably below what 2 epochs of MNIST reaches (~0.98).
    # Use a lower threshold for cifar10-cnn (e.g. 0.5).
    p.add_argument("--threshold", type=float, default=0.90)
    args = p.parse_args()

    ckpt = torch.load(args.model_path, map_location="cpu", weights_only=True)
    model = ARCHS[ckpt["arch"]]()
    model.load_state_dict(ckpt["model_state"])
    model.eval()

    loader = DataLoader(load_test(args.dataset, args.data_root), batch_size=256)
    total, correct, loss_sum = 0, 0, 0.0
    with torch.no_grad():
        for x, y in loader:
            out = model(x)
            loss_sum += torch.nn.functional.cross_entropy(out, y, reduction="sum").item()
            correct += (out.argmax(1) == y).sum().item()
            total += y.size(0)

    metrics = {
        "accuracy": round(correct / total, 4),
        "loss": round(loss_sum / total, 4),
        "samples": total,
        "threshold": args.threshold,
    }
    with open(args.metrics_out, "w") as f:
        json.dump(metrics, f, indent=2)
    print(json.dumps(metrics), flush=True)

    if metrics["accuracy"] < args.threshold:
        print(
            f"FAIL: accuracy {metrics['accuracy']} < threshold {args.threshold}",
            file=sys.stderr,
        )
        sys.exit(1)


if __name__ == "__main__":
    main()
