---
name: portainer-mcp-hygiene
description: "How to drive the Portainer MCP server's tools correctly — both reading and mutating. Reading: project responses with `select` (JMESPath), where the heavy fields live (snapshots, status blocks, managed fields), how to handle non-JSON Docker/K8s proxy endpoints (container and pod logs, stats, exec), and how to interpret results that are easy to misread (e.g. an edge environment's health comes from its heartbeat, not its `Status` field; typed K8s tools use different field names than the raw proxy). Mutating: deploying, scaling, restarting, and deleting Portainer-managed resources — where success payloads are empty and must be verified out-of-band, and where cleanup (orphaned volumes) and recovery (name-based vs id-based calls) have gotchas. Trigger this whenever you're about to call any Portainer MCP tool — including `docker_proxy`, `kubernetes_proxy`, `EndpointList`, `GetAllKubernetes*`, `StackList`, `StackCreateKubernetes*`, `StackDelete*`, `CreateKubernetes*`, `snapshot*`, `Helm*`, or any other `mcp__portainer__*` tool — and whenever the user asks you to inspect, deploy, scale, restart, or remove Docker containers/images/stacks/networks, Kubernetes resources, or Helm releases managed by Portainer. Use it even if the user doesn't mention Portainer by name, as long as the working answer requires one of these tools."
---

# Portainer MCP hygiene

The Portainer MCP server returns large JSON payloads by default — a list of environments with snapshots, a list of K8s pods with full status blocks, a stack with its complete manifest. Every tool the server exposes accepts an optional `select` (JMESPath) parameter applied server-side before the response reaches you. Responses are capped at ~50,000 chars; if you exceed the cap you get a truncation hint that names `select` and shows an example.

The cost of *not* projecting is real: 50K chars of dense JSON eats roughly 20K tokens out of your context for a question that usually needed a few hundred. Once truncation fires, you've wasted a round trip and the data past the cap is gone for that call. The default move on any list-shaped Portainer call is to pass `select` from the start.

## Resolve the environment first

Both proxy tools take an `environment_id`, and the IDs aren't predictable —
they're assigned in creation order, not "local = 1". So for any
environment-scoped question, resolve the target first with one call:

```
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
```

and use the ID whose name matches what the user means. Guessing the environment is a hard
failure, not a silent one: a wrong ID makes the proxy raise an `API request
failed (HTTP 404)` error rather than returning empty data — so a wrong environment is *not*
a source of `null` projections. Two other things are, though: an unquoted dotted key (see the
JMESPath notes below) and a field-name mismatch on the typed `GetAllKubernetes*` tools (see
their note under *Where the noise lives*). So read an all-`null` result as a wrong-keys signal
to investigate, not as proof the data is absent.

## The default pattern

For any call that returns a list of objects, ship a JMESPath that keeps only the fields the user's question actually needs:

```
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
docker_proxy(environment_id=N, path="/containers/json", select="[].{id:Id,name:Names[0],state:State,image:Image}")
kubernetes_proxy(environment_id=N, path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,node:spec.nodeName}")
```

JMESPath syntax notes that matter for these surfaces:
- List shape: start with `[]` to map over array elements.
- Wrapped list (Kubernetes `{items: [...]}`): start with `items[]`.
- Single object: `{field1:path.to.value,field2:other.path}` — no leading `[]`.
- Nested paths use dots: `Snapshots[0].RunningContainerCount`. But a key that
  *itself* contains a dot or hyphen — compose labels, K8s annotations — collides
  with that syntax, so quote it as an identifier:
  `Labels."com.docker.compose.project"`, or in a filter
  `[?Labels."com.docker.compose.project"=='myproj']`. Unquoted, JMESPath reads
  the dots as nested keys and silently returns `null` — which looks like a
  missing field rather than a broken expression. Quote with double quotes, not
  backslashes (`\"…\"` fails to parse) and not backticks (those denote JSON
  literals).
