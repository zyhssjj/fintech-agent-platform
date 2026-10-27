package communication

import (
	"context"
	"testing"

	"gorm.io/gorm"

	approval "wfagent/internal/module-approval"
)

func TestSeedBuiltinTemplatesCreatesReadonlySystemTemplates(t *testing.T) {
	db := openCommunicationTestDB(t)
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("seed demo data a second time: %v", err)
	}

	var templates []Template
	if err := db.Where("template_source = ?", templateSourceSystemBuiltin).Find(&templates).Error; err != nil {
		t.Fatalf("query builtin templates: %v", err)
	}
	if len(templates) < 2 {
		t.Fatalf("expected at least 2 builtin templates, got %d", len(templates))
	}
	for _, tpl := range templates {
		if tpl.IsEditable {
			t.Fatalf("builtin template %s should be read-only", tpl.TemplateCode)
		}
		if tpl.PublishedVersionID == nil {
			t.Fatalf("builtin template %s should have a published version", tpl.TemplateCode)
		}
		var versionCount int64
		if err := db.Model(&TemplateVersion{}).Where("template_id = ? AND is_deleted = false", tpl.ID).Count(&versionCount).Error; err != nil {
			t.Fatalf("count versions for %s: %v", tpl.TemplateCode, err)
		}
		if versionCount != 1 {
			t.Fatalf("builtin template %s should have one version after repeated startup seeding, got %d", tpl.TemplateCode, versionCount)
		}
	}
}

func TestPrivateTemplateVisibilityAndRestrictions(t *testing.T) {
	db := openCommunicationTestDB(t)
	service := NewService(db)

	created, err := service.SaveTemplate(context.Background(), "", TemplateRequest{
		Name:               "Private Notice",
		Category:           "notice",
		Scene:              "lp_communication",
		SubjectTemplate:    "{{fund_name}} notice",
		HTMLContent:        "<p>{{fund_name}}</p><p>{{lp_name}}</p>",
		DOCXTemplateName:   "private.docx",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
		Variables: []TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "Fund Name", VariableType: "text", IsRequired: true},
			{VariableName: "lp_name", VariableLabel: "LP Name", VariableType: "text", IsRequired: true},
			{VariableName: "notice_date", VariableLabel: "Notice Date", VariableType: "date", IsRequired: true},
		},
	}, "100", "owner_user")
	if err != nil {
		t.Fatalf("create private template: %v", err)
	}
	if created.TemplateSource != templateSourceUserPrivate {
		t.Fatalf("expected private template source, got %s", created.TemplateSource)
	}

	page, err := service.ListTemplates(context.Background(), TemplateQuery{}, "100")
	if err != nil {
		t.Fatalf("list templates for owner: %v", err)
	}
	if !containsTemplate(page.List, created.TemplateID) {
		t.Fatalf("owner should see created private template")
	}

	page, err = service.ListTemplates(context.Background(), TemplateQuery{}, "200")
	if err != nil {
		t.Fatalf("list templates for other user: %v", err)
	}
	if containsTemplate(page.List, created.TemplateID) {
		t.Fatalf("other user should not see private template")
	}

	if _, err := service.GetTemplate(context.Background(), created.TemplateID, "200"); err == nil {
		t.Fatalf("other user should not fetch private template detail")
	}
	if err := service.DeleteTemplate(context.Background(), created.TemplateID, "200"); err == nil {
		t.Fatalf("other user should not delete private template")
	}
}

