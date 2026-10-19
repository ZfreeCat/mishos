export interface RecommendationDto {
  tmdbId: number;
  mediaType: string;
  name: string;
  overview: string;
  posterUrl: string;
  date: string;
  voteAverage: number;
  showId?: number;
  followed: boolean;
}
