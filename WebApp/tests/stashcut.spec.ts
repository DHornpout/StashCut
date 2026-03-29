/**
 * Stashcut end-to-end tests
 *
 * Strategy
 * --------
 * The app runs as a local file:// page and relies on the File System Access API
 * (showSaveFilePicker / showOpenFilePicker) — native browser dialogs that
 * Playwright cannot drive.
 *
 * We remove those globals via addInitScript() before page load, which forces
 * the app onto its Safari-style fallback path:
 *   • "Create new library" → builds an empty in-memory store (no dialog)
 *   • "Open existing file" → triggers a hidden <input type="file"> that
 *     Playwright CAN intercept with page.waitForEvent('filechooser')
 *
 * All tests that need pre-populated data load tests/fixtures/shortcuts.json,
 * which contains two apps (Chrome + VS Code) matching the v1 spec example.
 */

import { test, expect, Page } from '@playwright/test';
import path from 'path';

const FILE_URL     = `file://${path.resolve(__dirname, '../stashcut.html')}`;
const FIXTURE_PATH = path.resolve(__dirname, 'fixtures/shortcuts.json');

// ─── HELPERS ──────────────────────────────────────────────────────────────────

/** Strip the File System Access API so the app uses its <input> fallback. */
async function disableFileAPI(page: Page) {
  await page.addInitScript(() => {
    delete (window as any).showSaveFilePicker;
    delete (window as any).showOpenFilePicker;
  });
}

/** Navigate to the app and wait for React to render something into #root. */
async function gotoApp(page: Page) {
  // Serve React/ReactDOM from local fixtures instead of the CDN.
  // This eliminates network flakiness across all browsers and test workers.
  await page.route(
    'https://cdnjs.cloudflare.com/ajax/libs/react/18.2.0/umd/react.production.min.js',
    route => route.fulfill({ path: path.resolve(__dirname, 'fixtures/react.production.min.js') })
  );
  await page.route(
    'https://cdnjs.cloudflare.com/ajax/libs/react-dom/18.2.0/umd/react-dom.production.min.js',
    route => route.fulfill({ path: path.resolve(__dirname, 'fixtures/react-dom.production.min.js') })
  );

  await page.goto(FILE_URL);
  await page.waitForSelector('#root > *', { timeout: 15_000 });
}

/**
 * Load the app and open the test fixture via the file-input fallback.
 * The fixture has two apps (Chrome, VS Code) with shortcuts pre-loaded.
 * Chrome is auto-selected by the app after load.
 */
async function openWithFixture(page: Page) {
  await disableFileAPI(page);
  await gotoApp(page);
  await page.waitForSelector('.welcome');

  const [fileChooser] = await Promise.all([
    page.waitForEvent('filechooser'),
    page.locator('.btn-secondary').click(),   // "Open existing file"
  ]);
  await fileChooser.setFiles(FIXTURE_PATH);
  await page.waitForSelector('.sidebar', { timeout: 8_000 });
}

/**
 * Load the app and create an empty in-memory library (no dialog, no fixture).
 * Use this for tests that start from a blank slate (Add App, etc.).
 */
async function createEmptyLibrary(page: Page) {
  await disableFileAPI(page);
  await gotoApp(page);
  await page.waitForSelector('.welcome');
  await page.locator('.btn-primary').click();  // "Create new library"
  await page.waitForSelector('.sidebar', { timeout: 8_000 });
}

// ─── WELCOME SCREEN ───────────────────────────────────────────────────────────

test.describe('Welcome screen', () => {
  test.beforeEach(async ({ page }) => {
    await disableFileAPI(page);
    await gotoApp(page);
    await page.waitForSelector('.welcome');
  });

  test('shows the Stashcut logo', async ({ page }) => {
    await expect(page.locator('.welcome-logo')).toContainText('Stashcut');
  });

  test('shows the tagline', async ({ page }) => {
    await expect(page.locator('.welcome-tag')).toHaveText(
      'Your personal keyboard shortcut reference'
    );
  });

  test('"Create new library" button is visible and enabled', async ({ page }) => {
    await expect(page.locator('.btn-primary')).toBeVisible();
    await expect(page.locator('.btn-primary')).toBeEnabled();
    await expect(page.locator('.btn-primary')).toContainText('Create new library');
  });

  test('"Open existing file" button is visible and enabled', async ({ page }) => {
    await expect(page.locator('.btn-secondary')).toBeVisible();
    await expect(page.locator('.btn-secondary')).toBeEnabled();
    await expect(page.locator('.btn-secondary')).toContainText('Open existing file');
  });

  test('"Create new library" transitions to the main layout', async ({ page }) => {
    await page.locator('.btn-primary').click();
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('.welcome')).toHaveCount(0);
  });
});

