# Academic Precedents for Reactive Dataflow and Effect Systems

> Research survey for Gas City's reactive bead calculus. Each entry maps
> academic work to Gas City concepts with citations.

---

## 1. Reactive Programming & Functional Reactive Programming (FRP)

### 1.1 Conal Elliott — Functional Reactive Animation (1997)

**Paper**: Elliott, C. & Hudak, P. "Functional Reactive Animation." ICFP 1997.

**Core idea**: FRP introduces two abstractions — *Behaviors* (time-varying values)
and *Events* (discrete occurrences) — composed with standard functional combinators.
Time is continuous and denotational, not discrete steps.

**State change propagation**: Pull-based: behaviors are sampled at render time.
No explicit propagation — the runtime evaluates the behavior function at each
time step, and dependencies are implicit in function composition.

**Effect/cost tracking**: None — pure FRP is side-effect-free by construction.
Cost is implicit in the depth of the behavior expression tree.

**Gas City mapping**: Gas City cells are analogous to FRP Behaviors — each cell
holds a time-varying value (its latest computation result). The `{{ref}}` wire
syntax is the analog of FRP's applicative lifting (`<*>`). The key difference:
Gas City is push-based (staleness propagates forward) while classic FRP is
pull-based (values are demanded).

**What Gas City could adopt**: FRP's denotational semantics provide a clean
specification language. The continuous-time model could inform a "logical time"
abstraction where cell versions replace wall-clock time, enabling replay and
debugging via time-travel.

### 1.2 Adaptive Functional Programming / Self-Adjusting Computation

**Paper**: Acar, U.A. "Self-Adjusting Computation." PhD thesis, CMU, 2005.
Also: Acar, U.A., Blelloch, G.E., Harper, R. "Adaptive Functional Programming."
POPL 2002.

**Core idea**: Programs automatically recompute only the parts affected by input
changes. A runtime maintains a dynamic dependency graph and uses *change
propagation* to incrementally update outputs.

**State change propagation**: Push-based with memoization. When an input changes,
the system walks the dependency graph forward, re-executing only nodes whose
inputs actually changed. Unchanged sub-computations are reused via memoization.

**Effect/cost tracking**: The key metric is *stability* — how much of the
computation graph needs to be re-executed for a given input change. Optimal
self-adjusting programs have update cost proportional to the size of the change,
not the size of the computation.

**Gas City mapping**: This is the closest academic analog to Gas City's staleness
propagation. When an upstream cell changes, staleness propagates through the DAG
exactly like change propagation in self-adjusting computation. The binary
`fresh`/`stale` flag is a coarse version of Acar's change propagation predicate.

**What Gas City could adopt**:
- *Granularity control*: Self-adjusting computation distinguishes between
  "the input changed" and "the input changed enough to affect the output."
  Gas City's v1.5 `estimatedDrift` magnitude captures this intuition.
- *Trace-based memoization*: Cache cell outputs keyed on input hashes. If the
  same inputs recur, skip recomputation entirely.

### 1.3 Adapton — Demand-Driven Incremental Computation

**Paper**: Hammer, M.A., Phang, K.Y., Hicks, M., Foster, J.S. "Adapton:
Composable, Demand-Driven Incremental Computation." PLDI 2014.

**Core idea**: Combines push-based change propagation with pull-based (lazy)
evaluation. Dirty flags propagate eagerly (push), but actual recomputation
happens lazily (pull) only when a value is demanded.

**State change propagation**: Two-phase: (1) *Dirtying* propagates forward through
the dependency graph (push); (2) *Cleaning* propagates backward from demanded
nodes (pull), re-executing only what's needed.

**Effect/cost tracking**: Amortized analysis shows that Adapton's total work is
bounded by the minimum of eager and lazy incremental strategies.

**Gas City mapping**: Gas City's staleness propagation is precisely Adapton's
dirtying phase. The missing piece is Adapton's cleaning phase — Gas City
currently recomputes all stale cells, not just demanded ones. For large DAGs
where only some outputs are actively used, demand-driven cleaning could save
significant tokens.

**What Gas City could adopt**: The two-phase dirty/clean protocol. Mark cells
stale eagerly (already done), but defer recomputation until output is demanded.
This enables "lazy sheets" where unused branches of the DAG don't consume tokens.

### 1.4 Differential Dataflow (Naiad / Timely Dataflow)

**Paper**: McSherry, D., Murray, D.G., Isaacs, R., Isard, M. "Differential
Dataflow." CIDR 2013. Also: Murray, D.G. et al. "Naiad: A Timely Dataflow
System." SOSP 2013.

**Core idea**: Represent computation over changing data as differences (deltas)
rather than full recomputation. Uses *partially ordered timestamps* (lattice
of logical times) to track when changes occur in nested loops.

**State change propagation**: Operators receive and produce *differences* —
`(data, time, diff)` triples. The `diff` field (+1 or -1) indicates additions
and retractions. Operators maintain state indexed by time and produce outputs
as new differences arrive.

**Effect/cost tracking**: Cost is proportional to the size of the *change*, not
the dataset. The lattice timestamp structure enables precise tracking of which
changes have been fully processed (the "frontier" moves forward monotonically).

**Gas City mapping**: Differential dataflow's partially ordered timestamps map
to Gas City's DAG structure — both use partial orders rather than total orders
for time. The `frontier` concept (all times below which are fully processed)
is analogous to the set of non-stale cells. The key difference: differential
dataflow operates on collections of records with efficient delta propagation,
while Gas City operates on opaque LLM outputs where "delta" is not well-defined.

**What Gas City could adopt**:
- *Frontier-based progress tracking*: Instead of binary staleness, track the
  "frontier" — the set of cell versions that are guaranteed settled. This
  enables fine-grained progress reporting.
- *Lattice timestamps for nested molecules*: When molecules nest, use
  product-lattice timestamps `(outer_step, inner_step)` to precisely track
  which sub-computations are current.

---

## 2. Spreadsheet Computation Models

### 2.1 Haskell Spreadsheet Models

