package campaigns

import (
	"context"
	"testing"
	"time"
)

type repositoryStub struct {
	template Template
	campaign Campaign
}

type actionsRepositoryStub struct {
	repositoryStub
	updated    bool
	cancelled  bool
	duplicated bool
}

func (r *actionsRepositoryStub) UpdateCampaign(context.Context, string, string, string, *time.Time, string) error {
	r.updated = true
	return nil
}
func (r *actionsRepositoryStub) CancelCampaign(context.Context, string, string) error {
	r.cancelled = true
	return nil
}
func (r *actionsRepositoryStub) DuplicateCampaign(context.Context, string, string) (Campaign, error) {
	r.duplicated = true
	return Campaign{ID: "copy-1", Status: "draft"}, nil
}

func (r *repositoryStub) ListTemplates(context.Context, string) ([]Template, error) {
	return []Template{r.template}, nil
}
func (r *repositoryStub) CreateTemplate(_ context.Context, item Template) (Template, error) {
	r.template = item
	item.ID = "template-1"
	return item, nil
}
func (r *repositoryStub) DeleteTemplate(context.Context, string, string) error {
	return nil
}
func (r *repositoryStub) ListCampaigns(context.Context, string) ([]Campaign, error) {
	return []Campaign{r.campaign}, nil
}
func (r *repositoryStub) CreateCampaign(_ context.Context, item Campaign) (Campaign, error) {
	r.campaign = item
	item.ID = "campaign-1"
	return item, nil
}
func (r *repositoryStub) DeleteCampaign(context.Context, string, string) error {
	return nil
}
func (r *repositoryStub) AddRecipients(context.Context, string, string, []string) error { return nil }

func TestCreateTemplateTrimsFields(t *testing.T) {
	r := &repositoryStub{}
	service, _ := NewService(r)
	item, err := service.CreateTemplate(context.Background(), "org-1", " Welcome ", " Subject ", " <p>Hello</p> ")
	if err != nil || item.Name != "Welcome" || item.Subject != "Subject" {
		t.Fatalf("unexpected template: %+v err=%v", item, err)
	}
}

func TestCreateCampaignSchedulesFutureCampaign(t *testing.T) {
	r := &repositoryStub{}
	service, _ := NewService(r)
	scheduled := time.Now().Add(time.Hour)
	item, err := service.CreateCampaign(context.Background(), "org-1", "template-1", "Launch", &scheduled)
	if err != nil || item.Status != "scheduled" {
		t.Fatalf("unexpected campaign: %+v err=%v", item, err)
	}
}

func TestCreateCampaignRejectsPastSchedule(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	scheduled := time.Now().Add(-time.Hour)
	if _, err := service.CreateCampaign(context.Background(), "org-1", "template-1", "Launch", &scheduled); err == nil {
		t.Fatal("expected past schedule to be rejected")
	}
}

func TestAddRecipientsValidatesRequiredValues(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	if err := service.AddRecipients(context.Background(), "org-1", "campaign-1", nil); err == nil {
		t.Fatal("expected recipients to be required")
	}
}

func TestCampaignActionsDelegateToRepository(t *testing.T) {
	repository := &actionsRepositoryStub{}
	service, _ := NewService(repository)
	if err := service.UpdateCampaign(context.Background(), "org-1", "campaign-1", "template-1", "Updated", nil); err != nil {
		t.Fatal(err)
	}
	if err := service.CancelCampaign(context.Background(), "org-1", "campaign-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.DuplicateCampaign(context.Background(), "org-1", "campaign-1"); err != nil {
		t.Fatal(err)
	}
	if !repository.updated || !repository.cancelled || !repository.duplicated {
		t.Fatal("expected all campaign actions to be delegated")
	}
}

func TestDeleteTemplateAndCampaignValidateInputs(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	if err := service.DeleteTemplate(context.Background(), "", "template-1"); err == nil {
		t.Fatal("expected template validation to fail")
	}
	if err := service.DeleteCampaign(context.Background(), "org-1", ""); err == nil {
		t.Fatal("expected campaign validation to fail")
	}
}
