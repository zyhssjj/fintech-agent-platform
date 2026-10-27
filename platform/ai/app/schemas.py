from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field


class HealthResponse(BaseModel):
    status: str
    provider: str
    model: str


class TemplateChatRequest(BaseModel):
    scene: str = ""
    goal: str = ""
    current_summary: str = ""
    latest_user_message: str = ""
    known_fields: list[dict[str, Any]] = Field(default_factory=list)


class TemplateImportRequest(BaseModel):
    file_name: str
    plain_text: str = ""
    paragraphs: list[str] = Field(default_factory=list)
    scene: str = ""


class TemplateRenderAssistRequest(BaseModel):
    template_name: str = ""
    template_scene: str = ""
    recipient_type: str = ""
    business_goal: str = ""
    known_values: dict[str, Any] = Field(default_factory=dict)


class TemplateField(BaseModel):
    variable_name: str
    variable_label: str
    variable_type: str = "text"
    is_required: bool = True


class TemplateDraftResponse(BaseModel):
    summary: str
    follow_up_questions: list[str] = Field(default_factory=list)
    suggested_fields: list[TemplateField] = Field(default_factory=list)
    subject_template: str = ""
    html_template: str = ""

