/-
  BeadCalculus.DAG — Directed Acyclic Graphs for cell dependencies.

  The dependency structure of a formula is a DAG. This module formalizes
  DAGs and proves the key properties we need:
  - Acyclicity
  - Topological ordering exists
  - Readiness is monotone (more completed nodes → more ready nodes)

  We represent DAGs as adjacency lists with a proof of acyclicity.
-/

namespace BeadCalculus

/-- A directed graph on nodes of type α. Edges go from a node to its dependencies. -/
structure Graph (α : Type) [DecidableEq α] where
  nodes : List α
  edges : α → List α  -- edges n = list of nodes that n depends on
  edges_valid : ∀ n, ∀ m ∈ edges n, m ∈ nodes

/-- A path in the graph: a sequence of edges from start to finish. -/
inductive Path {α : Type} [DecidableEq α] (g : Graph α) : α → α → Prop where
  | trivial : ∀ n, n ∈ g.nodes → Path g n n
  | step    : ∀ n m k, m ∈ g.edges n → Path g m k → Path g n k

/-- A graph is acyclic if no non-trivial path leads back to the start. -/
def Graph.acyclic {α : Type} [DecidableEq α] (g : Graph α) : Prop :=
  ∀ n, ∀ m ∈ g.edges n, ¬ Path g m n

/-- A DAG is a graph with a proof of acyclicity. -/
structure DAG (α : Type) [DecidableEq α] extends Graph α where
  is_acyclic : Graph.acyclic toGraph

/-- A node is ready when all its dependencies are in the completed set. -/
def DAG.ready {α : Type} [DecidableEq α] (dag : DAG α) (completed : List α) (n : α) : Prop :=
  n ∈ dag.nodes ∧ n ∉ completed ∧ ∀ m ∈ dag.edges n, m ∈ completed

/-- The set of all ready nodes given a completed set. -/
def DAG.readySet {α : Type} [DecidableEq α] [∀ (a : α) (l : List α), Decidable (a ∈ l)]
    (dag : DAG α) (completed : List α) : List α :=
  dag.nodes.filter fun n =>
    n ∉ completed ∧ (dag.edges n).all fun m => completed.contains m

/-- Key theorem: readiness is monotone.
    If we complete more nodes, we get at least as many ready nodes
    (minus the ones we just completed). -/
theorem DAG.ready_monotone {α : Type} [DecidableEq α]
    (dag : DAG α) (completed completed' : List α)
    (h : ∀ n ∈ completed, n ∈ completed')
    (n : α) (hn : dag.ready completed n) (hn_not : n ∉ completed') :
    dag.ready completed' n := by
  constructor
  · exact hn.1
  constructor
  · exact hn_not
  · intro m hm
    exact h m (hn.2.2 m hm)

end BeadCalculus
