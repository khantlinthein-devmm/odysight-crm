package leads

import "time"

type Status string

const (
	StatusNew       Status = "new"
	StatusContacted Status = "contacted"
	StatusQuoteSent Status = "quote_sent"
	StatusBooked    Status = "booked"
	StatusWon       Status = "won"
	StatusLost      Status = "lost"
)

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusContacted, StatusQuoteSent, StatusBooked, StatusWon, StatusLost:
		return true
	}
	return false
}

type Source string

const (
	SourceWebsite   Source = "website"
	SourceReferral  Source = "referral"
	SourceLine      Source = "line"
	SourceFacebook  Source = "facebook"
	SourceWalkIn    Source = "walk_in"
	SourceCampaign  Source = "campaign"
)

func (s Source) Valid() bool {
	switch s {
	case SourceWebsite, SourceReferral, SourceLine, SourceFacebook, SourceWalkIn, SourceCampaign:
		return true
	}
	return false
}

type Lead struct {
	ID             int64
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	Status         Status
	Source         Source
	LineUserID     string
	LinePictureURL string
	CreatedAt      time.Time
}
