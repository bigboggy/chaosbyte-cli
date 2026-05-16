package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Profile is the cached GitHub data we pull at /auth time. Token is discarded
// after the fetch; we never persist it.
type Profile struct {
	Login       string
	Name        string
	Bio         string
	AvatarURL   string
	Location    string
	Company     string
	Followers   int
	Following   int
	PublicRepos int

	Repos         []Repo
	Stars         []StarredRepo
	Contributions []ContribDay
}

// Repo is a public repo the authenticated user owns.
type Repo struct {
	Name        string
	Description string
	Language    string
	Stars       int
	Forks       int
	IsFork      bool
	UpdatedAt   time.Time
}

// StarredRepo is a repo (owned by anyone) the authenticated user starred.
type StarredRepo struct {
	FullName    string
	Description string
	Language    string
	Stars       int
}

// ContribDay is one day of contribution counts.
type ContribDay struct {
	Date  time.Time
	Count int
}

// FetchProfile resolves /user, the user's repos, starred repos, and last-year
// contribution graph in one go. Anything that fails individually leaves its
// field empty rather than aborting the whole sync — partial data on a profile
// beats no profile at all.
func (d *DeviceFlow) FetchProfile(ctx context.Context, token string) (*Profile, error) {
	user, err := d.fetchUser(ctx, token)
	if err != nil {
		return nil, err
	}
	p := &Profile{
		Login:       user.Login,
		Name:        user.Name,
		Bio:         user.Bio,
		AvatarURL:   user.AvatarURL,
		Location:    user.Location,
		Company:     user.Company,
		Followers:   user.Followers,
		Following:   user.Following,
		PublicRepos: user.PublicRepos,
	}

	if repos, err := d.fetchRepos(ctx, token); err == nil {
		p.Repos = repos
	}
	if stars, err := d.fetchStars(ctx, token); err == nil {
		p.Stars = stars
	}
	if days, err := d.fetchContributions(ctx, token); err == nil {
		p.Contributions = days
	}
	return p, nil
}

// userResponse holds the fields we care about from /user.
type userResponse struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Location    string `json:"location"`
	Company     string `json:"company"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	PublicRepos int    `json:"public_repos"`
}

func (d *DeviceFlow) fetchUser(ctx context.Context, token string) (*userResponse, error) {
	var u userResponse
	if err := d.getJSON(ctx, token, "https://api.github.com/user", &u); err != nil {
		return nil, err
	}
	if u.Login == "" {
		return nil, fmt.Errorf("github: empty login in /user response")
	}
	return &u, nil
}

func (d *DeviceFlow) fetchRepos(ctx context.Context, token string) ([]Repo, error) {
	var raw []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Stargazers  int    `json:"stargazers_count"`
		Forks       int    `json:"forks_count"`
		Fork        bool   `json:"fork"`
		UpdatedAt   string `json:"updated_at"`
	}
	if err := d.getJSON(ctx, token,
		"https://api.github.com/user/repos?per_page=100&sort=updated&affiliation=owner&visibility=public",
		&raw); err != nil {
		return nil, err
	}
	out := make([]Repo, 0, len(raw))
	for _, r := range raw {
		t, _ := time.Parse(time.RFC3339, r.UpdatedAt)
		out = append(out, Repo{
			Name:        r.Name,
			Description: r.Description,
			Language:    r.Language,
			Stars:       r.Stargazers,
			Forks:       r.Forks,
			IsFork:      r.Fork,
			UpdatedAt:   t,
		})
	}
	return out, nil
}

func (d *DeviceFlow) fetchStars(ctx context.Context, token string) ([]StarredRepo, error) {
	var raw []struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Stargazers  int    `json:"stargazers_count"`
	}
	if err := d.getJSON(ctx, token,
		"https://api.github.com/user/starred?per_page=100",
		&raw); err != nil {
		return nil, err
	}
	out := make([]StarredRepo, 0, len(raw))
	for _, r := range raw {
		out = append(out, StarredRepo{
			FullName:    r.FullName,
			Description: r.Description,
			Language:    r.Language,
			Stars:       r.Stargazers,
		})
	}
	return out, nil
}

// fetchContributions hits the GraphQL endpoint. The REST API doesn't expose
// the contribution calendar, but GraphQL's viewer.contributionsCollection does
// — and only needs the read:user scope we already request.
func (d *DeviceFlow) fetchContributions(ctx context.Context, token string) ([]ContribDay, error) {
	body := `{"query":"query{viewer{contributionsCollection{contributionCalendar{weeks{contributionDays{date contributionCount}}}}}}"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.github.com/graphql", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := d.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github graphql: %s: %s", resp.Status, bytes.TrimSpace(b))
	}
	var raw struct {
		Data struct {
			Viewer struct {
				ContributionsCollection struct {
					ContributionCalendar struct {
						Weeks []struct {
							ContributionDays []struct {
								Date              string `json:"date"`
								ContributionCount int    `json:"contributionCount"`
							} `json:"contributionDays"`
						} `json:"weeks"`
					} `json:"contributionCalendar"`
				} `json:"contributionsCollection"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw.Errors) > 0 {
		return nil, fmt.Errorf("github graphql: %s", raw.Errors[0].Message)
	}
	var out []ContribDay
	for _, w := range raw.Data.Viewer.ContributionsCollection.ContributionCalendar.Weeks {
		for _, day := range w.ContributionDays {
			t, err := time.Parse("2006-01-02", day.Date)
			if err != nil {
				continue
			}
			out = append(out, ContribDay{Date: t, Count: day.ContributionCount})
		}
	}
	return out, nil
}

func (d *DeviceFlow) getJSON(ctx context.Context, token, url string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := d.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("github %s: %s: %s", url, resp.Status, bytes.TrimSpace(body))
	}
	return json.NewDecoder(resp.Body).Decode(into)
}
