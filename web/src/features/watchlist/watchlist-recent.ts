import { Router } from '@vaadin/router';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { type TemplateResult, css, html } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

import { BaseElement } from '@/components/base-element';
import { watchEpisode } from '@/features/shows/shows-api';
import { askForRating, type RatingAskResult } from '@/features/shows/rating-dialog';
import { sharedStyles } from '@/styles/shared-styles';
import type { WatchlistEpisodeDto } from '@/types';
import { createToastEvent, formatAirTime } from '@/utils';

import { getWatchListRecent } from './wachlist-api';

import '@/features/shows/shows-search-form';
import '@/layout/app-layout';

import '@shoelace-style/shoelace/dist/components/button/button.js';
import '@shoelace-style/shoelace/dist/components/icon/icon.js';

dayjs.extend(relativeTime);

@customElement('watchlist-recent')
export class WatchListRecent extends BaseElement {
  @state() episodes?: WatchlistEpisodeDto[];
  @property({ type: String }) mediaType: 'all' | 'tv' | 'movie' = 'all';

  render() {
    const visibleEpisodes = this.getVisibleEpisodes();

    return html`
      ${
        visibleEpisodes
          ? visibleEpisodes.length > 0
            ? html`
              <ul class="episode-list">
                ${visibleEpisodes.map(episode => {
                  return html`<li>${this.renderEpisode(episode)}</li>`;
                })}
              </ul>
            `
            : html`<div>No recently watched episodes</div>`
          : null
      }
    `;
  }

  private getVisibleEpisodes(): WatchlistEpisodeDto[] | undefined {
    if (!this.episodes) return undefined;
    if (this.mediaType === 'all') return this.episodes;
    if (this.mediaType === 'tv') {
      // legacy TV entries have no showMediaType set - treat empty as TV
      return this.episodes.filter(episode => episode.showMediaType === 'tv' || !episode.showMediaType);
    }
    return this.episodes.filter(episode => episode.showMediaType === 'movie');
  }

  private renderEpisode(episode: WatchlistEpisodeDto): TemplateResult | null {
    const isMovie = episode.showMediaType === 'movie';
    return episode.id
      ? html`
          <div class="episode">
            <a href="#" class="show-image-container" @click=${(event: Event) => this.handleClickShow(event, episode)}>
              <img
                class="show-image"
                src=${episode.imageMedium ?? '/images/empty-image.svg'}
                alt=${episode.showName ?? 'Unknown show'}
              />
            </a>
            <div class="episode-info">
              <h1>${episode.showName}</h1>
              <div class="episode-name">${episode.name}</div>
              <div class="episode-summary">${episode.summary}</div>
              <ul class="detail-list">
                ${
                  isMovie
                    ? html`<li class="movie-badge">Movie</li>`
                    : html`<li>S${episode.seasonNumber} E${episode.number}</li>`
                }
                ${episode.network ? html`<li>${episode.network}</li>` : null}
                ${episode.aired ? html`<li>${formatAirTime(episode.aired)}</li>` : null}
                ${episode.runtime ? html`<li>${episode.runtime}m</li>` : null}
              </ul>
              <div class="watched-date">Watched ${dayjs(episode.watchedAt).fromNow()}</div>
            </div>
          </div>
        `
      : null;
  }

  private handleClickShow(event: Event, episode: WatchlistEpisodeDto) {
    event.preventDefault();
    if (episode.showMediaType === 'movie') {
      Router.go(`/show/view/${episode.showId}`);
      return;
    }
    Router.go(`/show/view/${episode.showId}?season=${episode.seasonNumber}`);
  }

  async handleWatch(episode: WatchlistEpisodeDto) {
    if (episode && episode.id) {
      const result: RatingAskResult = await askForRating(episode.showName ?? episode.name);
      if (result === 'cancel') return;
      const score = result === 'skip' ? 0 : result;
      await watchEpisode(episode.id, true, false, score);
      this.episodes = await this.callApi(() => getWatchListRecent());
      this.dispatchEvent(
        createToastEvent({
          variant: 'success',
          message: `"${episode.name}" marked as watched`,
        })
      );
    }
  }

  async connectedCallback() {
    super.connectedCallback();
    this.episodes = await this.callApi(() => getWatchListRecent());
  }

  static styles = [
    sharedStyles,
    css`
      .episode-list {
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-medium);
        list-style: none;
        margin: 0;
        padding: 0;
      }

      .episode {
        display: flex;
        align-items: start;
        gap: var(--sl-spacing-small);
        border-bottom: 1px solid var(--sl-color-neutral-100);
        padding-bottom: var(--sl-spacing-medium);
      }

      .episode h1 {
        margin: 0;
        padding: 0;
        font-weight: var(--sl-font-weight-semibold);
        font-size: var(--sl-font-size-small);
      }

      .episode-info {
        flex-grow: 1;
      }

      .show-image {
        width: 4rem;
        display: block;
        height: auto;
        border-radius: var(--sl-border-radius-medium);
      }

      .episode-summary {
        font-size: var(--sl-font-size-x-small);
        margin-block: var(--sl-spacing-2x-small);
        color: var(--sl-color-neutral-500);
      }

      .detail-list {
        list-style: none;
        padding: 0;
        font-size: var(--sl-font-size-small);
        color: var(--sl-color-neutral-600);
      }

      .detail-list li {
        padding: var(--sl-spacing-3x-small) 0;
      }

      .watched-date {
        font-size: var(--sl-font-size-x-small);
        margin-top: var(--sl-spacing-2x-small);
        color: var(--sl-color-neutral-700);
      }

      .movie-badge {
        font-weight: var(--sl-font-weight-semibold);
        color: var(--sl-color-primary-600);
      }

      @media screen and (min-width: 640px) {
        .detail-list {
          display: flex;
          align-items: center;
          padding-top: var(--sl-spacing-2x-small);
        }

        .detail-list li:not(:last-child):after {
          content: ' · ';
          padding-right: var(--sl-spacing-2x-small);
        }
      }
    `,
  ];
}
