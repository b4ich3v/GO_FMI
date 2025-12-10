# Query Relevance Stats

Small toolkit that answers the question:

> “Given a list of documents with scores, how many of them are **actually relevant** to the query?”

The code does this by:
- turning each `(document, score)` into a probability that the document is relevant;
- optionally **clustering** similar documents;
- combining all probabilities into:
  - a **Binomial** model, and
  - a **Poisson** model (approximation);
- returning a `QueryRelevanceStats` object with convenient properties.

The logic in this folder is deliberately thin and reuses the generic
probability utilities from the rest of the `app.quality_analyze` package.

---

## Files in `query_relevance_stats/`

- **`documentation.pdf`** – human‑oriented write‑up with more mathematical detail.
- **`bayesian_relevance_calibrator.py`** – converts a raw score into `P(relevant | score)`.
- **`bernoulli_clusterer.py`** – clusters similar documents and builds cluster‑level Bernoulli distributions.
- **`poisson_relevance_estimator.py`** – end‑to‑end pipeline from `(document, score)` to `QueryRelevanceStats`.
- **`query_relevance_stats.py`** – definition of the `QueryRelevanceStats` dataclass and its convenience properties.
- **`specified_discrete_variables/bernoulli_document_distribution.py`** – Bernoulli distribution + attached document metadata.
- **`specified_discrete_variables/binomial_document_distribution.py`** – Binomial distribution that remembers its component Bernoullis.

The rest of the project provides shared building blocks:

- `constants.py` – numeric and probability constants (`PROBABILITY_ONE`, `PROBABILITY_ZERO`, priors, tolerances, etc.).
- `discrete_variables/*` – generic Bernoulli / Binomial / Poisson distributions and approximations.
- `conditional_probability/*` – Bayes theorem utilities (`EventHypothesis`, `CompleteEventGroup`, `BayesConditional`).
- `contidional_exceptation/conditional_expectation.py` – helper functions for expectations.

---

## Core data structures

### `BernoulliDocumentDistribution`

File: `specified_discrete_variables/bernoulli_document_distribution.py`

Extends the generic `BernoulliDistribution` and adds document metadata:

- `probability: float` – `P(document is relevant | score)`;
- `document: Any | None` – whatever you use to represent a document;
- `score: float | None` – the original model score.

From `BernoulliDistribution` it inherits:

- `expected()` – returns `probability`;
- `variance()` – returns `probability * (1 - probability)`;
- `pmf(k)` – probability of success (1) or failure (0).

Each instance represents **one document (or cluster)** treated as a Bernoulli trial.

---

### `BinomialDocumentDistribution`

File: `specified_discrete_variables/binomial_document_distribution.py`

Extends the generic `BinomialDistribution` with:

- `bernoullis: Sequence[BernoulliDistribution]` – the underlying trials.

Factory:

```python
BinomialDocumentDistribution.from_bernoullis(bernoullis)
```

- If the sequence is empty:
  - returns a distribution with `number_of_experiments = 0`,
  - `probability = 0.0`,
  - and `bernoullis = ()`.
- Otherwise:
  - computes `lambda_total = sum(b.probability for b in bernoullis)`,
  - sets `p̄ = lambda_total / len(bernoullis)`,
  - returns a Binomial with `number_of_experiments = len(bernoullis)` and `probability = p̄`,
  - and keeps a tuple of the Bernoullis.

This is the “summary” distribution used before switching to Poisson.

---

### `QueryRelevanceStats`

File: `query_relevance_stats.py`

Simple dataclass that is the **final result** of the workflow:

```python
@dataclass
class QueryRelevanceStats:
    bernoullis: List[BernoulliDocumentDistribution]
    binomial: BinomialDocumentDistribution
    poisson: PoissonDistribution
```

It stores:

- `bernoullis` – document‑ or cluster‑level Bernoulli distributions;
- `binomial` – `BinomialDocumentDistribution` built from those Bernoullis;
- `poisson` – `PoissonDistribution` approximating the count of relevant docs.

Convenience properties:

- `lambda_` – Poisson rate parameter (`poisson.lamda`).
- `expected_relevant_docs` – exact expected count of relevant docs:

  - internally calls `__lambda_exact()`;
  - uses `ConditionalExpectation.sum_of_indicators_given_probabilities`.

