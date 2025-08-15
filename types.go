package xapi

import (
	"time"
)

// User represents a comprehensive Twitter user profile with all available metadata.
//
// This structure contains detailed information about a Twitter user including
// their basic profile information, statistics, verification status, and media URLs.
// All fields are populated from Twitter's API response when available.
//
// Key fields:
//   - ID/RestID: Unique Twitter user identifier
//   - Name: Display name (e.g., "NASA")
//   - ScreenName: Username without @ (e.g., "nasa")
//   - Description: User bio/description text
//   - FollowersCount: Number of followers
//   - FriendsCount: Number of accounts following
//   - StatusesCount: Total number of tweets
//   - Verified/IsBlueVerified: Verification status
//   - ProfileImageURL/ProfileBannerURL: Profile media
type User struct {
	ID                        string    `json:"id"`
	RestID                    string    `json:"rest_id"`
	Name                      string    `json:"name"`
	ScreenName                string    `json:"screen_name"`
	Description               string    `json:"description"`
	Location                  string    `json:"location"`
	URL                       string    `json:"url"`
	Protected                 bool      `json:"protected"`
	FollowersCount            int       `json:"followers_count"`
	FriendsCount              int       `json:"friends_count"`
	StatusesCount             int       `json:"statuses_count"`
	CreatedAt                 time.Time `json:"created_at"`
	Verified                  bool      `json:"verified"`
	VerifiedType              string    `json:"verified_type"`
	IsBlueVerified            bool      `json:"is_blue_verified"`
	ProfileImageURL           string    `json:"profile_image_url"`
	ProfileBannerURL          string    `json:"profile_banner_url"`
	DefaultProfile            bool      `json:"default_profile"`
	DefaultProfileImage       bool      `json:"default_profile_image"`
	FavouritesCount           int       `json:"favourites_count"`
	ListedCount               int       `json:"listed_count"`
	MediaCount                int       `json:"media_count"`
	PinnedTweetIDs            []string  `json:"pinned_tweet_ids_str"`
	PossiblySensitive         bool      `json:"possibly_sensitive"`
	ProfileInterstitialType   string    `json:"profile_interstitial_type"`
	TranslatorType            string    `json:"translator_type"`
	WithheldInCountries       []string  `json:"withheld_in_countries"`
	FastFollowersCount        int       `json:"fast_followers_count"`
	NormalFollowersCount      int       `json:"normal_followers_count"`
	HasCustomTimelines        bool      `json:"has_custom_timelines"`
	IsTranslator              bool      `json:"is_translator"`
}

// Profile combines user data with their tweets
type Profile struct {
	User   *User    `json:"user"`
	Tweets []*Tweet `json:"tweets"`
	Stats  *Stats   `json:"stats"`
}

// Stats provides profile analytics
type Stats struct {
	TotalEngagement int     `json:"total_engagement"`
	AvgEngagement   float64 `json:"avg_engagement"`
	TopTweet        *Tweet  `json:"top_tweet"`
}

// Cursor represents pagination cursor for timeline navigation
type Cursor struct {
	Value    string `json:"value"`
	CursorType string `json:"cursorType"`
}

// TweetPage represents a paginated response of tweets
type TweetPage struct {
	Tweets     []*Tweet `json:"tweets"`
	NextCursor *Cursor  `json:"next_cursor,omitempty"`
	PrevCursor *Cursor  `json:"prev_cursor,omitempty"`
	HasMore    bool     `json:"has_more"`
}

// UserPage represents a paginated response of users  
type UserPage struct {
	Users      []*User `json:"users"`
	NextCursor *Cursor `json:"next_cursor,omitempty"`
	PrevCursor *Cursor `json:"prev_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

// Broadcast represents a live stream/broadcast
type Broadcast struct {
	ID              string `json:"id"`
	MediaKey        string `json:"media_key"`
	Title           string `json:"title"`
	State           string `json:"state"`
	TotalWatching   int    `json:"total_watching"`
	Source          string `json:"source"`
	Location        string `json:"location"`
	Language        string `json:"language"`
	StartTime       string `json:"start_time"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	ChatToken       string `json:"chat_token"`
	ChatPermission  string `json:"chat_permission"`
	Status          string `json:"status"`
	IsLiveBroadcast bool   `json:"is_live_broadcast"`
}

