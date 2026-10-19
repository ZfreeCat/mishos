package modelsdto

// RatingDto mirrors one entry of the user's TMDB "My Ratings" list.
type RatingDto struct {
	TmdbId      int     `json:"tmdbId"`
	MediaType   string  `json:"mediaType"`
	Name        string  `json:"name"`
	Overview    string  `json:"overview"`
	PosterUrl   string  `json:"posterUrl"`
	Date        string  `json:"date"`
	VoteAverage float64 `json:"voteAverage"`
	Rating      float64 `json:"rating"`
	ShowId      *uint   `json:"showId"`
	Followed    bool    `json:"followed"`
}
