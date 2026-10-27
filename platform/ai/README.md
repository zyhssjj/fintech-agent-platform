# FinTechBase AI Service

This service is the standalone AI layer for `fintechbase`.

It is intended to:

- read its own `.env` from the `ai/` directory
- provide OpenAI-compatible template endpoints for the Go backend
- keep model prompts, parsing, and conversation orchestration outside business code

## Quick Start

1. Create or update `ai/.env`.
2. Fill at least:

```env
AI_BASE_URL=https://api.openai.com/v1
AI_API_KEY=your_key
AI_MODEL=gpt-5.4-mini
AI_PORT=9100
```

3. Install dependencies:

```bash
pip install -r requirements.txt
```

4. Start the service:

```bash
python -m app.main
```

The service listens on `http://127.0.0.1:9100` by default.