// Tweet represents a Twitter tweet/post with comprehensive metadata and engagement metrics.
//
// This structure contains all available information about a tweet including
// its content, engagement statistics, creation time, and related metadata.
// The Author field may be populated with user information when available.
//
// Key fields:
//   - ID/RestID: Unique tweet identifier
//   - FullText: Complete tweet text content
//   - CreatedAt: When the tweet was posted
//   - FavoriteCount: Number of likes/hearts
//   - RetweetCount: Number of retweets
//   - ReplyCount: Number of replies
//   - ViewCount: Number of views (when available)
//   - Author: User object for tweet author (optional)
//   - ConversationID: Thread identifier for replies
type Tweet struct {
	ID              string    `json:"id"`
	RestID          string    `json:"rest_id"`
	FullText        string    `json:"full_text"`
	CreatedAt       time.Time `json:"created_at"`
	ConversationID  string    `json:"conversation_id_str"`
	InReplyToUserID string    `json:"in_reply_to_user_id_str"`
	Author          *User     `json:"author,omitempty"`
	
	// Engagement metrics
	BookmarkCount int `json:"bookmark_count"`
	FavoriteCount int `json:"favorite_count"`
	QuoteCount    int `json:"quote_count"`
	ReplyCount    int `json:"reply_count"`
	RetweetCount  int `json:"retweet_count"`
	ViewCount     int `json:"view_count"`
	
	// Status flags
	Bookmarked       bool `json:"bookmarked"`
	Favorited        bool `json:"favorited"`
	Retweeted        bool `json:"retweeted"`
	IsQuoteStatus    bool `json:"is_quote_status"`
	PossiblySensitive bool `json:"possibly_sensitive"`
	
	// Content metadata
	DisplayTextRange []int    `json:"display_text_range"`
	Language         string   `json:"lang"`
	Source           string   `json:"source"`
	UserIDStr        string   `json:"user_id_str"`
	
	// Rich content
	Entities         *TweetEntities    `json:"entities,omitempty"`
	ExtendedEntities *ExtendedEntities `json:"extended_entities,omitempty"`
	
	// Edit information
	EditControl *EditControl `json:"edit_control,omitempty"`
	
	// Additional metadata
	IsTranslatable bool   `json:"is_translatable"`
	NoteType       string `json:"note_type,omitempty"`
}

// TweetEntities contains tweet text entities like URLs, mentions, hashtags
type TweetEntities struct {
	Hashtags     []Hashtag     `json:"hashtags"`
	Symbols      []Symbol      `json:"symbols"`
	UserMentions []UserMention `json:"user_mentions"`
	URLs         []URLEntity   `json:"urls"`
	Media        []Media       `json:"media,omitempty"`
}

// ExtendedEntities contains additional media information
type ExtendedEntities struct {
	Media []Media `json:"media"`
}

// Hashtag represents a hashtag entity in tweet text
type Hashtag struct {
	Indices []int  `json:"indices"`
	Text    string `json:"text"`
}

// Symbol represents a financial symbol entity in tweet text
type Symbol struct {
	Indices []int  `json:"indices"`
	Text    string `json:"text"`
}

