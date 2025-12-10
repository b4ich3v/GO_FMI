# Query Relevance Stats

This module turns a list of **scored documents** into calibrated estimates
of how many of them are relevant, using Bernoulli, Binomial and Poisson
distributions.

At the end of the pipeline you get a `QueryRelevanceStats` object that can
answer questions like:

* What is the expected number of relevant documents?
* What is the variance of that number?
* What is the probability that _none_ of the documents are relevant?
* What is the probability that there is _at least one_ relevant document?

The core idea is:

1. Convert each `(document, score)` into a calibrated probability
   `p_i = P(document i is relevant | score_i)`.
2. Treat each document as a Bernoulli random variable `X_i ~ Bernoulli(p_i)`.
3. Optionally **cluster** similar documents and work at cluster level.
4. Aggregate the Bernoullis into a Binomial distribution.
5. Use the Binomial → Poisson approximation to model the count of relevant
   documents with a Poisson distribution with rate `λ`.
6. Wrap all of this into `QueryRelevanceStats`, which gives convenient
   summary statistics.

---

## Directory layout

All files in this folder live under the package
`app.quality_analyze.query_relevance_stats`:

```text
query_relevance_stats/
├─ README.md                      # This file
├─ documentation.pdf              # Mathematical / conceptual background (human docs)
├─ query_relevance_stats.py       # Dataclass with final statistics
├─ poisson_relevance_estimator.py # End‑to‑end pipeline from scores to stats
├─ bayesian_relevance_calibrator.py
│                                  # Calibrates raw scores into probabilities
├─ bernoulli_clusterer.py         # Groups similar documents into clusters
└─ specified_discrete_variables/
   ├─ bernoulli_document_distribution.py
   │                                # Bernoulli + attached document + score
   └─ binomial_document_distribution.py
                                    # Binomial + remembers underlying Bernoullis
```

This package also depends on code in other folders:

* `discrete_variables/*` — generic Bernoulli, Binomial, Poisson distributions
  and the binomial→poisson approximation.
* `conditional_probability/*` — generic Bayes theorem machinery.
* `contidional_exceptation/conditional_expectation.py` — helper formulas for
  expectations (law of total expectation, linearity, etc.).
* `constants.py` — numeric and probability constants used across the codebase.

---

## Data model: document‑level distributions

### `BernoulliDocumentDistribution`

File: `specified_discrete_variables/bernoulli_document_distribution.py`

```python
@dataclass
class BernoulliDocumentDistribution(BernoulliDistribution):
    document: Any | None = None
    score: float | None = None
```

Inherits from `BernoulliDistribution` and adds **metadata** about the
underlying document and its original model score.

From `BernoulliDistribution` (in `discrete_variables/bernoulli_distribution`)
it inherits:

* `probability: float` — `p = P(relevant | score)`
* `expected() = p`
* `variance() = p * (1 - p)`
* `pmf(k)` for `k ∈ {0, 1}`

So each `BernoulliDocumentDistribution` is a **single document** together
with its probability of being relevant.

---

### `BinomialDocumentDistribution`

File: `specified_discrete_variables/binomial_document_distribution.py`

```python
@dataclass
class BinomialDocumentDistribution(BinomialDistribution):
    bernoullis: Sequence[BernoulliDistribution] = ()
```

This is a binomial distribution that **remembers** the individual
Bernoulli trials it was built from.

The factory method:

```python
@classmethod
def from_bernoullis(cls, bernoullis: Sequence[BernoulliDistribution])
```

does the aggregation:

* `n = len(bernoullis)` — number of independent “experiments”.
* `λ_total = sum(p_i)` — sum of the individual relevance probabilities.
* `p̄ = λ_total / n` — average probability per experiment.

Then it returns:

```python
BinomialDocumentDistribution(
    number_of_experiments=n,
    probability=p̄,
    bernoullis=tuple(bernoullis),
)
```

If `n == 0`, it returns a degenerate distribution with `n = 0`, `p̄ = 0`
and an empty sequence of Bernoullis.

---

## Calibrating scores into probabilities

### `BayesianRelevanceCalibrator`

File: `bayesian_relevance_calibrator.py`

```python
class BayesianRelevanceCalibrator:
    def __init__(
        self,
        prior_relevant: float = DEFAULT_RELEVANCE_PRIOR,
        temperature: float = DEFAULT_CALIBRATOR_TEMPERATURE,
        score_threshold: float | None = None,
        lower_score_is_better: bool = True,
    ) -> None:
        ...
```

Purpose: **map a raw ranking score to a probability of relevance**

* `prior_relevant` — prior probability that a random document is relevant.
* `temperature` — scales how “sharp” the transition is around the threshold.
* `score_threshold` — score at which a document is roughly 50% likely to be
  relevant; when `None`, a default constant is used.
