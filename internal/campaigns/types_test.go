package campaigns

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

func TestChangesetMetadata(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)

	githubActor := github.Actor{
		AvatarURL: "https://avatars2.githubusercontent.com/u/1185253",
		Login:     "mrnugget",
		URL:       "https://github.com/mrnugget",
	}

	githubPR := &github.PullRequest{
		ID:           "FOOBARID",
		Title:        "Fix a bunch of bugs",
		Body:         "This fixes a bunch of bugs",
		URL:          "https://github.com/sourcegraph/sourcegraph/pull/12345",
		Number:       12345,
		State:        "MERGED",
		Author:       githubActor,
		Participants: []github.Actor{githubActor},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	changeset := &Changeset{
		RepoID:              42,
		CreatedAt:           now,
		UpdatedAt:           now,
		Metadata:            githubPR,
		CampaignIDs:         []int64{},
		ExternalID:          "12345",
		ExternalServiceType: extsvc.TypeGitHub,
	}

	title, err := changeset.Title()
	if err != nil {
		t.Fatal(err)
	}

	if want, have := githubPR.Title, title; want != have {
		t.Errorf("changeset title wrong. want=%q, have=%q", want, have)
	}

	body, err := changeset.Body()
	if err != nil {
		t.Fatal(err)
	}

	if want, have := githubPR.Body, body; want != have {
		t.Errorf("changeset body wrong. want=%q, have=%q", want, have)
	}

	state, err := changeset.state()
	if err != nil {
		t.Fatal(err)
	}

	if want, have := ChangesetStateMerged, state; want != have {
		t.Errorf("changeset state wrong. want=%q, have=%q", want, have)
	}

	url, err := changeset.URL()
	if err != nil {
		t.Fatal(err)
	}

	if want, have := githubPR.URL, url; want != have {
		t.Errorf("changeset url wrong. want=%q, have=%q", want, have)
	}
}