- The backslash trap is easy to fall into *by accident*: the `select` string is
  itself an argument inside a JSON tool call, and escaping the inner quotes a
  second time sends literal `\"` to the server. The failure signature is a
  parse error mentioning `Unknown token \`. When you hit it, either resend with
  plain double quotes (encode the tool-call JSON once, not twice), or sidestep
  quoting entirely with a function filter — `[?contains(metadata.name,
  'ingress-nginx')]` uses only single quotes and survives any transport.

## Where the noise lives

These are the fields/sections that dominate Portainer payloads. Either project them out (when you don't need them) or project specifically into them (when they *are* the answer):

**`EndpointList` — `.Snapshots[0]` carries the heavy payload.**
Each environment includes a full Docker or Kubernetes snapshot — container list, image list, network list, etc. For counts and status questions you almost always want to project into specific snapshot fields rather than fetch them whole:

```
# Container counts per environment
EndpointList(select="[].{name:Name,running:Snapshots[0].RunningContainerCount,total:Snapshots[0].ContainerCount}")

# Just identity + status field
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status}")
```

`Status` is a reachability signal only for *direct* agents. For edge
environments it doesn't track reachability at all — see *Interpreting
results, not just shrinking them* below before reading it as up/down.

Two `EndpointList` query params don't surface over MCP: `updateInformation`
(agent-update availability) and `k8sEnvAdmin` (K8s admin flag). Portainer
returns their answer in HTTP response *headers* (`X-Update-Available`,
`X-K8S-Env-Admin`), and this server only ever returns the response *body* —
so passing them changes nothing you can read. Don't rely on them through the MCP.

**Kubernetes via `kubernetes_proxy` — `metadata.managedFields` and `status` are huge.**
`metadata.managedFields` alone is routinely 30-70% of an object. The `status` block on Deployments, StatefulSets, Pods, and Nodes is similarly verbose. Project them out unless the user is asking about reconciliation state or controller history:

```
# Pod summary
kubernetes_proxy(environment_id=N, path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,restarts:status.containerStatuses[0].restartCount,node:spec.nodeName}")

