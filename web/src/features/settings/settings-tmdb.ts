import { css, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { when } from 'lit/directives/when.js';

import { BaseElement } from '@/components/base-element';
import { sharedStyles } from '@/styles/shared-styles';
import { formatDate } from '@/utils';

import { completeTmdbAuth, createTmdbAuthToken, disconnectTmdb, getTmdbStatus, syncTmdbWatchlist } from './tmdb-api';

import '@shoelace-style/shoelace/dist/components/button/button.js';

interface TmdbStatus {
  connected: boolean;
  username?: string;
  lastSyncAt?: string;
}

@customElement('settings-tmdb')
export class SettingsTmdb extends BaseElement {
  @state() status?: TmdbStatus;
  @state() requestToken?: string;
  @state() authUrl?: string;
  @state() syncing = false;

  render() {
    return html`
      <section class="tmdb-section">
        <h3>TMDB</h3>
        <p class="help">
          Connect your TMDB account to sync your watchlist into Mishos. New movies and shows on your TMDB
          watchlist are added as followed shows automatically.
        </p>

        ${when(
          this.status?.connected,
          () => html`
            <ul class="tmdb-info">
              <li>Connected as <strong>${this.status?.username}</strong></li>
              ${
                this.status?.lastSyncAt
                  ? html`<li>Last synced: ${formatDate(new Date(this.status.lastSyncAt ?? ''))}</li>`
                  : html`<li>Not synced yet</li>`
              }
            </ul>
            <div class="actions">
              <sl-button variant="primary" ?loading=${this.syncing} @click=${this.handleSync}>Sync Watchlist</sl-button>
              <sl-button variant="danger" @click=${this.handleDisconnect}>Disconnect</sl-button>
            </div>
          `,
          () =>
            when(
              this.requestToken && this.authUrl,
              () => html`
                <p class="help">
                  A TMDB authorization window was opened. Approve the request in that window, then click
                  <strong>Complete Authorization</strong> below.
                </p>
                <div class="actions">
                  <sl-button variant="primary" @click=${this.handleCompleteAuth}>Complete Authorization</sl-button>
                  <sl-button variant="neutral" @click=${this.resetAuth}>Cancel</sl-button>
                </div>
              `,
              () => html`
                <div class="actions">
                  <sl-button variant="primary" @click=${this.handleConnect}>Connect TMDB</sl-button>
                </div>
              `
            )
        )}
      </section>
    `;
  }

  private async handleConnect() {
    const response = await this.callApi<{ requestToken: string; authUrl: string }>(() => createTmdbAuthToken());
    if (response) {
      this.requestToken = response.requestToken;
      this.authUrl = response.authUrl;
      window.open(response.authUrl, '_blank');
    }
  }

  private async handleCompleteAuth() {
    if (!this.requestToken) return;

    const response = await this.callApi<{ message?: string }>(() => completeTmdbAuth(this.requestToken ?? ''));
    if (response) {
      this.toast({ variant: 'success', message: 'TMDB account connected' });
      this.resetAuth();
      await this.loadStatus();
    }
  }

  private async handleSync() {
    this.syncing = true;
    const response = await this.callApi<{ added?: number; followed?: number }>(() => syncTmdbWatchlist());
    this.syncing = false;

    if (response) {
      this.toast({
        variant: 'success',
        message: `Watchlist synced: ${response.added ?? 0} added, ${response.followed ?? 0} followed`,
      });
      await this.loadStatus();
    }
  }

  private async handleDisconnect() {
    await this.callApi(() => disconnectTmdb());
    this.toast({ variant: 'success', message: 'TMDB account disconnected' });
    this.status = { connected: false };
  }

  private resetAuth() {
    this.requestToken = undefined;
    this.authUrl = undefined;
  }

  private async loadStatus() {
    const status = await this.callApi<TmdbStatus>(() => getTmdbStatus());
    if (status) {
      this.status = {
        connected: status.connected === true,
        username: status.username,
        lastSyncAt: status.lastSyncAt ?? undefined,
      };
    }
  }

  async connectedCallback() {
    super.connectedCallback();
    await this.loadStatus();
  }

  static styles = [
    sharedStyles,
    css`
      .tmdb-section {
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-medium);
      }

      h3 {
        margin: 0;
        font-size: var(--sl-font-size-medium);
      }

      .help {
        margin: 0;
        color: var(--sl-color-neutral-600);
        font-size: var(--sl-font-size-small);
        max-width: 40rem;
      }

      .tmdb-info {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: var(--sl-spacing-2x-small);
        font-size: var(--sl-font-size-small);
        color: var(--sl-color-neutral-700);
      }

      .actions {
        display: flex;
        gap: var(--sl-spacing-medium);
        flex-wrap: wrap;
      }
    `,
  ];
}