**Paper**: Peyton Jones, S., Blackwell, A., Burnett, M. "A User-Centred
Approach to Functions in Excel." ICFP 2003. Also see the "Forms/3" visual
language: Burnett, M. et al. "Forms/3: A First-Order Visual Language to
Explore the Boundaries of the Spreadsheet Paradigm." JFP 2001.

**Core idea**: Spreadsheets are implicitly functional reactive programs.
Each cell is a pure function of its inputs. The spreadsheet runtime maintains
the dependency DAG and propagates changes automatically.

**State change propagation**: Standard topological-sort evaluation. When a cell
changes, all downstream dependents are recalculated in dependency order.

**Effect/cost tracking**: None in traditional spreadsheets. Cost is implicit
in cell count and formula complexity. Excel's "calculation chain" is the
evaluation schedule.

**Gas City mapping**: Gas City IS a spreadsheet — this is the closest metaphor.
The `{{ref}}` syntax is the analog of cell references (`=A1+B2`). The DAG
topology, topological-sort scheduling, and change propagation are identical.
Gas City extends the spreadsheet model by: (1) making cells LLM-powered
(lossy codecs instead of exact functions); (2) adding an explicit effect algebra
for cost/quality; (3) using typed wires instead of untyped cell references.

**What Gas City could adopt**: Spreadsheet UI paradigms — cell formatting,
formula bar, named ranges. The "What-If Analysis" / Goal Seek feature maps to
Gas City's ability to ask "what token budget produces acceptable quality?"

### 2.2 Reactive Spreadsheets in PL Theory (Incremental View Maintenance)

**Paper**: Koch, C. "Incremental Query Evaluation in a Ring of Databases."
PODS 2010. Also: Koch, C. et al. "DBToaster: Higher-order Delta Processing
for Dynamic, Frequently Fresh Views." VLDB 2014.

**Core idea**: Maintain materialized views (computed tables) incrementally as
base data changes. Uses *higher-order deltas* — the delta of a delta — to
pre-compile update logic at view-definition time.

**State change propagation**: When base data changes by Δ, the view update is
computed as `∂Q/∂R · ΔR` (a "derivative" of the query with respect to the
changed relation). Higher-order deltas pre-compute these derivatives.

**Effect/cost tracking**: Update cost is proportional to the change size times
the derivative complexity. For common queries (joins, aggregations), the
derivative is much simpler than the full query.

**Gas City mapping**: Each cell is a "materialized view" of its upstream cells.
When upstream changes, Gas City currently recomputes the entire cell (full
recomputation). The IVM approach suggests computing only the *delta* — "what
changed in the output given what changed in the input." For LLM cells, this
maps to "diff-aware prompting" — feeding the LLM only the changed inputs and
asking it to update its previous output.

**What Gas City could adopt**: Delta-aware cell recomputation. Instead of
re-running a full LLM prompt, construct a "delta prompt" that says "your
previous output was X, the input changed by ΔY, update accordingly." This
could dramatically reduce token cost for incremental updates.

---

## 3. Effect Systems and Resource-Aware Type Systems

### 3.1 Graded Monads — Orchard, Petricek, et al.

**Papers**:
- Orchard, D., Wadler, P., Eades, H. "Unifying graded and parameterised
  monads." arXiv:2001.10274
- Gaboardi, M., Katsumata, S., Orchard, D., Breuvart, F., Uustalu, T.
  "Combining Effects and Coeffects via Grading." ICFP 2016.
- Katsumata, S. "Parametric Effect Monads and Semantics of Effect Systems."
  POPL 2014.

**Core idea**: A graded monad `T_r` is a monad indexed by a grade `r` from a
monoid (or semiring). The grade tracks a quantitative property of the
computation — resource usage, cost, effect labels, etc. Composition multiplies
grades: `bind : T_r a → (a → T_s b) → T_{r·s} b`.

**State change propagation**: Not directly about propagation. Graded monads
describe what effects a computation *may have*, enabling static analysis.

**Effect/cost tracking**: The grade IS the cost/effect tracker. For a resource
semiring `(R, +, 0, ·, 1)`:
- Sequential composition: `r · s` (multiplication in the semiring)
- Parallel composition (in duoidal grading): `r + s` or a separate product
- Return (pure): grade `1` (identity for multiplication)

**Gas City mapping**: Gas City's effect algebra `(Effect, seq, par, zero)` is
precisely a graded monad with duoidal structure. The Lean formalization proves
this: `seq` is the graded monad multiplication, `par` is the second monoidal
product, and `par_le_seq` is the interchange law. This is not an analogy — it
is a direct mathematical identification.

**What Gas City could adopt**:
- *Type-level effect tracking*: Annotate cell types with their effect grades,
  enabling compositional cost accounting after execution.
- *Coeffect grading*: Orchard's coeffects track what a computation *needs*
  from its context (capabilities, resources). This maps to cell requirements
  (model capabilities, API access, etc.).
- *Graded comonads for context*: The "context" a cell reads from is a graded
  comonadic structure. The coeffect grade tracks how much context is consumed.

### 3.2 Coeffects — Petricek, Orchard, Mycroft

**Paper**: Petricek, T., Orchard, D., Mycroft, A. "Coeffects: A Calculus of
Context-Dependent Computation." ICFP 2014.

**Core idea**: While effects describe what a computation *does* to the world,
coeffects describe what a computation *requires* from the world. Technically,
coeffects use graded comonads to track resource requirements.

**State change propagation**: Coeffects propagate *requirements backwards*
through the computation graph. If a downstream cell requires high quality,
that requirement propagates upstream as a coeffect demand.

**Effect/cost tracking**: Coeffect grades form a semiring dual to effect grades.
The key operations: `+` (requirement of choice/branching) and `·` (requirement
of sequential composition).

**Gas City mapping**: Currently Gas City tracks effects (what cells produce:
tokens, quality) but not coeffects (what cells require). Coeffects would enable
*demand-driven quality*: if a downstream decision cell only needs binary
yes/no, the upstream synthesis cell doesn't need "excellent" quality —
"adequate" suffices. This propagates backwards as a coeffect, reducing upstream
token budgets.