// ─── MAIN LAYOUT ──────────────────────────────────────────────────────────────

test.describe('Main layout (fixture loaded)', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
  });

  test('sidebar is visible', async ({ page }) => {
    await expect(page.locator('.sidebar')).toBeVisible();
  });

  test('sidebar shows Stashcut logo', async ({ page }) => {
    await expect(page.locator('.sidebar .logo')).toContainText('Stashcut');
  });

  test('sidebar lists both apps from the fixture', async ({ page }) => {
    await expect(page.locator('.app-item')).toHaveCount(2);
    await expect(page.locator('.app-item .app-name').nth(0)).toHaveText('Chrome');
    await expect(page.locator('.app-item .app-name').nth(1)).toHaveText('VS Code');
  });

  test('Chrome is auto-selected and shows its shortcut count', async ({ page }) => {
    await expect(page.locator('.app-item.active .app-name')).toHaveText('Chrome');
    await expect(page.locator('.app-item.active .app-count')).toHaveText('2');
  });

  test('panel title shows the selected app name and shortcut count', async ({ page }) => {
    await expect(page.locator('.panel-title h2')).toHaveText('Chrome');
    await expect(page.locator('.panel-title .count')).toContainText('2 shortcuts');
  });

  test('clicking a different app in the sidebar selects it', async ({ page }) => {
    await page.locator('.app-item').filter({ hasText: 'VS Code' }).click();
    await expect(page.locator('.app-item.active .app-name')).toHaveText('VS Code');
    await expect(page.locator('.panel-title h2')).toHaveText('VS Code');
  });

  test('toolbar contains the main search bar and OS filter toggle', async ({ page }) => {
    await expect(page.locator('.search-main')).toBeVisible();
    await expect(page.locator('.os-toggle')).toBeVisible();
  });

  test('empty state is shown when no app is selected', async ({ page }) => {
    // Deselect by refreshing to a blank library with no apps selected
    await createEmptyLibrary(page);
    await expect(page.locator('.empty-state')).toBeVisible();
    await expect(page.locator('.empty-state')).toContainText('Select an app');
  });
});

// ─── SHORTCUT LIST ────────────────────────────────────────────────────────────

test.describe('Shortcut list (fixture loaded)', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
    // Chrome is pre-selected
  });

  test('shows all Chrome shortcuts', async ({ page }) => {
    await expect(page.locator('.shortcut-row')).toHaveCount(2);
  });

  test('favorite shortcut appears in the Favorites section', async ({ page }) => {
    await expect(page.locator('.section-label').first()).toContainText('Favorites');
    await expect(page.locator('.shortcut-row.is-favorite .shortcut-desc'))
      .toHaveText('Reopen last closed tab');
  });

  test('non-favorite shortcut appears below favorites', async ({ page }) => {
    await expect(
      page.locator('.shortcut-row:not(.is-favorite) .shortcut-desc')
    ).toHaveText('Focus the address bar');
  });

  test('shortcut rows show key badges in All mode (default)', async ({ page }) => {
    const firstRow = page.locator('.shortcut-row').first();
    await expect(firstRow.locator('.key-badge')).toHaveCount(2);  // mac + win
    await expect(firstRow.locator('.key-os-label.mac')).toBeVisible();
    await expect(firstRow.locator('.key-os-label.win')).toBeVisible();
  });

  test('"both" shortcuts on VS Code show a single badge for all OS modes', async ({ page }) => {
    await page.locator('.app-item').filter({ hasText: 'VS Code' }).click();
    const firstRow = page.locator('.shortcut-row').first();
    await expect(firstRow.locator('.key-badge.both-badge')).toBeVisible();
  });
});

// ─── OS FILTER ────────────────────────────────────────────────────────────────

