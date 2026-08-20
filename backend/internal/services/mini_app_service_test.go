package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func newTestMiniAppService(t *testing.T) *MiniAppService {
	t.Helper()
	return NewMiniAppService(repositories.NewMiniAppRepository(newTestDB(t)))
}

func userID() uuid.UUID { return uuid.New() }

func validInput() MiniAppInput {
	return MiniAppInput{Name: "My Counter", Description: "A counter app", Icon: "🔢", HTML: "<button id=\"b\">0</button>", CSS: "button{font-size:2rem}", JS: "let n=0;document.getElementById('b').onclick=()=>{document.getElementById('b').textContent=++n}", Visibility: entities.MiniAppVisibilityPrivate}
}

func TestMiniAppCreateAndGet(t *testing.T) {
	service := newTestMiniAppService(t)
	owner := userID()
	app, err := service.Create(owner, validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if app.Slug == "" || app.UserID != owner {
		t.Fatalf("unexpected app %#v", app)
	}
	got, err := service.Get(owner, app.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "My Counter" || got.HTML == "" || got.Visibility != entities.MiniAppVisibilityPrivate {
		t.Fatalf("unexpected persisted app %#v", got)
	}
}

func TestMiniAppSlugUnique(t *testing.T) {
	service := newTestMiniAppService(t)
	first, err := service.Create(userID(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(uuid.New(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if first.Slug == second.Slug {
		t.Fatalf("slugs must be unique, both = %q", first.Slug)
	}
	if !strings.HasPrefix(second.Slug, first.Slug+"-") {
		t.Fatalf("expected suffix on second slug, got %q vs %q", first.Slug, second.Slug)
	}
}

func TestMiniAppValidation(t *testing.T) {
	service := newTestMiniAppService(t)
	cases := []MiniAppInput{
		{Name: "", HTML: "<p>x</p>", Visibility: entities.MiniAppVisibilityPublic},
		{Name: strings.Repeat("n", 81), HTML: "<p>x</p>", Visibility: entities.MiniAppVisibilityPublic},
		{Name: "ok", HTML: strings.Repeat("h", maxMiniAppSourceLength+1), Visibility: entities.MiniAppVisibilityPublic},
		{Name: "ok", HTML: "<p>x</p>", Visibility: "secret"},
	}
	for index, input := range cases {
		if _, err := service.Create(userID(), input); !errors.Is(err, ErrMiniAppInvalid) {
			t.Fatalf("case %d: expected ErrMiniAppInvalid, got %v", index, err)
		}
	}
}

func TestMiniAppUpdateAndDeleteScopesToOwner(t *testing.T) {
	service := newTestMiniAppService(t)
	owner, other := userID(), userID()
	app, err := service.Create(owner, validInput())
	if err != nil {
		t.Fatal(err)
	}
	// Another user cannot fetch or delete the app.
	if _, err := service.Get(other, app.ID); !errors.Is(err, ErrMiniAppNotFound) {
		t.Fatalf("expected ErrMiniAppNotFound for other user get, got %v", err)
	}
	if _, err := service.Update(other, app.ID, MiniAppPatch{Name: ptr("hijack")}); !errors.Is(err, ErrMiniAppNotFound) {
		t.Fatalf("expected ErrMiniAppNotFound for other user update, got %v", err)
	}
	// Partial update keeps untouched source fields.
	updated, err := service.Update(owner, app.ID, MiniAppPatch{Visibility: ptr(entities.MiniAppVisibilityPublic), CSS: ptr("button{color:red}")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.HTML != app.HTML || updated.CSS != "button{color:red}" || updated.Visibility != entities.MiniAppVisibilityPublic {
		t.Fatalf("partial update clobbered fields: %#v", updated)
	}
	// Renaming re-derives the slug.
	renamed, err := service.Update(owner, app.ID, MiniAppPatch{Name: ptr("Brand New")})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Slug == app.Slug || strings.Contains(renamed.Slug, " ") {
		t.Fatalf("expected re-slugified slug, got %q", renamed.Slug)
	}
	if err := service.Delete(owner, app.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := service.Get(owner, app.ID); !errors.Is(err, ErrMiniAppNotFound) {
		t.Fatalf("expected ErrMiniAppNotFound after delete, got %v", err)
	}
}

func TestMiniAppGetForRunPrivacy(t *testing.T) {
	service := newTestMiniAppService(t)
	owner, other := userID(), userID()
	private, err := service.Create(owner, validInput())
	if err != nil {
		t.Fatal(err)
	}
	// Private app: only the owner can run it.
	if _, err := service.GetForRun(private.Slug, nil); !errors.Is(err, ErrMiniAppPrivate) {
		t.Fatalf("expected ErrMiniAppPrivate for anonymous, got %v", err)
	}
	if _, err := service.GetForRun(private.Slug, &other); !errors.Is(err, ErrMiniAppPrivate) {
		t.Fatalf("expected ErrMiniAppPrivate for other user, got %v", err)
	}
	if _, err := service.GetForRun(private.Slug, &owner); err != nil {
		t.Fatalf("owner should run their private app: %v", err)
	}
	// Public app: anyone can run it, and it appears in PublicList.
	publicInput := validInput()
	publicInput.Visibility = entities.MiniAppVisibilityPublic
	publicInput.Name = "Public App"
	pub, err := service.Create(owner, publicInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetForRun(pub.Slug, nil); err != nil {
		t.Fatalf("anonymous should run a public app: %v", err)
	}
	summaries, err := service.PublicList()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, summary := range summaries {
		if summary.Slug == pub.Slug {
			found = true
		}
	}
	if !found {
		t.Fatal("public app missing from PublicList")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"My Counter":   "my-counter",
		"  Todo  List ": "todo-list",
		"123":          "123",
		"!!!":          "app",
		"Über Cool":    "ber-cool",
	}
	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseMiniAppDraft(t *testing.T) {
	raw := "```json\n{\"name\":\"Todo\",\"description\":\"a list\",\"icon\":\"✅\",\"html\":\"<ul></ul>\",\"css\":\"ul{}\",\"js\":\"\"}\n```"
	draft, err := parseMiniAppDraft(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if draft.Name != "Todo" || draft.HTML != "<ul></ul>" || draft.JS != "" {
		t.Fatalf("unexpected draft %#v", draft)
	}
	if _, err := parseMiniAppDraft("not json at all"); err == nil {
		t.Fatal("expected parse error for non-JSON")
	}
}

func ptr(s string) *string { return &s }