**What Gas City could adopt**: Bidirectional grading — effects flow forward
(cost accumulates), coeffects flow backward (requirements propagate). This
enables the optimal token allocation theorem: marginal distortion reduction
per token should be equal across cells (the Lagrangian condition from the
information-theoretic analysis).

### 3.3 Algebraic Effects and Handlers

**Papers**:
- Plotkin, G., Power, J. "Algebraic Operations and Generic Effects."
  Applied Categorical Structures 11(1), 2003.
- Plotkin, G., Pretnar, M. "Handlers of Algebraic Effects." ESOP 2009.
- Bauer, A., Pretnar, M. "Programming with Algebraic Effects and Handlers."
  JLAMP 84(1), 2015. arXiv:1203.1539

**Core idea**: Computational effects (exceptions, state, I/O) are modeled as
algebraic operations with equations. Effect handlers provide interpretations
(semantics) for these operations, much like exception handlers interpret `throw`.

**State change propagation**: Effect handlers can intercept and transform
effects as they propagate up the call stack. This gives compositional control
over effect interpretation.

**Effect/cost tracking**: Effect types statically track which effects a
computation may perform. Row-polymorphic effect types (as in Koka, Eff, Frank)
enable fine-grained tracking.

**Gas City mapping**: Each cell performs LLM effects (token consumption, quality
production). An "effect handler" at the molecule level could intercept these
effects to: (1) enforce token budgets (reject computations exceeding budget);
(2) provide mock responses for testing; (3) cache and replay for
deterministic evaluation. The handler pattern maps to Gas City's formula-level
configuration.

**What Gas City could adopt**: Handler-based cell configuration. Instead of
hard-coding cell behavior, define cells as effectful computations and let the
molecule-level handler determine how effects are interpreted. This enables
testing (mock handler), cost control (budget handler), and replay (cache handler)
without changing cell definitions.

### 3.4 Resource-Aware Type Systems — Linear and Affine Types

**Papers**:
- Girard, J.-Y. "Linear Logic." TCS 50(1), 1987.
- Walker, D. "Substructural Type Systems." Ch. 1 of Advanced Topics in Types
  and Programming Languages, MIT Press, 2004.
- Hofmann, M., Jost, S. "Static Prediction of Heap Space Usage for
  First-Order Functional Programs." POPL 2003.

**Core idea**: Linear types ensure resources are used exactly once. Affine types
allow at most once. This prevents resource leaks (forgetting to free) and
double-use (using freed memory). Hofmann-Jost extends this to automatic
heap-space prediction via type-level cost annotations.

**State change propagation**: Not directly applicable. Linear types ensure
correct resource flow rather than change propagation.

**Effect/cost tracking**: Hofmann-Jost's type system statically infers upper
bounds on heap usage by assigning "potential" to data structures. Each
operation consumes some potential and produces new potential. Total consumption
bounds total cost.

**Gas City mapping**: Token budgets are finite resources. Linear/affine
reasoning ensures tokens aren't "double-spent" (counted twice in the cost
algebra) and aren't "leaked" (consumed without contributing to any cell's
output). The potential method maps directly: each cell has a "token potential"
that is consumed during evaluation and transferred to downstream cells.

**What Gas City could adopt**: Linear reasoning about token flow. Ensure tokens
aren't double-counted in the cost algebra and that every token consumed
contributes to some cell's output. The potential method could track budget
*caps* (allocate a maximum spend, enforce at runtime) rather than predicting
actual consumption.

---

## 4. Information Theory in Computation

### 4.1 Rate-Distortion Theory (Shannon)

**Paper**: Shannon, C.E. "Coding Theorems for a Discrete Source with a
Fidelity Criterion." IRE National Convention Record, 1959.

**Core idea**: For any source and distortion measure, the *rate-distortion
function* R(D) gives the minimum bits per symbol needed to reproduce the
source within average distortion D. Below R(D), faithful reproduction is
impossible; above R(D), it's achievable.

**State change propagation**: Not directly applicable. Rate-distortion
characterizes the fundamental limit of lossy compression.

**Effect/cost tracking**: The rate R (bits/symbol) is the cost; the distortion
D is the quality loss. R(D) defines the Pareto frontier of cost vs. quality.

**Gas City mapping**: Each LLM cell is a lossy codec with a rate-distortion
curve. "Rate" = tokens consumed; "Distortion" = quality degradation. The
rate-distortion function of a cell gives the minimum tokens needed to achieve
a target quality level. Gas City's token budgeting problem is precisely
rate-distortion optimization: minimize total tokens subject to end-to-end
quality constraints.

**What Gas City could adopt**: Empirically estimate each cell's rate-distortion
curve by running at various token budgets and measuring output quality. Use
these curves for optimal token allocation: the Lagrangian condition (equal
marginal distortion per token across cells) is exactly the water-filling
solution from information theory.

### 4.2 Data Processing Inequality

**Reference**: Cover, T.M., Thomas, J.A. "Elements of Information Theory."
Wiley, 2006. Ch. 2.

**Core idea**: For a Markov chain X → Y → Z, the mutual information satisfies
I(X;Z) ≤ I(X;Y). Processing data cannot increase information about the source.

**Gas City mapping**: Already identified in the Gas City information-theoretic
analysis. Each cell in a chain loses information. A 10-cell chain preserves at
most what the bottleneck cell passes through. This motivates:
- Bottleneck identification: find the cell with lowest information throughput
- Parallel architectures: avoid long chains; use diamonds and fan-in patterns
- Quality-aware DAG design: wide and shallow beats deep and narrow

### 4.3 Information Bottleneck Method

**Paper**: Tishby, N., Pereira, F.C., Bialek, W. "The Information Bottleneck
Method." arXiv:physics/0004057, 2000.

**Core idea**: Find a compressed representation T of input X that maximizes
information about target Y. Formally: minimize I(X;T) - β·I(T;Y). The
parameter β trades compression against prediction quality.

**Gas City mapping**: Each synthesis cell is an information bottleneck — it
compresses upstream data while preserving what matters for downstream consumers.
The β parameter maps to the cell's token budget: higher budget = more faithful
representation = higher I(T;Y) but also higher I(X;T) (more tokens).

