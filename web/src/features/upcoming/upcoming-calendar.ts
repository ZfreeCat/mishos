import { Router } from '@vaadin/router';
import dayjs from 'dayjs';
import { LitElement, css, html } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import { map } from 'lit/directives/map.js';
import { repeat } from 'lit/directives/repeat.js';

import type { WatchlistEpisodeDto } from '@/types';

import '@shoelace-style/shoelace/dist/components/button/button.js';

@customElement('upcoming-calendar')
export class UpcomingCalendar extends LitElement {
  @property({ attribute: false }) episodes: WatchlistEpisodeDto[] = [];
  @state() private viewMonth = dayjs().startOf('month');

  render() {
    const monthStart = this.viewMonth;
    const daysInMonth = monthStart.daysInMonth();
    const firstWeekday = monthStart.day(); // Sunday = 0
    const todayKey = dayjs().format('YYYY-MM-DD');

    return html`
      <div class="calendar">
        <div class="calendar-header">
          <sl-button size="small" variant="neutral" @click=${this.prevMonth} aria-label="Previous month">‹</sl-button>
          <div class="calendar-title">${monthStart.format('MMMM YYYY')}</div>
          <sl-button size="small" variant="neutral" @click=${this.nextMonth} aria-label="Next month">›</sl-button>
        </div>
        <div class="calendar-grid">
          ${['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => html`<div class="weekday">${day}</div>`)}
          ${Array.from({ length: firstWeekday }, () => html`<div class="day empty"></div>`)}
          ${repeat(
            Array.from({ length: daysInMonth }, (_, i) => i + 1),
            day => day,
            day => {
              const date = monthStart.date(day);
              const dateKey = date.format('YYYY-MM-DD');
              const dayEpisodes = this.getEpisodesFor(dateKey);
              const isToday = dateKey === todayKey;
              return html`
              <div class="day ${dayEpisodes.length > 0 ? 'has-episodes' : ''} ${isToday ? 'today' : ''}">
                <div class="day-number">${day}</div>
                <div class="day-episodes">
                  ${map(dayEpisodes.slice(0, 3), episode => this.renderEpisodeItem(episode))}
                  ${dayEpisodes.length > 3 ? html`<div class="more">+${dayEpisodes.length - 3} more</div>` : null}
                </div>
              </div>
            `;
            }
          )}
        </div>
      </div>
    `;
  }

  private renderEpisodeItem(episode: WatchlistEpisodeDto) {
    const isMovie = episode.showMediaType === 'movie';
    return html`
      <a
        href="#"
        class="episode-item"
        @click=${(event: Event) => this.handleClickShow(event, episode)}
        title=${`${episode.showName}${isMovie ? '' : ` S${episode.seasonNumber}E${episode.number}`}`}
      >
        <span class="episode-show">${episode.showName}</span>
        ${isMovie ? html`<span class="episode-movie">Movie</span>` : html`<span class="episode-ep">S${episode.seasonNumber}E${episode.number}</span>`}
      </a>
    `;
  }

  private getEpisodesFor(dateKey: string): WatchlistEpisodeDto[] {
    return (this.episodes ?? []).filter(
      episode => episode.aired && dayjs(episode.aired).format('YYYY-MM-DD') === dateKey
    );
  }

  private handleClickShow(event: Event, episode: WatchlistEpisodeDto) {
    event.preventDefault();
    if (episode.showMediaType === 'movie') {
      Router.go(`/show/view/${episode.showId}`);
      return;
    }
    Router.go(`/show/view/${episode.showId}?season=${episode.seasonNumber}`);
  }

  private prevMonth() {
    this.viewMonth = this.viewMonth.subtract(1, 'month').startOf('month');
  }

  private nextMonth() {
    this.viewMonth = this.viewMonth.add(1, 'month').startOf('month');
  }

  static styles = css`
    .calendar {
      display: flex;
      flex-direction: column;
      gap: var(--sl-spacing-small);
    }

    .calendar-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--sl-spacing-small);
    }

    .calendar-header sl-button {
      font-size: var(--sl-font-size-x-large);
      font-weight: var(--sl-font-weight-semibold);
      line-height: 1;
    }

    .calendar-title {
      font-size: var(--sl-font-size-medium);
      font-weight: var(--sl-font-weight-semibold);
    }

    .calendar-grid {
      display: grid;
      grid-template-columns: repeat(7, 1fr);
      gap: var(--sl-spacing-2x-small);
    }

    .weekday {
      text-align: center;
      font-size: var(--sl-font-size-x-small);
      color: var(--sl-color-neutral-600);
      padding-block: var(--sl-spacing-2x-small);
    }

    .day {
      border: 1px solid var(--sl-color-neutral-100);
      border-radius: var(--sl-border-radius-small);
      min-height: 5.5rem;
      padding: var(--sl-spacing-2x-small);
      background-color: var(--sl-color-neutral-0);
      overflow: hidden;
    }

    .day.empty {
      border-color: transparent;
      background-color: transparent;
    }

    .day.today .day-number {
      background-color: var(--sl-color-primary-600);
      color: var(--sl-color-neutral-0);
      border-radius: 50%;
    }

    .day.has-episodes {
      border-color: var(--sl-color-primary-400);
    }

    .day-number {
      display: inline-block;
      min-width: 1.4rem;
      text-align: center;
      font-size: var(--sl-font-size-x-small);
      font-weight: var(--sl-font-weight-semibold);
      color: var(--sl-color-neutral-700);
      margin-bottom: var(--sl-spacing-2x-small);
      padding: 1px;
    }

    .day-episodes {
      display: flex;
      flex-direction: column;
      gap: 2px;
      overflow: hidden;
    }

    .episode-item {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: var(--sl-spacing-2x-small);
      text-decoration: none;
      font-size: var(--sl-font-size-x-small);
      padding: 1px 2px;
      border-radius: var(--sl-border-radius-small);
      background-color: var(--sl-color-primary-50);
      color: var(--sl-color-neutral-800);
      overflow: hidden;
      white-space: nowrap;
    }

    .episode-item:hover {
      background-color: var(--sl-color-primary-200);
    }

    .episode-show {
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .episode-ep,
    .episode-movie {
      flex-shrink: 0;
      color: var(--sl-color-neutral-500);
    }

    .episode-movie {
      color: var(--sl-color-primary-600);
      font-weight: var(--sl-font-weight-semibold);
    }

    .more {
      font-size: var(--sl-font-size-x-small);
      color: var(--sl-color-neutral-500);
      padding-left: 2px;
    }

    @media screen and (min-width: 640px) {
      .day {
        min-height: 7rem;
      }
    }
  `;
}
