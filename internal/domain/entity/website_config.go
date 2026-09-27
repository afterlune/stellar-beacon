package entity

// WebsiteConfig is the site's persisted configuration shape. JSON field names
// are retained for compatibility with existing database rows and API output.
type WebsiteConfig struct {
	Name              string `json:"name"`
	EnglishName       string `json:"englishName"`
	Logo              string `json:"logo"`
	MultiLanguage     int    `json:"multiLanguage"`
	Notice            string `json:"notice"`
	WebsiteCreateTime string `json:"websiteCreateTime"`
	BeianNumber       string `json:"beianNumber"`
	TouristAvatar     string `json:"touristAvatar"`
	UserAvatar        string `json:"userAvatar"`
	IsCommentReview   int    `json:"isCommentReview"`
	IsEmailNotice     int    `json:"isEmailNotice"`
}
