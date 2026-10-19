import { Router } from '@vaadin/router';
import { css, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { ifDefined } from 'lit/directives/if-defined.js';

import { BaseElement } from '@/components/base-element';
import { sharedStyles } from '@/styles/shared-styles';
import type { RatingDto, ShowCardBadge } from '@/types';

import { getMyRatings } from './shows-api';

import '@/components/header-button';
import '@/layout/app-layout';

import './show-card';
import './shows-search-form';

@customElement('shows-page')
export class ShowsPage extends BaseElement {
  @state() loading = false;
  @state() ratings?: RatingDto[];
  @state() mediaType: 'all' | 'tv' | 'movie' = 'all';

  private ratingBadge: ShowCardBadge = { variant: 'primary', label: 'My Rating' };

  render() {
    return html`
      <app-layout icon="tv" headerTitle="Shows" selected="shows">
        <shows-search-form slot="header"></shows-search-form>
        <ul class="view-tabs" slot="header">
          <li>
            <header-button
              ?active=${this.mediaType === 'all'}
              @click=${() => {
                this.mediaType = 'all';
              }}
            >
              All
            </header-button>
          </li>
          <li>
            <header-button
              ?active=${this.mediaType === 'tv'}
              @click=${() => {
                this.mediaType = 'tv';
              }}
            >
              TV
            </header-button>
          </li>
          <li>
            <header-button
              ?active=${this.mediaType === 'movie'}
              @click=${() => {
                this.mediaType = 'movie';
              }}
            >
              Movies
            </header-button>
          </li>
        </ul>
        <section class="content">
          ${
            this.ratings && this.ratings.length > 0
              ? this.ratings.map(
                  rating => html`
                  <a href="#" class="card-link" @click=${(event: Event) => this.handleSelectShow(event, rating)}>
                    <show-card
                      image=${ifDefined(rating.posterUrl)}
                      name=${ifDefined(rating.name)}
                      network=${ifDefined(this.getSubtitle(rating))}
                      .badge=${this.getBadge(rating)}
                    ></show-card>
                  </a>
                `
                )
              : this.renderEmpty()
          }
        </section>
      </app-layout>
    `;
  }

  private getSubtitle(rating: RatingDto): string | undefined {
    if (rating.mediaType === 'movie') return 'Movie';
    return rating.date || undefined;
  }

  private getBadge(rating: RatingDto): ShowCardBadge {
    return { ...this.ratingBadge, label: `★ ${rating.rating.toFixed(1)}` };
  }

  private renderEmpty() {
    if (this.loading) return null;
    return html`<div>No ratings found. Connect your TMDB account in Settings, then rate movies and shows on TMDB.</div>`;
  }

  private handleSelectShow(event: Event, rating: RatingDto) {
    event.preventDefault();
    if (rating.showId) {
      Router.go(`/show/view/${rating.showId}`);
      return;
    }
    this.toast({ variant: 'neutral', message: `${rating.name} is not in your Mishos library yet` });
  }

  async connectedCallback() {
    super.connectedCallback();
    await this.loadRatings();
  }

  private async loadRatings() {
    this.loading = true;
    this.ratings = await this.callApi(() => getMyRatings(this.mediaType));
    this.loading = false;
  }

  updated(changedProperties: Map<string, unknown>) {
    super.updated?.(changedProperties);
    if (changedProperties.has('mediaType')) {
      void this.loadRatings();
    }
  }

  static styles = [
    sharedStyles,
    css`
      .content {
        display: flex;
        align-items: stretch;
        flex-wrap: wrap;
        gap: var(--sl-spacing-medium);
        margin: var(--sl-spacing-large);
      }

      .view-tabs {
        display: flex;
        gap: var(--sl-spacing-2x-small);
        list-style: none;
        margin: 0;
        padding: 0;
      }
    `,
  ];
}
