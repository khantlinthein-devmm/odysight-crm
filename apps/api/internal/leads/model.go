package leads

import "time"

type Status string

const (
	StatusNew       Status = "new"
	StatusContacted Status = "contacted"
	StatusQualified Status = "qualified"
	StatusProposal  Status = "proposal"
	StatusWon       Status = "won"
	StatusLost      Status = "lost"
)

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusContacted, StatusQualified, StatusProposal, StatusWon, StatusLost:
		return true
	}
	return false
}

type Source string

const (
	SourceWebsite     Source = "website"
	SourceReferral    Source = "referral"
	SourceSocialMedia Source = "social_media"
	SourceWalkIn      Source = "walk_in"
	SourceCampaign    Source = "campaign"
)

func (s Source) Valid() bool {
	switch s {
	case SourceWebsite, SourceReferral, SourceSocialMedia, SourceWalkIn, SourceCampaign:
		return true
	}
	return false
}

type Lead struct {
	ID        int64
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Status    Status
	Source    Source
	CreatedAt time.Time
}
