"""KServe custom predictor for the MNIST/CIFAR CNN.

The KServe storage-initializer downloads STORAGE_URI (s3://models/...) into
/mnt/models before this container starts; load() reads model.pt from there.
"""

import os

import kserve
import torch
import torch.nn as nn
import torch.nn.functional as F


class MnistCNN(nn.Module):
    """Must stay in sync with training/src/train.py."""

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
SHAPES = {"mnist-cnn": (1, 28, 28), "cifar10-cnn": (3, 32, 32)}


class CNNModel(kserve.Model):
    def __init__(self, name: str):
        super().__init__(name)
        self.model = None
        self.shape = None

    def load(self):
        path = os.path.join(os.environ.get("MODEL_DIR", "/mnt/models"), "model.pt")
        ckpt = torch.load(path, map_location="cpu", weights_only=True)
        self.model = ARCHS[ckpt["arch"]]()
        self.model.load_state_dict(ckpt["model_state"])
        self.model.eval()
        self.shape = SHAPES[ckpt["arch"]]
        self.ready = True

    def predict(self, payload, headers=None):
        # Accepts {"instances": [...]} with flat (e.g. 784 floats) or nested
        # (28x28 / 3x32x32) instances.
        instances = payload["instances"]
        x = torch.tensor(instances, dtype=torch.float32).reshape(-1, *self.shape)
        with torch.no_grad():
            out = self.model(x)
        return {"predictions": out.argmax(1).tolist()}


if __name__ == "__main__":
    model = CNNModel(os.environ.get("MODEL_NAME", "mnist-demo"))
    model.load()
    kserve.ModelServer().start([model])
