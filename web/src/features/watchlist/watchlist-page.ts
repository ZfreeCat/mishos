import { LitElement, css, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { choose } from 'lit/directives/choose.js';

import { sharedStyles } from '@/styles/shared-styles';
import type { EpisodeDto } from '@/types';

import '@/components/header-button';
import '@/features/shows/shows-search-form';
import '@/layout/app-layout';

import '@shoelace-style/shoelace/dist/components/button/button.js';
import '@shoelace-style/shoelace/dist/components/icon/icon.js';

import './watchlist-unwatched';
import './watchlist-recent';

@customElement('watchlist-page')
export class WatchListPage extends LitElement {
  @state() episodes?: EpisodeDto[];
  @state() section = 'unwatched';
  @state() mediaType: 'all' | 'tv' | 'movie' = 'all';

  render() {
    return html`
      <app-layout icon="check-badge" headerTitle="Watch List" selected="watchlist">
        <shows-search-form slot="header"></shows-search-form>
        <ul class="header-tabs" slot="header">
          <li>
            <header-button
              ?active=${this.section === 'unwatched'}
              @click=${() => {
                this.section = 'unwatched';
              }}
            >
              Unwatched
            </header-button>
          </li>
          <li>
            <header-button
              ?active=${this.section === 'recent'}
              @click=${() => {
                this.section = 'recent';
              }}
            >
              Recently Watched
            </header-button>
          </li>
        </ul>
        <section class="content">
          <ul class="type-tabs">
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
          ${choose(this.section, [
            ['unwatched', () => html`<watchlist-unwatched .mediaType=${this.mediaType}></watchlist-unwatched>`],
            ['recent', () => html`<watchlist-recent .mediaType=${this.mediaType}></watchlist-recent>`],
          ])}
        </section>
      </app-layout>
    `;
  }

  static styles = [
    sharedStyles,
    css`
      .content {
        margin: var(--sl-spacing-large);
      }

      .type-tabs {
        display: flex;
        gap: var(--sl-spacing-2x-small);
        list-style: none;
        margin: 0 0 var(--sl-spacing-medium);
        padding: 0;
      }
    `,
  ];
}
