import { kyWrapper } from '@/lib/ky-wrapper';
import type { RecommendationDto } from '@/types';

export const getRecommendations = async (category: string): Promise<RecommendationDto[]> => {
  const response: any = await kyWrapper.get(`recommendations?category=${category}`).json();
  return response ? response.results : [];
};

export const followRecommendation = async (tmdbId: number, mediaType: string) => {
  const response: any = await kyWrapper
    .post('tmdb/follow', {
      json: { tmdbId, mediaType },
    })
    .json();

  return response;
};