# Deployment readiness
kubernetes_proxy(environment_id=N, path="/apis/apps/v1/deployments", select="items[].{name:metadata.name,ns:metadata.namespace,replicas:spec.replicas,ready:status.readyReplicas}")
```

A caveat on that pod projection: `status.containerStatuses[0].restartCount` —
and `status.containerStatuses[0].state.waiting.reason` — comes back `null` for
pods that never started a container, because Pending and many Failed pods have
an empty `containerStatuses`. That's exactly the unhealthy set you're usually
chasing, so don't read a `null` restart count as "zero restarts, healthy."
Lead with `status.phase` and the pod-level `status.reason` (e.g. `Evicted`),
and treat `restartCount` as extra detail that only exists once a container has
actually run:

```
# Pod health, robust for not-yet-running pods
kubernetes_proxy(environment_id=N, path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,reason:status.reason,restarts:status.containerStatuses[0].restartCount,node:spec.nodeName}")
```

**`GetAllKubernetes*` tools — full object body per element, in a Portainer-specific shape.**
The OpenAPI-generated `GetAllKubernetesApplications`, `GetAllKubernetesPersistentVolumes`, `GetAllKubernetesConfigMaps`, etc. return arrays where each element carries its full body — so project to the fields you need. But mind the field names: these typed tools do **not** use the raw-K8s paths the proxy uses (`metadata.name`, `status.phase`, `spec.…`), and they don't reliably match the PascalCase of `EndpointList` (`Id`, `Name`) either. They tend to be flattened camelCase, and some fields collapse — e.g. `GetAllKubernetesPersistentVolumes` returns identity as a top-level `name` (not `metadata.name`) and `status` as a bare string `"Bound"` (not `status.phase`):

```
# Actual shape — camelCase, flattened, status-as-string
GetAllKubernetesPersistentVolumes(select="[].{name:name,status:status,claimNs:claimRef.namespace,sc:storageClassName,reclaim:persistentVolumeReclaimPolicy}")
```

Because the convention varies by tool, an all-`null` projection here almost always means you guessed the wrong keys — reusing the proxy's `metadata.name` or `EndpointList`'s `Name` — not that the data is absent. When unsure, **fetch one element with no `select` first** to read the real field names, then project. If you'd rather work in raw-K8s field names, use `kubernetes_proxy` instead.

**`StackList` and `StackInspect` — config and env vars.**
Stacks carry the full compose/manifest content plus environment variable dictionaries. If the user asked "which stacks exist?", project to `{id, name, type, status}`. If they asked about a specific stack's config, fetch it directly and only then look at the body. Env *values* come back redacted by default — see *Env values are redacted by default* below.

**Snapshot inspects (`snapshotInspect`, `snapshotContainersList`, etc.) — entire snapshots.**
These return the *whole* snapshot blob by design. Always project.

**Helm endpoints — full chart values and manifests.**
`HelmList` carries release status + chart metadata; `HelmGet` returns the rendered manifest. Project to release names and status when listing; only fetch the manifest when the user asked to see it.

**`EndpointGetCharts`, `dockerDashboard`, `EndpointSummaryCounts` — already aggregated.**
These are the lightweight "summary" tools. Prefer them over `EndpointList` + projection when the user's question is purely a count or rollup — fewer characters, less work, more accurate (server-side aggregation).

**Env values are redacted by default.**
Stack, container, and Kubernetes env values come back as `[REDACTED]`. The response also carries a one-line summary: `[N env value(s) redacted; set PORTAINER_EXPOSE_ENV_VALUES=1 on the MCP server to disclose]`.

- Don't waste a tool call fishing for them via `select` — the projection runs *after* redaction, so any field path lands on the sentinel.
- If the user genuinely needs an env value (troubleshooting a deploy), tell them to set `PORTAINER_EXPOSE_ENV_VALUES=1` on the MCP server and reconnect. Don't invoke the toggle yourself.
- The sentinel `[REDACTED]` is a literal placeholder — never quote it back to the user as if it were the real value.
- Redaction covers `Env` / `EnvVars` shapes (stack `Env` pairs, Docker `KEY=VAL` strings, K8s `env[].value`). K8s `valueFrom` references are preserved — they're references to a Secret/ConfigMap, not the secret material itself.

## Interpreting results, not just shrinking them

Projecting the right fields only helps if you read them correctly. The
highest-frequency misread on this surface:

**Edge environments — `Status` is not the health signal; `Heartbeat` is.**
The endpoint `Status` field means up/down only for *direct* agents (`1 = up`,
`2 = down`). For *edge* agents (`Type` 4 = EdgeAgentOnDocker, 7 =
EdgeAgentOnKubernetes) `Status` is not a reachability signal in either
direction: Portainer sets it to `1` when the edge environment is created and
again on every check-in, and never flips it to `2` (down) on its own. So a dead
edge agent that last checked in an hour ago still reads `Status: 1`, and you may
also see `Status: 0` on an edge env that was provisioned but never completed a
normal create/check-in path — neither value reflects current health. Don't read
edge `Status` as up *or* down. Portainer
judges an edge agent by its **heartbeat**: up if it checked in within
`2 × interval + 20` seconds (the interval is `EdgeCheckinInterval`, falling
back to the global Edge check-in setting, for standard agents; the smallest of
the ping/command/snapshot intervals for async agents). The server exposes this
as a computed `Heartbeat` boolean — which is exactly what the dashboard's
environment badge renders ("Heartbeat" vs "Down").

For a reachability check that's correct across both kinds, project the
heartbeat inputs, not just `Status`:

```
EndpointList(select="[].{id:Id,name:Name,type:Type,status:Status,heartbeat:Heartbeat,lastCheckIn:LastCheckInDate}")
```

Read it as: **direct agent → trust `status`** (`1` = up); **edge agent
(`type` 4/7) → trust `heartbeat`** (`true` = up) and ignore `status`.

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

A handful of `docker_proxy` and `kubernetes_proxy` paths return plain text or streamed data rather than JSON — logs, stats, exec output. On these the proxy detects the non-JSON body and passes it through unchanged, so any `select` you pass is silently ignored — a no-op, not an error, since there's no JSON to project. The response-size cap still applies, so a noisy stream can still truncate. **Narrow the upstream query parameters instead of projecting.**

**Container logs** — `/containers/{id}/logs`:
- Set `tail` to limit lines (`tail=100` for the last hundred).
- Set `since` to limit time range (Unix timestamp).
- Always pass `stdout=true` and/or `stderr=true` — without them Docker rejects the call with a 400.
- Don't set `follow=true` — it streams indefinitely and will burn your context.

**Kubernetes pod logs** — `kubernetes_proxy` path `/api/v1/namespaces/{ns}/pods/{pod}/log`:
- Cap the output with `tailLines` (e.g. `tailLines=100`) and/or `limitBytes` — the K8s equivalents of Docker's `tail`. Both are query params.
- Add `previous=true` to read the *prior* container after a crash/restart — the current container may be too young to show what failed.
- In a multi-container pod, pass `container=<name>` to pick one (otherwise the API errors).
- Don't set `follow=true` — like the Docker case it streams without end. `select` is a no-op here too; the body is plain text.

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

## Mutations: verify, recover, clean up

Most of this skill is about reading; the same care applies when you *change* things. The Portainer mutation tools have a few habits worth planning around.

**Success is usually silent — verify out-of-band.** Create/update/delete tools tend to return nothing useful: `CreateKubernetesNamespace` returns an empty body, and `StackCreateKubernetesFile` / `StackDelete` return `{"Output":""}` on success. An empty response is *not* proof of success — it's indistinguishable from a swallowed error. After any mutation, confirm with a read: list the resource or fetch it and check its status. Don't report "done" off a non-error alone.

**For a deployed K8s app, the strongest verification is a request through the
service proxy.** A pod in `Running` phase proves the container started, not
that the app answers. The Kubernetes API's service proxy lets you hit an
endpoint inside the cluster with no external network reach:

```
kubernetes_proxy(environment_id=N,
                 path="/api/v1/namespaces/{ns}/services/{name}:{port}/proxy/{route}")
