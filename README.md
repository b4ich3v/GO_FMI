# Query Relevance Stats

Тази папка съдържа малък, но завършен модул, който отговаря на въпроса:

> „Имам списък от документи с някакъв **score**. Колко от тях са **релевантни** към заявката и с каква вероятност?“

Идеята е:
- всеки `(document, score)` се превръща в **вероятност документът да е релевантен**;
- тези вероятности се разглеждат като Бернули променливи (успех = „документът е релевантен“);
- по желание се прави **клъстеризация** на подобни документи, за да не броим дубликати;
- от всички Бернули се строи биномиално и после пуасоново разпределение за броя релевантни документи;
- резултатът се събира в обект `QueryRelevanceStats`, от който лесно четем очакван брой, вариация, вероятност да няма нито един релевантен документ и т.н.

Целият модул ползва общи помощни класове от останалата част на `app.quality_analyze` за вероятности, очаквания и Байес.

---

## Груба схема на целия workflow

```text
(doc, score) списък
        │
        ▼
BayesianRelevanceCalibrator
(score → P(RELEVANT | score))
        │
        ▼
BernoulliDocumentDistribution
(по един за всеки документ)
        │
        ├─ (по избор) BernoulliClusterer
        │             (групира сходни документи → клъстерни Бернули)
        ▼
BinomialDocumentDistribution.from_bernoullis(...)
        │
        ▼
BinomialToPoissonApproximation
        │
        ▼
PoissonDistribution
        │
        ▼
QueryRelevanceStats
  - bernoullis (документи/клъстери)
  - binomial
  - poisson
  - метрики (очакване, вариация, P(0), P(≥1))
```

За един query това е „тръбопроводът“ от сурови резултати до статистики.

---

## Файлове и какво правят

Папка: `analyze/query_relevance_stats/`

- **`documentation.pdf`**  
  Човешка документация – математически обяснения и мотивация. Кодът тук е по-прагматичен и следва тези идеи.

- **`bayesian_relevance_calibrator.py`**  
  Отговаря за **преобразуването на score в вероятност за релевантност**.  
  Получава суров score от ранкера и връща число между 0 и 1: `P(RELEVANT | score)`.

- **`bernoulli_clusterer.py`**  
  Взима списък от Bernoulli-дистрибуции за отделни документи и ги **групира в клъстери** според подадена от потребителя функция за прилика. От всеки клъстер прави един нов Bernoulli, който описва „има ли поне един релевантен документ в този клъстер“.

- **`poisson_relevance_estimator.py`**  
  Това е **основният pipeline клас**.  
  Той свързва калибратора, документните Бернули, клъстеризацията (ако е включена), биномиалното и пуасоновото разпределение и накрая връща `QueryRelevanceStats`.

- **`query_relevance_stats.py`**  
  Съдържа датакласа `QueryRelevanceStats`, който е крайният контейнер с всички статистики за един query.

- **`specified_discrete_variables/bernoulli_document_distribution.py`**  
  Дефинира `BernoulliDocumentDistribution` – Бернули разпределение, в което освен вероятност има и **document** и **score**.

- **`specified_discrete_variables/binomial_document_distribution.py`**  
  Дефинира `BinomialDocumentDistribution` – биномиално разпределение, което освен параметрите си пази и **списъка от изходни Бернули** и има удобен конструктор `from_bernoullis(...)`.

Модулът стъпва върху:
- `discrete_variables.*` – общи имплементации на Bernoulli, Binomial, Poisson, плюс аппроксимации;
- `conditional_probability.*` – обща Байесова инфраструктура;
- `contidional_exceptation.conditional_expectation` – общи формули за очаквания;
- `constants` – числови и вероятностни константи (0, 1, толеранси, прагове).

---

## Какво означават основните обекти

### 1. `BernoulliDocumentDistribution` – един документ като „монета“

Файл: `specified_discrete_variables/bernoulli_document_distribution.py`

Това е обикновено Бернули разпределение с допълнителна информация:

- наследява `BernoulliDistribution` (има `probability`, `expected()`, `variance()` и т.н.);
- добавя:
  - `document` – самият документ (обект по твой избор);
  - `score` – суровият score, по който документът е бил подреден.

Интуитивно: *„Имаме документ X. Вероятността той да е релевантен е p. Хвърляме монета с шанс p за успех.“*  
Всеки документ или клъстер, с който работим по-нататък, е такава монета.

