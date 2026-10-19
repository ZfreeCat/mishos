import { kyWrapper } from '@/lib/ky-wrapper';
import type { FollowShowResponseDto, RatingDto, ShowDto } from '@/types';

const RATINGS_CACHE_PREFIX = 'mishos_ratings_cache_';
const RATINGS_CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes short TTL

export const getFollowedShows = async (): Promise<ShowDto[]> => {
  const response: any = await kyWrapper.get('show/followed').json();
  return response ? response.followedShows : [];
};

// Ratings are cached in localStorage with a short TTL: first paint is instant,
// and a stale cache is silently refreshed by the network call below.
export const getMyRatings = async (mediaType: 'all' | 'tv' | 'movie' = 'all'): Promise<RatingDto[]> => {
  const cacheKey = `${RATINGS_CACHE_PREFIX}${mediaType}`;

  try {
    const cached = localStorage.getItem(cacheKey);
    if (cached) {
      const parsed = JSON.parse(cached);
      if (parsed && typeof parsed.ts === 'number' && Date.now() - parsed.ts < RATINGS_CACHE_TTL_MS) {
        return Array.isArray(parsed.data) ? parsed.data : [];
      }
    }
  } catch {
    // corrupted cache - ignore and refetch
  }

  const response: any = await kyWrapper.get(`tmdb/ratings?mediaType=${mediaType}`).json();
  const results = response ? response.results : [];

  try {
    localStorage.setItem(cacheKey, JSON.stringify({ ts: Date.now(), data: results }));
  } catch {
    // storage full / blocked - cache is optional
  }

  return results;
};

// Import a movie/TV show (by TMDB id + media type) into the local library,
// then jump to its detail page. Returns the local show id.
export const importShow = async (tmdbId: number, mediaType: string): Promise<{ showId?: number }> => {
  const response: any = await kyWrapper
    .post('tmdb/import', {
      json: { tmdbId, mediaType },
    })
    .json();
  return response ?? {};
};

export const getShowDetails = async (showId: string): Promise<ShowDto | undefined> => {
  //const show: ShowDto | undefined = await kyWrapper.get(`show/view/${showId}`).json();
  const response: any = await kyWrapper.get(`show/view/${showId}`).json();
  return response.show as ShowDto;
};

export const watchEpisode = async (episodeId: number, watch: boolean, previous = false) => {
  // toggle previous should only apply when marking watched
  if (watch && previous) {
    const response: any = await kyWrapper.post('episode/watchprevious', {
      json: {
        episodeId,
      },
    });
    return response?.watched === true;
  }

  const response: any = await kyWrapper.post(watch ? 'episode/watch' : 'episode/unwatch', {
    json: {
      episodeId,
    },
  });
  return response?.watched === true;
};

export const getProviderPreview = async (providerId: string): Promise<ShowDto> => {
  const response: any = await kyWrapper.get(`show/preview/${providerId}`).json();
  return response.show as ShowDto;
};

export const followShow = async (providerId: string): Promise<FollowShowResponseDto | undefined> => {
  const response: any = await kyWrapper
    .post('show/add', {
      json: {
        providerId: providerId.toString(),
      },
    })
    .json();

  return {
    id: response?.show?.id,
    name: response?.show?.name,
  };
};

export const unfollowShow = async (show: ShowDto): Promise<void> => {
  await kyWrapper.post('show/unfollow', {
    json: {
      showId: show.id,
    },
  });
};

export const refreshShow = async (showId: string): Promise<void> => {
  await kyWrapper.get(`show/update/${showId}`);
};

export const watchShow = async (showId: number): Promise<boolean> => {
  const response: any = await kyWrapper
    .post('show/watch', {
      json: { showId },
    })
    .json();
  return response?.watched === true;
};

export const unwatchShow = async (showId: number): Promise<void> => {
  await kyWrapper.post('show/unwatch', {
    json: { showId },
  });
};