* `lower_score_is_better` — flips the sign of the margin depending on
  whether lower or higher scores are better.

Internal workflow:

1. Constructs two hypotheses using `EventHypothesis`:
   * `"RELEVANT"` with prior `π_rel`
   * `"IRRELEVANT"` with prior `π_irrel = 1 - π_rel`
2. Wraps them in a `CompleteEventGroup`, which normalizes the priors.
3. Builds a `BayesConditional` object, which will apply Bayes’ theorem.

Score → likelihoods:

* The method `__normalized_margin(score)` computes a **margin** between the
  incoming score and `score_threshold`, then divides by `temperature`.
* The margin is clamped to `±CALIBRATOR_MAX_NORMALIZED_MARGIN_ABS`.
* A logistic curve is applied:

  ```python
  p_b_given_rel = σ(normalized_margin)
  p_b_given_irrel = 1 - p_b_given_rel
  ```

  producing `P(score | RELEVANT)` and `P(score | IRRELEVANT)` up to a
  common scale factor.

These likelihoods are passed into `BayesConditional`, which computes the
posterior probabilities, and `calibrate(score)` returns:

```python
P(RELEVANT | score)
```

This is the probability that will become `BernoulliDocumentDistribution.probability`.

---

## Clustering documents

### `BernoulliClusterer`

File: `bernoulli_clusterer.py`

```python
class BernoulliClusterer:
    def __init__(self, are_documents_similar: Callable[[Any, Any], bool]) -> None:
        ...
```

The clusterer groups document‑level Bernoulli distributions into clusters
using a user‑provided similarity function `are_documents_similar(doc_a, doc_b)`.

Workflow:

1. `__cluster_distributions(bernoulli_distributions)`
   * Iterates through all Bernoulli distributions.
   * For each distribution, tries to place it in an existing cluster that
     already contains at least one “similar” document.
   * If no such cluster is found, it creates a new cluster.

2. `build_cluster_bernoullis(bernoulli_distributions)`
   * Calls `__cluster_distributions`.
   * For each cluster:
     * Computes the probability that **at least one document is relevant**:

       ```python
       P(cluster relevant) = 1 - Π_j (1 - p_j)
       ```

     * Clips this probability to the `[0, 1]` interval.
     * Chooses a **representative document** in the cluster: the one with
       the highest individual probability `p_j`.
     * Creates a new `BernoulliDocumentDistribution` with:
       * `probability = P(cluster relevant)`
       * `document = representative.document`
       * `score = representative.score`

   * Returns a flat list of **cluster‑level** Bernoulli distributions.

This allows you to de‑duplicate near‑identical documents and treat each
cluster as a single “experiment” when estimating how many **distinct**
relevant results you have.

---

## Final statistics container

### `QueryRelevanceStats`

File: `query_relevance_stats.py`

```python
@dataclass
class QueryRelevanceStats:
    bernoullis: List[BernoulliDocumentDistribution]
    binomial: BinomialDocumentDistribution
    poisson: PoissonDistribution
```

The object `QueryRelevanceStats` is the final output of the pipeline.
It stores:

* `bernoullis` — document‑ or cluster‑level Bernoulli distributions.
* `binomial` — aggregated `BinomialDocumentDistribution` built from
  `bernoullis`.
* `poisson` — `PoissonDistribution` approximating the distribution of the
  number of relevant documents.

It also exposes convenience properties:

* `expected_relevant_docs` — exact expectation

  ```python
  # N = Σ_i X_i, X_i ~ Bernoulli(p_i)
  E[N | info] = Σ_i p_i
  ```

  implemented via `__lambda_exact()` and `ConditionalExpectation.sum_of_indicators_given_probabilities`.

* `variance_relevant_docs` — exact variance

  ```python
  Var(N) = Σ_i p_i (1 - p_i)
  ```

* `prob_no_relevant` — probability that **none** of the documents is relevant

  ```python
  P(N = 0 | info) = Π_i (1 - p_i)
  ```

* `prob_at_least_one_relevant` — probability of at least one relevant document

  ```python
  P(N ≥ 1 | info) = 1 - P(N = 0 | info)
  ```

* `lambda_` — the `λ` parameter of the Poisson approximation

  ```python
  lambda_ = poisson.lamda
  ```

Note: `expected_relevant_docs` is computed **exactly** from the Bernoullis,
while `lambda_` comes from the Poisson approximation via the binomial
aggregate; in practice they should match.

---

## End‑to‑end workflow

### 1. Without clustering

Entry point: `PoissonRelevanceEstimator.estimate_from_scored_docs`

File: `poisson_relevance_estimator.py`

```python
class PoissonRelevanceEstimator:
    def __init__(self, calibrator: BayesianRelevanceCalibrator) -> None:
        self.__calibrator = calibrator

    def estimate_from_scored_docs(
        self,
        scored_docs: Sequence[tuple[Any, float]],
    ) -> QueryRelevanceStats:
        ...
```