```

Aim that request at a **representative functional endpoint** — one that
exercises the app's real logic — not at `/healthz`. A health route is the
thing most likely to keep returning `200` while the app is broken: it usually
doesn't touch the code path that failed, so a green health check tells you the
process is listening, not that the release works (a health-only check will
happily declare a fully-broken deploy green). A good response from a real
endpoint is strong end-to-end evidence; a health check alone is necessary, not
sufficient. The response is whatever the app returns — often non-JSON, so
`select` may not apply (see *Non-JSON endpoints*).

**A mutation call with no body fields can be rejected with `400 … EOF`.** If
every body field of a write tool is optional and you supply none, the server
receives *no request body at all*, and some Portainer handlers refuse to decode
that — the signature is `Invalid request payload` with details `EOF`. The
canonical case is `StackGitRedeploy`, where a bare redeploy ("pull the
configured ref and re-apply") conceptually needs no arguments: pass one
harmless body field to make the request well-formed, e.g. `Prune: false`.
Read that error as "send at least one body field", not as a broken tool.

**`StackUpdateGit` changes the Git *settings*, not the running deployment —
and its response looks like it already redeployed.** Repointing a ref or
toggling AutoUpdate with `StackUpdateGit` writes the new Git config and returns
a full, populated `stackResponse` — but it doesn't pull or re-apply on its own.
The returned `GitConfig.ReferenceName` shows your *new* target while
`GitConfig.ConfigHash` and `CurrentDeploymentInfo` still describe the *old*
commit: an internally inconsistent object that reads as a coherent post-state
if you trust it at face value. This is a nastier cousin of the silent-success
trap above — the payload isn't empty, so "no error" plus "populated body"
tempts you to report the upgrade as live when nothing rolled out. To actually
apply the change, follow with `StackGitRedeploy`, then verify out-of-band that
`CurrentDeploymentInfo` (and `ConfigHash`) advanced to the new ref — don't read
the running state off the `StackUpdateGit` response.

**`StackCreateKubernetesGit` — the inline `Repository*` fields work but are
the legacy path.** Most of its schema (`RepositoryURL`,
`RepositoryReferenceName`, `RepositoryAuthentication`, …) is marked
*Deprecated: use SourceID instead*. Both paths function: passing an inline
`RepositoryURL` auto-creates a git Source and the stack references it via
`SourceID`. Prefer `SourceID` when a Source for that repo already exists
(list via the GitOps tools); use the inline fields when deploying from a repo
Portainer hasn't seen — that's what they're for, deprecated or not. Don't send
both: when `SourceID` is set, the inline URL and authentication fields are
ignored.

**When a name-based call misbehaves, fall back to id.** Some `…ByName` convenience tools need parameters you can't supply through the MCP and will reject the call. The reliable pattern is `list → act-by-id`: resolve the object first (`StackList` with a `{id,name,type}` projection), then act on the numeric id (`StackDelete`). Reach for this whenever a name-based mutation errors on something you can't satisfy.

**Deletions can leave orphans — check the reclaim policy.** Deleting a PVC (or a stack that owns one) does not necessarily reclaim its storage: under a `Retain` storage class the underlying PersistentVolume is left behind in `Released` state, still holding disk. After deleting volume-backed resources, list `/persistentvolumes` (via `kubernetes_proxy`, for accurate field names) and delete the released PV with `DeleteKubernetesPersistentVolumes` if you meant to free the space — but only the one you intend, never a `Bound` PV another workload is using.

**Restart by scaling, not by deleting the pod.** There's no first-class scale tool, so to stop or restart a workload, merge-PATCH its replica count through the proxy:

```
kubernetes_proxy(environment_id=N, method="PATCH",
                 path="/apis/apps/v1/namespaces/{ns}/deployments/{name}",
                 headers={"Content-Type": "application/merge-patch+json"},
                 body='{"spec":{"replicas":0}}')