test.describe('OS filter', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
    // Chrome is pre-selected; Chrome shortcuts have separate mac / win keys
  });

  test('"All" is active by default', async ({ page }) => {
    await expect(page.locator('.os-btn.active')).toHaveText('All');
  });

  test('Mac filter shows only mac-style key badges', async ({ page }) => {
    await page.locator('.os-btn.mac').click();
    await expect(page.locator('.os-btn.mac')).toHaveClass(/active/);
    await expect(page.locator('.key-badge.mac-badge').first()).toBeVisible();
    await expect(page.locator('.key-badge.win-badge')).toHaveCount(0);
  });

  test('Win filter shows only win-style key badges', async ({ page }) => {
    await page.locator('.os-btn.win').click();
    await expect(page.locator('.os-btn.win')).toHaveClass(/active/);
    await expect(page.locator('.key-badge.win-badge').first()).toBeVisible();
    await expect(page.locator('.key-badge.mac-badge')).toHaveCount(0);
  });

  test('All filter restores both mac and win badges', async ({ page }) => {
    await page.locator('.os-btn.mac').click();
    await page.locator('.os-btn').filter({ hasText: 'All' }).click();
    await expect(page.locator('.key-badge.mac-badge').first()).toBeVisible();
    await expect(page.locator('.key-badge.win-badge').first()).toBeVisible();
  });

  test('"both" shortcut is visible under every OS filter', async ({ page }) => {
    await page.locator('.app-item').filter({ hasText: 'VS Code' }).click();
    for (const filter of ['.os-btn.mac', '.os-btn.win']) {
      await page.locator(filter).click();
      await expect(page.locator('.key-badge').first()).toBeVisible();
    }
  });
});

// ─── ADD APP ──────────────────────────────────────────────────────────────────

test.describe('Add app', () => {
  test.beforeEach(async ({ page }) => {
    await createEmptyLibrary(page);
  });

  test('"Add application" button opens the modal', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await expect(page.locator('.modal-title')).toHaveText('Add application');
    await expect(page.locator('.modal')).toBeVisible();
  });

  test('Cancel closes the modal without adding an app', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await page.locator('.modal-footer .btn-sm-ghost').click();
    await expect(page.locator('.modal')).toHaveCount(0);
    await expect(page.locator('.app-item')).toHaveCount(0);
  });

  test('"Add app" button is disabled when the name field is empty', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await expect(page.locator('.modal-footer .btn-sm')).toBeDisabled();
  });

  test('adding an app with a built-in icon slug name (Slack)', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await page.locator('.modal .form-input').fill('Slack');
    // Icon picker should auto-highlight the slack icon
    await expect(page.locator('.icon-option.selected')).toBeVisible();
    await page.locator('.modal-footer .btn-sm').click();

    // Modal closes
    await expect(page.locator('.modal')).toHaveCount(0);
    // App appears in sidebar and is selected
    await expect(page.locator('.app-item .app-name')).toHaveText('Slack');
    await expect(page.locator('.app-item.active .app-name')).toHaveText('Slack');
  });

  test('adding an app with a custom name (no built-in icon)', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await page.locator('.modal .form-input').fill('MyCustomApp');
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.app-item .app-name')).toHaveText('MyCustomApp');
  });

  test('new app shows a toast confirmation', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await page.locator('.modal .form-input').fill('Figma');
    await page.locator('.modal-footer .btn-sm').click();
    await expect(page.locator('.toast').filter({ hasText: '"Figma" added' })).toBeVisible();
  });

  test('new app starts with zero shortcuts and shows an empty state', async ({ page }) => {
    await page.locator('.add-app-btn').click();
    await page.locator('.modal .form-input').fill('TestApp');
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.empty-state')).toContainText('No shortcuts yet');
    await expect(page.locator('.panel-title .count')).toContainText('0 shortcuts');
  });

  test('can add multiple apps and they appear in the sidebar', async ({ page }) => {
    for (const name of ['App One', 'App Two', 'App Three']) {
      await page.locator('.add-app-btn').click();
      await page.locator('.modal .form-input').fill(name);
      await page.locator('.modal-footer .btn-sm').click();
    }
    await expect(page.locator('.app-item')).toHaveCount(3);
  });
});

// ─── ADD SHORTCUT ─────────────────────────────────────────────────────────────

