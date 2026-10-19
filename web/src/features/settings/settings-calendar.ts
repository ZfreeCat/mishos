import { css, html } from 'lit';
import { customElement, state } from 'lit/decorators.js';

import { BaseElement } from '@/components/base-element';
import { sharedStyles } from '@/styles/shared-styles';
import { createToastEvent } from '@/utils';

import { getCalendarToken } from './calendar-api';

import '@shoelace-style/shoelace/dist/components/button/button.js';
import '@shoelace-style/shoelace/dist/components/input/input.js';

@customElement('settings-calendar')
export class SettingsCalendar extends BaseElement {
  @state() token?: string;
  @state() copied = false;

  render() {
    return html`
      <section class="calendar-section">
        <h3>Apple Calendar Sync</h3>
        <p class="help">
          Subscribe to your Mishos watch calendar in Apple Calendar (iPhone, iPad, Mac):
          open <strong>Calendar &gt; Add Calendar &gt; Add Subscription Calendar</strong> (on iOS:
          Settings &gt; Calendar &gt; Accounts &gt; Add Account &gt; Other &gt; Add Subscribed Calendar) and
          paste the URL below. Future episodes and movie premieres update automatically.
        </p>

        ${
          this.token
            ? html`
              <sl-input
                class="url-input"
                readonly
                value=${this.subscriptionUrl}
                aria-label="Calendar subscription URL"
              ></sl-input>
              <div class="actions">
                <sl-button variant="primary" @click=${this.handleCopy}>${this.copied ? 'Copied!' : 'Copy URL'}</sl-button>
                <sl-button variant="neutral" @click=${this.handleOpenIos}>Open on iOS</sl-button>
              </div>
            `
            : html`<sl-button variant="primary" ?loading=${!this.tokenLoaded} @click=${this.loadToken}>Generate Subscription URL</sl-button>`
        }
      </section>
    `;
  }

  @state() tokenLoaded = false;

  private get subscriptionUrl(): string {
    const origin = window.location.origin;
    return `${origin}/api/calendar/${this.token}.ics`;
  }

  private get webcalUrl(): string {
    return this.subscriptionUrl.replace(/^https:/, 'webcal:');
  }

  private async loadToken() {
    const token = await this.callApi<string>(() => getCalendarToken());
    if (token) {
      this.token = token;
      this.tokenLoaded = true;
    }
  }

  private async handleCopy() {
    if (!this.token) return;

    try {
      await navigator.clipboard.writeText(this.subscriptionUrl);
      this.copied = true;
      setTimeout(() => {
        this.copied = false;
      }, 2000);
    } catch {
      this.dispatchEvent(
        createToastEvent({
          variant: 'danger',
          message: 'Could not copy. Please copy the URL manually.',
        })
      );
    }
  }

  private handleOpenIos() {
    if (!this.token) return;
    window.location.href = this.webcalUrl;
  }

  async connectedCallback() {
    super.connectedCallback();
    await this.loadToken();
  }

  static styles = [
    sharedStyles,
    css`
      .calendar-section {
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

      .url-input {
        max-width: 40rem;
      }

      .actions {
        display: flex;
        gap: var(--sl-spacing-medium);
        flex-wrap: wrap;
      }
    `,
  ];
}
