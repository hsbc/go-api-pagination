package pagination

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

type processFunc struct {
	client *github.Client
}

func (p *processFunc) Process(ctx context.Context, item *github.Repository) error {
	return nil
}

type processErrorFunc struct {
	client *github.Client
}

func (p *processErrorFunc) Process(ctx context.Context, item *github.Repository) error {
	return errors.New("mock error")
}

type rateLimitReturnNowFunc struct {
}

func (r *rateLimitReturnNowFunc) RateLimit(ctx context.Context, resp *github.Response) (bool, error) {
	return false, nil
}

type rateLimitFunc struct {
}

func (r *rateLimitFunc) RateLimit(ctx context.Context, resp *github.Response) (bool, error) {
	return true, nil
}

type rateLimitErrorFunc struct {
}

func (r *rateLimitErrorFunc) RateLimit(ctx context.Context, resp *github.Response) (bool, error) {
	return true, errors.New("mock error")
}

type listFunc struct {
	client *github.Client
}

func (l *listFunc) List(ctx context.Context, opt *github.ListOptions) ([]*github.Repository, *github.Response, error) {
	rOpts := github.RepositoryListByAuthenticatedUserOptions{
		Visibility:  "public",
		ListOptions: *opt,
	}
	t, r, err := l.client.Repositories.ListByAuthenticatedUser(ctx, &rOpts)

	return t, r, err
}

type listErrorFunc struct {
	client *github.Client
}

func (l *listErrorFunc) List(ctx context.Context, opt *github.ListOptions) ([]*github.Repository, *github.Response, error) {
	return nil, nil, errors.New("mock error")
}

func Test_Paginator(t *testing.T) {
	t.Run("should return a list of items via pagination", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processFunc{client: client}
		rFunc := &rateLimitFunc{}
		lFunc := &listFunc{client: client}
		opts := PaginatorOpts{ListOptions: &github.ListOptions{Page: 1, PerPage: 10}}

		resp, err := Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, &opts)
		assert.NoError(t, err)
		assert.Len(t, resp, 378)
	})

	t.Run("should return when ratelimter returns a false response", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-with-opts")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		want := 2
		pFunc := &processFunc{client: client}
		rFunc := &rateLimitReturnNowFunc{}
		lFunc := &listFunc{client: client}
		opts := PaginatorOpts{ListOptions: &github.ListOptions{Page: 1, PerPage: want}}

		resp, err := Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, &opts)
		assert.NoError(t, err)
		assert.Len(t, resp, want)
	})

	t.Run("should use default opts if none provided", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-default-opts")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processFunc{client: client}
		rFunc := &rateLimitFunc{}
		lFunc := &listFunc{client: client}

		resp, err := Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, nil)
		assert.NoError(t, err)
		assert.Len(t, resp, 378)
	})
	t.Run("should use 100 per page if per page is 0 (resource exhaustion)", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-opts-min-per-page")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processFunc{client: client}
		rFunc := &rateLimitFunc{}
		lFunc := &listFunc{client: client}
		opts := PaginatorOpts{ListOptions: &github.ListOptions{Page: 1, PerPage: 0}}

		resp, err := Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, &opts)
		assert.NoError(t, err)
		// at time of creating the fixture there were 73 public repos
		assert.Len(t, resp, 378)
	})

	t.Run("should return any error encountered by the list function", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-list")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processFunc{client: client}
		rFunc := &rateLimitFunc{}
		lFunc := &listErrorFunc{client: client}

		resp, err := Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, nil)
		assert.Error(t, err)
		assert.Len(t, resp, 0)
	})

	t.Run("should return any error encountered by the rate limit function", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-rate-limit")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processFunc{client: client}
		rFunc := &rateLimitErrorFunc{}
		lFunc := &listFunc{client: client}

		_, err = Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, nil)
		assert.Error(t, err)
	})

	t.Run("should return any error encountered by the process function", func(t *testing.T) {
		client, r, err := newVcrGithubClient("fixtures/paginator-process")
		assert.NoError(t, err)
		//nolint:errcheck // this is used as a cleanup
		defer r.Stop()

		pFunc := &processErrorFunc{client: client}
		rFunc := &rateLimitFunc{}
		lFunc := &listFunc{client: client}

		_, err = Paginator[*github.Repository](context.Background(), lFunc, pFunc, rFunc, nil)
		assert.Error(t, err)
	})
}

func newVcrGithubClient(vcrPath string) (*github.Client, *recorder.Recorder, error) {

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "PLACEHOLDER"},
	)

	t, _ := ts.Token()

	tr := &oauth2.Transport{
		Base:   http.DefaultTransport,
		Source: oauth2.ReuseTokenSource(t, ts),
	}

	r, err := recorder.New(vcrPath, recorder.WithRealTransport(tr), recorder.WithMatcher(
		cassette.NewDefaultMatcher(
			cassette.WithIgnoreUserAgent(),
			cassette.WithIgnoreAuthorization(),
		),
	), recorder.WithMode(recorder.ModeRecordOnce), recorder.WithSkipRequestLatency(true))
	if err != nil {
		return nil, nil, err
	}

	httpClient := http.DefaultClient
	httpClient.Transport = r

	return github.NewClient(httpClient), r, nil
}
