FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
# CPU wheels only — no CUDA, keeps the image small and demo-friendly.
RUN pip install --no-cache-dir -r requirements.txt \
    --extra-index-url https://download.pytorch.org/whl/cpu

COPY src/ src/
# No ENTRYPOINT: the Argo WorkflowTemplate sets the command per step.