```

Scale to `0`, confirm the pod is gone, then back to the target count. For any workload that must not run two copies at once — anything writing to a single shared volume, such as a game server, a database, or any app with non-shared backing storage — this scale-to-zero-then-up cycle is the *only* safe restart. Never raise replicas above one to "roll" such a workload: two pods writing the same volume can corrupt it.

## Tool selection cheatsheet

- Environment-level summary (counts, status, reachability) → `EndpointList` with snapshot projection, or `EndpointSummaryCounts`/`dockerDashboard` if the question is purely aggregate.
- Docker things on a specific environment → `docker_proxy`. The OpenAPI-generated `dockerContainerGpusInspect`, `containerImageStatus`, etc. are specific helpers; use them when they directly answer the question, otherwise the proxy is more flexible.
- Kubernetes things on a specific environment → either the OpenAPI-generated `GetAllKubernetes*` / `GetKubernetes*` tools (Portainer-aware, often already filtered) or `kubernetes_proxy` (raw K8s API, full flexibility). Prefer the typed tool when it exists; fall back to the proxy for paths Portainer doesn't surface natively.
- Helm releases → `HelmList`, `HelmGet`, `HelmGetHistory`. Don't try to route Helm through the K8s proxy — Portainer's Helm tools see the release metadata the K8s API alone doesn't.
- Mutations (POST/PUT/DELETE) → only in read-write mode. If the server is in `PORTAINER_READ_ONLY=1`, non-GET calls are rejected at the tool with a clear error. Don't retry mutations as GET when this happens — surface the read-only state to the user. See *Mutations: verify, recover, clean up* above for verifying success and cleaning up after writes.

## When the skill itself is wrong

If reality contradicts this skill — a `select` example fails to parse, redaction behaves
differently than described, a documented tool is missing or renamed, the truncation hint
doesn't match what's written here — offer to file an issue on
[`portainer/portainer-mcp`](https://github.com/portainer/portainer-mcp), the repo this
skill ships from, so the gap gets fixed for everyone. Server misbehaviour (a tool errors
on valid input, the cap or redaction is broken) belongs there too. That repo is the *only*
destination this section covers: user errors, instance misconfiguration, and Portainer
product bugs are out of scope — handle them in conversation, don't offer to file them
anywhere.

First make sure the evidence points at the skill rather than at your own expression:
re-run the *verbatim* example from this file, not your adaptation of it. A `null`
projection usually means an unquoted dotted key in your own expression (see the JMESPath
notes above), not a skill gap. Only a verbatim example failing, or behaviour that
contradicts an explicit claim in this file, is reportable.

1. **Offer once per session.** One line — "this looks like a gap in the
   portainer-mcp-hygiene skill; want me to file an issue on `portainer/portainer-mcp`?"
   If declined, drop it for the rest of the session. If more mismatches surface later,
   fold them into the one offer (and one issue) rather than asking per finding.
2. **Draft and scrub.** Replace hostnames/IPs/URLs, usernames, and resource names with
   placeholders (`<portainer-host>`, `<stack-name>`); never include tokens, env values,
   or response dumps. Quote the failing line, not the whole response — short bodies also
   survive the prefilled-link fallback below.
3. **Include**: the skill version from the footer of this file; the Portainer version
   (`systemVersion` is one cheap call) and the `mcp-portainer` server version if the user
   knows it; a title prefixed `[portainer-mcp-hygiene]` for skill-guidance gaps (plain
   titles for server bugs); what the skill said (quote it); the tool call made (tool name
   + sanitized arguments including the `select` expression); what actually happened; and
   what you expected.
4. **Show the draft, then file on approval** — `gh issue create --repo
   portainer/portainer-mcp --title … --body …` if `gh` is available and authenticated;
   otherwise hand the user a prefilled link they can open themselves:
   `https://github.com/portainer/portainer-mcp/issues/new?title=<url-encoded>&body=<url-encoded>`.
   Never file silently.

---

Skill version: 2.43.1 (matches the `mcp-portainer` release tag this file shipped with).
