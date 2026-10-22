# Cline Pass: Model Guide, Benchmarks, and CLI Control Reference

Research date: **2026-10-22** · Target environment: **Cline CLI v3.0.73 on Windows 11** (`provider`: `cline-pass`)

> **Lead's verification (Claude, 2026-10-22).** Researched by Antigravity (Gemini 4 Argon
> High). Checked live on this PC after the owner's `cline auth cline-pass`: these answered a
> test prompt - `cline-pass/glm-5.3`, `cline-pass/glm-5.3-flash`, `cline-pass/deepseek-v4-pro`,
> `cline-pass/kimi-k3` (each ~60 s for a one-line answer, mostly CLI start-up). A bare
> `glm-5.3` is rejected ("Expected format: modelType/model") - the `cline-pass/` prefix is
> required. `stealth/ox-alpha` is retired (404). The $9.99/month price, the 2-5x quota claim
> and the GLM / Kimi / DeepSeek / Qwen / MiniMax / MiMo families match independent pages
> ([cline.bot/cline-pass](https://cline.bot/cline-pass), [makerstack review](https://makerstack.co/reviews/clinepass-review/),
> [1devtool](https://1devtool.com/compare/clinepass-vs-openrouter)). **Benchmark percentages
> and reference prices below were NOT verified** - many cite only a site's home page; treat
> them as indicative. The other 8 model ids were not live-tested.
> HERDR profiles (agents.json): `cline` = glm-5.3, `cline-flash` = glm-5.3-flash,
> `deepseek` = deepseek-v4-pro, `kimi` = kimi-k3 - all through Cline Pass.

---

## 1. What Cline Pass is

**Cline Pass** (`cline-pass`, styled **ClinePass** in official docs) is an optional flat-rate monthly subscription from Cline (`cline.bot`) that provides managed, high-quota access to a curated roster of frontier open-weight and partner coding models through Cline's unified inference gateway (`https://api.cline.bot/api/v1`).

* **Price:** **$9.99 / month** (flat-rate recurring monthly subscription; occasional first-month promotional discounts have been offered, e.g., $4.99 first month).
* **What is included:**
  * Access to 12 curated active coding models across 7 model families (**Z.ai GLM**, **Moonshot Kimi**, **DeepSeek**, **Alibaba Qwen**, **Xiaomi MiMo**, **MiniMax**, and **Meta Muse Spark Contributor**) without managing separate provider accounts or API keys.
  * Works across the **VS Code / JetBrains extension**, **Cline Desktop**, **Cline CLI** (`-P cline-pass`), and external OpenAI-compatible tools via a `CLINE_API_KEY` generated at `https://app.cline.bot` (*Settings → API Keys*).
* **Usage limits, quotas, and fair use:**
  * **Quota-based, not unlimited:** Cline Pass advertises **2x to 5x the usage capacity** on supported open-weight models compared to buying equivalent spend at standard pay-as-you-go API rates.
  * **Three rolling quota windows:** Consumption is metered dynamically across three simultaneous windows (official docs + secondary community/analyst tracking):
    1. **5-hour rolling window:** Caps short-term burst usage (secondary community tracking estimates one 5-hour window is ~20% of the monthly allotment).
    2. **7-day (weekly) rolling window:** Caps sustained multi-day heavy agent loops.
    3. **Monthly billing-cycle window:** Overall monthly cap.
  * **Unpublished hard numbers (*Unverified*):** Cline **does not publish exact numerical token caps or request limits** for the 5-hour, weekly, or monthly windows. Internally, models consume quota weighted by their reference token cost (so a high-cost/heavy model like `kimi-k3` at $3/$15 per 1M reference rate depletes your quota window much faster than `glm-5.3-flash` at $0.15/$0.50 or `deepseek-v4.1-flash` at $0.15/$0.60).
* **Rate limits:**
  * Cline routes requests across dedicated upstream providers (e.g., Fireworks, Together, native lab endpoints) with higher RPM/TPM concurrency than standard free/tier-1 open-model APIs so multi-turn agentic loops (file reads, AST searches, command execution, test-fix cycles) are not throttled mid-task. Exact RPM/TPM figures are not publicly disclosed by Cline (*Unverified*).
* **What happens at the limit:**
  * When any of the active windows (5-hour, weekly, or monthly) reaches 100%, the API (`api.cline.bot`) rejects requests with **`HTTP 429 Too Many Requests`** (often accompanied by `INFERENCE_CAP_ERROR` or rate-limit ceiling messages).
  * **No automatic overage billing or provider fallback:** Cline Pass **does not** automatically roll over into pay-per-token billing when a window is exhausted, and Cline CLI does not auto-switch providers mid-turn. The CLI retries `--retries` times with backoff and then stops; you must either wait for the rolling 5-hour/weekly window to reset, switch to a cheaper/flash model before hitting the cap, or manually switch `-P` to another provider (such as `cline` pay-as-you-go usage billing or a BYOK provider).
* **How it differs from paying per token:**

| Dimension | Cline Pass (`-P cline-pass`) | Cline Usage-Billing (`-P cline`) / BYOK (`-P openrouter`, etc.) |
|---|---|---|
| **Billing** | Flat **$9.99/month** subscription | Pay-per-token credit balance or direct provider invoice |
| **Models** | Curated open-weight + partner coding models only (12 active models) | All frontier proprietary + open models (Claude Opus/Sonnet 5.5, GPT-6.1, Gemini 4, etc.) |
| **Predictability** | Capped at $9.99/mo; hard stop (`HTTP 429`) at window limit | Uncapped spend; multi-turn agent loops with 1M context can cost $5–$20+ in one session |
| **Effective value** | **2x–5x more tokens per dollar** on supported open models | 1x standard API pricing (with prompt-cache discounts where supported) |

---

## 2. Every model available in Cline Pass today

### CLI Model ID Format Verification
* **Verified CLI format:** Official `clinepass.mdx` (`github.com/cline/cline/blob/main/docs/getting-started/clinepass.mdx`) and CLI documentation (`docs.cline.bot/getting-started/clinepass`) specify that Cline Pass model IDs use the **`cline-pass/<model-slug>`** prefix:
  ```powershell
  cline -P cline-pass -m cline-pass/glm-5.3 "Your task"
  ```
* **Note on `cline auth` flag:** In Cline CLI `3.0.73` (`cline auth --help`), the model flag for the `auth` subcommand is `-m, --modelid <id>`, whereas for top-level `cline` runs (`cline --help`) it is `-m, --model <model-id>`. *(Whether the `api.cline.bot` gateway also strips or accepts bare slugs without `cline-pass/` when `-P cline-pass` is set could not be live-tested on this PC because `cline-pass` currently requires interactive owner re-authentication; always pass the documented `cline-pass/<slug>` ID).*

### Why `stealth/ox-alpha` Failed
In August 2026, Z.ai (Zhipu AI) tested a stealth preview model on OpenRouter, OpenCode, and Cline under the anonymous ID **`stealth/ox-alpha`**. After community fingerprinting, Z.ai officially revealed `stealth/ox-alpha` as **GLM-5.3-Flash** (320B total / 18B active MoE with hybrid sparse-and-linear `IndexPool` attention) and retired the `stealth/ox-alpha` slug. Its direct replacement in Cline Pass is **`cline-pass/glm-5.3-flash`** (or its flagship sibling **`cline-pass/glm-5.3`**).

### Active and Retired Cline Pass Models (as of 2026-10-22)

| Exact Model ID (`-m`) | Maker | Context Window | Thinking / Reasoning Support | Speed (Median Throughput & TTFT) | Coding Strengths | Weaknesses | Quota Weight / Extra Cost (Ref. API Price per 1M in/out) | Status |
|---|---|---|---|---|---|---|---|---|
| **`cline-pass/glm-5.3`** | Z.ai (Zhipu AI) | 1,000,000 (1M) | **Yes** (`low`, `medium`, `high`, `xhigh` via `--thinking`) | **~73–85 tok/s** · TTFT ~1.45 s *(secondary: OpenRouter / Ampere)* | Top-tier SWE-bench Verified (95.4% vendor) & Terminal-Bench 2.1 (88.2); trained with SAO + compaction RL specifically for long-horizon agent loops | Slower than Flash; medium-high quota burn rate | Included in $9.99/mo (Ref: **$1.40 / $4.40** per 1M) | **Active** (Replaced `glm-5.2`) |
| **`cline-pass/glm-5.3-flash`** | Z.ai (Zhipu AI) | 1,000,000 (1M) | **Yes** (controllable reasoning + text/image multimodal) | **~105–166 tok/s** (up to 220 tok/s) · TTFT ~0.37–1.46 s *(OpenRouter p50)* | Direct production version of `stealth/ox-alpha`; 320B/18B active MoE; 92.0% SWE-bench Verified, 84.3 Terminal-Bench 2.1; tiny quota footprint | Slightly shallower than full `glm-5.3` on deep multi-file architectural edge cases | Included in $9.99/mo (Ref: **$0.15 / $0.50** per 1M — very low quota burn) | **Active** (Replaced `stealth/ox-alpha`) |
| **`cline-pass/kimi-k3`** | Moonshot AI | 1,000,000 (1M) | **Yes** (native reasoning + text/image multimodal) | **~62–96 tok/s** (up to 165 tok/s on Fireworks/Morph) · TTFT ~0.8–2.5 s | 2.8T MoE with Kimi Delta Attention; #1 on SWE-bench Pro V2 Hard (88.2%) and Terminal-Bench 2.1 (88.3); strongest on hard multi-file refactors & UI/frontend | Highest reference price ($3/$15) → burns 5-hour/weekly Cline Pass quota fastest | Included in $9.99/mo (Ref: **$3.00 / $15.00** per 1M — **highest quota burn**) | **Active** (Replaced `kimi-k2.6` & `kimi-k2.7-code`) |
| **`cline-pass/deepseek-v4-pro`** | DeepSeek | 1,000,000 (1M) | **Yes** (deep reasoning mode, text-only) | **~45–75 tok/s** *(est. secondary; exact Cline gateway p50 unverified)* | Highest LiveCodeBench (93.5% Pass@1) and strong SWE-bench Verified (80.6%); excellent algorithmic logic, math, and polyglot code generation | Text-only (no image input); slower on long agent tool chains than Kimi K3 or GLM-5.3 | Included in $9.99/mo (Ref: **$0.44–$1.32 in / $0.87–$3.96 out** per 1M) | **Active** |
| **`cline-pass/deepseek-v4.1-flash`** | DeepSeek | 1,000,000 (1M) | **Yes** (reasoning + text/image multimodal) | **~214 tok/s** · sub-second TTFT *(Artificial Analysis)* | Fastest model in Cline Pass; 90.6% on Terminal-Bench 2.1 and 91.6% LiveCodeBench; 8B–16B active sparse FP4 MoE | Can over-edit or miss subtle repository constraints if not guided by `.clinerules` | Included in $9.99/mo (Ref: **$0.14–$0.30 in / $0.28–$1.20 out** per 1M — very low quota burn) | **Active** (Replaced `deepseek-v4-flash`) |
| **`cline-pass/qwen3.8-max`** | Alibaba Cloud (Qwen) | 1,000,000 (1M) | **Yes** (hybrid reasoning + omnimodal text/image) | **~39.5 tok/s** · TTFT ~1.41 s *(secondary benchmark)* | Strong SWE-bench Pro (67.7%) & Terminal-Bench 2.1 (86.6%); strict adherence to diff formatting and complex multi-file instructions | Slower token generation (~40 tok/s) on large file rewrites; high quota weight | Included in $9.99/mo (Ref: **$2.00 / $6.00** per 1M) | **Active** |
| **`cline-pass/qwen3.7-max`** | Alibaba Cloud (Qwen) | 262,144 (262K) *(1M on some hosts)* | **Yes** (reasoning, text-focused) | **~40–55 tok/s** *(secondary estimate; exact Cline gateway speed unverified)* | Tuned for creative/frontend design and structured code; 80.4% SWE-bench Verified (vendor), 60.6% SWE-bench Pro | Superseded by `qwen3.8-max` on both context window and Terminal-Bench (74.5 vs 86.6); higher ref price | Included in $9.99/mo (Ref: **$2.50 / $7.50** per 1M) | **Active** (Legacy flagship) |
| **`cline-pass/qwen3.7-plus`** | Alibaba Cloud (Qwen) | 1,000,000 (1M) | **Yes** (reasoning + text/image multimodal) | **~70–100 tok/s** *(secondary estimate; exact Cline gateway speed unverified)* | Balanced mid-tier coding & code review; 89.6% LiveCodeBench v6; low quota weight | Lower SWE-bench Pro (57.6%) and Terminal-Bench (52.8–70.3%) than GLM-5.3 Flash | Included in $9.99/mo (Ref: **$0.40 / $1.60** per 1M) | **Active** |
| **`cline-pass/mimo-v2.5-pro`** | Xiaomi | 262,144–1,048,576 *(262K in pi.dev catalog; 1M on OpenRouter)* | **Yes** (thinking mode, 1.02T/42B active MoE) | **~33–66 tok/s** · TTFT ~0.9–4.2 s *(OpenRouter)* | Built for sustained 1,000+ tool-call agent runs; 78.9% SWE-bench Verified, 57.2% SWE-bench Pro | Deprecated upstream on Xiaomi's native platform in mid-Oct 2026 in favor of MiMo-V2.6 | Included in $9.99/mo (Ref: **$0.435 / $0.87** per 1M) | **Active in Cline Pass** (Deprecated upstream by Xiaomi) |
| **`cline-pass/mimo-v2.5`** | Xiaomi | 262,144 (262K) *(up to 1M upstream)* | **Yes** (light reasoning + omnimodal) | **~80–120 tok/s** *(secondary estimate)* | Ultra-cheap (310B/15B active MoE) for localized edits and Multimodal perception | Mid-tier coding depth (~44–46.3% aggregate coding index) | Included in $9.99/mo (Ref: **$0.14 / $0.28** per 1M) | **Active** |
| **`cline-pass/minimax-m3`** | MiniMax | 1,000,000 (1M) *(512K in older Cline docs)* | **Yes** (reasoning + text/image multimodal) | **~60–90 tok/s** *(secondary estimate; exact Cline gateway speed unverified)* | Good general task decomposition and SWE-bench Pro (59.0%); low-to-moderate quota burn | Weak score on SWE-bench Verified (30.6%) & Aider Polyglot (30.5%) in third-party tests | Included in $9.99/mo (Ref: **$0.30 / $1.20** per 1M ≤512K; $0.60/$2.40 >512K) | **Active** |
| **`cline-pass/muse-spark-1.3-contributor`** | Meta (Superintelligence Labs) | 1,048,576 (1M) | **Yes** (`low`, `medium`, `high`, `xhigh` via `--thinking`) | **~70–110 tok/s** *(secondary estimate; exact Cline gateway speed unverified)* | 98.5% long-context retrieval; 88.8% Terminal-Bench 2.1 (`max` tier) & 75.4% DeepSWE v1.1; uses ~20% fewer tool calls and ~25% fewer tokens | **Privacy / Data warning:** Contributor tier permits Meta to train on prompts and outputs; do **not** use on proprietary/company code | Included in $9.99/mo (Ref: **$0.10 / $0.20** per 1M — Contributor tier) | **Active** (**Contributor data-sharing terms apply**) |
| `stealth/ox-alpha` | Z.ai (Zhipu AI) | 1,000,000 (1M) | Yes | ~105–166 tok/s | August 2026 anonymous stealth preview of GLM-5.3 Flash | Fails with model-not-found / retired error | N/A | **Retired** → Use `cline-pass/glm-5.3-flash` |
| `cline-pass/glm-5.2` | Z.ai (Zhipu AI) | 1,000,000 (1M) | Yes | — | Previous GLM generation | Removed from active Cline Pass routing | N/A | **Retired** → Use `cline-pass/glm-5.3` |
| `cline-pass/kimi-k2.6` / `cline-pass/kimi-k2.7-code` | Moonshot AI | 262,144 (262K) | Yes | — | Previous Kimi coding models (K2.6 SWE-bench Pro 58.6%; K2.7 MCP Mark 81.1) | Removed from active Cline Pass routing | N/A | **Retired** → Use `cline-pass/kimi-k3` |
| `cline-pass/deepseek-v4-flash` | DeepSeek | 1,000,000 (1M) | Yes | — | Initial V4 Flash release | Removed from active Cline Pass routing | N/A | **Retired** → Use `cline-pass/deepseek-v4.1-flash` |

---

## 3. Coding benchmarks

All numbers below list their source and publication/snapshot date. Vendor-reported numbers often use custom scaffolds or `max` effort; independent harness runs (Terminal-Bench `tbench.ai`, Scale SWE-bench Pro, RankLLMs, Artificial Analysis) are noted explicitly.

| Model (`cline-pass/...`) | SWE-bench Verified (%) | SWE-bench Pro (%) | Terminal-Bench (%) | LiveCodeBench (%) | Aider Polyglot / Coding Index (%) | Sources & Dates |
|---|---|---|---|---|---|---|
| **`glm-5.3`** | **95.4%** *(vendor/OpenRouter card, 2026-09)* | *Unverified* (not on public Scale board) | **88.2%** (TB 2.1, Z.ai, 2026-09) · **28.3%** (TB 3.0) · **41.8%** (TB 4.0 Max, `tbench.ai`, 2026-10) | **80.5%** *(OpenRouter / Z.ai card, 2026-09)* | **~74.8%** *(integrated polyglot coding index, secondary, 2026-10)* | OpenRouter `z-ai/glm-5.3` (2026-09); Z.ai blog (2026-09); `tbench.ai` (2026-10); Multiple.chat (2026-10) |
| **`glm-5.3-flash`** *(was `stealth/ox-alpha`)* | **92.0%** *(vendor/OpenRouter card, 2026-08)* | *Unverified* | **84.3%** (TB 2.1, DataCamp/Z.ai, 2026-09) · **35.8%** (TB 4.0 None effort, `tbench.ai`, 2026-10) | **80.5%** *(OpenRouter / Z.ai card, 2026-08)* | **~71.5%** *(integrated polyglot coding index, secondary, 2026-10)* | OpenRouter `z-ai/glm-5.3-flash` (2026-08); DataCamp (2026-09); `tbench.ai` (2026-10) |
| **`kimi-k3`** | **93.4%** *(vendor/leaderboard, 2026-09)* · **76.8%** *(mini-SWE-agent baseline, 2026-09)* | **88.20%** *(SWE-bench Pro V2 Hard, Max effort, Scale/arXiv, 2026-09)* | **88.3%** (TB 2.1, 2026-09) · **17.4%** (TB 3.0, 2026-09) | *Unverified* (K2.6 scored **89.6%** on LCB v6) | **~76.2%** *(multi-language coding index, secondary, 2026-10)* | Scale Labs / SWE-bench Pro (2026-09); ExplainX / Wan27 (2026-09); Z.ai TB comparison (2026-09) |
| **`deepseek-v4-pro`** | **80.6%** *(Max reasoning, HF model card, 2026-08)* | **55.4%** *(HF model card, 2026-08)* | **67.9%** (TB 2.0 Acc, HF card, 2026-08) | **93.5%** *(Pass@1 Max reasoning, HF card, 2026-08)* | **~77%–80%+** *(secondary polyglot suite, 2026-09)* | Hugging Face `deepseek-ai` card (2026-08); Medium / Cerit.io (2026-09) |
| **`deepseek-v4.1-flash`** | **79.0%** *(vendor card; 66.0% strict prompt, 2026-10)* | **52.6%** *(model card / Macaron, 2026-10)* | **90.6%** (TB 2.1 public board leader, 2026-10) · **56.9%** (TB 2.0) | **91.6%** *(Pass@1 Max reasoning, 2026-10)* | **85%–96%** per-language pass on Rust/JS lanes *(secondary, 2026-10)* | DeepSeek release notes (2026-10); ContraCollective / APXML TB 2.1 report (2026-10) |
| **`qwen3.8-max`** | **79.4%–80.2%** *(0902 snapshot / RankLLMs, 2026-09)* | **67.7%** *(HF model card, 2026-09)* | **86.6%** (TB 2.1, HF card, 2026-09) | **87.9%–90.3%** *(LCB v6, OpenRouter / HF, 2026-09)* | **~68.9%–76.2%** *(multi-language coding index, secondary, 2026-10)* | Hugging Face `Qwen` card (2026-09); RankLLMs / LLMCheck (2026-09); OpenRouter (2026-09) |
| **`qwen3.7-max`** | **80.4%** *(vendor card; 65.4% RankLLMs independent, 2026-06)* | **60.6%** *(HF card, 2026-06)* | **74.5%** (TB 2.1, HF card, 2026-06) | *Unverified* | *Unverified* | RankLLMs (2026-06); Beam.ai / Hugging Face (2026-06) |
| **`qwen3.7-plus`** | *Unverified* | **57.6%** *(HF ecosystem eval, 2026-06)* | **70.3%** (Harbor harness, 2026-06) · **52.8%–61.0%** (TB 2.1 independent) | **89.6%** *(LCB v6 reasoning, BenchLM, 2026-07)* | *Unverified* | Hugging Face (2026-06); BenchLM / OpenRouter (2026-07) |
| **`mimo-v2.5-pro`** | **78.6%–78.9%** *(Thinking mode, Xiaomi card, 2026-08)* | **57.2%** *(Spheron / Xiaomi report, 2026-08)* | **68.4%** (TB 2.0, Neura Market / Xiaomi, 2026-08) | **~85%+** *(upper-mid tier, secondary, 2026-08)* | *Unverified* | Xiaomi `mi.com` tech report (2026-08); OpenRouter `xiaomi/mimo-v2.5-pro` (2026-10) |
| **`mimo-v2.5`** | *Unverified* (Pro variant is benchmarked) | *Unverified* | *Unverified* | *Unverified* | **~44.0%–46.3%** *(general intelligence/coding index, 2026-08)* | OpenRouter `xiaomi/mimo-v2.5` (2026-08); Multiple.chat (2026-10) |
| **`minimax-m3`** | **30.6%** *(VibeCoderJournal independent run, 2026-08)* | **59.0%** *(CometAPI / vendor card, 2026-08)* | **65.2%** (TB 2.1, OpenRouter, 2026-08) · **30.0%** (TB 2.0) | **82.2%** *(OpenRouter card, 2026-08)* | **30.5%** *(Aider Polyglot 12-language run, VibeCoderJournal, 2026-08)* | OpenRouter `minimax/minimax-m3` (2026-08); VibeCoderJournal (2026-08); CometAPI (2026-08) |
| **`muse-spark-1.3-contributor`** | **75.4%** *(DeepSWE v1.1 long-horizon SWE benchmark, Meta, 2026-09)* | *Unverified* | **88.8%** (TB 2.1 `max` effort, Meta, 2026-09) · **~85% / 14.5%** (`xhigh` external harness, `tbench.ai`) | **59.4%** *(SWEAtlas CodeBase QnA, Meta, 2026-09)* | *Unverified* | DataCamp / ExplainX Meta Muse Spark 1.3 coverage (2026-09); `tbench.ai` (2026-10) |

---

## 4. Recommendation for daily coding

Because Cline Pass enforces rolling **5-hour, weekly, and monthly quotas** weighted by model cost, using `kimi-k3` ($3/$15 ref) for every trivial edit will exhaust your 5-hour window quickly. Use this 4-tier routing strategy:

### 1. Default Daily Coding Model: `cline-pass/glm-5.3`
* **Exact CLI command:**
  ```powershell
  cline -P cline-pass -m cline-pass/glm-5.3 --thinking medium "<task>"
  ```
* **Why:** `glm-5.3` is the best all-around daily driver in Cline Pass. It is the full flagship brother of the retired `stealth/ox-alpha`, trained specifically with SAO + context compaction RL for multi-step coding agents. It pairs 1M context, 95.4% SWE-bench Verified, and 88.2% Terminal-Bench 2.1 with ~75–85 tok/s throughput at less than one-third the quota burn of `kimi-k3` ($1.40/$4.40 vs $3.00/$15.00 reference rate).

### 2. Fast / Cheap (Quota-Saver) Model: `cline-pass/glm-5.3-flash` *(or `cline-pass/deepseek-v4.1-flash`)*
* **Exact CLI command:**
  ```powershell
  cline -P cline-pass -m cline-pass/glm-5.3-flash --thinking low "<task>"
  ```
  *(Alternative for pure terminal/script speed at ~214 tok/s: `cline -P cline-pass -m cline-pass/deepseek-v4.1-flash --thinking low "<task>"`)*
* **Why:** `cline-pass/glm-5.3-flash` **is the exact production model that `stealth/ox-alpha` was previewing**. At $0.15/$0.50 reference pricing (nearly 10x lighter on quota than `glm-5.3` and 30x lighter than `kimi-k3`), 105–166 tok/s throughput, 1M context, and 92.0% SWE-bench Verified, it handles bulk tests, mechanical refactors, file searches, and everyday edits while barely touching your 5-hour quota window.

### 3. "Hard Problem" Model: `cline-pass/kimi-k3` *(or `cline-pass/deepseek-v4-pro` for pure algorithms)*
* **Exact CLI command:**
  ```powershell
  cline -P cline-pass -m cline-pass/kimi-k3 --thinking high "<task>"
  ```
  *(For competitive-programming/algorithmic bugs: `cline -P cline-pass -m cline-pass/deepseek-v4-pro --thinking high "<task>"`)*
* **Why:** `kimi-k3` (2.8T MoE) leads all Cline Pass models on **SWE-bench Pro V2 Hard (88.2%)** and **Terminal-Bench 2.1 (88.3%)**. Reserve it for complex multi-file bugs, cross-module refactors, or frontend/vision tasks where `glm-5.3` gets stuck—because its $3/$15 reference cost burns quota ~3.5x faster than `glm-5.3`.

### 4. Plan-Mode Model: `cline-pass/glm-5.3` (or `cline-pass/qwen3.8-max` for second-opinion architecture)
* **Exact CLI command:**
  ```powershell
  cline -p -P cline-pass -m cline-pass/glm-5.3 --thinking high "<task>"
  ```
  *(Second-opinion architectural plan: `cline -p -P cline-pass -m cline-pass/qwen3.8-max --thinking high "<task>"`)*
* **Why:** Passing `-p` (`--plan`) puts Cline in read-only **Plan Mode** so it inspects the repository and designs an implementation plan without editing files. `glm-5.3` at `--thinking high` (or `qwen3.8-max`, which scores 67.7% on SWE-bench Pro and 86.6% on Terminal-Bench 2.1) excels at reading large 1M-token contexts and producing structured step-by-step plans before switching to Act mode.

---

## 5. How to control Cline for the best results

### Plan vs. Act Mode (`-p` / `--plan`)
* **Act Mode (default):** When you run `cline "<prompt>"`, Cline starts in **Act Mode** with `--auto-approve true` by default—immediately reading files, editing code, and running shell commands.
* **Plan Mode (`-p` / `--plan`):** Restricts Cline to read-only exploration and planning. In interactive TUI (`cline --tui`), toggle between Plan and Act modes anytime; in headless/delegated workflows, run `cline -p -P cline-pass -m cline-pass/glm-5.3 --thinking high "<task>"` first to review the plan before running Act mode.

### Thinking Levels (`--thinking none|low|medium|high|xhigh`)
* Controls the reasoning token budget on models that support dynamic effort (`glm-5.3`, `glm-5.3-flash`, `kimi-k3`, `deepseek-v4-pro`, `deepseek-v4.1-flash`, `qwen3.8-max`, `muse-spark-1.3-contributor`).
* Bare `--thinking` defaults to `medium`; omitting it uses the level last saved via `/model` or the provider default.
* **Rule of thumb:** Reasoning tokens count as output tokens against your Cline Pass rolling quota. Use `low` for mechanical edits/tests, `medium` for standard feature specs, `high` for architecture/hard bugs, and `xhigh` only when `high` fails.

### Context Compaction (`--compaction agentic|basic|off`)
* **Default:** `agentic`. When a session nears the model's context limit, Cline automatically summarizes earlier turns and tool outputs while keeping critical file states and task goals intact.
* **`basic`:** Faster, simpler truncation/summarization with less overhead.
* **`off`:** Disables automatic compaction (useful only when debugging raw prompt behavior; risks hitting context limits on long runs).
* **Best practice:** Leave `--compaction agentic` on, especially with `glm-5.3`, which was specifically RL-trained with compaction awareness.

### `.clinerules` / Rules Files
* Cline automatically injects Markdown rule files into its system prompt:
  * **Workspace rules:** `.clinerules` (single file) or `.clinerules/*.md` / `.cline/rules/*.md` in the repository root. Also respects `AGENTS.md` / `CLAUDE.md` when configured.
  * **Global rules:** Stored under `%USERPROFILE%\.cline\rules\`.
* Use `.clinerules` to enforce PowerShell 7 (`pwsh`) syntax on Windows, ban destructive git commands, and keep edits scoped to specific directories.

### Skills (`cline skill`)
* Modular task instructions managed via the open skills CLI (`npx skills`).
* **Commands:** `cline skill list`, `cline skill add <owner/repo>`, `cline skill remove <name>`.
* **Locations:** Workspace skills live in `.cline/skills/` or `.clinerules/skills/`; global skills on Windows live in `%USERPROFILE%\.cline\skills\`.

### MCP (`cline mcp`)
* Connects Cline to external Model Context Protocol servers (databases, docs, browser tools).
* **Commands:** `cline mcp add <name> [targetArgs...]`, `cline mcp remove <name>`.
* **Config file:** `%USERPROFILE%\.cline\data\settings\cline_mcp_settings.json`.
* **Quota tip:** Each enabled MCP server adds tool schemas to every turn's input prompt. Disable unused MCP servers to save context and Cline Pass quota.

### Worktrees (`--worktree`) & Kanban (`cline kanban`)
* Passing `--worktree` automatically creates an isolated, detached Git worktree under `%USERPROFILE%\.cline\worktrees\` and runs the agent there (symlinking heavy ignored folders like `node_modules`).
* **In this Herdr setup:** Prefer `herdr-agents.ps1 start cline -Worktree <branch>` (which uses plain `git worktree add` under `~\.herdr\worktrees\<repo>\<branch>`, per `LESSONS.md` #19) so `herdr-agents diff` and `watch` track it cleanly.

### Checkpoints
* Cline automatically snapshots workspace files into a shadow Git repository after each tool step so you can roll back broken edits while keeping the chat history.
* **Watch out on Windows:** In very large repositories or repos with large binaries, shadow Git checkpoints can cause high disk I/O or temporary `.git_disabled` lock states if a process is killed mid-checkpoint. Keep repositories clean with `.gitignore` and `.clineignore`.

### Auto-Approve (`--auto-approve`) and YOLO Mode (`-y` / `--yolo`) — Risks
* **`--auto-approve <boolean>` (default `true` in CLI):** Approves standard file reads/edits and tool calls without interactive confirmation. On this PC, `agents.json` passes `--tui --auto-approve true` by owner decision (`CLAUDE.md` Safety Rule 3, Phase 18).
* **`-y` / `--yolo`:** Intended only for isolated CI/benchmark containers. It bypasses **all** confirmations AND disables runtime safety guards like `.clineignore` protections. **Never use `--yolo` on this PC** (`check.ps1` and `CLAUDE.md` forbid widening approval flags).

### Retries (`--retries <count>`) and Timeouts (`-t, --timeout <seconds>`)
* **`--retries <count>` (default `3`):** Maximum consecutive tool/model mistakes or failed turns before Cline exits. Lower to `--retries 2` on expensive models (`kimi-k3`) so a stuck loop does not burn your 5-hour quota window.
* **`-t, --timeout <seconds>` (default `0` = no timeout):** Hard wall-clock limit for headless CLI runs (e.g., `-t 600` for a 10-minute cap).

### How to Keep Costs / Quota Limits Under Control
1. **Default to `glm-5.3-flash` or `glm-5.3`**, not `kimi-k3`.
2. **Match `--thinking` to task difficulty:** Use `--thinking low` for routine edits; `--thinking high` multiplies output reasoning tokens 2–4x.
3. **Avoid `muse-spark-1.3-contributor` on private code:** Although its quota weight is tiny ($0.10/$0.20 ref), the Contributor tier allows Meta to train on your prompts/code.
4. **Keep sessions short:** Start a fresh session for each distinct task rather than carrying a 400k-token context across unrelated tasks.

### How to Switch the Saved Default Model (Exact Commands)
Because `cline-pass` on this PC currently reports *"requires re-authentication"* (`docs/AGENT-ROSTER.md`) and still has the retired `stealth/ox-alpha` saved, the **owner** should run one of the following in a terminal pane (agents must never run `cline auth` or touch `%USERPROFILE%\.cline`):

1. **Re-authenticate and set the saved default model via CLI (`cline auth`):**
   ```powershell
   cline auth -p cline-pass -m cline-pass/glm-5.3
   ```
   *(If authenticating headless with a `CLINE_API_KEY` from `https://app.cline.bot`: `cline auth -p cline-pass -k <YOUR_KEY> -m cline-pass/glm-5.3`)*
2. **Or switch interactively inside the Cline TUI:**
   ```powershell
   cline --tui
   ```
   Then type **`/settings`** (to ensure provider is `ClinePass` and sign in) or **`/model`** and select **`cline-pass/glm-5.3`** (or **`cline-pass/glm-5.3-flash`**).
3. **Verify the saved configuration (read-only):**
   ```powershell
   cline config --json
   ```

### Known Problems and Fixes on Windows
1. **PowerShell command completion hangs ("Command completion could not be observed"):**
   * *Symptom:* Cline runs a PowerShell command that finishes, but Cline hangs waiting for the shell-integration exit marker (especially with progress bars, spinners, or multi-line quotes).
   * *Fix:* Switch terminal execution mode to **Background execution mode** in Cline settings, or pass `-t <seconds>` on CLI runs; if stuck, run `cline doctor fix` to kill orphaned child processes.
2. **Blank stdout/stderr on non-`cmd.exe` shells (older CLI/extension issue, largely fixed in v3.x):**
   * *Symptom:* Commands exit `0` but return empty output when default shell is Git Bash or mismatched PowerShell 5.1 vs `pwsh` 7.
   * *Fix:* Explicitly set `CLINE_SHELL=pwsh.exe` (PowerShell 7) in the environment and avoid interactive pagers (`git --no-pager`).
3. **Spurious `nul` files created when using Git Bash:**
   * *Symptom:* Models emit Windows `> nul` redirects inside Git Bash, creating literal `nul` files that break Git/Windows tools.
   * *Fix:* Use `pwsh` (PowerShell 7) instead of Git Bash, and add a `.clinerules` line stating: *"Shell is PowerShell 7 (`pwsh`) on Windows; use `$null`, never `> nul`."*
4. **MCP server `EINVAL` / `npx` spawn failure on Windows:**
   * *Symptom:* MCP servers configured with bare `npx` fail to spawn on Windows (`spawn npx EINVAL` / `ENOENT`).
   * *Fix:* In `cline_mcp_settings.json`, invoke `cmd.exe /c npx -y <package>` or `npx.cmd` instead of bare `npx`.
5. **Corporate proxy / firewall returning HTML on direct model endpoints:**
   * *Symptom:* Direct DeepSeek API calls on this PC returned an HTML block page (`LESSONS.md` #33).
   * *Fix:* Routing DeepSeek V4 Pro / V4.1 Flash through **Cline Pass** (`-P cline-pass -m cline-pass/deepseek-v4-pro`) sends traffic to `https://api.cline.bot/api/v1` instead of `api.deepseek.com`, bypassing direct-provider blocks once `cline auth` is completed.

---

## 6. Sources

Every URL consulted for this report, read on **2026-10-22**:

### Official Cline Sources (`cline.bot`, `docs.cline.bot`, `github.com/cline/cline`)
1. `https://docs.cline.bot/getting-started/clinepass` — Official ClinePass documentation (pricing $9.99/mo, 2–5x usage, provider setup, model table) · Read 2026-10-22
2. `https://github.com/cline/cline/blob/main/docs/getting-started/clinepass.mdx` — Official source file for ClinePass documentation and exact `cline-pass/<slug>` model ID table (`glm-5.3`, `glm-5.3-flash`, `kimi-k3`, `deepseek-v4-pro`, `deepseek-v4.1-flash`, `mimo-v2.5`, `mimo-v2.5-pro`, `minimax-m3`, `muse-spark-1.3-contributor`, `qwen3.8-max`, `qwen3.7-max`, `qwen3.7-plus`) · Read 2026-10-22
3. `https://cline.bot/pricing` — Cline pricing page (ClinePass $9.99/month vs. pay-as-you-go usage billing and BYOK) · Read 2026-10-22
4. `https://cline.bot/` — Cline main product overview · Read 2026-10-22
5. `https://docs.cline.bot/cli/cli-overview` — Cline CLI overview and interactive slash commands (`/settings`, `/model`) · Read 2026-10-22
6. `https://docs.cline.bot/cli/cli-reference` — Cline CLI flags reference (`--provider`, `--model`, `--plan`, `--thinking`, `--compaction`, `--worktree`, `--auto-approve`, `--yolo`, `--retries`, `--timeout`, `cline auth`, `cline config`) · Read 2026-10-22
7. `https://docs.cline.bot/getting-started/authorization` — Cline provider authentication and `CLINE_API_KEY` usage · Read 2026-10-22
8. `https://docs.cline.bot/features/plan-and-act-mode` — Plan Mode vs. Act Mode documentation · Read 2026-10-22
9. `https://docs.cline.bot/configurations/rules` — `.clinerules` and workspace/global rules documentation · Read 2026-10-22
10. `https://docs.cline.bot/features/skills` — Cline Skills documentation · Read 2026-10-22
11. `https://docs.cline.bot/mcp` — Model Context Protocol (MCP) configuration in Cline · Read 2026-10-22
12. `https://docs.cline.bot/features/checkpoints` — Shadow Git checkpoints documentation and storage behavior · Read 2026-10-22
13. `https://docs.cline.bot/configurations/hooks` — Runtime hooks documentation · Read 2026-10-22
14. `https://github.com/cline/cline/blob/main/CHANGELOG.md` — Cline repository changelog (CLI 3.0.x, Windows shell fixes, model deprecations) · Read 2026-10-22
15. `https://github.com/cline/cline` — Cline GitHub repository issues & discussions (Windows PowerShell completion hangs, Git Bash `nul` file issue, MCP `npx.cmd` spawn, HTTP 429 behavior) · Read 2026-10-22

### OpenRouter Model Cards (`openrouter.ai`)
16. `https://openrouter.ai/z-ai/glm-5.3` — Z.ai GLM-5.3 context window (1M), pricing ($1.40/$4.40 per 1M), SWE-bench Verified (95.4%), LiveCodeBench (80.5%) · Read 2026-10-22
17. `https://openrouter.ai/z-ai/glm-5.3-flash` — Z.ai GLM-5.3 Flash context window (1M), pricing ($0.15/$0.50 per 1M), p50 throughput (~166 tok/s), SWE-bench Verified (92.0%), `stealth/ox-alpha` lineage · Read 2026-10-22
18. `https://openrouter.ai/moonshotai/kimi-k3` — Moonshot AI Kimi K3 context window (1M), throughput (~96 tok/s p50), multimodal & reasoning capabilities · Read 2026-10-22
19. `https://openrouter.ai/deepseek/deepseek-v4-pro` — DeepSeek V4 Pro context window (1M), pricing, and reasoning specs · Read 2026-10-22
20. `https://openrouter.ai/deepseek/deepseek-v4.1-flash` — DeepSeek V4.1 Flash context window (1M), pricing, and throughput specs · Read 2026-10-22
21. `https://openrouter.ai/qwen/qwen3.8-max` — Alibaba Qwen3.8 Max context window (1M), pricing ($2.00/$6.00 per 1M), LiveCodeBench scores · Read 2026-10-22
22. `https://openrouter.ai/qwen/qwen3.7-max` — Alibaba Qwen3.7 Max specs and pricing ($2.50/$7.50 per 1M) · Read 2026-10-22
23. `https://openrouter.ai/qwen/qwen3.7-plus` — Alibaba Qwen3.7 Plus context window (1M), pricing ($0.40/$1.60 per 1M), Terminal-Bench scores · Read 2026-10-22
24. `https://openrouter.ai/minimax/minimax-m3` — MiniMax M3 context window (1M), pricing ($0.30/$1.20 per 1M), LiveCodeBench (82.2%) & Terminal-Bench 2.1 (65.2%) · Read 2026-10-22
25. `https://openrouter.ai/xiaomi/mimo-v2.5-pro` — Xiaomi MiMo-V2.5-Pro context window (1M), pricing ($0.435/$0.87 per 1M), throughput (33–66 tok/s), SWE-bench Pro · Read 2026-10-22
26. `https://openrouter.ai/xiaomi/mimo-v2.5` — Xiaomi MiMo-V2.5 specs and pricing ($0.14/$0.28 per 1M) · Read 2026-10-22
27. `https://openrouter.ai/meta/muse-spark-1.3-contributor` — Meta Muse Spark 1.3 Contributor context window (1M), pricing ($0.10/$0.20 per 1M), and Contributor data-training terms · Read 2026-10-22

### Official Benchmark Leaderboards & First-Party Lab Reports
28. `https://www.swebench.com/` — SWE-bench Verified and SWE-bench official leaderboards · Read 2026-10-22
29. `https://labs.scale.com/` — Scale AI SWE-bench Pro leaderboard (Kimi K3 88.20% SWE-bench Pro V2 Hard) · Read 2026-10-22
30. `https://www.tbench.ai/` — Terminal-Bench 2.0 / 2.1 / 3.0 / 4.0 official leaderboard (GLM-5.3, GLM-5.3 Flash, Kimi K3, DeepSeek V4.1 Flash, Qwen3.8 Max, Muse Spark 1.3) · Read 2026-10-22
31. `https://huggingface.co/spaces/livecodebench/leaderboard` — LiveCodeBench official leaderboard · Read 2026-10-22
32. `https://aider.chat/docs/leaderboards/` — Aider LLM code editing and Polyglot leaderboards · Read 2026-10-22
33. `https://z.ai/` — Z.ai official GLM-5.3 and GLM-5.3 Flash (`IndexPool`, SAO RL, Terminal-Bench 2.1 88.2 / 84.3) release notes · Read 2026-10-22
34. `https://www.deepseek.com/` — DeepSeek V4 Pro and DeepSeek V4.1 Flash official API pricing (cache hit/miss, peak/off-peak) and benchmark cards · Read 2026-10-22
35. `https://kimi.ai/` — Moonshot AI Kimi K3 (2.8T KDA/AttnRes, 1M context) technical overview · Read 2026-10-22
36. `https://qwen.ai/` — Alibaba Qwen3.8 Max and Qwen3.7 series technical reports · Read 2026-10-22
37. `https://www.minimax.io/` — MiniMax M3 (MiniMax Sparse Attention, 1M context) official release notes · Read 2026-10-22
38. `https://www.mi.com/` — Xiaomi MiMo-V2.5 and MiMo-V2.5-Pro technical reports and October 2026 deprecation notice · Read 2026-10-22

### Secondary Sources (Marked as Secondary — Used to Fill Quota Window, Deprecation, and Benchmark Gaps)
39. `https://kingy.ai/` — Secondary analysis of ClinePass 5-hour, weekly, and monthly rolling quota windows and unpublished hard token caps · Read 2026-10-22
40. `https://1devtool.com/` — Secondary review of ClinePass $9.99/mo pricing, GLM-5.3 reference pricing ($1.40/$4.40), and HTTP 429 behavior · Read 2026-10-22
41. `https://pi.dev/` — Pi coding agent provider catalog (`cline-pass/` active model definitions, context windows, and deprecated slug replacements for `glm-5.2`, `kimi-k2.6`, `kimi-k2.7-code`, `deepseek-v4-flash`) · Read 2026-10-22
42. `https://mastra.ai/` — Mastra AI `cline-pass/` model router documentation · Read 2026-10-22
43. `https://artificialanalysis.ai/` — Artificial Analysis throughput and intelligence index tracking (DeepSeek V4.1 Flash ~214 tok/s, Muse Spark 1.3 `xhigh`/`max`) · Read 2026-10-22
44. `https://www.datacamp.com/` — Secondary benchmark coverage for Meta Muse Spark 1.3 (88.8% Terminal-Bench 2.1, 75.4% DeepSWE v1.1) and GLM-5.3 Flash (84.3% Terminal-Bench 2.1) · Read 2026-10-22
45. `https://www.truefoundry.com/` — Secondary coverage of Meta Muse Spark 1.3 (20% fewer tool calls, 25% fewer tokens) and Kimi K2.7 MCP Mark (81.1) · Read 2026-10-22
46. `https://rankllms.com/` — Independent third-party SWE-bench Verified evaluations for Qwen3.8 Max (79.4%–80.2%) and Qwen3.7 Max (65.4% strict vs 80.4% vendor) · Read 2026-10-22
47. `https://vibecoderjournal.com/` — Independent SWE-bench Verified (30.6%) and Aider Polyglot (30.5%) evaluation of MiniMax M3 · Read 2026-10-22
48. `https://multiple.chat/` — Secondary multi-language coding index comparisons for GLM-5.3 (74.8%), GLM-5.3 Flash (71.5%), Kimi K3 (76.2%), Qwen3.8 Max (68.9%–76.2%), and MiMo-V2.5 (44%–46.3%) · Read 2026-10-22