**What Gas City could adopt**: Frame cell design as an information bottleneck
problem. For each cell, identify what information downstream cells actually
need (the "relevant information") and compress aggressively along irrelevant
dimensions. This gives principled guidance for prompt design: "summarize X
*with respect to Y's needs*."

### 4.4 Lossy Source Coding in Pipelines (Successive Refinement)

**Papers**:
- Equitz, W.H.R., Cover, T.M. "Successive Refinement of Information."
  IEEE TIT 37(2), 1991.
- Rimoldi, B. "Successive Refinement of Information: Characterization of
  Achievable Rates." IEEE TIT 40(1), 1994.

**Core idea**: A source is "successively refinable" if layered encoding achieves
the same rate-distortion performance as single-shot encoding at each layer.
Gaussian sources are successively refinable; most others are not.

**Gas City mapping**: Multi-resolution cell evaluation. A cell might first
produce a "draft" (low rate, high distortion), then "refine" to higher quality
if demanded. Successive refinability determines whether this layered approach
is optimal. If LLM outputs are not successively refinable (likely), then
draft-then-refine incurs overhead vs. single-shot at the target quality.

**What Gas City could adopt**: Test whether common cell types are approximately
successively refinable. If so, implement multi-resolution evaluation; if not,
prefer single-shot at the target quality level.

---

## 5. Dataflow Architectures

### 5.1 Kahn Process Networks

**Paper**: Kahn, G. "The Semantics of a Simple Language for Parallel
Programming." IFIP Congress, 1974.

**Core idea**: A network of deterministic sequential processes communicating
via unbounded FIFO channels. Each process reads from input channels and writes
to output channels. The network semantics is the least fixed point of the
process equations.

**State change propagation**: Data-driven: processes block on empty input
channels and produce output when input is available. Propagation follows the
data flow.

**Effect/cost tracking**: None explicitly. The semantic model is denotational —
processes are continuous functions on streams.

**Gas City mapping**: Kahn networks are the theoretical foundation for dataflow
computing. Gas City cells are Kahn processes (deterministic functions of inputs),
and cell wires are channels. The key property: Kahn networks are deterministic
(same inputs always produce same outputs) regardless of scheduling order.
Gas City inherits this property from its DAG structure — evaluation order
within a topological layer doesn't matter.

**What Gas City could adopt**: The Kahn fixed-point semantics provides a formal
foundation for iterative/recursive cell evaluation. If Gas City ever supports
cycles (iterative refinement), Kahn's least-fixed-point semantics gives the
correct convergence criterion.

### 5.2 Synchronous Dataflow (SDF)

**Paper**: Lee, E.A., Messerschmitt, D.G. "Synchronous Data Flow." Proc. IEEE
75(9), 1987.

**Core idea**: A restricted dataflow model where each process declares how many
tokens it consumes and produces per firing. This static rate information enables
compile-time scheduling — no dynamic scheduling overhead.

**State change propagation**: Static schedule computed at compile time via
balance equations. Each process fires a predetermined number of times per period.

**Effect/cost tracking**: Token rates (consumption/production per firing) are
the key static property. Buffer sizes can be computed at compile time.

**Gas City mapping**: SDF's static token rates do NOT map to Gas City. LLM
token consumption is fundamentally unpredictable — it varies with input content,
model behavior, and output length. Cells cannot declare token rates ahead of
time. This is where the SDF analogy breaks: SDF processes have fixed,
deterministic consumption rates; LLM cells do not.

**What Gas City could adopt**: NOT static cost scheduling. Instead, Gas City
should track actual token consumption after evaluation, record it in digests,
and use historical data from prior runs to inform (not predict) budget caps.
Runtime budget enforcement (abort if cap exceeded) is feasible; pre-execution
cost prediction is not.

### 5.3 Demand-Driven vs. Data-Driven Evaluation

**Reference**: Arvind, Nikhil, R.S. "Executing a Program on the MIT
Tagged-Token Dataflow Architecture." IEEE TPDS 1(3), 1990.

**Core idea**: Data-driven (eager) evaluation computes everything reachable from
changed inputs. Demand-driven (lazy) evaluation computes only what's needed for
demanded outputs. Hybrid approaches combine both.

**Gas City mapping**: Current Gas City is data-driven (staleness propagates
eagerly through the entire downstream DAG). For large DAGs where only some
outputs are observed, demand-driven evaluation avoids computing unobserved
cells. The Adapton hybrid (Section 1.3) is the best known combination.

---

## 6. Agent Coordination and Multi-Agent Systems

### 6.1 BDI Architecture (Belief-Desire-Intention)

**Papers**:
- Rao, A.S., Georgeff, M.P. "BDI Agents: From Theory to Practice."
  ICMAS 1995.
- Bratman, M.E. "Intention, Plans, and Practical Reason." Harvard UP, 1987.

**Core idea**: Agents maintain three mental attitudes: Beliefs (information
about the world), Desires (goals), and Intentions (committed plans). The BDI
loop: perceive → update beliefs → deliberate (select desires) → plan (form
intentions) → execute.

**State change propagation**: Beliefs update from perception. Desire generation
is triggered by belief changes. Intentions are formed from desires via
means-ends reasoning.

**Effect/cost tracking**: None formally. Some extensions add utility functions
to desires for cost-benefit analysis.

**Gas City mapping**: Each polecat (worker agent) operates in a BDI-like loop:
- Beliefs: bead status, DAG state, git state
- Desires: assigned bead (work goal)
- Intentions: molecule formula (committed plan)
The key insight: Gas City externalizes BDI state into the bead system. Beliefs
are shared (beads are visible to all), desires are assigned (hooks), and
intentions are formalized (formulas). This eliminates the coordination failure
mode where agents have inconsistent beliefs.

**What Gas City could adopt**: The BDI literature on *teamwork and joint
intentions* (Cohen & Levesque, Grosz & Kraus) addresses how multiple agents
coordinate on shared goals. Gas City's molecule system implicitly implements
"shared plans" — the formula IS the joint intention, and beads ARE the shared
beliefs.

### 6.2 Stigmergy — Indirect Communication Through Environment