test.describe('Add shortcut', () => {
  test.beforeEach(async ({ page }) => {
    // Start from a blank library, add one app, then run each test
    await createEmptyLibrary(page);
    await page.locator('.add-app-btn').click();
    await page.locator('.modal .form-input').fill('Chrome');
    await page.locator('.modal-footer .btn-sm').click();
    await page.waitForSelector('.empty-state');
  });

  test('"Add shortcut" button in the empty state opens the form', async ({ page }) => {
    await page.locator('.empty-state .btn-sm').click();
    await expect(page.locator('.modal-title')).toHaveText('Add shortcut');
  });

  test('"Add shortcut" button in the panel title opens the form', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();
    await expect(page.locator('.modal-title')).toHaveText('Add shortcut');
  });

  test('"Add shortcut" button is disabled when description is empty', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();
    await expect(page.locator('.modal-footer .btn-sm')).toBeDisabled();
  });

  test('Cancel closes the shortcut form without saving', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();
    await page.locator('.modal-footer .btn-sm-ghost').click();
    await expect(page.locator('.modal')).toHaveCount(0);
    await expect(page.locator('.shortcut-row')).toHaveCount(0);
  });

  test('adding a shortcut with separate Mac and Win key combos', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();

    await page.locator('.modal .form-input').fill('Reopen last closed tab');
    await page.locator('.key-input').nth(0).fill('Cmd + Shift + T');   // Mac
    await page.locator('.key-input').nth(1).fill('Ctrl + Shift + T');  // Win
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.modal')).toHaveCount(0);
    await expect(page.locator('.shortcut-row')).toHaveCount(1);
    await expect(page.locator('.shortcut-desc')).toHaveText('Reopen last closed tab');
    await expect(page.locator('.key-badge.mac-badge')).toBeVisible();
    await expect(page.locator('.key-badge.win-badge')).toBeVisible();
  });

  test('adding a shortcut with "same on both platforms" checked', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();

    await page.locator('.modal .form-input').fill('Quick open file');
    await page.locator('.key-input').nth(0).fill('Ctrl + P');
    await page.locator('.same-both input[type="checkbox"]').check();
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.shortcut-row')).toHaveCount(1);
    // "both" badge shown — no separate mac/win labels
    await expect(page.locator('.key-badge.both-badge')).toBeVisible();
    await expect(page.locator('.key-os-label')).toHaveCount(0);
  });

  test('"same on both platforms" hides the Win field and keeps the Mac value', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();

    // Both Mac and Win fields are present before checking the box
    await expect(page.locator('.key-input')).toHaveCount(2);
    await page.locator('.key-input').nth(0).fill('Ctrl + P');

    await page.locator('.same-both input[type="checkbox"]').check();

    // Win field is removed from DOM when sameBoth is active (!sameBoth && ...)
    await expect(page.locator('.key-input')).toHaveCount(1);
    // Mac field still holds the typed value (will be saved as keys_by_os.both)
    await expect(page.locator('.key-input').first()).toHaveValue('Ctrl + P');
  });

  test('adding a shortcut shows a success toast', async ({ page }) => {
    await page.locator('.panel-title .btn-sm').click();
    await page.locator('.modal .form-input').fill('New Tab');
    await page.locator('.modal-footer .btn-sm').click();
    await expect(page.locator('.toast').filter({ hasText: 'Shortcut added' })).toBeVisible();
  });

  test('shortcut count in the panel title increments after adding', async ({ page }) => {
    await expect(page.locator('.panel-title .count')).toContainText('0 shortcuts');

    await page.locator('.panel-title .btn-sm').click();
    await page.locator('.modal .form-input').fill('First shortcut');
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.panel-title .count')).toContainText('1 shortcut');
  });

  test('"Add shortcut" inline row at the bottom of the list opens the form', async ({ page }) => {
    // Add one shortcut first so the list renders
    await page.locator('.panel-title .btn-sm').click();
    await page.locator('.modal .form-input').fill('First');
    await page.locator('.modal-footer .btn-sm').click();

    await page.locator('.add-shortcut-row').click();
    await expect(page.locator('.modal-title')).toHaveText('Add shortcut');
  });
});

// ─── EDIT SHORTCUT ────────────────────────────────────────────────────────────

