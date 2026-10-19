import { css, html, LitElement } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

import '@shoelace-style/shoelace/dist/components/button/button.js';
import '@shoelace-style/shoelace/dist/components/dialog/dialog.js';
import '@shoelace-style/shoelace/dist/components/range/range.js';

/**
 * TMDB-style rating dialog + generic confirm dialog.
 * Usage:
 *   const score = await askForRating('Dune: Part Two');
 *   // score: number (0.5-10) | 'skip' (mark watched without rating) | 'cancel'
 *   const ok = await confirmAction('This will delete the TMDB rating.');
 */

export type RatingAskResult = number | 'skip' | 'cancel';

// ---------------------------------------------------------------------------
// Rating dialog (TMDB look: orange-red progress ring + 0-10 slider)
// ---------------------------------------------------------------------------

@customElement('rating-ask-dialog')
export class RatingAskDialog extends LitElement {
  @property({ type: String }) title = '';
  @state() score = 10;

  static styles = css`
    .rating-wrap {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 18px;
      padding: 6px 4px 2px;
    }
    .ring {
      width: 156px;
      height: 156px;
    }
    .hint {
      color: var(--sl-color-neutral-500);
      font-size: 14px;
      text-align: center;
      max-width: 260px;
      line-height: 1.4;
    }
    .actions {
      display: flex;
      gap: 10px;
      justify-content: flex-end;
      width: 100%;
    }
    sl-range {
      width: 240px;
      --sl-color-primary-600: #ff6a00;
      --sl-color-primary-500: #ff6a00;
    }
  `;

  render() {
    const circumference = 2 * Math.PI * 52;
    const ratio = Math.max(0, Math.min(10, this.score)) / 10;
    return html`
      <sl-dialog
        label="Rate it on TMDB"
        @sl-request-close=${() => this.resolve('cancel')}
      >
        <div class="rating-wrap">
          <svg class="ring" viewBox="0 0 120 120">
            <defs>
              <linearGradient id="ratingGrad" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0%" stop-color="#ff6a00" />
                <stop offset="100%" stop-color="#ff2d55" />
              </linearGradient>
            </defs>
            <circle cx="60" cy="60" r="52" fill="none" stroke="#2b3448" stroke-width="9" />
            <circle
              cx="60"
              cy="60"
              r="52"
              fill="none"
              stroke="url(#ratingGrad)"
              stroke-width="9"
              stroke-linecap="round"
              stroke-dasharray="${circumference}"
              stroke-dashoffset="${circumference * (1 - ratio)}"
              transform="rotate(-90 60 60)"
            />
            <text x="60" y="66" text-anchor="middle" fill="#ffffff" font-size="26" font-weight="700">
              ${this.score.toFixed(1)}
            </text>
            <text x="60" y="84" text-anchor="middle" fill="#8b93a7" font-size="11">/10</text>
          </svg>

          <sl-range
            min="0"
            max="10"
            step="0.5"
            .value=${this.score}
            @sl-input=${(event: CustomEvent) => {
              this.score = Number((event.currentTarget as HTMLInputElement & { value?: number }).value ?? 10);
            }}
          ></sl-range>

          <div class="hint">${this.title}</div>

          <div class="actions">
            <sl-button variant="neutral" @click=${() => this.resolve('skip')}>Skip</sl-button>
            <sl-button variant="primary" @click=${() => this.resolve(this.score)}>Rate</sl-button>
          </div>
        </div>
      </sl-dialog>
    `;
  }

  private resolver?: (value: RatingAskResult) => void;

  private resolve(value: RatingAskResult) {
    this.resolver?.(value);
    this.remove();
  }

  private dialogEl?: Element & { show: () => void };

  async open(): Promise<RatingAskResult> {
    return new Promise((resolvePromise) => {
      this.resolver = resolvePromise;
      document.body.appendChild(this);
      void this.updateComplete.then(() => {
        this.dialogEl = this.shadowRoot?.querySelector('sl-dialog') as Element & { show: () => void };
        this.dialogEl?.show();
      });
    });
  }
}

export function askForRating(title: string): Promise<RatingAskResult> {
  const dialog = new RatingAskDialog();
  dialog.title = title;
  return dialog.open();
}

// ---------------------------------------------------------------------------
// Generic confirm dialog (used before deleting a TMDB rating)
// ---------------------------------------------------------------------------

@customElement('confirm-action-dialog')
export class ConfirmActionDialog extends LitElement {
  @property({ type: String }) title = 'Confirm';
  @property({ type: String }) message = '';
  @property({ type: String }) confirmLabel = 'Confirm';

  static styles = css`
    .message {
      color: var(--sl-color-neutral-200);
      font-size: 14px;
      line-height: 1.5;
      margin-bottom: 4px;
    }
    .actions {
      display: flex;
      gap: 10px;
      justify-content: flex-end;
      width: 100%;
      margin-top: 8px;
    }
  `;

  render() {
    return html`
      <sl-dialog label=${this.title} @sl-request-close=${() => this.resolve(false)}>
        <div class="message">${this.message}</div>
        <div class="actions">
          <sl-button variant="neutral" @click=${() => this.resolve(false)}>Cancel</sl-button>
          <sl-button variant="danger" @click=${() => this.resolve(true)}>${this.confirmLabel}</sl-button>
        </div>
      </sl-dialog>
    `;
  }

  private resolver?: (value: boolean) => void;

  private resolve(value: boolean) {
    this.resolver?.(value);
    this.remove();
  }

  private dialogEl?: Element & { show: () => void };

  async open(): Promise<boolean> {
    return new Promise((resolvePromise) => {
      this.resolver = resolvePromise;
      document.body.appendChild(this);
      void this.updateComplete.then(() => {
        this.dialogEl = this.shadowRoot?.querySelector('sl-dialog') as Element & { show: () => void };
        this.dialogEl?.show();
      });
    });
  }
}

export function confirmAction(message: string, title = 'Confirm', confirmLabel = 'Confirm'): Promise<boolean> {
  const dialog = new ConfirmActionDialog();
  dialog.title = title;
  dialog.message = message;
  dialog.confirmLabel = confirmLabel;
  return dialog.open();
}
