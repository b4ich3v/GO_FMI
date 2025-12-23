# Kubernetes Agent

Read-only diagnostics agent for Kubernetes clusters. Backed by kubeconfig; no write operations.

## What it does
- Executes a fixed set of read-only tools (`K8S_READONLY_TOOLS`): pods, logs, events, debug bundle, describe, service/endpoints/ingress.
- Works in two modes:
  - **Deterministic**: `/kube ...` commands parsed into tool payloads.
  - **LLM fallback**: free-text `/kube ...` routed to the same tools, with validation and context injection to prevent hallucinated contexts.
- Requires a kubeconfig per session (uploaded via `/kube upload` UI modal).

## Main pieces
- `k8s_agent.py`: orchestrates prompt → tool call → formatting; selects kubeconfig backend.
- `command_parser/`: parses `/kube ...` into structured payloads; `kube_command_pipeline` injects kubeconfig path, validates context, and normalizes params.
- `context/context_resolver.py`: loads contexts from kubeconfig; resolves requested vs current-context.
- `clients/`: kubeconfig backend (client pool, resource clients, formatters, sanitizers). `KubeApiClientPool` caches API clients per context with TTL/size limits.
- `tools/`: LangChain toolset + factory mapping the backend to structured tools.
- `validation/`: `K8SValidator` (agent lifecycle args) and `K8SParamsValidator` (payload shape, required fields per tool, tail bounds, allowed tool set).
- `k8s_history_recorder.py`: writes a concise “K8S: …” summary + response into shared history.

## Flow (deterministic command)
1) User sends `/kube pods -A --context ctx1`.
2) `KubeParser` builds a payload (`tool=list_pods`, flags, context).
3) `prepare_kube_request` injects kubeconfig path, validates context via `ContextResolver`, caps tail, ensures required fields.
4) `K8SAgent` builds prompt with defaults, runs the LangChain tool, formats response, records history.

## Flow (LLM fallback)
1) User sends `/kube why is my pod broken?`.
2) Parser returns an empty query + AgentRequest(K8S) with params possibly None.
3) `prepare_kube_request` injects kubeconfig path and, if no context provided, fills current-context to avoid hallucinations; rejects invalid contexts early.
4) LLM chooses tools from the read-only set; validation still applies before execution.

## Validation & limits
- Context must exist in kubeconfig; otherwise a clear validation error.
- Required fields per tool enforced (`pod`, `service`, `deployment`, `ingress` as applicable).
- Tail capped (`K8S_MAX_TAIL`); default tail (`K8S_DEFAULT_TAIL`).
- Tool must be in `K8S_READONLY_TOOLS`.

## Kubeconfig handling
- One kubeconfig per session, stored under `instance/kubeconfigs/<user_hash>/<session_id>.yaml`.
- Context indicator in the UI shows current-context after upload.
- `ContextResolver` lists contexts and returns the current one if none is requested.

## Usage examples
- List all pods: `/kube pods -A`
- Logs from pod: `/kube logs mypod -n default --tail 300`
- Debug bundle: `/kube bundle mypod -n default --logs --tail 400`
- Describe deployment: `/kube describe deployment my-deploy -n default`
- Service: `/kube service my-svc -n default`

## Read-only guarantee
- Only read tools are exposed; validators block anything outside the allowed set.

## Commands reference

**Format**
```
/kube
/kube help
/kube pods -n <namespace> [-l <label_selector>] [--all-namespaces|-A] [--context <ctx>]
/kube logs <pod> -n <namespace> [-c <container>] [--tail <N>] [--previous] [--context <ctx>]
/kube events -n <namespace> [--context <ctx>]
/kube bundle <pod> -n <namespace> [--logs|--include-logs] [--no-logs] [--tail <N>] [--context <ctx>]
/kube describe pod <pod> -n <namespace> [--context <ctx>]
/kube describe deployment <deployment> -n <namespace> [--context <ctx>]
/kube service <service> -n <namespace> [--context <ctx>]
/kube endpoints <service> -n <namespace> [--context <ctx>]
/kube ingress <ingress> -n <namespace> [--context <ctx>]
```

**Example commands (parser)**
```
/kube
/kube help
/kube pods -n default
/kube pods -A
/kube events -n default
/kube events -n kube-system
/kube logs my-pod -n default --tail 200
/kube logs my-pod -n default --tail 200 --previous
/kube logs my-pod -n default -c my-container --tail 200
/kube bundle my-pod -n default --no-logs
/kube bundle my-pod -n default --tail 200
/kube describe pod my-pod -n default
/kube describe deployment firedrill -n firedrill
/kube service kubernetes -n default
/kube endpoints kubernetes -n default
/kube ingress firedrill-ingress -n firedrill
```

**Example LLM queries**
```
/kube is there any issues with the cluster
/kube show service kubernetes in namespace default
/kube show endpoints for service kubernetes in namespace default
/kube look for recent warning events in kube-system
/kube give me a quick health overview of the cluster (pods + warning events in kube-system, ingress-nginx, monitoring)
/kube are there any warning events or failing pods across firedrill, monitoring and ingress-nginx
/kube summarize what you can tell about namespace default and pod my-pod (status, recent events, logs tail)
/kube investigate my-pod in default and tell me if it looks healthy (events + bundle without logs first)
/kube check whether ingress-nginx looks healthy (pods + events)
```
