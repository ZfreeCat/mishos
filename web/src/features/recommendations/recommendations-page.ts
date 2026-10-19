import { Router } from '@vaadin/router';
import { css, html, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { when } from 'lit/directives/when.js';

import { BaseElement } from '@/components/base-element';
import { sharedStyles } from '@/styles/shared-styles';
import type { RecommendationDto } from '@/types';

import { followRecommendation, getRecommendations } from './recommendations-api';

import '@/components/header-button';
import '@/features/shows/shows-search-form';
import '@/layout/app-layout';

import '@shoelace-style/shoelace/dist/components/badge/badge.js';
import '@shoelace-style/shoelace/dist/components/button/button.js';

type RecommendationCategory = 'upcoming' | 'new' | 'trending';

@customElement('recommendations-page')
export class RecommendationsPage extends BaseElement {
  @state() category: RecommendationCategory = 'upcoming';
  @state() results?: RecommendationDto[];

  render() {
    return html`
      <app-layout icon="fire" headerTitle="Discover" selected="recommendations">
        <shows-search-form slot="header"></shows-search-form>
        <ul class="header-tabs" slot="header">
          <li>
            <header-button
              ?active=${this.category === 'upcoming'}
              @click=${() => this.changeCategory('upcoming')}
            >
              Upcoming
            </header-button>
          </li>
          <li>
            <header-button ?active=${this.category === 'new'} @click=${() => this.changeCategory('new')}>
              New
            </header-button>
          </li>
          <li>
            <header-button
              ?active=${this.category === 'trending'}
              @click=${() => this.changeCategory('trending')}
            >
              Trending
            </header-button>
          </li>
        </ul>
        <section class="content">
          ${
            this.results
              ? this.results.length > 0
                ? html`
                    <div class="grid">
                      ${this.results.map(item => this.renderCard(item))}
                    </div>
                  `
                : html`<div class="no-results">No recommendations found</div>`
              : null
          }
        </section>
      </app-layout>
    `;
  }

  private renderCard(item: RecommendationDto) {
    return html`
      <div class="card">
        <div class="card-image">
          ${
            item.posterUrl
              ? html`<img src=${item.posterUrl} alt=${item.name ?? 'Unknown'} loading="lazy" />`
              : html`<div class="no-image-placeholder">No Image</div>`
          }
          <sl-badge variant=${item.mediaType === 'movie' ? 'primary' : 'neutral'} class="media-badge">
            ${item.mediaType === 'movie' ? 'Movie' : 'TV'}
          </sl-badge>
        </div>
        <div class="card-body">
          <div
            class="card-title ${item.showId ? 'clickable' : ''}"
            @click=${() => this.goToShow(item)}
          >
            ${item.name}
          </div>
          <div class="card-meta">
            ${item.date ? html`<span>${item.date}</span>` : nothing}
            ${when(item.voteAverage > 0, () => html`<span class="rating">★ ${item.voteAverage.toFixed(1)}</span>`)}
          </div>
          ${when(
            item.overview,
            () => html`<p class="card-summary">${item.overview}</p>`,
            () => nothing
          )}
          <div class="card-actions">
            ${
              item.followed
                ? html`<sl-button size="small" variant="success" disabled>Following</sl-button>`
                : html`
                  <sl-button size="small" variant="primary" @click=${() => this.handleFollow(item)}>
                    Follow
                  </sl-button>
                `
            }
          </div>
        </div>
      </div>
    `;
  }

  private async changeCategory(category: RecommendationCategory) {
    this.category = category;
    this.results = undefined;
    this.results = await this.callApi(() => getRecommendations(category));
  }

  private async handleFollow(item: RecommendationDto) {
    const response = await this.callApi(() => followRecommendation(item.tmdbId, item.mediaType));
    if (response) {
      this.toast({ variant: 'success', message: `${item.name} has been added to your shows` });
      if (this.results) {
        this.results = this.results.map(result =>
          result.tmdbId === item.tmdbId ? { ...result, followed: true } : result
        );
      }
    }
  }

  private goToShow(item: RecommendationDto) {
    if (item.showId) {
      Router.go(`/show/view/${item.showId}`);
    }
  }

  async connectedCallback() {
    super.connectedCallback();
    this.results = await this.callApi(() => getRecommendations(this.category));
  }

  static styles = [
    sharedStyles,
    css`
      .content {
        margin: var(--sl-spacing-large);
      }

      .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
        gap: var(--sl-spacing-medium);
      }

      .card {
        display: flex;
        flex-direction: column;
        background: var(--sl-color-neutral-0);
        border: 1px solid var(--sl-color-neutral-200);
        border-radius: var(--sl-border-radius-medium);
        overflow: hidden;
      }

      .card-image {
        position: relative;
        aspect-ratio: 2 / 3;
        background: var(--sl-color-neutral-100);
      }

      .card-image img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        display: block;
      }

      .media-badge {
        position: absolute;
        top: var(--sl-spacing-2x-small);
        right: var(--sl-spacing-2x-small);
      }

      .no-image-placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--sl-color-neutral-500);
        font-size: var(--sl-font-size-small);
      }

      .card-body {
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-2x-small);
        padding: var(--sl-spacing-small);
        flex-grow: 1;
      }

      .card-title {
        font-size: var(--sl-font-size-medium);
        font-weight: var(--sl-font-weight-semibold);
        line-height: 1.2;
      }

      .card-title.clickable {
        cursor: pointer;
      }

      .card-title.clickable:hover {
        text-decoration: underline;
      }

      .card-meta {
        display: flex;
        align-items: center;
        gap: var(--sl-spacing-x-small);
        font-size: var(--sl-font-size-x-small);
        color: var(--sl-color-neutral-600);
      }

      .rating {
        color: var(--sl-color-warning-600);
        font-weight: var(--sl-font-weight-semibold);
      }

      .card-summary {
        margin: 0;
        font-size: var(--sl-font-size-x-small);
        color: var(--sl-color-neutral-700);
        display: -webkit-box;
        -webkit-line-clamp: 3;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }

      .card-actions {
        margin-top: auto;
        padding-top: var(--sl-spacing-2x-small);
      }

      .no-results {
        color: var(--sl-color-neutral-600);
        font-size: var(--sl-font-size-small);
      }
    `,
  ];
}
