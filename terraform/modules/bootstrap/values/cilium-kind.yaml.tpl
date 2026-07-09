# Cilium on kind: replaces both kindnet and kube-proxy.
kubeProxyReplacement: true
k8sServiceHost: ${api_server_host}
k8sServicePort: ${api_server_port}

ipam:
  mode: kubernetes
routingMode: tunnel

operator:
  replicas: 1
  resources:
    requests: { cpu: 25m, memory: 128Mi }
    limits: { memory: 256Mi }

resources:
  requests: { cpu: 100m, memory: 256Mi }
  limits: { memory: 512Mi }

hubble:
  enabled: true
  relay:
    enabled: true
    resources:
      requests: { cpu: 10m, memory: 64Mi }
  # UI off by default to save RAM; enable for the netpol demo screenshots:
  # helm upgrade cilium ... --set hubble.ui.enabled=true
  ui:
    enabled: false
