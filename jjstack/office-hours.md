# Forge - Office Hours Design Session

**Date:** 2026-04-21
**Format:** YC-style forcing questions + premise challenge + strategic alternatives

---

## Phase 1: Q&A Summary

**Q1 - Demand:** The builder uses it himself. Pain is not cost but friction: every new project requires
re-learning and re-wiring 4-6 disconnected tools that cannot communicate with each other.
No existing user is paying, but the builder's own repeated pain is the validated demand signal.

**Q2 - Status Quo:** Users write shell scripts to automate the gaps. Or they pick a bundled
platform (Coolify, Render, Railway) and accept its lock-in. The shell-script crowd still has to
understand every underlying tool. The platform crowd surrenders control.

**Q3 - Target User (refined):** Originally identified as "solo dev on a cheap VPS." User corrected
this to two distinct segments after reviewing HearthForge's isolation model:
- Solo devs running personal/side projects on a single VPS
- Small companies where source code must stay confidential: developers work inside isolated
  containers and never touch the host or each other's workspaces

**MCP Server Insight (user-generated):** A Forge MCP server would let an AI agent (e.g. Claude)
issue natural-language deploy commands. After building a site, say "deploy to vps-1 with
healthchecks and CI/CD" and Forge handles the full pipeline: mesh routing, blue-green deploy,
WatchForge monitor, SparkForge notification. AI-native infrastructure management.

---

## Phase 2: Premises

| # | Premise | Source |
|---|---------|--------|
| P1 | The pain is tool fragmentation, not cost | Q1 answers |
| P2 | Solo devs are the primary beachhead | Q3 initial answer |
| P3 | Small teams with IP isolation needs are a second segment | User correction after HearthForge review |
| P4 | Competitors (Coolify/CapRover/Dokploy) are deployment platforms, not infrastructure suites | Q2 + research |
| P5 | Integration across modules IS the product (modules know about each other) | SRS + use cases |
| P6 | A Forge MCP server is a viable differentiator for AI-native workflows | User insight |
| P7 | The suite must be installable module-by-module (no big-bang install) | Vision doc G-03 |

---

## Phase 3: Premise Challenge

**P1 - "Pain is tool fragmentation, not cost"**
HOLD. This is accurate but incomplete. The deeper pain is *cognitive overhead*: not just that
tools are separate, but that each new project forces the developer to context-switch into
infrastructure thinking when they want to be building product. Fragmentation is the symptom.
Cognitive drain is the actual pain. This distinction matters for positioning.

**P2 - "Solo devs are the primary beachhead"**
CHALLENGE. Solo devs have low willingness to invest setup time in a new tool unless the
first-run experience is near-zero friction. The 150-requirement scope (all pending) risks
producing a complex tool that requires 2 hours to understand before the first deploy. The
beachhead may be better defined as: "developers who already self-host and are already
maintaining shell scripts." They have proven pain tolerance and will adopt a structured
alternative. Pure newcomers will not.

**P3 - "HearthForge opens the small-company segment"**
HOLD, BUT TIME-BOXED. HearthForge (FR-065 to FR-084, 20 requirements) is the largest
module. It requires wildcard TLS DNS setup, SSH gateway config, and per-developer container
provisioning. Small companies evaluating this will ask: "who maintains this when my
infrastructure person leaves?" The segment is real but arrives after the solo-dev beachhead
proves the deployment core. Do not build HearthForge first.

**P4 - "Competitors are deployment platforms, not infrastructure suites"**
HOLD. Coolify is the relevant comparison point (500-700MB idle RAM, Docker-based, web UI).
Forge's CLI-first approach and mesh networking are genuinely different. But Coolify has a
web UI, marketplace, and a large install base. Forge needs a story for the "why not just use
Coolify?" question: the answer is mesh networking + dev workspaces + security scanning +
no web UI attack surface + single static binary. This is a credible answer for a specific
buyer but will not win on features alone.

**P5 - "Integration across modules IS the product"**
HOLD, CRITICAL. The use-case relationships document this precisely: UC-10 (deploy) includes
WatchForge pause/resume, SparkForge notification, AND PenForge post-deploy scan. No
single-purpose tool delivers this chain. This is the strongest moat and should be the
primary marketing message. "Every deploy is automatically monitored, notified, and scanned."