### 2. `BinomialDocumentDistribution` – събиране на много документи

Файл: `specified_discrete_variables/binomial_document_distribution.py`

Този клас:
- наследява общия `BinomialDistribution` (брой опити + шанс за успех);
- пази списък `bernoullis` – изходните Бернули;
- има класов метод `from_bernoullis(bernoullis)`:

  - ако списъкът е празен → връща биномиално с 0 опита и вероятност 0;
  - иначе:
    - сумира всички вероятности `p_i`;
    - дели на броя им → получава средна вероятност `p̄`;
    - създава биномиално разпределение с:
      - `number_of_experiments = len(bernoullis)`,
      - `probability = p̄`,
      - `bernoullis = tuple(bernoullis)`.

Това е компактният „обобщен“ изглед върху всички документи или клъстери.

### 3. `QueryRelevanceStats` – финалният резултат

Файл: `query_relevance_stats.py`

Това е датакласът, който събира крайния резултат за един query:

- `bernoullis`: списък `List[BernoulliDocumentDistribution]` – може да са по един на документ или по един на клъстер (при клъстеризация);
- `binomial`: `BinomialDocumentDistribution` – обобщеното биномиално разпределение;
- `poisson`: `PoissonDistribution` – Пуасонова аппроксимация за броя релевантни.

Вътре има две помощни частни функции:

- `__lambda_exact()` – изчислява **точното очакване** на броя релевантни документи като сума на вероятностите;
- `__prob_zero_exact()` – изчислява **точната вероятност** да няма нито един релевантен документ (произведение от `(1 - p_i)`).

Има и няколко property-та, които са удобният интерфейс навън:

- `lambda_` – λ на пуасоновото разпределение (директно от `poisson.lamda`);
- `expected_relevant_docs` – очакван брой релевантни документи (ползва `__lambda_exact`);
- `variance_relevant_docs` – вариация на броя релевантни (сума на вариациите от всеки Bernoulli);
- `prob_no_relevant` – вероятност **да няма нито един** релевантен документ;
- `prob_at_least_one_relevant` – вероятност да има **поне един** релевантен документ (1 – `prob_no_relevant`).

Така `QueryRelevanceStats` е лицето на модула: подаваш му се чрез `PoissonRelevanceEstimator`, а после четеш от него най-важните числа.

---

## Score → вероятност: `BayesianRelevanceCalibrator`

Файл: `bayesian_relevance_calibrator.py`

Този клас отговаря за най-важното преобразуване в началото на pipeline-а:

- вход: суров `score` за документ (каквото дава ранкерът);
- изход: `P(RELEVANT | score)` – вероятност документът да е релевантен.

Как го прави:

1. В конструктора се дефинират две хипотези чрез `EventHypothesis`:
   - `"RELEVANT"` с prior `prior_relevant`;
   - `"IRRELEVANT"` с prior `1 - prior_relevant`.
2. Тези хипотези се събират в `CompleteEventGroup`, после в `BayesConditional`.
3. За даден score:
   - изчислява се **нормализиран margin** спрямо `score_threshold` (със знак в зависимост от `lower_score_is_better`) и се дели на `temperature`;
   - margin-ът се ограничава в интервал `[-CALIBRATOR_MAX_NORMALIZED_MARGIN_ABS, +...]`;
   - през този margin се пуска логистична функция (σ), която дава „колко по-вероятно е да сме в RELEVANT“;
   - това става `p(score | RELEVANT)` в относителна скала, а допълнението е `p(score | IRRELEVANT)`;
   - тези две стойности влизат в `BayesConditional`, което връща постериорните вероятности.

Краен ефект: за всеки score имаме калибрирана, гладка и ограничена вероятност за релевантност, която се използва в `BernoulliDocumentDistribution`.

---

## Клъстеризация на документи: `BernoulliClusterer`

Файл: `bernoulli_clusterer.py`

Цел: ако имаме много почти идентични документи (напр. копия на една страница), да ги третираме като **един „клъстерен“ резултат**, а не да ги броим поотделно.

Работи така:

1. В конструктора му подаваш функция `are_documents_similar(doc_a, doc_b) -> bool`.
2. Методът `__cluster_distributions(...)` минава през всички Bernoulli-дистрибуции:
   - за текущото разпределение търси дали вече има клъстер, в който документът е „подобен“ на поне един от елементите;
   - ако намери → добавя го към този клъстер;
   - ако не → създава нов клъстер само с него.
