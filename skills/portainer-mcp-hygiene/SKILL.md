---
name: portainer-mcp-hygiene
description: How to efficiently query the Portainer MCP server's tools — when to project responses with `select` (JMESPath), where the heavy fields live (snapshots, status blocks, managed fields), and how to handle non-JSON Docker/K8s proxy endpoints (logs, stats, exec). Trigger this whenever you're about to call any Portainer MCP tool — including `docker_proxy`, `kubernetes_proxy`, `EndpointList`, `GetAllKubernetes*`, `StackList`, `snapshot*`, `Helm*`, or any other `mcp__portainer__*` tool — and whenever the user asks about Portainer environments, Docker containers/images/stacks/networks managed by Portainer, Kubernetes resources via Portainer, or Helm releases. Use it even if the user doesn't mention Portainer by name, as long as the working answer requires one of these tools.
---

# Portainer MCP hygiene

The Portainer MCP server returns large JSON payloads by default — a list of environments with snapshots, a list of K8s pods with full status blocks, a stack with its complete manifest. Every tool the server exposes accepts an optional `select` (JMESPath) parameter applied server-side before the response reaches you. Responses are capped at ~50,000 chars; if you exceed the cap you get a truncation hint that names `select` and shows an example.

The cost of *not* projecting is real: 50K chars of dense JSON eats roughly 20K tokens out of your context for a question that usually needed a few hundred. Once truncation fires, you've wasted a round trip and the data past the cap is gone for that call. The default move on any list-shaped Portainer call is to pass `select` from the start.

## The default pattern

For any call that returns a list of objects, ship a JMESPath that keeps only the fields the user's question actually needs:

```
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
docker_proxy(path="/containers/json", select="[].{id:Id,name:Names[0],state:State,image:Image}")
kubernetes_proxy(path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,node:spec.nodeName}")
```

JMESPath syntax notes that matter for these surfaces:
- List shape: start with `[]` to map over array elements.
- Wrapped list (Kubernetes `{items: [...]}`): start with `items[]`.
- Single object: `{field1:path.to.value,field2:other.path}` — no leading `[]`.
- Nested paths use dots: `Snapshots[0].RunningContainerCount`, `metadata.labels."app.kubernetes.io/name"` (quote keys that contain dots or hyphens).

## Where the noise lives

