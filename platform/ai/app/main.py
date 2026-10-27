from __future__ import annotations

from fastapi import FastAPI
import uvicorn

from app.config import SETTINGS
from app.schemas import (
    HealthResponse,
    TemplateChatRequest,
    TemplateDraftResponse,
    TemplateImportRequest,
    TemplateRenderAssistRequest,
)
from app.services.template_agent import assist_render, draft_template_from_chat, draft_template_from_import


app = FastAPI(title="FinTechBase AI Service", version="0.1.0")


@app.get("/healthz", response_model=HealthResponse)
async def healthz() -> HealthResponse:
    return HealthResponse(status="ok", provider=SETTINGS.provider, model=SETTINGS.model)


@app.post("/template/chat", response_model=TemplateDraftResponse)
async def template_chat(payload: TemplateChatRequest) -> TemplateDraftResponse:
    return await draft_template_from_chat(payload)


@app.post("/template/import", response_model=TemplateDraftResponse)
async def template_import(payload: TemplateImportRequest) -> TemplateDraftResponse:
    return await draft_template_from_import(payload)


@app.post("/template/render-assist", response_model=TemplateDraftResponse)
async def template_render_assist(payload: TemplateRenderAssistRequest) -> TemplateDraftResponse:
    return await assist_render(payload)


if __name__ == "__main__":
    uvicorn.run("app.main:app", host=SETTINGS.host, port=SETTINGS.port, reload=False)

