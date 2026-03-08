# Molecule Lifecycle Paradigm: Three Example Use Cases

Paradigm C handles staleness through **re-instantiation**, not mutation.
Completed work is immutable. When upstream changes, you distill the completed
molecule back into a proto and pour a fresh instance. Version history emerges
as a chain of digests.

**Lifecycle:** Proto → Pour → Execute → Squash → (upstream changes) → Distill → Re-Pour

---

## Example 1: Security Audit Pipeline

A recurring security audit of a deployed service. Each audit cycle produces
an immutable report; when the service changes, a new audit cycle is poured.

### Proto Definition

```toml
# mol-security-audit.formula.toml
description = """
Security audit pipeline for a deployed service.
Scans for vulnerabilities, reviews configurations, tests auth flows,
and produces a consolidated report with risk scores.
"""

[variables]
service = { description = "Service to audit", required = true }
environment = { description = "Target environment", default = "production" }
previous_digest = { description = "Prior audit digest for delta comparison", default = "" }

[[steps]]
id = "dependency-scan"
title = "Scan dependencies for known CVEs"
description = """
Run dependency scanner against {{service}} in {{environment}}.
Output: list of CVEs with severity scores.
"""

[[steps]]
id = "config-review"
title = "Review runtime configuration"
description = """
Audit TLS settings, CORS policy, header security, secrets rotation.
Flag any deviations from the security baseline.
"""

[[steps]]
id = "auth-pentest"
title = "Test authentication and authorization flows"
description = """
Verify token expiry, privilege escalation vectors, session fixation.
Requires: dependency-scan (CVE context informs attack surface).
"""
deps = ["dependency-scan"]

[[steps]]
id = "network-exposure"
title = "Map network exposure and attack surface"
description = """
Port scan, DNS enumeration, certificate chain validation.
Requires: config-review (to know expected exposure).
"""
deps = ["config-review"]

[[steps]]
id = "risk-matrix"
title = "Build consolidated risk matrix"
description = """
Combine findings from all prior cells into a scored risk matrix.
Compare against {{previous_digest}} if available.
"""
deps = ["auth-pentest", "network-exposure"]

[[steps]]
id = "remediation-plan"
title = "Produce prioritized remediation plan"
description = """
For each HIGH/CRITICAL finding, propose fix with effort estimate.
Final deliverable of the audit cycle.
"""
deps = ["risk-matrix"]
```

**DAG structure:**
```
dependency-scan ──→ auth-pentest ──┐
                                   ├──→ risk-matrix ──→ remediation-plan
config-review ────→ network-exposure┘
```

### First Run: Pour → Execute → Squash

```bash
# 1. Pour: instantiate the audit for payments-api
bd mol pour mol-security-audit --var service=payments-api --var environment=production
# Creates molecule gt-mol-a1 with 6 open beads

# 2. Execute: work cells in dependency order
bd update gt-mol-a1-dependency-scan --claim
# ... run scanner, record CVEs ...
bd close gt-mol-a1-dependency-scan --reason "Found 3 HIGH, 7 MEDIUM CVEs"

bd update gt-mol-a1-config-review --claim
# ... audit config ...
bd close gt-mol-a1-config-review --reason "TLS 1.2 still enabled, HSTS missing on /api"

# auth-pentest and network-exposure are now unblocked (parallel)
bd update gt-mol-a1-auth-pentest --claim
bd close gt-mol-a1-auth-pentest --reason "Session fixation vector in /oauth/callback"

bd update gt-mol-a1-network-exposure --claim
bd close gt-mol-a1-network-exposure --reason "Port 9090 debug endpoint exposed"

# risk-matrix unblocked
bd update gt-mol-a1-risk-matrix --claim
bd close gt-mol-a1-risk-matrix --reason "4 CRITICAL, 3 HIGH findings. Score: 7.2/10"

bd update gt-mol-a1-remediation-plan --claim
bd close gt-mol-a1-remediation-plan --reason "6 remediations filed, ETA 2 sprints"

# 3. Squash: compress to immutable digest
bd mol squash gt-mol-a1
# Digest: gt-digest-a1 (immutable record of Q1 audit)
```

### Upstream Changes: Distill → Re-Pour

The payments-api deploys a major update. Time for a new audit cycle.

```bash
# 4. Distill: extract reusable proto from completed audit
bd mol distill gt-mol-a1 mol-security-audit
# Proto updated with any refinements learned during execution

# 5. Re-pour: fresh instance referencing the prior digest
bd mol pour mol-security-audit \
  --var service=payments-api \
  --var environment=production \
  --var previous_digest=gt-digest-a1
# Creates molecule gt-mol-a2 — all cells open, fresh execution

# The risk-matrix cell can now delta against gt-digest-a1:
# "3 of 4 CRITICAL findings from Q1 are resolved. 1 new HIGH finding."
```

### Version History

```
gt-digest-a1 (Q1 audit) → gt-digest-a2 (post-deploy audit) → gt-digest-a3 (Q2 audit) → ...
```

Each digest is immutable. The chain shows how the security posture evolved.
Auditors can diff any two digests to see what changed.

