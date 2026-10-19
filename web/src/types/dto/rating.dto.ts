export interface RatingDto {
  tmdbId: number;
  mediaType: 'tv' | 'movie';
  name: string;
  overview: string;
  posterUrl: string;
  date: string;
  voteAverage: number;
  rating: number;
  showId?: number;
  followed: boolean;
}