**P6 - "MCP server is a differentiator"**
STRONG HOLD. FR-012 (JSON output mode for all commands) is already in the requirements.
The architecture is already designed for machine consumption. An MCP server is 200-400 lines
of Go wrapping existing CLI commands. No other self-hosted infrastructure tool has this.
The timing is right: AI-assisted development is becoming mainstream. This should be treated
as a first-class feature, not an add-on.

**P7 - "Module-by-module installation"**
HOLD. FR-001/FR-002 (install/uninstall module) + FR-011 (dynamic module discovery) support
this. The "forge install <module>" UX is a direct answer to the fragmentation pain: instead
of "install 6 separate tools," it's "install forge, then add what you need." This is a
compelling on-ramp story.

---

## Phase 3.5: Second Opinion

*Simulated adversarial review from an infrastructure-skeptic VC perspective:*

**Concern 1: Scope creep risk is extreme.**
150 requirements at status "Pending" across 7 modules means zero is shipped. A solo builder
completing even 50% of Must requirements is 63 features. At 1 feature/week, that's 15 months
before an MVP. The answer to this is a strict MVP cut: which 3 modules deliver the core loop
that makes everything else worth building?

**Concern 2: "Better Coolify" is not a business.**
Coolify is free and open-source. Forge being also free and also open-source means the
competition is "community size + ecosystem + marketing" not just features. Forge needs either
a monetization path or a reason why community gravity will flow to it.