Steps:

1. **Empty input** → `empty_stats()`
   * Builds `BinomialDocumentDistribution.from_bernoullis([])`.
   * Approximates to a Poisson distribution with `λ = 0`.
   * Returns `QueryRelevanceStats` with all parts degenerate.

2. **Non‑empty input**:
   * For each `(doc, score)`:
     * Compute `p = calibrator.calibrate(score)`.
     * Use `ConditionalExpectation.indicator_given_probability(p)`
       (essentially an identity) to represent `E[1_{doc is relevant}]`.
     * Build a `BernoulliDocumentDistribution(
           probability=p,
           document=doc,
           score=float(score),
       )`.
   * Collect all document‑level Bernoullis into a list `bernoullis`.

3. Build `BinomialDocumentDistribution`:

   ```python
   binomial = BinomialDocumentDistribution.from_bernoullis(bernoullis)
   ```

   This computes `n`, `p̄` and stores the original Bernoullis.

4. Approximate with Poisson:

   ```python
   binomial_to_poisson = BinomialToPoissonApproximation.from_binomial(binomial)
   poisson = binomial_to_poisson.poisson
   ```

5. Return:

   ```python
   QueryRelevanceStats(
       bernoullis=bernoullis,
       binomial=binomial,
       poisson=poisson,
   )
   ```

---

### 2. With clustering

Entry point: `PoissonRelevanceEstimator.estimate_with_clustering`

```python
def estimate_with_clustering(
    self,
    scored_docs: Sequence[tuple[Any, float]],
    are_documents_similar: Callable[[Any, Any], bool],
) -> QueryRelevanceStats:
    ...
```

Steps:

1. If `scored_docs` is empty → same as `empty_stats()`.
2. Otherwise, build document‑level Bernoullis exactly as in the non‑clustered
   case.
3. Create a `BernoulliClusterer` with the provided similarity function.
4. Build **cluster‑level Bernoullis**:

   ```python
   clusterer = BernoulliClusterer(are_documents_similar=are_documents_similar)
   cluster_bernoullis = clusterer.build_cluster_bernoullis(bernoullis)
   ```

5. Build a `BinomialDocumentDistribution` from the cluster‑level Bernoullis,
   approximate with Poisson, and wrap everything in `QueryRelevanceStats`:

   ```python
   binomial = BinomialDocumentDistribution.from_bernoullis(cluster_bernoullis)
   poisson = BinomialToPoissonApproximation.from_binomial(binomial).poisson

   return QueryRelevanceStats(
       bernoullis=cluster_bernoullis,
       binomial=binomial,
       poisson=poisson,
   )
   ```

In this mode, `QueryRelevanceStats.bernoullis` refers to **clusters**, not
individual raw documents.

---

## Example usage (pseudocode)

```python
from app.quality_analyze.query_relevance_stats.bayesian_relevance_calibrator import (
    BayesianRelevanceCalibrator,
)
from app.quality_analyze.query_relevance_stats.poisson_relevance_estimator import (
    PoissonRelevanceEstimator,
)

# 1. Prepare scored documents: (document, score)
scored_docs = [
    (doc1, 0.12),
    (doc2, 0.40),
    (doc3, 0.95),
    # ...
]

# 2. Create a calibrator
calibrator = BayesianRelevanceCalibrator(
    prior_relevant=0.5,
    temperature=1.0,
    score_threshold=None,          # or some tuned value
    lower_score_is_better=True,    # set False if higher score is better
)

# 3. Create the estimator
estimator = PoissonRelevanceEstimator(calibrator=calibrator)

# (a) Without clustering
stats = estimator.estimate_from_scored_docs(scored_docs)

# (b) With clustering (optional)
def are_documents_similar(doc_a, doc_b) -> bool:
    # your own similarity logic here
    return doc_a.url == doc_b.url

clustered_stats = estimator.estimate_with_clustering(
    scored_docs,
    are_documents_similar=are_documents_similar,
)

# 4. Consume the statistics
print("Expected relevant docs:", stats.expected_relevant_docs)
print("Variance:", stats.variance_relevant_docs)
print("P(no relevant docs):", stats.prob_no_relevant)
print("P(at least one relevant):", stats.prob_at_least_one_relevant)
```

---

## Notes and assumptions

* Individual Bernoulli trials (documents or clusters) are treated as
  **independent**.
* The Poisson approximation relies on the usual conditions for approximating
  a Binomial with a Poisson (many trials, small probabilities, moderate
  expected count).
* Clustering is purely heuristic and depends entirely on the user‑supplied
  `are_documents_similar` function.
* The calibration quality depends on good choices for
  `prior_relevant`, `temperature`, and `score_threshold`.