- `variance_relevant_docs` – exact variance:

  - computed as `sum(b.variance() for b in self.bernoullis)`.

- `prob_no_relevant` – probability that **no** document is relevant:

  - multiplies `(1 - p_i)` for all Bernoullis.

- `prob_at_least_one_relevant` – probability of at least one relevant doc:

  - `1 - prob_no_relevant`.

---

## Calibrating scores → probabilities

### `BayesianRelevanceCalibrator`

File: `bayesian_relevance_calibrator.py`

Purpose: convert a raw score into a probability that a document is relevant.

Key ideas:

- Define two hypotheses using `EventHypothesis`:
  - `"RELEVANT"` with prior `prior_relevant`,
  - `"IRRELEVANT"` with prior `1 - prior_relevant`.
- Build a `CompleteEventGroup` to normalize the priors.
- Use `BayesConditional` to later compute the posterior probabilities.

Constructor (simplified):

```python
BayesianRelevanceCalibrator(
    prior_relevant: float = DEFAULT_RELEVANCE_PRIOR,
    temperature: float = DEFAULT_CALIBRATOR_TEMPERATURE,
    score_threshold: float | None = None,
    lower_score_is_better: bool = True,
)
```

Parameters:

- `prior_relevant` – prior belief that a random document is relevant.
- `temperature` – smoothness of the transition around the threshold
  (smaller → steeper curve).
- `score_threshold` – score where relevance and irrelevance are about 50/50;
  `None` falls back to `CALIBRATOR_DEFAULT_SCORE_THRESHOLD`.
- `lower_score_is_better` – if `True`, lower scores are considered better.

Internal steps in `calibrate(score)`:

1. Compute a **normalized margin** between the score and the threshold,
   with sign depending on `lower_score_is_better`, divided by `temperature`.
2. Clamp the margin to `[-CALIBRATOR_MAX_NORMALIZED_MARGIN_ABS, +CALIBRATOR_MAX_NORMALIZED_MARGIN_ABS]`.
3. Apply a logistic function to get a likelihood for `"RELEVANT"`:

   ```python
   sigma(x) = 1 / (1 + exp(-x))
   ```

4. Derive the likelihood for `"IRRELEVANT"` as `1 - sigma(x)`.
5. Feed both likelihoods into `BayesConditional` and return the posterior
   probability of `"RELEVANT"`:

   ```python
   # P(RELEVANT | score)
   p = calibrator.calibrate(score)
   ```

This `p` is the probability used in `BernoulliDocumentDistribution.probability`.

---

## Clustering documents

### `BernoulliClusterer`

File: `bernoulli_clusterer.py`

Purpose: group “similar enough” documents and treat each group as **one**
Bernoulli trial.

Constructor:

```python
BernoulliClusterer(are_documents_similar: Callable[[Any, Any], bool])
```

You provide a function that says whether two documents belong to the same cluster.

Internal methods:

- `__cluster_distributions(bernoulli_distributions)`:
  - iterates over document‑level Bernoullis;
  - for each one, tries to place it into an existing cluster where at least one
    document is “similar” according to `are_documents_similar`;
  - otherwise starts a new cluster;
  - returns a list of clusters (each a list of Bernoullis).

- `build_cluster_bernoullis(bernoulli_distributions)`:
  - calls `__cluster_distributions`;
  - for each cluster:
    - if the cluster is non‑empty:
      - computes the probability that **none** of the documents is relevant
        as a product of `(1 - p_i)`;
      - sets

        ```python
        cluster_probability = 1 - prob_all_not_relevant
        ```

      - clips this to `[0.0, 1.0]`;
    - picks a representative document – the one with the **highest** probability;
    - creates a new `BernoulliDocumentDistribution` with:
      - `probability = cluster_probability`,
      - `document = representative.document`,
      - `score = representative.score`.
  - returns a flat list of cluster‑level Bernoullis.

If there are no clusters (empty input), it returns an empty list.

---

## End‑to‑end pipeline

### `PoissonRelevanceEstimator`

File: `poisson_relevance_estimator.py`

This class wires everything together.

Constructor:

