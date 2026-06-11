---
name: portainer-mcp-hygiene
description: How to efficiently query the Portainer MCP server's tools and read the responses correctly — when to project responses with `select` (JMESPath), where the heavy fields live (snapshots, status blocks, managed fields), how to handle non-JSON Docker/K8s proxy endpoints (logs, stats, exec), and how to interpret results that are easy to misread (e.g. an edge environment's health comes from its heartbeat, not its `Status` field). Trigger this whenever you're about to call any Portainer MCP tool — including `docker_proxy`, `kubernetes_proxy`, `EndpointList`, `GetAllKubernetes*`, `StackList`, `snapshot*`, `Helm*`, or any other `mcp__portainer__*` tool — and whenever the user asks about Portainer environments, Docker containers/images/stacks/networks managed by Portainer, Kubernetes resources via Portainer, or Helm releases. Use it even if the user doesn't mention Portainer by name, as long as the working answer requires one of these tools.
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

and use the ID whose name matches what the user means. Guessing is a hard
failure, not a silent one: a wrong ID makes the proxy raise an `API request
failed (HTTP 404)` error, so a projection that returns `null` fields means the
data is genuinely absent, not that you hit the wrong environment.

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

`Status` is the right reachability signal only for *direct* agents — for
edge environments it stays `0` regardless of health. See *Interpreting
results, not just shrinking them* below before reading it as up/down.

**Kubernetes via `kubernetes_proxy` — `metadata.managedFields` and `status` are huge.**
`metadata.managedFields` alone is routinely 30-70% of an object. The `status` block on Deployments, StatefulSets, Pods, and Nodes is similarly verbose. Project them out unless the user is asking about reconciliation state or controller history:

```
# Pod summary
kubernetes_proxy(environment_id=N, path="/api/v1/pods", select="items[].{name:metadata.name,ns:metadata.namespace,phase:status.phase,restarts:status.containerStatuses[0].restartCount,node:spec.nodeName}")

# Deployment readiness
kubernetes_proxy(environment_id=N, path="/apis/apps/v1/deployments", select="items[].{name:metadata.name,ns:metadata.namespace,replicas:spec.replicas,ready:status.readyReplicas}")
```

**`GetAllKubernetes*` tools — full status blocks per object.**
The OpenAPI-generated `GetAllKubernetesApplications`, `GetAllKubernetesConfigMaps`, `GetAllKubernetesIngresses`, etc. return arrays where each element carries its full object body. Same rules as the proxy: project to the named fields you need.

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
EdgeAgentOnKubernetes) `Status` is left at its zero value `0` and never tracks
reachability — so reading `Status: 0` as "down" is a false alarm. Portainer
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

Skill version: 2.42.5 (matches the `mcp-portainer` release tag this file shipped with).