func TestChangesetEvents(t *testing.T) {
	type testCase struct {
		name      string
		changeset Changeset
		events    []*ChangesetEvent
	}

	var cases []testCase

	{ // Github

		now := time.Now().UTC()

		reviewComments := []*github.PullRequestReviewComment{
			{DatabaseID: 1, Body: "foo"},
			{DatabaseID: 2, Body: "bar"},
			{DatabaseID: 3, Body: "baz"},
		}

		actor := github.Actor{Login: "john-doe"}

		assignedEvent := &github.AssignedEvent{
			Actor:     actor,
			Assignee:  actor,
			CreatedAt: now,
		}

		unassignedEvent := &github.UnassignedEvent{
			Actor:     actor,
			Assignee:  actor,
			CreatedAt: now,
		}

		closedEvent := &github.ClosedEvent{
			Actor:     actor,
			CreatedAt: now,
		}

		commit := &github.PullRequestCommit{
			Commit: github.Commit{
				OID: "123",
			},
		}

		cases = append(cases, testCase{"github",
			Changeset{
				ID: 23,
				Metadata: &github.PullRequest{
					TimelineItems: []github.TimelineItem{
						{Type: "AssignedEvent", Item: assignedEvent},
						{Type: "PullRequestReviewThread", Item: &github.PullRequestReviewThread{
							Comments: reviewComments[:2],
						}},
						{Type: "UnassignedEvent", Item: unassignedEvent},
						{Type: "PullRequestReviewThread", Item: &github.PullRequestReviewThread{
							Comments: reviewComments[2:],
						}},
						{Type: "ClosedEvent", Item: closedEvent},
						{Type: "PullRequestCommit", Item: commit},
					},
				},
			},
			[]*ChangesetEvent{{
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubAssigned,
				Key:         assignedEvent.Key(),
				Metadata:    assignedEvent,
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubReviewCommented,
				Key:         reviewComments[0].Key(),
				Metadata:    reviewComments[0],
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubReviewCommented,
				Key:         reviewComments[1].Key(),
				Metadata:    reviewComments[1],
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubUnassigned,
				Key:         unassignedEvent.Key(),
				Metadata:    unassignedEvent,
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubReviewCommented,
				Key:         reviewComments[2].Key(),
				Metadata:    reviewComments[2],
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubClosed,
				Key:         closedEvent.Key(),
				Metadata:    closedEvent,
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubCommit,
				Key:         commit.Key(),
				Metadata:    commit,
			}},
		})

		reviewRequestedActorEvent := &github.ReviewRequestedEvent{
			RequestedReviewer: github.Actor{Login: "the-great-tortellini"},
			Actor:             actor,
			CreatedAt:         now,
		}
		reviewRequestedTeamEvent := &github.ReviewRequestedEvent{
			RequestedTeam: github.Team{Name: "the-belgian-waffles"},
			Actor:         actor,
			CreatedAt:     now,
		}

		cases = append(cases, testCase{"github-blank-review-requested",
			Changeset{
				ID: 23,
				Metadata: &github.PullRequest{
					TimelineItems: []github.TimelineItem{
						{Type: "ReviewRequestedEvent", Item: reviewRequestedActorEvent},
						{Type: "ReviewRequestedEvent", Item: reviewRequestedTeamEvent},
						{Type: "ReviewRequestedEvent", Item: &github.ReviewRequestedEvent{
							// Both Team and Reviewer are blank.
							Actor:     actor,
							CreatedAt: now,
						}},
					},
				},
			},
			[]*ChangesetEvent{{
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubReviewRequested,
				Key:         reviewRequestedActorEvent.Key(),
				Metadata:    reviewRequestedActorEvent,
			}, {
				ChangesetID: 23,
				Kind:        ChangesetEventKindGitHubReviewRequested,
				Key:         reviewRequestedTeamEvent.Key(),
				Metadata:    reviewRequestedTeamEvent,
			}},
		})
	}

	{ // Bitbucket Server

		user := bitbucketserver.User{Name: "john-doe"}
		reviewer := bitbucketserver.User{Name: "jane-doe"}

		activities := []*bitbucketserver.Activity{{
			ID:     1,
			User:   user,
			Action: bitbucketserver.OpenedActivityAction,
		}, {
			ID:     2,
			User:   reviewer,
			Action: bitbucketserver.ReviewedActivityAction,
		}, {
			ID:     3,
			User:   reviewer,
			Action: bitbucketserver.DeclinedActivityAction,
		}, {
			ID:     4,
			User:   user,
			Action: bitbucketserver.ReopenedActivityAction,
		}, {
			ID:     5,
			User:   user,
			Action: bitbucketserver.MergedActivityAction,
		}}

		cases = append(cases, testCase{"bitbucketserver",
			Changeset{
				ID: 24,
				Metadata: &bitbucketserver.PullRequest{
					Activities: activities,
				},
			},
			[]*ChangesetEvent{{
				ChangesetID: 24,
				Kind:        ChangesetEventKindBitbucketServerOpened,
				Key:         activities[0].Key(),
				Metadata:    activities[0],
			}, {
				ChangesetID: 24,
				Kind:        ChangesetEventKindBitbucketServerReviewed,
				Key:         activities[1].Key(),
				Metadata:    activities[1],
			}, {
				ChangesetID: 24,
				Kind:        ChangesetEventKindBitbucketServerDeclined,
				Key:         activities[2].Key(),
				Metadata:    activities[2],
			}, {
				ChangesetID: 24,
				Kind:        ChangesetEventKindBitbucketServerReopened,
				Key:         activities[3].Key(),
				Metadata:    activities[3],
			}, {
				ChangesetID: 24,
				Kind:        ChangesetEventKindBitbucketServerMerged,
				Key:         activities[4].Key(),
				Metadata:    activities[4],
			}},
		})
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			have := tc.changeset.Events()
			want := tc.events

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestChangesetDiffStat(t *testing.T) {
	var (
		added   int32 = 77
		changed int32 = 88
		deleted int32 = 99
	)

	for name, tc := range map[string]struct {
		c    Changeset
		want *diff.Stat
	}{
		"added missing": {
			c: Changeset{
				DiffStatAdded:   nil,
				DiffStatChanged: &changed,
				DiffStatDeleted: &deleted,
			},
			want: nil,
		},
		"changed missing": {
			c: Changeset{
				DiffStatAdded:   &added,
				DiffStatChanged: nil,
				DiffStatDeleted: &deleted,
			},
			want: nil,
		},
		"deleted missing": {
			c: Changeset{
				DiffStatAdded:   &added,
				DiffStatChanged: &changed,
				DiffStatDeleted: nil,
			},
			want: nil,
		},
		"all present": {
			c: Changeset{
				DiffStatAdded:   &added,
				DiffStatChanged: &changed,
				DiffStatDeleted: &deleted,
			},
			want: &diff.Stat{
				Added:   added,
				Changed: changed,
				Deleted: deleted,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			have := tc.c.DiffStat()
			if (tc.want == nil && have != nil) || (tc.want != nil && have == nil) {
				t.Errorf("mismatched nils in diff stats: have %+v; want %+v", have, tc.want)
			} else if tc.want != nil && have != nil {
				if d := cmp.Diff(*have, *tc.want); d != "" {
					t.Errorf("incorrect diff stat: %s", d)
				}
			}
		})
	}
}

type changesetSyncStateTestCase struct {
	state [2]ChangesetSyncState
	want  bool
}

func TestChangesetSyncStateEquals(t *testing.T) {
	testCases := make(map[string]changesetSyncStateTestCase)

	for baseName, basePairs := range map[string][2]string{
		"base equal":     {"abc", "abc"},
		"base different": {"abc", "def"},
	} {
		for headName, headPairs := range map[string][2]string{
			"head equal":     {"abc", "abc"},
			"head different": {"abc", "def"},
		} {
			for completeName, completePairs := range map[string][2]bool{
				"complete both true":  {true, true},
				"complete both false": {false, false},
				"complete different":  {true, false},
			} {
				key := fmt.Sprintf("%s; %s; %s", baseName, headName, completeName)

				testCases[key] = changesetSyncStateTestCase{
					state: [2]ChangesetSyncState{
						{
							BaseRefOid: basePairs[0],
							HeadRefOid: headPairs[0],
							IsComplete: completePairs[0],
						},
						{
							BaseRefOid: basePairs[1],
							HeadRefOid: headPairs[1],
							IsComplete: completePairs[1],
						},
					},
					// This is icky, but works, and means we're not just
					// repeating the implementation of Equals().
					want: strings.HasPrefix(key, "base equal; head equal; complete both"),
				}
			}
		}
	}

	for name, tc := range testCases {
		if have := tc.state[0].Equals(&tc.state[1]); have != tc.want {
			t.Errorf("%s: unexpected Equals result: have %v; want %v", name, have, tc.want)
		}
	}
}
