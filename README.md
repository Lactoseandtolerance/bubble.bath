# Brain Fart API

An open-source question generation, validation, and grading service powered by open-source LLMs. Generates intellectually challenging questions, verifies their answerability against curated academic sources, and grades free-form responses with confidence scoring.

---

## Overview

The Brain Fart is not a custom-trained model. It is an **orchestration layer** — a service that wraps existing open-source LLMs with structured prompt engineering, retrieval-augmented generation (RAG), and a multi-stage validation pipeline. The intelligence is in the harness, not the weights.

The underlying model can be swapped as the open-source ecosystem evolves. What makes this service valuable is the validation pipeline that ensures generated questions are answerable, sources are verified, and player answers are graded with transparent confidence.

Built as a standalone, independently deployable API. Originally developed for the [hard.think](https://hard.think) game but designed to be consumed by any project needing reliable, source-verified question generation and answer evaluation.

---

## Core Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Brain Fart API                        │
│                                                              │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────┐  │
│  │   Stage 1   │──▶│   Stage 2    │──▶│     Stage 3      │  │
│  │ Constrained │   │   Source     │   │  Answer Grading   │  │
│  │ Generation  │   │ Verification │   │  + Confidence     │  │
│  └─────────────┘   └──────────────┘   └──────────────────┘  │
│        │                  │                    │              │
│        ▼                  ▼                    ▼              │
│   [Open-source      [Source APIs:        [Open-source        │
│    LLM via           Wikipedia,          LLM comparison      │
│    Ollama/API]       Scholarpedia,       task]               │
│                      SEP, etc.]                              │
└──────────────────────────────────────────────────────────────┘
```

---

## Three-Stage Validation Pipeline

### Stage 1 — Constrained Generation

The LLM generates a question with enforced structured metadata:

```json
{
  "question": "What mathematical framework did Emmy Noether develop that fundamentally links symmetry and conservation laws in physics?",
  "answer": "Noether's theorem",
  "domain": ["mathematics", "physics"],
  "difficulty": 7,
  "source_path": {
    "source": "wikipedia",
    "article": "Noether's_theorem",
    "section": "Statement"
  }
}
```

**Required fields:** question, answer, domain(s), difficulty rating, source path.

**Rejection criteria:** If the model cannot provide a source path, the question is discarded before reaching Stage 2. No source claim → no question served.

### Stage 2 — Source Verification (RAG)

The claimed source path is verified against the actual source content:

1. Call the approved source API (e.g., Wikipedia MediaWiki API) to retrieve the specified article/section
2. Run a second LLM call with the prompt: *"Does this passage support this answer to this question?"*
3. The LLM returns a verification judgment (confirmed / not confirmed / partially confirmed)
4. **Not confirmed → question is rejected and never served**
5. **Partially confirmed → flagged for manual review or prompt refinement**

This is the critical reliability layer. The key insight: the model is not being asked to *know* things — it's being asked to *compare* things. A constrained comparison task is significantly more reliable than open-ended knowledge recall.

### Stage 3 — Answer Grading with Confidence Scoring

When a player submits an answer:

1. The LLM compares the player's submission against the known correct answer
2. Returns a structured grading response:

```json
{
  "judgment": "correct",
  "confidence": 0.92,
  "explanation": "The player's answer matches the expected answer.",
  "canonical_answer": "Noether's theorem"
}
```

**Confidence thresholds:**
- **High confidence (≥ 0.85):** Auto-resolved as correct or incorrect
- **Low confidence (< 0.85):** Flagged as ambiguous. Surfaced to the player as: *"Your answer might be correct — here's what we were looking for"*

**The confidence score serves three purposes:**
- **Gameplay:** Graceful handling of ambiguous, partial, or alternatively-phrased answers
- **Scoring:** Confidence level feeds into the Wheel of Knowledge scoring formula
- **Improvement:** Low-confidence cases are logged for review, enabling iterative prompt refinement and potential future fine-tuning

---

## API Endpoints

### `POST /api/question/generate`

Generate a validated question.

**Request body:**
```json
{
  "domain": "science",
  "difficulty_range": [5, 8],
  "exclude_ids": ["q_abc123", "q_def456"]
}
```

**Parameters:**
- `domain` (optional) — Target knowledge domain. If omitted, domain is chosen randomly
- `difficulty_range` (optional) — Min/max difficulty (1–10 scale). Default: [3, 7]
- `exclude_ids` (optional) — Array of previously served question IDs to avoid repeats

**Response:**
```json
{
  "question_id": "q_ghi789",
  "question": "What mathematical framework did Emmy Noether develop that fundamentally links symmetry and conservation laws in physics?",
  "domain": ["mathematics", "physics"],
  "difficulty": 7,
  "approved_sources": [
    {
      "name": "Wikipedia",
      "url": "https://en.wikipedia.org/wiki/Noether%27s_theorem"
    }
  ],
  "generated_at": "2025-01-15T22:30:00Z"
}
```

**Note:** The correct answer and source path are *not* returned to the client. They are stored server-side for grading.

**Error cases:**
- `503` — LLM unavailable or generation failed after retries
- `422` — Invalid domain or difficulty range

---

### `POST /api/question/grade`

Grade a player's answer.

**Request body:**
```json
{
  "question_id": "q_ghi789",
  "player_answer": "Noether's first theorem",
  "time_elapsed_seconds": 145
}
```

**Response:**
```json
{
  "question_id": "q_ghi789",
  "judgment": "correct",
  "confidence": 0.88,
  "canonical_answer": "Noether's theorem",
  "explanation": "The player's answer is substantially correct. Noether's first theorem is the commonly referenced result.",
  "domain": ["mathematics", "physics"],
  "difficulty": 7,
  "source": {
    "name": "Wikipedia",
    "url": "https://en.wikipedia.org/wiki/Noether%27s_theorem",
    "relevant_section": "Statement"
  }
}
```

**Response fields:**
- `judgment` — `correct`, `incorrect`, or `ambiguous`
- `confidence` — Float 0–1. Below 0.85 triggers `ambiguous` judgment
- `canonical_answer` — The expected correct answer (revealed after grading)
- `explanation` — Brief explanation of the grading decision
- `source` — The verified source, for player reference after answering

---

### `GET /api/question/:id`

Retrieve metadata for a previously generated question (post-grading only — does not expose the answer before grading).

**Response:**
```json
{
  "question_id": "q_ghi789",
  "question": "What mathematical framework did Emmy Noether develop...",
  "domain": ["mathematics", "physics"],
  "difficulty": 7,
  "generated_at": "2025-01-15T22:30:00Z",
  "times_served": 12,
  "average_confidence": 0.81,
  "correct_rate": 0.42
}
```

---

### `GET /api/domains`

List all available knowledge domains.

**Response:**
```json
{
  "domains": [
    {
      "id": "science",
      "label": "Natural Sciences",
      "description": "Physics, chemistry, biology, earth sciences"
    },
    {
      "id": "mathematics",
      "label": "Mathematics",
      "description": "Pure and applied mathematics, logic, statistics"
    },
    {
      "id": "philosophy",
      "label": "Philosophy",
      "description": "Ethics, epistemology, metaphysics, logic, political philosophy"
    },
    {
      "id": "history",
      "label": "History",
      "description": "World history, civilizations, historical events and figures"
    },
    {
      "id": "technology",
      "label": "Technology & Computing",
      "description": "Computer science, engineering, information technology"
    },
    {
      "id": "arts",
      "label": "Arts & Literature",
      "description": "Visual arts, music, literature, cultural studies"
    },
    {
      "id": "social_sciences",
      "label": "Social Sciences",
      "description": "Psychology, sociology, economics, political science"
    },
    {
      "id": "geography",
      "label": "Geography & Earth",
      "description": "Physical geography, geopolitics, climate, ecology"
    }
  ]
}
```

**Note:** Domain list is provisional (6–8 categories). Final categorization is an open design question.

---

### `GET /api/sources`

List all approved sources in the corpus.

**Response:**
```json
{
  "sources": [
    {
      "id": "wikipedia",
      "name": "Wikipedia",
      "api": "MediaWiki API",
      "url": "https://en.wikipedia.org",
      "status": "active"
    },
    {
      "id": "scholarpedia",
      "name": "Scholarpedia",
      "url": "http://www.scholarpedia.org",
      "status": "active"
    },
    {
      "id": "sep",
      "name": "Stanford Encyclopedia of Philosophy",
      "url": "https://plato.stanford.edu",
      "status": "active"
    }
  ]
}
```

---

## Model Layer

### Primary Strategy: Open-Source LLMs

The Brain Fart prioritizes open-source models for cost control, transparency, and community alignment.

**Candidates (to be benchmarked at implementation time):**
- Llama 3 (Meta) — strong general-purpose instruction following
- Mistral / Mixtral — competitive performance, efficient inference
- Other emerging open-source models at time of development

**Serving options:**
- **Local development:** Ollama (simple local model serving)
- **Production:** Hosted open-source inference provider (e.g., Together AI, Replicate, Groq) or self-hosted with vLLM/TGI

**Selection criteria:**
- Strong instruction-following for structured output generation
- Low hallucination rate on factual content
- Active community support and ongoing development
- Cost — architecture should function within free/low-cost inference tiers where possible

### Model Abstraction Layer

The service should abstract the model layer behind a consistent interface so the underlying model can be swapped without changing the validation pipeline, prompts, or API contracts.

```
┌────────────────────────┐
│   Model Interface      │
│   generate(prompt)     │
│   compare(a, b)        │
│   verify(passage, q&a) │
├────────────────────────┤
│   OllamaProvider       │  ← local dev
│   TogetherProvider     │  ← production option A
│   ReplicateProvider    │  ← production option B
│   AnthropicProvider    │  ← fallback / comparison
└────────────────────────┘
```

---

## Approved Source Corpus

### Initial Sources
| Source | API | Use Case |
|--------|-----|----------|
| Wikipedia | MediaWiki API | Broad knowledge base across all domains |
| Scholarpedia | Direct access | Peer-reviewed science and mathematics |
| Stanford Encyclopedia of Philosophy | Direct access | Philosophy, logic, ethics |

### Planned Expansions
| Source | Status | Use Case |
|--------|--------|----------|
| PubMed abstracts | Future | Medical and life sciences |
| Khan Academy references | Future | Educational mathematics and science |
| MIT OpenCourseWare | Future | Advanced STEM topics |
| Project Gutenberg | Future | Literature and humanities |

### Source Integration Interface

Each source needs an adapter that implements:
```
{
  search(query) → results[]
  getArticle(id) → full_text
  getSection(id, section) → section_text
}
```

Source adapters should handle API rate limits, caching, and error recovery independently.

---

## Configuration

```env
# .env.example

# Model Configuration
LLM_PROVIDER=ollama                  # ollama | together | replicate | anthropic
LLM_MODEL=llama3                     # Model identifier for chosen provider
LLM_API_KEY=<key_if_needed>          # API key for hosted providers
LLM_BASE_URL=http://localhost:11434  # For Ollama or self-hosted

# Validation Pipeline
GENERATION_MAX_RETRIES=3             # Max attempts to generate a valid question
VERIFICATION_THRESHOLD=0.80          # Minimum verification confidence to pass Stage 2
GRADING_AMBIGUITY_THRESHOLD=0.85     # Below this confidence → ambiguous judgment

# Source APIs
WIKIPEDIA_API_URL=https://en.wikipedia.org/w/api.php
SOURCE_CACHE_TTL_HOURS=24            # Cache source content to reduce API calls

# Rate Limiting
MAX_QUESTIONS_PER_HOUR=30            # Per-client question generation limit

# Logging
LOG_LEVEL=info
LOG_LOW_CONFIDENCE=true              # Log all low-confidence grading cases for review

# Server
PORT=5000
```

---

## Data Model

### Generated Question Record
```
{
  question_id:       string (UUID)
  question_text:     string
  canonical_answer:  string
  domain:            string[]
  difficulty:        integer (1–10)
  source_path: {
    source_id:       string
    article_id:      string
    section:         string
  }
  verification: {
    verified:        boolean
    confidence:      float
    verified_at:     timestamp
  }
  generated_at:      timestamp
  times_served:      integer
  grading_stats: {
    total_graded:    integer
    correct_count:   integer
    avg_confidence:  float
  }
}
```

### Grading Event Record
```
{
  event_id:          string (UUID)
  question_id:       string (foreign key)
  player_id:         string (from Bubble Bath identity)
  player_answer:     string
  judgment:          string (correct | incorrect | ambiguous)
  confidence:        float
  time_elapsed_sec:  integer
  graded_at:         timestamp
}
```

### Low-Confidence Log
```
{
  event_id:          string (foreign key to grading event)
  question_id:       string
  player_answer:     string
  canonical_answer:  string
  confidence:        float
  reviewed:          boolean
  review_outcome:    string (null until reviewed)
}
```

---

## Project Structure

```
brain-fart/
├── README.md
├── package.json
├── .env.example
├── src/
│   ├── index.js                    # Entry point / server setup
│   ├── config/
│   │   └── settings.js             # Environment and configuration
│   ├── routes/
│   │   ├── question.js             # Question generation and retrieval endpoints
│   │   ├── grading.js              # Answer grading endpoint
│   │   ├── domains.js              # Domain listing endpoint
│   │   └── sources.js              # Source listing endpoint
│   ├── pipeline/
│   │   ├── stage1_generation.js    # Constrained question generation
│   │   ├── stage2_verification.js  # Source verification (RAG)
│   │   ├── stage3_grading.js       # Answer grading with confidence
│   │   └── prompts/
│   │       ├── generation.js       # Generation prompt templates
│   │       ├── verification.js     # Verification prompt templates
│   │       └── grading.js          # Grading prompt templates
│   ├── models/
│   │   ├── question.js             # Question data model
│   │   └── gradingEvent.js         # Grading event data model
│   ├── providers/
│   │   ├── modelInterface.js       # Abstract model interface
│   │   ├── ollamaProvider.js       # Ollama integration
│   │   ├── togetherProvider.js     # Together AI integration
│   │   └── anthropicProvider.js    # Anthropic fallback
│   ├── sources/
│   │   ├── sourceInterface.js      # Abstract source adapter
│   │   ├── wikipedia.js            # Wikipedia MediaWiki adapter
│   │   ├── scholarpedia.js         # Scholarpedia adapter
│   │   └── sep.js                  # Stanford Encyclopedia adapter
│   └── utils/
│       ├── cache.js                # Source content caching
│       ├── logging.js              # Event and low-confidence logging
│       └── validation.js           # Input validation
├── tests/
│   ├── pipeline/
│   │   ├── generation.test.js
│   │   ├── verification.test.js
│   │   └── grading.test.js
│   ├── providers/
│   │   └── ollama.test.js
│   └── sources/
│       └── wikipedia.test.js
└── docs/
    ├── pipeline-architecture.md    # Detailed pipeline documentation
    ├── prompt-engineering.md       # Prompt design rationale and iterations
    ├── model-benchmarks.md         # Model comparison results (TBD)
    └── source-integration.md       # Source adapter documentation
```

---

## Prompt Engineering

Prompt design is central to the Brain Fart's reliability. All prompts live in `src/pipeline/prompts/` and follow these principles:

- **Structured output enforcement:** Every prompt specifies exact JSON output format
- **Constraint-first:** Prompts tell the model what it *cannot* do before what it should do
- **Source grounding:** Generation prompts require source attribution as a mandatory field
- **Comparison framing:** Verification and grading prompts frame tasks as comparison (not knowledge recall)
- **Iterative refinement:** Prompts are versioned. Low-confidence logs drive prompt improvements over time

**Status:** Prompt templates to be developed during implementation. This is expected to be the most iteration-heavy part of the project.

---

## Open Questions & Future Exploration

### Model Selection & Benchmarking
Specific model choice deferred to implementation. Requires benchmarking candidates on:
- Structured output reliability (does it consistently produce valid JSON?)
- Factual accuracy on knowledge-domain questions
- Comparison task accuracy (Stage 2 and Stage 3 reliability)
- Inference speed and cost at expected query volume

**Status:** Awaiting implementation phase. Will document results in `docs/model-benchmarks.md`.

### Source API Rate Limits & Caching
Wikipedia and other source APIs have rate limits. The caching strategy (TTL, invalidation, storage) needs specification. High-traffic gameplay could exceed free-tier limits.

**Status:** Needs research during implementation.

### Cross-Domain Question Handling
Questions that span multiple domains (e.g., philosophy of mathematics, history of science) need clear attribution rules for how scoring weight is distributed across domains.

**Status:** Design decision pending. Impacts Wheel of Knowledge scoring in Service 3.

### Fine-Tuning on Gameplay Data
Long-term possibility: use logged question-answer pairs, especially low-confidence cases with manual review outcomes, to create a curated fine-tuning dataset. This could improve grading accuracy over time.

**Status:** Long-term goal. Requires significant logged data volume first.

### Question Deduplication & Quality Scoring
As the question bank grows, need strategies for detecting semantically duplicate questions and scoring question quality based on player engagement metrics (skip rate, average confidence, etc.).

**Status:** Future feature.

### Difficulty Calibration
Initial difficulty ratings are LLM-estimated. Over time, actual player performance data should recalibrate difficulty ratings empirically.

**Status:** Designed conceptually. Implementation after sufficient play data.

---

## Contributing

This project is open-source. Contributions are welcome, particularly in:
- Prompt engineering and validation pipeline improvements
- Source adapter implementations for new academic sources
- Model provider integrations
- Benchmarking and reliability testing
- Question quality analysis tooling

---

## License

TBD — will be open-source. License selection pending.