**Papers**:
- Grassé, P.-P. "La reconstruction du nid et les coordinations
  inter-individuelles chez Bellicositermes natalensis." Insectes Sociaux 6,
  1959.
- Parunak, H.V.D. "'Go to the Ant': Engineering Principles from Natural
  Multi-Agent Systems." Annals of OR 75, 1997.
- Theraulaz, G., Bonabeau, E. "A Brief History of Stigmergy." Artificial
  Life 5(2), 1999.

**Core idea**: Agents coordinate by modifying a shared environment rather than
communicating directly. A termite deposits a pheromone-laden mud ball; another
termite, encountering the ball, is stimulated to deposit its own ball nearby.
No direct communication — the environment mediates all coordination.

**State change propagation**: Agents poll the environment. Changes to the
environment (new artifacts) trigger behavioral responses in agents that
encounter them. Propagation is indirect and asynchronous.

**Effect/cost tracking**: Pheromone evaporation provides natural cost decay —
old signals fade, preventing stale coordination. Signal strength encodes
urgency/confidence.

**Gas City mapping**: Gas City IS a stigmergic system. Beads are the
environment; polecats are the agents. A polecat doesn't message another polecat
directly — it modifies a bead (closes it, files a new one, updates status),
and other agents react to the changed environment. The `gt nudge` mechanism is
a slight departure from pure stigmergy (it's direct communication), but `bd`
operations are fully stigmergic.

**What Gas City could adopt**:
- *Pheromone-like priority decay*: Issue priority could decay over time,
  ensuring stale issues don't block fresh ones.
- *Stigmergic load balancing*: Instead of centralized dispatch (Witness assigns
  work), let polecats self-select from the bead pool. In ant colonies, this
  produces near-optimal task allocation.
- *Environmental memory*: The bead system already provides this — completed
  beads form a "trail" that informs future decisions.

### 6.3 Contract Net Protocol

**Paper**: Smith, R.G. "The Contract Net Protocol: High-Level Communication
and Control in a Distributed Problem Solver." IEEE TC C-29(12), 1980.

**Core idea**: Task allocation via a market-like protocol. A manager announces
a task; contractor agents bid based on capability and availability; the manager
awards the contract to the best bidder.

**State change propagation**: Broadcast announcement → unicast bids → unicast
award. Failed contracts can be re-announced.

**Effect/cost tracking**: Bids include cost estimates. The manager selects
based on expected cost/quality.

**Gas City mapping**: The Witness + molecule dispatch system is a simplified
Contract Net. The Witness (manager) announces work; polecats (contractors)
are assigned based on availability. The simplification: Gas City doesn't use
bidding — the Witness assigns directly. This is more efficient for the current
scale but limits self-organization.

**What Gas City could adopt**: Bidding for complex molecules. If a molecule
requires specific capabilities (vision model, high context window), polecats
could bid with their capabilities, and the dispatcher selects the best match.
This maps to the "model-aware dispatch" in the model-aware molecules design.

### 6.4 Tuple Spaces (Linda)

**Paper**: Gelernter, D. "Generative Communication in Linda." ACM TOPLAS
7(1), 1985.

**Core idea**: Processes communicate by depositing, reading, and removing tuples
from a shared associative memory ("tuple space"). Operations: `out` (deposit),
`in` (remove-and-read, blocking), `rd` (read without removing).

**State change propagation**: Event-driven: processes block on `in`/`rd` until
matching tuples appear. No explicit propagation — the tuple space mediates.

**Gas City mapping**: The bead system is a typed tuple space. `bd create` is
`out`; `bd show` is `rd`; `bd close` removes the tuple. The beads query
language (`bd ready`, `bd blocked`) provides pattern matching over the space.
Linda's elegance is in its simplicity — three operations cover all
coordination patterns. Gas City has evolved a richer interface but the
underlying model is identical.

**What Gas City could adopt**: Linda's theoretical foundation enables formal
analysis of coordination patterns. The "chemical abstract machine" (Section 7.1)
is Linda + reaction rules, which directly maps to molecule formulas.

---

## 7. Chemical and Membrane Computing

### 7.1 Chemical Abstract Machine (CHAM)

**Paper**: Berry, G., Boudol, G. "The Chemical Abstract Machine." TCS 96(1),
1992. Also: Banâtre, J.-P., Le Métayer, D. "Programming by Multiset
Transformation." CACM 36(1), 1993.

**Core idea**: Computation as chemical reactions. The program state is a
"solution" (multiset) of molecules. Reactions transform molecules according to
rules. Rules fire non-deterministically whenever reactants are present. The
*membrane* (denoted `{| ... |}`) scopes reactions — molecules in different
membranes don't interact.

**State change propagation**: Reaction rules fire when reactants match.
Products become available for further reactions. No explicit propagation order —
reactions are asynchronous and concurrent.

**Effect/cost tracking**: None natively. Extensions add "energy" or "priority"
to reactions. The "heating/cooling" rules (`≡` structural congruence) allow
rearranging molecules without consuming resources.

**Gas City mapping**: This is the etymological source of Gas City's "molecule"
terminology. The mapping is direct:
- CHAM solution = bead pool
- CHAM molecules = individual beads
- CHAM reactions = formula steps (transforming input beads into output beads)
- CHAM membrane = rig boundary (polecats in different rigs don't interact)
- Heating/cooling = organizational operations (reordering, grouping) vs.
  computational operations (LLM evaluation)

**What Gas City could adopt**:
- *Formal reaction semantics*: Define formula steps as CHAM reactions with
  explicit reactants (input beads) and products (output beads). This enables
  formal analysis of molecule execution — deadlock detection, confluence
  checking, etc.
- *Membrane hierarchy*: CHAM membranes scope reactions. Gas City rigs are
  membranes. This could extend to sub-rig scoping for team-level isolation.
- *Airlock pattern*: The `⟨...⟩` syntax for molecules that can cross membranes.
  In Gas City, this maps to cross-rig bead references (e.g., HQ beads
  visible to all rigs).

### 7.2 Membrane Computing (P Systems)

**Paper**: Păun, G. "Computing with Membranes." JCSS 61(1), 2000.
Also: Păun, G. "Membrane Computing: An Introduction." Springer, 2002.

**Core idea**: Computation occurs in a hierarchy of membranes (nested regions).
Each membrane contains a multiset of objects and a set of evolution rules.
Objects evolve according to rules and can cross membranes (in or out). The
skin membrane (outermost) is the interface with the environment.

**State change propagation**: Rules fire maximally parallel — all applicable
rules fire simultaneously in each step. Objects can be sent to inner membranes
(`here`), outer membranes (`out`), or dissolved membranes (`δ`).

**Effect/cost tracking**: P systems have been studied for computational
complexity. Key result: P systems with active membranes (membranes can divide)
solve NP-complete problems in polynomial time (using exponential space from
membrane division).

**Gas City mapping**:
- Membranes = rigs (each rig has its own bead pool and rules)
- Membrane hierarchy = Town > Rig > Worktree
- Skin membrane = Mayor (interface with external world)
- `out` rule = escalation (bead moves to outer rig)
- `in` rule = dispatch (bead moves to inner rig)
- `δ` (dissolution) = polecat self-cleanup after `gt done`

**What Gas City could adopt**:
- *Active membranes*: Rigs that can spawn sub-rigs dynamically when workload
  exceeds capacity. This is "elastic scaling" via membrane division.
- *Maximal parallelism*: P systems fire all possible rules simultaneously.
  Gas City's current dispatch is sequential (one bead per polecat). Maximal
  parallelism within a molecule (all independent steps fire at once) is an
  optimization opportunity.
- *Membrane dissolution*: After a polecat completes, its worktree membrane
  dissolves, releasing products (committed code) to the outer rig.

### 7.3 Reaction Networks as Computation

**Papers**:
- Soloveichik, D., Seelig, G., Winfree, E. "DNA as a Universal Substrate
  for Chemical Kinetics." PNAS 107(12), 2010.
- Cardelli, L. "Morphisms of Reaction Networks That Couple Structure to
  Function." BMC Systems Biology 8, 2014.
- Fages, F., Soliman, S. "Abstract Interpretation and Types for Systems
  Biology." TCS 403(1), 2008.

**Core idea**: Chemical reaction networks (CRNs) are Turing-complete — any
computable function can be implemented as a set of chemical reactions. CRNs
are naturally analog (concentrations are continuous) but can implement discrete
computation via population protocols.

**State change propagation**: Governed by mass-action kinetics (rate
proportional to reactant concentrations) or stochastic kinetics (Gillespie
algorithm). The system evolves through a continuous state space.

**Gas City mapping**: If beads are molecular species and formulas are
reactions, then Gas City is a discrete CRN. The "concentration" of a bead
type (how many open issues of that kind) affects the "reaction rate" (how
quickly they get processed). High concentration of bugs → high fix rate
(more polecats assigned) — this is mass-action-like behavior emerging from
the dispatch mechanism.

**What Gas City could adopt**: CRN analysis tools. Chemical reaction network
theory provides conditions for:
- *Persistence*: No species goes extinct (no bead type is permanently
  neglected)
- *Detailed balance*: The system reaches equilibrium (workload stabilizes)
- *Deficiency theory*: Predicts steady-state behavior from network structure
  alone. Gas City's workload dynamics might be analyzable via CRN deficiency.

---

## 8. Categorical and Algebraic Foundations

### 8.1 Duoidal Categories (Aguiar & Mahajan)

**Paper**: Aguiar, M., Mahajan, S. "Monoidal Functors, Species, and Hopf
Algebras." CRM Monograph Series, AMS, 2010.

**Core idea**: A duoidal category has two monoidal structures (⊗, I) and (⋆, J)
related by interchange maps. The two products need not distribute over each
other — only a lax interchange law holds: `(A ⋆ B) ⊗ (C ⋆ D) → (A ⊗ C) ⋆ (B ⊗ D)`.

**Gas City mapping**: Gas City's effect algebra has exactly this structure:
- `seq` (⊗): sequential composition, monoid over Effect
- `par` (⋆): parallel composition, monoid over Effect
- `par_le_seq`: the interchange/lax distributivity law
This has been formalized and proven in Lean 4 (Section 18 of the codebase).

**What Gas City could adopt**: Duoidal categories have rich theory for:
- *Monoidal functors between duoidal categories*: Natural transformations that
  respect both products. This could formalize how molecule transformations
  (refactoring a sequential pipeline to a parallel one) preserve semantics.
- *Species*: Combinatorial species (Joyal) in duoidal categories provide a
  theory of "structures indexed by finite sets" — potentially relevant to
  molecule templates with variable numbers of steps.

### 8.2 Sheaf Theory and Staleness

**References**:
- Mac Lane, S., Moerdijk, I. "Sheaves in Geometry and Logic." Springer, 1994.
- Grothendieck, A. "Sur quelques points d'algèbre homologique." Tôhoku Math.
  J. 9, 1957.

**Core idea**: A sheaf on a topological space assigns data to open sets such
that consistent local data can be uniquely glued to global data. The
sheafification functor converts a presheaf (arbitrary assignment) into a sheaf
(consistent assignment).

**Gas City mapping**: As identified in the Gas City synthesis:
- Open sets = "cones" in the DAG (a cell and all its downstream dependents)
- Presheaf = assignment of values to cells (possibly inconsistent after updates)
- Staleness propagation = the sheafification functor turning an inconsistent
  assignment (some cells updated, some not) into a consistent one (all cells
  recomputed)
- Gluing axiom = if two adjacent sub-DAGs are both fresh, their union is fresh

**What Gas City could adopt**: Sheaf cohomology measures "how far from being
a sheaf" a presheaf is. Applied to Gas City: cohomological invariants could
measure "how inconsistent" the current DAG state is (how many cells are stale
and how badly). This gives a single number for "DAG health."

### 8.3 Operad Theory and Modular Composition

**Papers**:
- May, J.P. "The Geometry of Iterated Loop Spaces." Springer LNM 271, 1972.
- Leinster, T. "Higher Operads, Higher Categories." Cambridge UP, 2004.
  arXiv:math/0305049
- Spivak, D.I. "The Operad of Wiring Diagrams." arXiv:1305.0297, 2013.

**Core idea**: An operad is an algebraic structure describing composition of
multi-input operations. Operations with arities (n inputs → 1 output) compose
by plugging outputs into inputs. Wiring diagrams (Spivak) extend this to
arbitrary wirings, not just tree-shaped compositions.

**Gas City mapping**: Gas City's DAG topology is an operad algebra:
- Operations = cells (each cell has typed inputs and a typed output)
- Composition = wiring cells together via `{{ref}}`
- The operad axioms (associativity, identity) ensure that nested composition
  works correctly

Spivak's wiring diagrams are particularly relevant — they formalize exactly
the kind of "plug cells together" composition that Gas City uses, including
feedback loops and parallel composition.

**What Gas City could adopt**: Operadic typing would enable:
- *Type-safe composition*: Verify at DAG-construction time that cell types
  match at connection points.
- *Modular molecule construction*: Build molecules from reusable sub-molecules
  using operadic composition, with guaranteed type safety.
- *Graphical language*: Wiring diagrams provide a rigorous visual notation
  for Gas City DAGs.

---

## 9. Thermodynamics of Computation

### 9.1 Landauer's Principle

**Paper**: Landauer, R. "Irreversibility and Heat Generation in the Computing
Process." IBM J. Res. Dev. 5(3), 1961.

**Core idea**: Erasing one bit of information necessarily dissipates at least
kT ln 2 joules of energy. Computation itself need not be dissipative (reversible
computing), but information erasure (forgetting) has a fundamental cost.

**Gas City mapping**: LLM cells are lossy — they compress information, erasing
details. By Landauer's principle, this erasure has a minimum cost. In Gas City,
the "energy" is tokens: compressing n bits of upstream data into m bits of
cell output requires at least f(n-m) tokens (where f maps information erasure
to LLM computational cost). The exact form of f is empirical but the principle
is fundamental.

**What Gas City could adopt**: Landauer's principle gives a *lower bound* on
cell token cost given the compression ratio. If a cell compresses a 10,000-word
document into a 100-word summary (roughly 100:1 compression), there's a
minimum token cost to achieve acceptable quality. This bound could be computed
per cell type to detect "impossible budgets" at planning time.

### 9.2 Bennett's Reversible Computing

**Paper**: Bennett, C.H. "Logical Reversibility of Computation." IBM J. Res.
Dev. 17(6), 1973.

**Core idea**: Any computation can be made thermodynamically reversible (zero
energy dissipation) by keeping a complete history. The trade-off: reversibility
requires space proportional to time (you must store the history).

**Gas City mapping**: Git IS Bennett's reversible computing. Every cell
evaluation is logged (git commits), enabling perfect reversal (revert to any
previous state). The space cost: Dolt storage grows with history. The benefit:
any coordination failure can be unwound without information loss.

### 9.3 Free Energy Principle (Active Inference)

**Papers**:
- Friston, K. "The Free-Energy Principle: A Unified Brain Theory?"
  Nature Reviews Neuroscience 11(2), 2010.
- Parr, T., Pezzulo, G., Friston, K.J. "Active Inference: The Free Energy
  Principle in Mind, Brain, and Behavior." MIT Press, 2022.

**Core idea**: Biological systems minimize variational free energy — a bound
on surprise (negative log evidence). This drives both perception (updating
beliefs to match observations) and action (changing the world to match
predictions).

**Gas City mapping**: Each polecat minimizes "surprise" — the discrepancy
between expected state (from formulas) and actual state (from beads). When
a polecat encounters unexpected state (a stale cell, a failed build), it acts
to reduce surprise (recompute the cell, fix the build). The Witness monitors
global free energy (aggregate staleness/inconsistency) and intervenes when it
exceeds a threshold.

**What Gas City could adopt**: Free energy minimization as an objective function
for agent scheduling. Instead of FIFO or priority-based scheduling, select the
next action that maximally reduces DAG free energy (inconsistency). This
naturally prioritizes: (1) highly stale critical-path cells; (2) cells with
many dependents; (3) cells whose staleness blocks other agents.

---

## 10. Synthesis: Mapping to Gas City

### Concept Correspondence Table

| Gas City Concept | Academic Precedent | Key Reference |
|------------------|--------------------|---------------|
| Cell | FRP Behavior, Kahn process, CHAM molecule | Elliott 1997, Kahn 1974, Berry & Boudol 1992 |
| `{{ref}}` wire | FRP lifting, Kahn channel, tuple space pattern | Elliott 1997, Kahn 1974, Gelernter 1985 |
| Staleness propagation | Self-adjusting computation, Adapton dirtying, sheafification | Acar 2002, Hammer 2014, Grothendieck 1957 |
| Effect algebra (seq/par) | Graded monad with duoidal structure | Katsumata 2014, Orchard 2020, Aguiar & Mahajan 2010 |
| Token budget | Rate-distortion, Landauer cost | Shannon 1959, Landauer 1961 |
| Quality tracking | Distortion measure, information bottleneck | Shannon 1959, Tishby 2000 |
| Molecule formula | CHAM reaction rule, P system evolution rule | Berry & Boudol 1992, Păun 2000 |
| Rig/membrane | CHAM membrane, P system membrane | Berry & Boudol 1992, Păun 2000 |
| Polecat | BDI agent, ant in stigmergic colony | Rao & Georgeff 1995, Parunak 1997 |
| Bead system | Tuple space, stigmergic environment | Gelernter 1985, Grassé 1959 |
| DAG topology | Operad algebra, wiring diagram | Spivak 2013, Leinster 2004 |
| Git history | Bennett's reversible computing | Bennett 1973 |
| Witness monitoring | Free energy minimization | Friston 2010 |
| Cross-rig dispatch | Contract Net Protocol | Smith 1980 |

### Priority Adoption Recommendations

**High priority** (directly applicable, proven in related systems):
1. **Adapton two-phase protocol** — Lazy recomputation saves tokens for large DAGs
2. **Rate-distortion optimization** — Principled token budgeting via water-filling
3. **Coeffect-based demand propagation** — Quality requirements flow backward
4. **CHAM formal semantics** — Enables deadlock detection and confluence analysis

**Medium priority** (requires design work but high potential):
5. **Delta-aware recomputation** (IVM) — Incremental cell updates via diff prompting
6. **Operadic typing** — Type-safe DAG composition with modular molecules
7. **Runtime budget enforcement** — Hard caps with historical cost data

**Lower priority** (theoretical interest, long-term foundation):
8. **Sheaf cohomology for DAG health** — Quantitative inconsistency measure
9. **CRN deficiency theory** — Predict workload steady-state from structure
10. **Active inference scheduling** — Free-energy-minimizing dispatch

---

## References (Alphabetical)

- Acar, U.A. "Self-Adjusting Computation." PhD thesis, CMU, 2005.
- Acar, U.A., Blelloch, G.E., Harper, R. "Adaptive Functional Programming." POPL 2002.
- Aguiar, M., Mahajan, S. "Monoidal Functors, Species, and Hopf Algebras." CRM Monograph Series, AMS, 2010.
- Banâtre, J.-P., Le Métayer, D. "Programming by Multiset Transformation." CACM 36(1), 1993.
- Bauer, A., Pretnar, M. "Programming with Algebraic Effects and Handlers." JLAMP 84(1), 2015. arXiv:1203.1539
- Bennett, C.H. "Logical Reversibility of Computation." IBM J. Res. Dev. 17(6), 1973.
- Berry, G., Boudol, G. "The Chemical Abstract Machine." TCS 96(1), 1992.
- Bratman, M.E. "Intention, Plans, and Practical Reason." Harvard UP, 1987.
- Burnett, M. et al. "Forms/3: A First-Order Visual Language." JFP 2001.
- Cardelli, L. "Morphisms of Reaction Networks That Couple Structure to Function." BMC Systems Biology 8, 2014.
- Cover, T.M., Thomas, J.A. "Elements of Information Theory." Wiley, 2006.
- Elliott, C., Hudak, P. "Functional Reactive Animation." ICFP 1997.
- Equitz, W.H.R., Cover, T.M. "Successive Refinement of Information." IEEE TIT 37(2), 1991.
- Fages, F., Soliman, S. "Abstract Interpretation and Types for Systems Biology." TCS 403(1), 2008.
- Friston, K. "The Free-Energy Principle: A Unified Brain Theory?" Nature Reviews Neuroscience 11(2), 2010.
- Gaboardi, M. et al. "Combining Effects and Coeffects via Grading." ICFP 2016.
- Gelernter, D. "Generative Communication in Linda." ACM TOPLAS 7(1), 1985.
- Girard, J.-Y. "Linear Logic." TCS 50(1), 1987.
- Grassé, P.-P. "La reconstruction du nid..." Insectes Sociaux 6, 1959.
- Grothendieck, A. "Sur quelques points d'algèbre homologique." Tôhoku Math. J. 9, 1957.
- Hammer, M.A. et al. "Adapton: Composable, Demand-Driven Incremental Computation." PLDI 2014.
- Hofmann, M., Jost, S. "Static Prediction of Heap Space Usage." POPL 2003.
- Kahn, G. "The Semantics of a Simple Language for Parallel Programming." IFIP 1974.
- Katsumata, S. "Parametric Effect Monads and Semantics of Effect Systems." POPL 2014.
- Koch, C. "Incremental Query Evaluation in a Ring of Databases." PODS 2010.
- Koch, C. et al. "DBToaster: Higher-order Delta Processing." VLDB 2014.
- Landauer, R. "Irreversibility and Heat Generation in the Computing Process." IBM J. Res. Dev. 5(3), 1961.
- Lee, E.A., Messerschmitt, D.G. "Synchronous Data Flow." Proc. IEEE 75(9), 1987.
- Leinster, T. "Higher Operads, Higher Categories." Cambridge UP, 2004. arXiv:math/0305049
- Mac Lane, S., Moerdijk, I. "Sheaves in Geometry and Logic." Springer, 1994.
- May, J.P. "The Geometry of Iterated Loop Spaces." Springer LNM 271, 1972.
- McSherry, D. et al. "Differential Dataflow." CIDR 2013.
- Murray, D.G. et al. "Naiad: A Timely Dataflow System." SOSP 2013.
- Orchard, D., Wadler, P., Eades, H. "Unifying graded and parameterised monads." arXiv:2001.10274
- Parr, T., Pezzulo, G., Friston, K.J. "Active Inference." MIT Press, 2022.
- Parunak, H.V.D. "'Go to the Ant': Engineering Principles from Natural Multi-Agent Systems." Annals of OR 75, 1997.
- Păun, G. "Computing with Membranes." JCSS 61(1), 2000.
- Petricek, T., Orchard, D., Mycroft, A. "Coeffects: A Calculus of Context-Dependent Computation." ICFP 2014.
- Peyton Jones, S. et al. "A User-Centred Approach to Functions in Excel." ICFP 2003.
- Plotkin, G., Power, J. "Algebraic Operations and Generic Effects." Applied Categorical Structures 11(1), 2003.
- Plotkin, G., Pretnar, M. "Handlers of Algebraic Effects." ESOP 2009.
- Rao, A.S., Georgeff, M.P. "BDI Agents: From Theory to Practice." ICMAS 1995.
- Rimoldi, B. "Successive Refinement of Information." IEEE TIT 40(1), 1994.
- Shannon, C.E. "Coding Theorems for a Discrete Source with a Fidelity Criterion." IRE 1959.
- Smith, R.G. "The Contract Net Protocol." IEEE TC C-29(12), 1980.
- Soloveichik, D. et al. "DNA as a Universal Substrate for Chemical Kinetics." PNAS 107(12), 2010.
- Spivak, D.I. "The Operad of Wiring Diagrams." arXiv:1305.0297, 2013.
- Theraulaz, G., Bonabeau, E. "A Brief History of Stigmergy." Artificial Life 5(2), 1999.
- Tishby, N. et al. "The Information Bottleneck Method." arXiv:physics/0004057, 2000.
- Walker, D. "Substructural Type Systems." ATTAPL, MIT Press, 2004.