---

## Example 2: Data Pipeline Validation

A multi-stage data pipeline that ingests, transforms, validates, and publishes
a dataset. When source data changes, the entire pipeline re-executes as a
new molecule — no partial re-runs, no stale intermediate state.

### Proto Definition

```toml
# mol-data-pipeline.formula.toml
description = """
End-to-end data pipeline: ingest raw data, apply transformations,
run quality checks, generate statistics, and publish to the warehouse.
"""

[variables]
dataset = { description = "Dataset identifier", required = true }
source_version = { description = "Source data version/hash", required = true }
schema_version = { description = "Target schema version", default = "v3" }

[[steps]]
id = "ingest"
title = "Ingest raw data from source"
description = """
Pull {{dataset}} at version {{source_version}}.
Validate file integrity (checksums, row counts).
"""

[[steps]]
id = "schema-migrate"
title = "Apply schema migrations"
description = """
Transform ingested data to match {{schema_version}}.
Handle column renames, type coercions, null policies.
"""
deps = ["ingest"]

[[steps]]
id = "dedup"
title = "Deduplicate records"
description = """
Remove exact and fuzzy duplicates.
Log dedup statistics (count removed, merge strategy).
"""
deps = ["ingest"]

[[steps]]
id = "quality-gate"
title = "Run data quality checks"
description = """
Validate constraints: nullability, referential integrity, range checks.
Requires both schema-migrated and deduped data.
Fail the pipeline if critical quality rules are violated.
"""
deps = ["schema-migrate", "dedup"]

[[steps]]
id = "statistics"
title = "Compute aggregate statistics"
description = """
Row counts, distributions, anomaly detection.
Compare against previous digest for drift detection.
"""
deps = ["quality-gate"]

[[steps]]
id = "publish"
title = "Publish to data warehouse"
description = """
Write validated dataset to warehouse.
Update catalog metadata and lineage graph.
"""
deps = ["quality-gate"]

[[steps]]
id = "notify"
title = "Send completion notification"
description = """
Alert downstream consumers that fresh data is available.
Include statistics summary and quality report.
"""
deps = ["statistics", "publish"]
```

**DAG structure:**
```
            ┌──→ schema-migrate ──┐
ingest ─────┤                     ├──→ quality-gate ──┬──→ statistics ──┐
            └──→ dedup ───────────┘                   │                 ├──→ notify
                                                      └──→ publish ─────┘
```

### First Run: Pour → Execute → Squash

```bash
# Pour for the March customer dataset
bd mol pour mol-data-pipeline \
  --var dataset=customers \
  --var source_version=2026-03-01-abc123 \
  --var schema_version=v3
# Creates gt-mol-b1 with 7 open beads

# Execute in dependency order
bd update gt-mol-b1-ingest --claim
bd close gt-mol-b1-ingest --reason "1.2M rows ingested, checksums valid"

# schema-migrate and dedup run in parallel
bd update gt-mol-b1-schema-migrate --claim
bd close gt-mol-b1-schema-migrate --reason "Migrated to v3: 3 columns renamed, 1 type coercion"

bd update gt-mol-b1-dedup --claim
bd close gt-mol-b1-dedup --reason "847 exact dupes removed, 23 fuzzy merges"

# quality-gate now unblocked
bd update gt-mol-b1-quality-gate --claim
bd close gt-mol-b1-quality-gate --reason "All 14 quality rules passed"

# statistics and publish in parallel
bd update gt-mol-b1-statistics --claim
bd close gt-mol-b1-statistics --reason "Mean order value up 12%, no anomalies"

bd update gt-mol-b1-publish --claim
bd close gt-mol-b1-publish --reason "Published to warehouse table customers_v3"

bd update gt-mol-b1-notify --claim
bd close gt-mol-b1-notify --reason "Notified 3 downstream pipelines"

# Squash
bd mol squash gt-mol-b1
# Digest: gt-digest-b1
```

### Upstream Changes: Distill → Re-Pour

Source data gets a correction — 500 rows updated, 30 new rows added.

```bash
# Distill completed pipeline back to proto
bd mol distill gt-mol-b1 mol-data-pipeline

# Re-pour with new source version
bd mol pour mol-data-pipeline \
  --var dataset=customers \
  --var source_version=2026-03-05-def456 \
  --var schema_version=v3
# Creates gt-mol-b2 — entirely fresh execution

# The statistics cell detects drift against gt-digest-b1:
# "530 rows changed vs prior run. Mean order value stable."
```

### Version History

```
gt-digest-b1 (March 1 load) → gt-digest-b2 (March 5 correction) → gt-digest-b3 (April 1 load) → ...
```

Each digest captures the exact transformation applied to a specific source
version. Lineage is traceable: "which source version produced the data in
the warehouse on March 5?" → check the digest chain.

---

## Example 3: Infrastructure Provisioning Cycle

Provisioning a multi-tier application environment. Each provisioning cycle
is a molecule; infrastructure changes trigger a new cycle rather than
mutating the existing state.

### Proto Definition