// UserMention represents a user mention in tweet text
type UserMention struct {
	IDStr      string `json:"id_str"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	Indices    []int  `json:"indices"`
}

// URLEntity represents a URL in tweet text
type URLEntity struct {
	URL         string `json:"url"`
	ExpandedURL string `json:"expanded_url"`
	DisplayURL  string `json:"display_url"`
	Indices     []int  `json:"indices"`
}

// Media represents media attachments in tweets
type Media struct {
	ID          string      `json:"id_str"`
	MediaKey    string      `json:"media_key"`
	MediaURL    string      `json:"media_url_https"`
	URL         string      `json:"url"`
	DisplayURL  string      `json:"display_url"`
	ExpandedURL string      `json:"expanded_url"`
	Type        string      `json:"type"`
	Indices     []int       `json:"indices"`
	Sizes       MediaSizes  `json:"sizes"`
	VideoInfo   *VideoInfo  `json:"video_info,omitempty"`
}

// MediaSizes contains different sizes for media
type MediaSizes struct {
	Thumb  MediaSize `json:"thumb"`
	Small  MediaSize `json:"small"`
	Medium MediaSize `json:"medium"`
	Large  MediaSize `json:"large"`
}

// MediaSize represents dimensions and resize info for media
type MediaSize struct {
	Width  int    `json:"w"`
	Height int    `json:"h"`
	Resize string `json:"resize"`
}

// VideoInfo contains video-specific information
type VideoInfo struct {
	AspectRatio    []int          `json:"aspect_ratio"`
	DurationMillis int            `json:"duration_millis"`
	Variants       []VideoVariant `json:"variants"`
}

// VideoVariant represents different quality versions of a video
type VideoVariant struct {
	Bitrate     int    `json:"bitrate,omitempty"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

// EditControl contains tweet edit information
type EditControl struct {
	EditTweetIDs      []string `json:"edit_tweet_ids"`
	EditableUntilMsec string   `json:"editable_until_msecs"`
	IsEditEligible    bool     `json:"is_edit_eligible"`
	EditsRemaining    string   `json:"edits_remaining"`
}

// Timeline represents a collection of tweets in chronological order
type Timeline struct {
	Instructions []TimelineInstruction `json:"instructions"`
}

// TimelineInstruction represents instructions for rendering timeline content
type TimelineInstruction struct {
	Type    string           `json:"type"`
	Entries []TimelineEntry  `json:"entries,omitempty"`
}

// TimelineEntry represents an entry in the timeline
type TimelineEntry struct {
	EntryID   string               `json:"entryId"`
	SortIndex string               `json:"sortIndex"`
	Content   TimelineEntryContent `json:"content"`
}

// TimelineEntryContent contains the actual content of a timeline entry
type TimelineEntryContent struct {
	EntryType   string                 `json:"entryType"`
	Typename    string                 `json:"__typename"`
	ItemContent *TimelineItemContent   `json:"itemContent,omitempty"`
	Value       string                 `json:"value,omitempty"`
	CursorType  string                 `json:"cursorType,omitempty"`
}

// TimelineItemContent contains tweet content within timeline items
type TimelineItemContent struct {
	ItemType     string       `json:"itemType"`
	Typename     string       `json:"__typename"`
	TweetResults *TweetResult `json:"tweet_results,omitempty"`
	UserResults  *UserResult  `json:"user_results,omitempty"`
}

// TweetResult wraps tweet data in API responses
type TweetResult struct {
	Result *TweetData `json:"result"`
}

// TweetData contains the main tweet information
type TweetData struct {
	Typename string     `json:"__typename"`
	RestID   string     `json:"rest_id"`
	Core     *TweetCore `json:"core,omitempty"`
	Legacy   *Tweet     `json:"legacy,omitempty"`
	Views    *ViewCount `json:"views,omitempty"`
}

// TweetCore contains core tweet information including user data
type TweetCore struct {
	UserResults *UserResult `json:"user_results,omitempty"`
}

// UserResult wraps user data in API responses
type UserResult struct {
	Result *UserData `json:"result"`
}

// UserData contains the main user information
type UserData struct {
	Typename string    `json:"__typename"`
	ID       string    `json:"id"`
	RestID   string    `json:"rest_id"`
	Core     *UserCore `json:"core,omitempty"`
	Legacy   *User     `json:"legacy,omitempty"`
}

// UserCore contains core user information
type UserCore struct {
	CreatedAt  string `json:"created_at"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
}

// ViewCount represents tweet view statistics
type ViewCount struct {
	Count string `json:"count"`
	State string `json:"state"`
}

// APIError represents errors returned by the Twitter API
type APIError struct {
	Message    string                 `json:"message"`
	Code       int                    `json:"code"`
	Kind       string                 `json:"kind"`
	Name       string                 `json:"name"`
	Source     string                 `json:"source"`
	Locations  []ErrorLocation        `json:"locations,omitempty"`
	Path       []string               `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	Tracing    *ErrorTracing          `json:"tracing,omitempty"`
}

// ErrorLocation represents the location of an error in a GraphQL query
type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ErrorTracing contains error tracing information
type ErrorTracing struct {
	TraceID string `json:"trace_id"`
}







// GetHashtags extracts hashtag texts from tweet entities
func (t *Tweet) GetHashtags() []string {
	if t.Entities == nil {
		return nil
	}
	
	var hashtags []string
	for _, hashtag := range t.Entities.Hashtags {
		hashtags = append(hashtags, hashtag.Text)
	}
	
	return hashtags
}

// GetMentions extracts mentioned usernames from tweet entities
func (t *Tweet) GetMentions() []string {
	if t.Entities == nil {
		return nil
	}
	
	var mentions []string
	for _, mention := range t.Entities.UserMentions {
		mentions = append(mentions, mention.ScreenName)
	}
	
	return mentions
}

// GetURLs extracts URLs from tweet entities
func (t *Tweet) GetURLs() []string {
	if t.Entities == nil {
		return nil
	}
	
	var urls []string
	for _, url := range t.Entities.URLs {
		if url.ExpandedURL != "" {
			urls = append(urls, url.ExpandedURL)
		} else {
			urls = append(urls, url.URL)
		}
	}
	
	return urls
}

// GetMediaURLs extracts media URLs from tweet entities
func (t *Tweet) GetMediaURLs() []string {
	var mediaURLs []string
	
	// Check entities media
	if t.Entities != nil {
		for _, media := range t.Entities.Media {
			if media.MediaURL != "" {
				mediaURLs = append(mediaURLs, media.MediaURL)
			}
		}
	}
	
	// Check extended entities media
	if t.ExtendedEntities != nil {
		for _, media := range t.ExtendedEntities.Media {
			if media.MediaURL != "" {
				mediaURLs = append(mediaURLs, media.MediaURL)
			}
		}
	}
	
	return mediaURLs
}

// HasMedia checks if the tweet contains any media
func (t *Tweet) HasMedia() bool {
	if t.Entities != nil && len(t.Entities.Media) > 0 {
		return true
	}
	
	if t.ExtendedEntities != nil && len(t.ExtendedEntities.Media) > 0 {
		return true
	}
	
	return false
}