```python
estimator = PoissonRelevanceEstimator(calibrator=BayesianRelevanceCalibrator(...))
```

Helper:

- `__bernoullis_from_scored_docs(scored_docs)`:
  - input is `Sequence[tuple[Any, float]]` (document, score);
  - for each `(doc, score)`:
    - compute `p = calibrator.calibrate(score)`;
    - call `ConditionalExpectation.indicator_given_probability(p)`
      to get the expected value of the indicator variable;
    - build a `BernoulliDocumentDistribution(probability=p, document=doc, score=float(score))`;
  - returns a list of Bernoullis.

Shared “empty” case:

- `empty_stats()`:
  - builds a `BinomialDocumentDistribution` from an empty list of Bernoullis;
  - uses `BinomialToPoissonApproximation.from_binomial` to create a Poisson
    with `lambda_ = 0` (because the expectation is 0);
  - returns a `QueryRelevanceStats` with:
    - `bernoullis = []`,
    - `binomial` degenerate at 0,
    - `poisson` degenerate with `lamda = 0`.

#### 1. Estimation **without** clustering

```python
estimate_from_scored_docs(
    scored_docs: Sequence[tuple[Any, float]],
) -> QueryRelevanceStats
```

Steps:

1. If `scored_docs` is empty → return `empty_stats()`.
2. Build document‑level Bernoullis with `__bernoullis_from_scored_docs`.
3. Build a `BinomialDocumentDistribution` from them:

   ```python
   binomial = BinomialDocumentDistribution.from_bernoullis(bernoullis)
   ```

4. Approximate the Binomial with a Poisson:

   ```python
   binomial_to_poisson = BinomialToPoissonApproximation.from_binomial(binomial)
   poisson = binomial_to_poisson.poisson
   ```

5. Return a `QueryRelevanceStats` with these three objects.

In this mode, `QueryRelevanceStats.bernoullis` contains **one entry per document**.

#### 2. Estimation **with** clustering

```python
estimate_with_clustering(
    scored_docs: Sequence[tuple[Any, float]],
    are_documents_similar: Callable[[Any, Any], bool],
) -> QueryRelevanceStats
```

Steps:

1. If `scored_docs` is empty → return `empty_stats()`.
2. Build document‑level Bernoullis with `__bernoullis_from_scored_docs`.
3. Create a `BernoulliClusterer` with your similarity function.
4. Build cluster‑level Bernoullis:

   ```python
   clusterer = BernoulliClusterer(are_documents_similar=are_documents_similar)
   cluster_bernoullis = clusterer.build_cluster_bernoullis(bernoullis)
   ```

5. Build a `BinomialDocumentDistribution` from `cluster_bernoullis`.
6. Approximate with Poisson using `BinomialToPoissonApproximation`.
7. Return a `QueryRelevanceStats` where:
   - `bernoullis` now represent **clusters** (not raw documents).

---

## Typical usage (short pseudocode)

```python
from app.quality_analyze.query_relevance_stats.bayesian_relevance_calibrator import BayesianRelevanceCalibrator
from app.quality_analyze.query_relevance_stats.poisson_relevance_estimator import PoissonRelevanceEstimator

scored_docs = [
    (doc1, 0.12),
    (doc2, 0.40),
    (doc3, 0.95),
]

calibrator = BayesianRelevanceCalibrator(
    prior_relevant=0.5,
    temperature=1.0,
    score_threshold=0.0,
    lower_score_is_better=True,
)

estimator = PoissonRelevanceEstimator(calibrator=calibrator)

# Without clustering
stats = estimator.estimate_from_scored_docs(scored_docs)

# Or with clustering
def are_documents_similar(a, b) -> bool:
    return getattr(a, "url", None) == getattr(b, "url", None)

clustered_stats = estimator.estimate_with_clustering(
    scored_docs,
    are_documents_similar=are_documents_similar,
)

print(stats.expected_relevant_docs)
print(stats.prob_no_relevant)
print(stats.prob_at_least_one_relevant)
print(stats.lambda_)
```

This is the entire workflow: scores → probabilities → (optional) clusters →
Bernoulli list → Binomial summary → Poisson approximation → `QueryRelevanceStats`.