```toml
# mol-infra-provision.formula.toml
description = """
Provision a multi-tier application environment: networking, compute,
storage, application deployment, and health verification.
"""

[variables]
env_name = { description = "Environment name", required = true }
region = { description = "Cloud region", default = "us-east-1" }
app_version = { description = "Application version to deploy", required = true }
tier = { description = "Environment tier (dev/staging/prod)", default = "staging" }

[[steps]]
id = "network"
title = "Provision network layer"
description = """
Create VPC, subnets, security groups, load balancer.
Tag all resources with env_name={{env_name}}, tier={{tier}}.
"""

[[steps]]
id = "storage"
title = "Provision storage layer"
description = """
Create database instances, object storage buckets, cache clusters.
Apply encryption and backup policies for {{tier}}.
"""

[[steps]]
id = "compute"
title = "Provision compute layer"
description = """
Launch instances/containers in the network from step 1.
Attach storage volumes from step 2.
"""
deps = ["network", "storage"]

[[steps]]
id = "deploy"
title = "Deploy application"
description = """
Deploy {{app_version}} to the compute layer.
Run database migrations, seed initial data.
"""
deps = ["compute"]

[[steps]]
id = "smoke-test"
title = "Run smoke tests"
description = """
Verify the deployed application is healthy.
Test critical paths: auth, API, database connectivity.
"""
deps = ["deploy"]

[[steps]]
id = "dns-cutover"
title = "Update DNS and routing"
description = """
Point {{env_name}}.example.com to the new load balancer.
Only if smoke tests pass. For prod, use weighted routing for canary.
"""
deps = ["smoke-test"]
```

**DAG structure:**
```
network ──────┐
              ├──→ compute ──→ deploy ──→ smoke-test ──→ dns-cutover
storage ──────┘
```

### First Run: Pour → Execute → Squash

```bash
# Pour: provision staging for app v2.4.0
bd mol pour mol-infra-provision \
  --var env_name=staging-blue \
  --var region=us-east-1 \
  --var app_version=v2.4.0 \
  --var tier=staging
# Creates gt-mol-c1 with 6 open beads

# Execute
bd update gt-mol-c1-network --claim
bd close gt-mol-c1-network --reason "VPC vpc-abc123, ALB alb-xyz789 created"

bd update gt-mol-c1-storage --claim
bd close gt-mol-c1-storage --reason "RDS pg-staging-blue, S3 bucket, Redis cluster created"

# compute unblocked
bd update gt-mol-c1-compute --claim
bd close gt-mol-c1-compute --reason "3x c5.xlarge in ASG, EBS volumes attached"

bd update gt-mol-c1-deploy --claim
bd close gt-mol-c1-deploy --reason "v2.4.0 deployed, migrations complete, 12 tables"

bd update gt-mol-c1-smoke-test --claim
bd close gt-mol-c1-smoke-test --reason "All 8 smoke tests passed (auth, api, db, cache)"

bd update gt-mol-c1-dns-cutover --claim
bd close gt-mol-c1-dns-cutover --reason "staging-blue.example.com → alb-xyz789"

# Squash
bd mol squash gt-mol-c1
# Digest: gt-digest-c1 (staging v2.4.0 environment snapshot)
```

### Upstream Changes: Distill → Re-Pour

Application v2.5.0 is ready. Instead of mutating the running environment,
provision a fresh parallel environment (blue-green pattern):

```bash
# Distill the completed provisioning cycle
bd mol distill gt-mol-c1 mol-infra-provision

# Re-pour with new app version (green deployment)
bd mol pour mol-infra-provision \
  --var env_name=staging-green \
  --var region=us-east-1 \
  --var app_version=v2.5.0 \
  --var tier=staging
# Creates gt-mol-c2 — entirely fresh provisioning cycle

# Old environment (staging-blue, gt-digest-c1) remains running
# until staging-green is verified, then DNS cuts over
# staging-blue can be decommissioned (another molecule for that)
```

### Version History

```
gt-digest-c1 (v2.4.0 staging-blue) → gt-digest-c2 (v2.5.0 staging-green) → gt-digest-c3 (v2.5.1 hotfix) → ...
```

Each digest captures the complete infrastructure state for a specific app
version. Rollback = re-pour from an older digest's proto. The chain provides
a complete audit trail of every environment that was ever provisioned.

---

## Summary: Why Re-Instantiation Beats Mutation

| Concern | Mutation (Paradigms A/B) | Re-Instantiation (Paradigm C) |
|---------|--------------------------|-------------------------------|
| Staleness | Reopen/relabel closed beads | Pour a fresh molecule |
| History | Mutable state, hard to audit | Chain of immutable digests |
| Concurrency | Lock contention on shared beads | Each cycle is independent |
| Rollback | Undo mutations (error-prone) | Re-pour from prior proto |
| Debugging | "What state was this bead in at time T?" | Check the digest for that cycle |

The molecule lifecycle makes every execution cycle a first-class, immutable
artifact. Staleness propagation IS molecule re-instantiation — the most
natural expression of "do the work again with fresh inputs."
