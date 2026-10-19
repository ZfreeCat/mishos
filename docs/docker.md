# MISHOS

## Docker Configuration

Here is an example of the `docker-compose.yml` file that I'm personally using:

```yaml
services:
  mishos:
    image: bljohnsondev/mishos:latest
    container_name: mishos
    ports:
      - 8080:80
    environment:
      - TZ=America/Chicago
      - DB_TYPE=sqlite
      - DB_URL=/db/mishos.db
      - SECRET=YOUR_SECRET_KEY_HERE
      - WATCHLIST_RECENT_LIMIT=30
      - TOKEN_DURATION=90
      # - CRON_PROVIDER_UPDATE=21 8 * * *
      # - RUN_MIGRATION=1
    volumes:
      - /home/brent/temp/mishos:/db
    restart: unless-stopped
```

Here is a quick breakdown of the environment variables:

**SQLite**

```
DB_TYPE=sqlite
DB_URL=/db/mishos.db
```

When running this in a docker container don't forget to add a volume mapping for your `DB_URL` SQLite file location.

**MySQL**

```
DB_TYPE=mysql
DB_URL=username:password@tcp(mariadb:3306)/mishos?charset=utf8mb4&parseTime=True&loc=Local
```

Here are some other important environment variables:

- `TZ` - Your preferred timezone if different than the system.
- `SECRET` - Set this to whatever you want but definitely set it to something.  This is the signing key used for authentication.
- `WATCHLIST_RECENT_LIMIT` - This is how many episodes to show on the recently watched tab.
- `CRON_PROVIDER_UPDATE` - This is a cron string that defines the interval to get show updates from the provider.  If unset it defaults to 12 AM.
- `RUN_MIGRATION` - This should be set to `1` for the **first run of the app** only.  This instructs the backend to initialize the empty database. **Set it to `1` again after upgrading to pull in the new tables/columns.**
- `TOKEN_DURATION` - This is the number of days the authentication token is valid.

**TMDB**

- `TMDB_API_KEY` - Your TMDB API key (https://www.themoviedb.org/settings/api). Required for the TMDB watchlist sync and the Discover (recommendations) feature.
- `TMDB_API_URL` - Defaults to `https://api.themoviedb.org/3`. If your host cannot reach api.themoviedb.org directly (e.g. mainland China), point this at a reverse proxy / mirror.
- `TMDB_IMAGE_URL` - Defaults to `https://image.tmdb.org/t/p`. Same reachability caveat as above.

**Local image cache**

TVmaze and TMDB image CDNs are unreachable from mainland China, so images are downloaded into a local cache and served from `/api/images/...`:

- `IMAGE_CACHE_DIR` - Directory where cached images are stored. Defaults to `./data/images`. In Docker, map a volume here (e.g. `IMAGE_CACHE_DIR=/images` and a `- /your/path:/images` volume) so the cache survives container rebuilds.
- `IMAGE_ALLOWED_DOMAINS` - Comma-separated extra domains allowed for image downloads (defaults to `tvmaze.com` and `tmdb.org`).
- `CRON_IMAGE_PREFETCH` - Cron string to prefetch all followed show images. Defaults to 4 AM.

**TMDB watchlist sync**

- `CRON_TMDB_SYNC` - Cron string to automatically sync every connected user's TMDB watchlist. Defaults to 3 AM. Users can also sync manually from Settings -> TMDB.

## First Run

**IMPORTANT**: For the first run with an empty database you need to set `RUN_MIGRATION=1` environment variable in the docker compose file. This causes the backend to initialize the database.  Once this has been completed and your initial user has been created I suggest commenting out `RUN_MIGRATION=1` in the docker compose file and then doing a docker compose `down` and `up` for the change to take effect.  This flag is only needed the first time the application is started (or in the future if any database schema changes are made).

Once everything is set and Mishos has been started for the first time visit `/onboarding` in your browser.
This is a one-time only screen that creates your initial username and password.
Once your user has been created this URL will no longer be available.