These are the fields/sections that dominate Portainer payloads. Either project them out (when you don't need them) or project specifically into them (when they *are* the answer):

**`EndpointList` — `.Snapshots[0]` carries the heavy payload.**
Each environment includes a full Docker or Kubernetes snapshot — container list, image list, network list, etc. For counts and status questions you almost always want to project into specific snapshot fields rather than fetch them whole:

```
# Container counts per environment
EndpointList(select="[].{name:Name,running:Snapshots[0].RunningContainerCount,total:Snapshots[0].ContainerCount}")

# Just identity + reachability
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
```

**Kubernetes via `kubernetes_proxy` — `metadata.managedFields` and `status` are huge.**
`metadata.managedFields` alone is routinely 30-70% of an object. The `status` block on Deployments, StatefulSets, Pods, and Nodes is similarly verbose. Project them out unless the user is asking about reconciliation state or controller history:

```
# Pod summary
kubernetes_proxy(path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,restarts:status.containerStatuses[0].restartCount,node:spec.nodeName}")

# Deployment readiness
kubernetes_proxy(path="/apis/apps/v1/deployments", select="items[].{name:metadata.name,ns:metadata.namespace,replicas:spec.replicas,ready:status.readyReplicas}")
```

**`GetAllKubernetes*` tools — full status blocks per object.**
The OpenAPI-generated `GetAllKubernetesApplications`, `GetAllKubernetesConfigMaps`, `GetAllKubernetesIngresses`, etc. return arrays where each element carries its full object body. Same rules as the proxy: project to the named fields you need.

**`StackList` and `StackInspect` — config and env vars.**
Stacks carry the full compose/manifest content plus environment variable dictionaries. If the user asked "which stacks exist?", project to `{id, name, type, status}`. If they asked about a specific stack's config, fetch it directly and only then look at the body.

**Snapshot inspects (`snapshotInspect`, `snapshotContainersList`, etc.) — entire snapshots.**
These return the *whole* snapshot blob by design. Always project.

**Helm endpoints — full chart values and manifests.**
`HelmList` carries release status + chart metadata; `HelmGet` returns the rendered manifest. Project to release names and status when listing; only fetch the manifest when the user asked to see it.

**`EndpointGetCharts`, `dockerDashboard`, `EndpointSummaryCounts` — already aggregated.**
These are the lightweight "summary" tools. Prefer them over `EndpointList` + projection when the user's question is purely a count or rollup — fewer characters, less work, more accurate (server-side aggregation).

## Patterns for common questions

A few high-frequency questions and the projection that gets them in one call:

**"How many running containers in each environment?"**
```
EndpointList(select="[].{name:Name,type:Type,running:Snapshots[0].RunningContainerCount,total:Snapshots[0].ContainerCount}")
```

**"List containers in environment N."**
```
docker_proxy(environment_id=N, path="/containers/json",
             select="[].{id:Id,name:Names[0],state:State,image:Image,status:Status}")
```

**"Which images are in use, grouped by name?"**
Fetch with projection, group client-side:
```
docker_proxy(environment_id=N, path="/containers/json", select="[].Image")
```

**"One-line pod summary in environment N."**
```
kubernetes_proxy(environment_id=N, path="/api/v1/pods",
                 select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,node:spec.nodeName}")
```

**"Which deployments aren't fully ready?"**
Project readiness fields, then filter in the response. (JMESPath can also filter inline with `items[?status.readyReplicas != spec.replicas]`, but expressions like that are easy to get wrong — projection + your own filter is usually safer.)

**"Inspect deployment X in namespace Y."**
A single-object fetch. Project out `metadata.managedFields` and `status.conditions` if you only need the spec; keep them if the user is asking about reconciliation:
```
kubernetes_proxy(environment_id=N, path="/apis/apps/v1/namespaces/Y/deployments/X",
                 select="{name:metadata.name,replicas:spec.replicas,ready:status.readyReplicas,image:spec.template.spec.containers[0].image}")
```

## Non-JSON endpoints — `select` does not apply

A handful of `docker_proxy` and `kubernetes_proxy` paths return plain text or streamed data rather than JSON. `select` is a no-op on these (the proxy detects non-JSON and passes the body through unchanged), but the response-size cap still fires, and `_select_wrapper` will raise a JSON parse error if you do pass `select`. **Narrow the upstream query parameters instead.**

**Container logs** — `/containers/{id}/logs`:
- Set `tail` to limit lines (`tail=100` for the last hundred).
- Set `since` to limit time range (Unix timestamp).
- Always pass `stdout=true` and/or `stderr=true` — without them the daemon returns nothing.
- Don't set `follow=true` — it streams indefinitely and will burn your context.

**Container stats** — `/containers/{id}/stats`:
- Always pass `stream=false` to get a single snapshot. The streaming form is unbounded.

**Container exec output** — chunked stream.
- If you need command output, prefer `docker_proxy` against `/containers/{id}/top` for process listing, or run the command another way. Exec attach over HTTP returns multiplexed binary frames and won't render usefully through the cap.

**Image pulls / archives / build context** — binary or streamed.
- Don't fetch these through the proxy for inspection. Use the specific Portainer endpoints (`endpointDockerhubStatus`, `ServiceImageStatus`, `dockerImagesList`) which return parseable JSON summaries.

If the cap fires on a non-JSON endpoint, the truncation hint will suggest `select` — ignore that suggestion in this case and retry with narrower upstream parameters.

## When *not* to project

Projecting isn't always right:

- **Small single-object reads** that you already know are under a few KB — `SettingsInspect`, `MOTD`, `StatusInspect`, `systemVersion`. Projecting just adds a round of cognitive overhead for no win.

- **Exploratory scans where you don't know what you're looking for** — "anything unusual in this stack's config", "is there an error somewhere in this deployment's status". Here you want the full body so you can scan for patterns. Pull the full object; if it truncates, narrow the *path* (one resource, not the whole list) rather than projecting fields.

- **When the user asked for "everything"** — sometimes they really do want the raw object. Respect that, but warn them once if you're about to retrieve something that will eat their context.

## Reading the truncation hint

When you do hit the cap, the response ends with a bracketed `[truncated: ... Retry with a JMESPath `select` ...]` message that includes a concrete example. Your next move should almost always be: retry the *same* call with a `select` projection — not pivot to reading the spilled file with `jq`, not paginate by guessing offsets, not call a different tool. The server-side projection is cheaper (no re-fetch from Portainer if the data was already cached upstream, and far fewer tokens shipped back).

The exception is non-JSON endpoints (see above) — there, ignore the `select` suggestion and re-shape the upstream query instead.

## Tool selection cheatsheet

- Environment-level summary (counts, status, reachability) → `EndpointList` with snapshot projection, or `EndpointSummaryCounts`/`dockerDashboard` if the question is purely aggregate.
- Docker things on a specific environment → `docker_proxy`. The OpenAPI-generated `dockerContainerGpusInspect`, `containerImageStatus`, etc. are specific helpers; use them when they directly answer the question, otherwise the proxy is more flexible.
- Kubernetes things on a specific environment → either the OpenAPI-generated `GetAllKubernetes*` / `GetKubernetes*` tools (Portainer-aware, often already filtered) or `kubernetes_proxy` (raw K8s API, full flexibility). Prefer the typed tool when it exists; fall back to the proxy for paths Portainer doesn't surface natively.
- Helm releases → `HelmList`, `HelmGet`, `HelmGetHistory`. Don't try to route Helm through the K8s proxy — Portainer's Helm tools see the release metadata the K8s API alone doesn't.
- Mutations (POST/PUT/DELETE) → only in read-write mode. If the server is in `PORTAINER_READ_ONLY=1`, non-GET calls are rejected at the tool with a clear error. Don't retry mutations as GET when this happens — surface the read-only state to the user.
