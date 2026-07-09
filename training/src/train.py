"""Train a small CNN on MNIST or CIFAR-10.

Pipeline contract: invoked by the `train` step of the training-pipeline
WorkflowTemplate with the workflow parameters mapped to CLI flags.
Writes the checkpoint to --out (shared /workspace PVC).
"""

import argparse
import json
import time

import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.utils.data import DataLoader
from torchvision import datasets, transforms


class MnistCNN(nn.Module):
    """Tiny 2-conv CNN. Kept in sync with serving/runtime/model_server.py."""

    def __init__(self):
        super().__init__()
        self.conv1 = nn.Conv2d(1, 16, 3, padding=1)
        self.conv2 = nn.Conv2d(16, 32, 3, padding=1)
        self.fc1 = nn.Linear(32 * 7 * 7, 128)
        self.fc2 = nn.Linear(128, 10)

    def forward(self, x):
        x = F.max_pool2d(F.relu(self.conv1(x)), 2)
        x = F.max_pool2d(F.relu(self.conv2(x)), 2)
        x = x.flatten(1)
        x = F.relu(self.fc1(x))
        return self.fc2(x)


class Cifar10CNN(nn.Module):
    def __init__(self):
        super().__init__()
        self.conv1 = nn.Conv2d(3, 32, 3, padding=1)
        self.conv2 = nn.Conv2d(32, 64, 3, padding=1)
        self.fc1 = nn.Linear(64 * 8 * 8, 256)
        self.fc2 = nn.Linear(256, 10)

    def forward(self, x):
        x = F.max_pool2d(F.relu(self.conv1(x)), 2)
        x = F.max_pool2d(F.relu(self.conv2(x)), 2)
        x = x.flatten(1)
        x = F.relu(self.fc1(x))
        return self.fc2(x)


ARCHS = {"mnist-cnn": MnistCNN, "cifar10-cnn": Cifar10CNN}


def load_data(dataset: str, batch_size: int, root: str):
    if dataset == "mnist":
        tf = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.1307,), (0.3081,))]
        )
        train = datasets.MNIST(root, train=True, download=True, transform=tf)
    elif dataset == "cifar10":
        tf = transforms.Compose(
            [
                transforms.ToTensor(),
                transforms.Normalize((0.49, 0.48, 0.45), (0.25, 0.24, 0.26)),
            ]
        )
        train = datasets.CIFAR10(root, train=True, download=True, transform=tf)
    else:
        raise SystemExit(f"unknown dataset: {dataset}")
    return DataLoader(train, batch_size=batch_size, shuffle=True, num_workers=0)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--model", choices=sorted(ARCHS), default="mnist-cnn")
    p.add_argument("--dataset", default="mnist")
    p.add_argument("--epochs", type=int, default=2)
    p.add_argument("--batch-size", type=int, default=128)
    p.add_argument("--lr", type=float, default=0.001)
    p.add_argument("--out", default="/workspace/model.pt")
    p.add_argument("--data-root", default="/workspace/data")
    args = p.parse_args()

    torch.manual_seed(0)
    model = ARCHS[args.model]()
    loader = load_data(args.dataset, args.batch_size, args.data_root)
    opt = torch.optim.Adam(model.parameters(), lr=args.lr)

    start = time.time()
    for epoch in range(args.epochs):
        model.train()
        total, correct, loss_sum = 0, 0, 0.0
        for x, y in loader:
            opt.zero_grad()
            out = model(x)
            loss = F.cross_entropy(out, y)
            loss.backward()
            opt.step()
            loss_sum += loss.item() * y.size(0)
            correct += (out.argmax(1) == y).sum().item()
            total += y.size(0)
        print(
            f"epoch {epoch + 1}/{args.epochs} "
            f"loss={loss_sum / total:.4f} acc={correct / total:.4f} "
            f"elapsed={time.time() - start:.0f}s",
            flush=True,
        )

    torch.save(
        {"model_state": model.state_dict(), "arch": args.model, "classes": 10},
        args.out,
    )
    meta = {"arch": args.model, "epochs": args.epochs, "train_seconds": round(time.time() - start, 1)}
    print(f"saved {args.out} {json.dumps(meta)}", flush=True)


if __name__ == "__main__":
    main()