test.describe('Edit shortcut', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
    // Chrome is selected; "Reopen last closed tab" (favorite) and
    // "Focus the address bar" are the two Chrome shortcuts
  });

  test('clicking the edit button opens the form pre-filled', async ({ page }) => {
    const editBtn = page.locator('.shortcut-row').first().locator('.row-action-btn').first();
    await editBtn.click();
    await expect(page.locator('.modal-title')).toHaveText('Edit shortcut');
    await expect(page.locator('.modal .form-input')).toHaveValue('Reopen last closed tab');
    await expect(page.locator('.key-input').nth(0)).toHaveValue('Cmd + Shift + T');
    await expect(page.locator('.key-input').nth(1)).toHaveValue('Ctrl + Shift + T');
  });

  test('updating the description saves correctly', async ({ page }) => {
    const editBtn = page.locator('.shortcut-row').first().locator('.row-action-btn').first();
    await editBtn.click();

    await page.locator('.modal .form-input').fill('Restore closed tab');
    await page.locator('.modal-footer .btn-sm').click();

    await expect(page.locator('.modal')).toHaveCount(0);
    await expect(page.locator('.shortcut-desc').first()).toHaveText('Restore closed tab');
  });

  test('updating a key combo saves correctly', async ({ page }) => {
    const editBtn = page.locator('.shortcut-row').first().locator('.row-action-btn').first();
    await editBtn.click();

    await page.locator('.key-input').nth(0).fill('Cmd + Z');
    await page.locator('.modal-footer .btn-sm').click();

    // formatKeyDisplay converts 'Cmd + Z' → '⌘Z' for the badge display
    await expect(page.locator('.key-badge.mac-badge').first()).toContainText('⌘Z');
  });

  test('edit shows a success toast', async ({ page }) => {
    const editBtn = page.locator('.shortcut-row').first().locator('.row-action-btn').first();
    await editBtn.click();
    await page.locator('.modal .form-input').fill('Updated desc');
    await page.locator('.modal-footer .btn-sm').click();
    await expect(page.locator('.toast').filter({ hasText: 'Shortcut updated' })).toBeVisible();
  });
});

// ─── FAVORITES ────────────────────────────────────────────────────────────────

test.describe('Favorites', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
    // Chrome selected: "Reopen last closed tab" is already a favorite
  });

  test('starring a non-favorite shortcut adds it to favorites', async ({ page }) => {
    const nonFavoriteRow = page.locator('.shortcut-row:not(.is-favorite)');
    await expect(nonFavoriteRow.locator('.shortcut-desc')).toHaveText('Focus the address bar');

    await nonFavoriteRow.locator('.star-btn').click();

    await expect(page.locator('.shortcut-row.is-favorite')).toHaveCount(2);
  });

  test('un-starring a favorite removes it from the favorites section', async ({ page }) => {
    await expect(page.locator('.shortcut-row.is-favorite')).toHaveCount(1);

    await page.locator('.shortcut-row.is-favorite .star-btn').click();

    await expect(page.locator('.shortcut-row.is-favorite')).toHaveCount(0);
    await expect(page.locator('.section-label').filter({ hasText: '★ Favorites' })).toHaveCount(0);
  });

  test('favorited shortcut shows a filled star (★)', async ({ page }) => {
    await expect(
      page.locator('.shortcut-row.is-favorite .star-btn')
    ).toContainText('★');
  });

  test('un-favorited shortcut shows an empty star (☆)', async ({ page }) => {
    await expect(
      page.locator('.shortcut-row:not(.is-favorite) .star-btn')
    ).toContainText('☆');
  });

  test('Favorites section label disappears when there are no favorites', async ({ page }) => {
    await page.locator('.shortcut-row.is-favorite .star-btn').click();
    await expect(page.locator('.section-label').filter({ hasText: '★ Favorites' })).toHaveCount(0);
  });
});

// ─── DELETE SHORTCUT ──────────────────────────────────────────────────────────

test.describe('Delete shortcut', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
  });

  test('deleting a shortcut removes it from the list', async ({ page }) => {
    await expect(page.locator('.shortcut-row')).toHaveCount(2);

    const deleteBtn = page.locator('.shortcut-row').last().locator('.row-action-btn.delete');
    await deleteBtn.click();

    await expect(page.locator('.shortcut-row')).toHaveCount(1);
  });

  test('deleting all shortcuts shows the empty state', async ({ page }) => {
    for (let i = 0; i < 2; i++) {
      await page.locator('.row-action-btn.delete').first().click();
    }
    await expect(page.locator('.empty-state')).toContainText('No shortcuts yet');
  });

  test('shortcut count in the panel title decrements after deleting', async ({ page }) => {
    await expect(page.locator('.panel-title .count')).toContainText('2 shortcuts');
    await page.locator('.row-action-btn.delete').first().click();
    await expect(page.locator('.panel-title .count')).toContainText('1 shortcut');
  });
});

// ─── DELETE APP ───────────────────────────────────────────────────────────────