3. Методът `build_cluster_bernoullis(...)`:
   - взима клъстерите;
   - за всеки клъстер:
     - ако е празен → прескача;
     - умножава `(1 - p_i)` за всички документи вътре и получава вероятност **никой** да не е релевантен;
     - от това прави `1 - ...` и така получава вероятност **поне един да е релевантен в клъстера**;
     - ако числото е извън [0, 1] заради числена грешка, го клипва;
     - избира представителния документ (този с най-голяма `probability`);
     - създава `BernoulliDocumentDistribution` за целия клъстер, с:
       - `probability = P(клъстерът съдържа релевантен документ)`,
       - `document = представителният документ`,
       - `score = score-а на представителния документ`.
   - връща списък от тези нови клъстерни Бернули.

Така на следващите стъпки работим със „смислени“ уникални резултати, а не с шум от дубликати.

---

## Крайният pipeline: `PoissonRelevanceEstimator`

Файл: `poisson_relevance_estimator.py`

Това е класът, който реално свързва всичко.

Има три основни части:

1. **Конструктор**  
   Взима инстанция на `BayesianRelevanceCalibrator` и я пази.

2. **Преобразуване на `(doc, score)` → Bernoulli**  
   Частният метод `__bernoullis_from_scored_docs(scored_docs)`:
   - `scored_docs` е списък от `(document, score)` двойки;
   - за всяка двойка:
     - вика `calibrator.calibrate(score)`, за да вземе вероятност `p`;
     - през `ConditionalExpectation.indicator_given_probability(p)` получава очакването на индикаторната променлива (което е същото число p);
     - създава `BernoulliDocumentDistribution` с:
       - `probability = p`,
       - `document = doc`,
       - `score = float(score)`;
     - добавя го в списъка.
   - връща списък от документни Бернули.

3. **Два начина за оценка: с и без клъстеризация**

   - `empty_stats()`  
     Ако няма документи:
     - строи биномиално разпределение от празен списък (0 опита, вероятност 0);
     - прави пуасонова аппроксимация (λ = 0);
     - връща `QueryRelevanceStats` с празен `bernoullis` и дегенератни `binomial` и `poisson`.

   - `estimate_from_scored_docs(scored_docs)` – **без клъстеризация**  
     - ако списъкът е празен → връща `empty_stats()`;
     - иначе:
       - прави списък от Bernoulli-дистрибуции чрез `__bernoullis_from_scored_docs`;
       - от тях строи `BinomialDocumentDistribution.from_bernoullis(...)`;
       - от биномиалното прави пуасонова аппроксимация (`BinomialToPoissonApproximation`);
       - създава `QueryRelevanceStats` със:
         - оригиналните Bernoulli-та (по едно на документ),
         - биномиалното разпределение,
         - пуасоновото разпределение.

   - `estimate_with_clustering(scored_docs, are_documents_similar)` – **с клъстеризация**  
     - ако списъкът е празен → връща `empty_stats()`;
     - иначе:
       - пак строи документните Бернули;
       - създава `BernoulliClusterer` със зададената функция `are_documents_similar`;
       - от него взима клъстерните Бернули;
       - от тях прави `BinomialDocumentDistribution` и после пуасонова аппроксимация;
       - връща `QueryRelevanceStats`, където:
         - `bernoullis` вече са **по клъстер**, а не по документ.

---

## Как да „четеш“ резултата

Накратко, за един query:

- подаваш списък `(doc, score)` на `PoissonRelevanceEstimator`;
- получаваш `QueryRelevanceStats stats`;
- от `stats` можеш да разбереш:
  - **колко релевантни документа очакваме** (`expected_relevant_docs`);
  - **колко несигурно е това число** (`variance_relevant_docs`);
  - **какъв е шансът да няма нито един релевантен документ** (`prob_no_relevant`);
  - **какъв е шансът да има поне един** (`prob_at_least_one_relevant`);
  - каква е **λ на пуасоновото разпределение** (`lambda_`), ако искаш да работиш директно с него;
  - ако гледаш по-дълбоко – цялото разпределение на броя релевантни (през `poisson` или `binomial`).

Това е реалният „живот“ на този модул: от сурови score-ове по документи до интуитивни, обясними статистики за качеството на резултатите за един query.