func TestSubmitPrivateTemplatePublishesPublicCopy(t *testing.T) {
	db := openCommunicationTestDB(t)
	if err := approvalSeedForTemplateTests(db); err != nil {
		t.Fatalf("approval seed: %v", err)
	}
	service := NewService(db)

	created, err := service.SaveTemplate(context.Background(), "", TemplateRequest{
		Name:               "Upgrade Me",
		Category:           "notice",
		Scene:              "strategic_cooperation",
		SubjectTemplate:    "{{fund_name}} notice",
		HTMLContent:        "<p>{{fund_name}}</p><p>{{lp_name}}</p><p>{{notice_date}}</p>",
		DOCXTemplateName:   "upgrade.docx",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
		Variables: []TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "Fund Name", VariableType: "text", IsRequired: true},
			{VariableName: "lp_name", VariableLabel: "LP Name", VariableType: "text", IsRequired: true},
			{VariableName: "notice_date", VariableLabel: "Notice Date", VariableType: "date", IsRequired: true},
		},
	}, "101", "ops_manager")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	result, err := service.SubmitTemplate(context.Background(), created.TemplateID, "101", "ops_manager")
	if err != nil {
		t.Fatalf("submit template: %v", err)
	}
	approvalID, _ := result["approval_id"].(string)
	if approvalID == "" {
		t.Fatalf("approval id should not be empty")
	}

	if _, err := service.approvalService.ActionApproval(context.Background(), approvalID, approval.ApprovalActionRequest{
		Action:  "approve",
		Comment: "step1",
	}, "101", "ops_manager", []string{"OPS_MANAGER"}); err != nil {
		t.Fatalf("step1 approve: %v", err)
	}
	if _, err := service.approvalService.ActionApproval(context.Background(), approvalID, approval.ApprovalActionRequest{
		Action:  "approve",
		Comment: "step2",
	}, "102", "legal_manager", []string{"LEGAL_MANAGER"}); err != nil {
		t.Fatalf("step2 approve: %v", err)
	}
	if err := service.syncTemplatePublication(context.Background()); err != nil {
		t.Fatalf("sync template publication: %v", err)
	}

	var publicTpl Template
	if err := db.Where("template_source = ? AND source_ref_id IS NOT NULL", templateSourcePublishedPublic).First(&publicTpl).Error; err != nil {
		t.Fatalf("find published public copy: %v", err)
	}
	if publicTpl.IsEditable {
		t.Fatalf("published public template should be read-only")
	}
	if publicTpl.PublishedVersionID == nil {
		t.Fatalf("published public template should have published version")
	}
}

func TestSaveTemplateAllowsSuggestedVariablesNotYetUsedInHTML(t *testing.T) {
	db := openCommunicationTestDB(t)
	service := NewService(db)

	created, err := service.SaveTemplate(context.Background(), "", TemplateRequest{
		Name:               "Suggested Fields Draft",
		Category:           "notice",
		Scene:              "lp_communication",
		SubjectTemplate:    "{{fund_name}} notice",
		HTMLContent:        "<p>{{fund_name}}</p><p>{{lp_name}}</p>",
		DOCXTemplateName:   "suggested.docx",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
		Variables: []TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "Fund Name", VariableType: "text", IsRequired: true},
			{VariableName: "lp_name", VariableLabel: "LP Name", VariableType: "text", IsRequired: true},
			{VariableName: "notice_date", VariableLabel: "Notice Date", VariableType: "date", IsRequired: true},
		},
	}, "100", "owner_user")
	if err != nil {
		t.Fatalf("save template with extra suggested variable: %v", err)
	}
	if created.TemplateID == "" {
		t.Fatalf("expected template id")
	}
}

func TestSaveTemplateCanBeCreatedWithoutDocxForChatMode(t *testing.T) {
	db := openCommunicationTestDB(t)
	service := NewService(db)

	created, err := service.SaveTemplate(context.Background(), "", TemplateRequest{
		Name:            "Chat First Draft",
		Category:        "notice",
		Scene:           "lp_communication",
		SubjectTemplate: "{{fund_name}} notice",
		HTMLContent:     "<p>{{fund_name}}</p><p>{{lp_name}}</p>",
		Variables: []TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "Fund Name", VariableType: "text", IsRequired: true},
			{VariableName: "lp_name", VariableLabel: "LP Name", VariableType: "text", IsRequired: true},
		},
	}, "100", "owner_user")
	if err != nil {
		t.Fatalf("save template without docx: %v", err)
	}
	if created.TemplateID == "" {
		t.Fatalf("expected template id")
	}
}

func TestSaveTemplateRejectsPlaceholderWithoutVariableDefinition(t *testing.T) {
	db := openCommunicationTestDB(t)
	service := NewService(db)

	_, err := service.SaveTemplate(context.Background(), "", TemplateRequest{
		Name:               "Broken Draft",
		Category:           "notice",
		Scene:              "lp_communication",
		SubjectTemplate:    "{{fund_name}} notice",
		HTMLContent:        "<p>{{fund_name}}</p><p>{{unknown_field}}</p>",
		DOCXTemplateName:   "broken.docx",
		DOCXTemplateBase64: encodeBase64(sampleDOCXTemplate()),
		Variables: []TemplateVariableInput{
			{VariableName: "fund_name", VariableLabel: "Fund Name", VariableType: "text", IsRequired: true},
			{VariableName: "notice_date", VariableLabel: "Notice Date", VariableType: "date", IsRequired: true},
		},
	}, "100", "owner_user")
	if err == nil {
		t.Fatalf("expected missing variable definition error")
	}
}

func containsTemplate(items []TemplateView, templateID string) bool {
	for _, item := range items {
		if item.TemplateID == templateID {
			return true
		}
	}
	return false
}

func approvalSeedForTemplateTests(db *gorm.DB) error {
	return approval.SeedDefaults(db)
}