test.describe('Delete app', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
  });

  test('deleting an app removes it from the sidebar', async ({ page }) => {
    await expect(page.locator('.app-item')).toHaveCount(2);

    // Accept the browser confirm() dialog
    page.once('dialog', dialog => dialog.accept());
    await page.locator('button[title="Delete app"]').click();

    await expect(page.locator('.app-item')).toHaveCount(1);
    await expect(page.locator('.app-item .app-name')).toHaveText('VS Code');
  });

  test('dismissing the confirmation dialog keeps the app', async ({ page }) => {
    page.once('dialog', dialog => dialog.dismiss());
    await page.locator('button[title="Delete app"]').click();

    await expect(page.locator('.app-item')).toHaveCount(2);
  });
});

// ─── GLOBAL SEARCH ────────────────────────────────────────────────────────────

test.describe('Global search', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
  });

  test('typing a query shows results grouped by app', async ({ page }) => {
    await page.locator('.search-main').fill('tab');
    await expect(page.locator('.search-group-header')).toHaveCount(1);
    await expect(page.locator('.search-group-header')).toContainText('Chrome');
    await expect(page.locator('.shortcut-row')).toHaveCount(1);
    await expect(page.locator('.shortcut-desc')).toContainText('Reopen last closed tab');
  });

  test('search matches on key combo text', async ({ page }) => {
    await page.locator('.search-main').fill('Ctrl + P');
    await expect(page.locator('.search-group-header')).toContainText('VS Code');
    await expect(page.locator('.shortcut-desc')).toContainText('Quick open file');
  });

  test('search matches across multiple apps', async ({ page }) => {
    await page.locator('.search-main').fill('Ctrl');
    // Both Chrome (win keys) and VS Code (both keys) have "Ctrl"
    await expect(page.locator('.search-group-header').first()).toBeVisible();
  });

  test('clearing the query returns to the normal app view', async ({ page }) => {
    await page.locator('.search-main').fill('tab');
    await page.locator('.search-main').fill('');
    // Normal panel title should be back
    await expect(page.locator('.panel-title h2')).toBeVisible();
  });

  test('a query with no matches shows the empty search state', async ({ page }) => {
    await page.locator('.search-main').fill('zzzzzz');
    await expect(page.locator('.empty-state')).toContainText('No results');
  });
});

// ─── SIDEBAR SEARCH ───────────────────────────────────────────────────────────

test.describe('Sidebar app filter', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
  });

  test('typing in the sidebar filter hides non-matching apps', async ({ page }) => {
    await page.locator('.sidebar .search-input').fill('Chrome');
    await expect(page.locator('.app-item')).toHaveCount(1);
    await expect(page.locator('.app-item .app-name')).toHaveText('Chrome');
  });

  test('clearing the sidebar filter restores all apps', async ({ page }) => {
    await page.locator('.sidebar .search-input').fill('Chrome');
    await page.locator('.sidebar .search-input').fill('');
    await expect(page.locator('.app-item')).toHaveCount(2);
  });

  test('a filter with no match shows "No apps yet" message', async ({ page }) => {
    await page.locator('.sidebar .search-input').fill('zzzzz');
    await expect(page.locator('.app-list')).toContainText('No apps yet');
  });
});

// ─── SETTINGS PANEL ───────────────────────────────────────────────────────────

test.describe('Settings panel', () => {
  test.beforeEach(async ({ page }) => {
    await openWithFixture(page);
    await page.locator('button[title="Settings"]').click();
    await expect(page.locator('.settings-modal')).toBeVisible();
  });

  test('settings modal is visible', async ({ page }) => {
    await expect(page.locator('.modal-title')).toContainText('Settings');
  });

  test('shows the correct app and shortcut counts', async ({ page }) => {
    // fixture has 2 apps and 4 shortcuts
    await expect(page.locator('.settings-modal')).toContainText('2 apps, 4 shortcuts');
  });

  test('Export button is present', async ({ page }) => {
    await expect(page.locator('.btn-mini.accent').filter({ hasText: 'Export' })).toBeVisible();
  });

  test('"Import & merge" button is present', async ({ page }) => {
    await expect(page.locator('.btn-mini').filter({ hasText: 'Import & merge' })).toBeVisible();
  });

  test('closing the modal via × hides it', async ({ page }) => {
    await page.locator('.settings-modal .modal-close').click();
    await expect(page.locator('.settings-modal')).toHaveCount(0);
  });

  test('closing the modal via overlay click hides it', async ({ page }) => {
    await page.locator('.overlay').click({ position: { x: 5, y: 5 } });
    await expect(page.locator('.settings-modal')).toHaveCount(0);
  });
});
