import { kyWrapper } from '@/lib/ky-wrapper';

export const getTmdbStatus = async () => {
  const response: any = await kyWrapper.get('tmdb/status').json();
  return response;
};

export const createTmdbAuthToken = async () => {
  const response: any = await kyWrapper.post('tmdb/auth/token').json();
  return response;
};

export const completeTmdbAuth = async (requestToken: string) => {
  const response: any = await kyWrapper
    .post('tmdb/auth/session', {
      json: { requestToken },
    })
    .json();

  return response;
};

export const disconnectTmdb = async () => {
  await kyWrapper.post('tmdb/disconnect');
};

export const syncTmdbWatchlist = async () => {
  const response: any = await kyWrapper.post('tmdb/sync').json();
  return response;
};