**Concern 3: HearthForge is a separate product.**
The SSH dev workspace module serves a different buyer (a small company's CTO, not a solo dev)
via a different distribution channel (enterprise eval, not "just install it on your VPS").
Bundling it forces the solo-dev story to carry enterprise complexity from day one.

**Response to concerns:**
- Concern 1: Answered by MVP sequencing (see Phase 4)
- Concern 2: Forge is a personal infrastructure tool first; monetization (Forge Cloud, managed
  hosted controller) is a future-state problem. The OSS path is intentional: build community,
  then offer hosted tier.
- Concern 3: HearthForge should ship last, not first. The architecture supports this
  (module-by-module install). Position it as "when your team grows, add HearthForge."

---

## Phase 4: Strategic Alternatives

### Alternative A: CLI-first deployment core (Recommended)

**Sequence:** Core -> SmeltForge -> WatchForge -> SparkForge -> FluxForge -> HearthForge -> PenForge

**Rationale:** SmeltForge is the highest-value first module: it delivers the deploy story
(UC-09 to UC-14) with blue-green deploys, Caddy TLS, webhook CI/CD, rollback, and env vars.
WatchForge + SparkForge complete the "deploy-monitor-alert" loop that is the core product
promise. This 3-module combination (Core + SmeltForge + WatchForge + SparkForge) is the MVP.
FluxForge adds the mesh story. HearthForge and PenForge come last.

**MVP scope (Must requirements only, first 3 modules):**
- Core: FR-001 to FR-008, FR-011 (9 of 12 Must)
- SmeltForge: FR-024 to FR-041 (14 Must)
- WatchForge: FR-042 to FR-053 (12 Must)
- SparkForge: FR-055 to FR-059, FR-062 to FR-064 (10 Must)
Total: ~45 requirements for a shippable MVP with a complete story.

**Risk:** No mesh networking in MVP. Multi-VPS users must wait for FluxForge.

### Alternative B: Mesh-first networking layer

**Sequence:** Core -> FluxForge -> SmeltForge -> ...

**Rationale:** If the mesh is the moat (no competitor has WireGuard mesh), lead with it.
FluxForge (FR-013 to FR-023, 11 requirements, 9 Must) is actually the smallest module after
Core. Ship Core + FluxForge first as "self-hosted Tailscale replacement," build community on
that positioning, then layer SmeltForge on top.

**Risk:** FluxForge alone is not a complete product. A mesh without deployment on top of it
is a tool for sysadmins, not developers. Longer time to "I deployed my first app."

### Alternative C: MCP-first AI-native platform

**Sequence:** Core -> SmeltForge -> MCP Server -> FluxForge -> WatchForge -> SparkForge

**Rationale:** Lead with the differentiator. No infrastructure tool has an MCP server. Ship
Core + SmeltForge + a Forge MCP server as "the first self-hosted infrastructure you can
control from your AI assistant." Target the vibe-coding / AI-first developer persona directly.
The MCP server wraps existing CLI commands (FR-012 JSON output makes this trivial).

**Risk:** MCP protocol adoption is still maturing. The market may not be large enough yet
for this to be the lead story. Better as a feature launch on top of Alternative A's MVP.

---

## Phase 5: Recommendation

**Go with Alternative A, ship the MCP server alongside SmeltForge as a bonus feature.**

The core product loop is: `forge install smeltforge` -> register project -> deploy -> monitor ->
get notified. This loop is complete in ~45 Must requirements. The MCP server is a 200-400 line
wrapper and should ship with or immediately after SmeltForge because FR-012 (JSON output) is
already planned.

The HearthForge isolation model is a real second segment, but it must not drive v1 scope.
It is a "when you outgrow a single VPS and need to onboard a contractor safely" story, which
comes after the deployment core proves itself.

**Positioning:** "Everything a developer needs to self-host, in one CLI. Deploy. Monitor. Alert.
No dependencies. No cloud account. Yours."

**The MCP angle:** "The first self-hosted infrastructure you can manage from Claude." This is a
headline feature that no competitor can match and costs almost nothing to build given the
existing JSON output design.

---

## Strategic Design Summary

### Product
Forge is a self-hosted infrastructure suite. Seven composable modules installed via a single
CLI. No web UI, no SaaS dependency, no privileged containers. Statically linked Go binary
(+ Rust for PenForge engines). Runs on any Ubuntu 22.04 / Debian 12 VPS.

### Core Differentiators
1. **Integration chain**: Deploy triggers monitor pause, health check, Caddy routing switch,
   monitor resume, SparkForge alert, PenForge scan. One `forge smeltforge deploy` command
   orchestrates all of this automatically.
2. **Mesh networking**: WireGuard mesh (10.forge.0.0/16) across VPS nodes. No Tailscale
   account needed. Secrets synced across the mesh.
3. **AI-native**: JSON output on every command + MCP server = Claude can deploy your app.
4. **Dev workspaces with IP isolation**: HearthForge gives each developer a container with
   no host access. Source code stays inside. Admin controls everything. SSH config snippet
   auto-generated.
5. **Security baked in**: PenForge runs Nuclei, Nmap, testssl.sh, dnsx, Trivy as isolated
   Docker containers against declared scope. Runs post-deploy automatically.

### Target Users
**Beachhead:** Developers who already self-host and maintain shell scripts. They know the
pain, they have a server, they will adopt a structured solution if the first-run experience
is fast.

**Second wave:** Small product companies where contractors must not access source code.
HearthForge solves this. Admin provisions workspace, developer gets SSH shell in container,
never touches host.

**Third wave:** AI-first developers using Claude/Cursor to build projects. Forge MCP server
lets them deploy from their AI assistant without leaving the development context.

### MVP Build Order
1. Forge Core (module install/uninstall/status + secrets + JSON output)
2. SmeltForge (register + deploy + rollback + env vars + webhook + Caddy TLS)
3. WatchForge (HTTP/TCP/Docker/SSL/heartbeat monitors + incident tracking + status page)
4. SparkForge (Gotify + email + webhook channels + priority routing + deduplication)
5. Forge MCP Server (wraps CLI with JSON output, exposes as MCP tools)
6. FluxForge (WireGuard mesh + join tokens + RBAC + DERP relay)
7. HearthForge (SSH gateway + dev containers + preview domains + wildcard TLS)
8. PenForge (Nuclei/Nmap/testssl/dnsx/Trivy engines + finding store + delta detection)

### Architecture Constraints (non-negotiable from SRS)
- No cross-module Go imports. HTTP APIs or shared filesystem only.
- No privileged containers (NFR-010). No Docker socket in containers (NFR-011).
- All secrets age-encrypted (NFR-009). No plaintext ever written to disk.
- 80% test coverage on shared/ and internal/ packages (NFR-028).
- Statically linked binary, no external runtime deps (NFR-041).

### The Question Forge Must Answer in 10 Seconds
"I can install Coolify in 5 minutes and it has a web UI. Why would I use Forge instead?"

Answer: "Coolify is a web app. Forge is infrastructure. No browser, no web UI to attack,
no 500MB daemon to babysit. One binary. Install what you need. Your VPS, your data, your
mesh. And you can control it from Claude."

---

## Quality Score: 9/10

**Strengths:** Full requirement coverage, realistic MVP sequencing, two distinct user segments
clearly separated, architecture constraints preserved, MCP differentiator justified with
implementation cost estimate.

**Gap (-1):** No go-to-market channel identified. The beachhead user ("developer who already
self-hosts") has no obvious distribution channel. Hacker News, personal blogs, and GitHub
are likely vectors, but this was not addressed in the session because the user did not raise
it as a concern.
