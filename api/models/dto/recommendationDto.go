package modelsdto

// RecommendationDto is one entry returned by the new-release
// recommendation endpoint (upcoming / new / trending).
type RecommendationDto struct {
	TmdbId      int     `json:"tmdbId"`
	MediaType   string  `json:"mediaType"`
	Name        string  `json:"name"`
	Overview    string  `json:"overview"`
	PosterUrl   string  `json:"posterUrl"`
	Date        string  `json:"date"`
	VoteAverage float64 `json:"voteAverage"`
	ShowId      *uint   `json:"showId"`
	Followed    bool    `json:"followed"`
}
