# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Brain Fart API** — an orchestration layer (not a custom model) that wraps open-source LLMs with structured prompt engineering and RAG to generate, verify, and grade trivia-style questions. Originally built for the hard.think game but designed as a standalone API.

The repository is currently in the **specification phase**. The README.md contains the full architecture spec, API contracts, and data models. No source code has been implemented yet.

## Planned Tech Stack

- **Runtime:** Node.js with Express.js
- **LLM Serving:** Ollama (local dev), Together AI / Replicate / Groq (production)
- **Target Models:** Llama 3, Mistral/Mixtral, or best available open-source model at implementation time
- **Source APIs:** Wikipedia (MediaWiki API), Scholarpedia, Stanford Encyclopedia of Philosophy
- **Server Port:** 5000 (default)

## Core Architecture: Three-Stage Validation Pipeline

This is the central design pattern. Every question flows through all three stages sequentially:

1. **Stage 1 — Constrained Generation:** LLM generates a question with enforced structured JSON metadata (question, answer, domain, difficulty, source_path). No source claim = question discarded.
2. **Stage 2 — Source Verification (RAG):** Fetches the claimed source via API, then a second LLM call compares the source content against the question/answer pair. Not confirmed = question rejected.
3. **Stage 3 — Answer Grading:** LLM compares player answer against canonical answer. Returns judgment + confidence score. Confidence >= 0.85 auto-resolves; < 0.85 = ambiguous.

**Key design principle:** The LLM is always used for *comparison tasks*, never knowledge recall. This is deliberate — comparison is more reliable than open-ended generation.

## Project Structure (Planned)

```
src/
├── index.js                     # Express server entry point
├── config/settings.js           # Env-based configuration
├── routes/                      # Express route handlers
├── pipeline/                    # Three-stage validation pipeline
│   ├── stage1_generation.js
│   ├── stage2_verification.js
│   ├── stage3_grading.js
│   └── prompts/                 # Prompt templates (generation, verification, grading)
├── providers/                   # LLM provider abstraction layer
│   ├── modelInterface.js        # Abstract interface: generate(), compare(), verify()
│   ├── ollamaProvider.js
│   ├── togetherProvider.js
│   └── anthropicProvider.js
├── sources/                     # Source adapter abstraction layer
│   ├── sourceInterface.js       # Abstract interface: search(), getArticle(), getSection()
│   ├── wikipedia.js
│   ├── scholarpedia.js
│   └── sep.js
└── utils/                       # Cache, logging, validation
tests/
├── pipeline/
├── providers/
└── sources/
```

## Key Abstraction Layers

- **Model providers** implement a common interface (`generate`, `compare`, `verify`) so the underlying LLM can be swapped without changing the pipeline or API contracts.
- **Source adapters** implement a common interface (`search`, `getArticle`, `getSection`) with independent rate limiting, caching, and error recovery.

## API Endpoints

- `POST /api/question/generate` — Generate a validated question (domain, difficulty_range, exclude_ids)
- `POST /api/question/grade` — Grade a player answer (question_id, player_answer, time_elapsed)
- `GET /api/question/:id` — Retrieve question metadata (post-grading only; answer not exposed pre-grading)
- `GET /api/domains` — List knowledge domains (8 categories)
- `GET /api/sources` — List approved source corpus

## Prompt Engineering Conventions

Prompts live in `src/pipeline/prompts/` and must follow these principles:
- Enforce exact JSON output format in every prompt
- Constraints before instructions (tell the model what it *cannot* do first)
- Source attribution is a mandatory field in generation prompts
- Verification and grading prompts frame tasks as comparison, not recall
- Prompts are versioned; low-confidence logs drive iterative refinement

## Configuration

Uses `.env` files. Key thresholds:
- `VERIFICATION_THRESHOLD=0.80` — minimum confidence to pass Stage 2
- `GRADING_AMBIGUITY_THRESHOLD=0.85` — below this = ambiguous judgment
- `GENERATION_MAX_RETRIES=3` — max attempts per question generation
- `SOURCE_CACHE_TTL_HOURS=24` — source content cache duration

## Important Design Decisions

- Correct answers are **never returned to clients** during generation — only after grading
- Low-confidence grading cases are logged separately for review and prompt improvement
- The 8 knowledge domains: science, mathematics, philosophy, history, technology, arts, social_sciences, geography
- Questions can span multiple domains (cross-domain attribution rules are an open design question)
