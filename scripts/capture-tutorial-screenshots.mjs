#!/usr/bin/env node
// Capture all tutorial screenshots referenced by app/web/src/tutorial/sections/*.md
// into app/web/public/tutorial/ using Playwright (Chromium).
//
// Usage:
//   pnpm dlx playwright install chromium   # one-time
//   node scripts/capture-tutorial-screenshots.mjs
//
// Assumes:
//   - opendray dev/prod server reachable at BASE_URL (default http://localhost:5173)
//   - admin/12345678 from config.toml (override via OPENDRAY_USER / OPENDRAY_PASS)

import { chromium } from 'playwright'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { mkdirSync, existsSync } from 'node:fs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const OUT_DIR = resolve(__dirname, '..', 'app', 'web', 'public', 'tutorial')
const BASE_URL = process.env.BASE_URL ?? 'http://localhost:5173'
const USER = process.env.OPENDRAY_USER ?? 'admin'
const PASS = process.env.OPENDRAY_PASS ?? '12345678'

const VIEWPORT = { width: 1440, height: 900 }
const DEVICE_SCALE_FACTOR = 2 // crisp retina output

mkdirSync(OUT_DIR, { recursive: true })

const log = (...args) => console.log('[tutorial-shots]', ...args)

async function login(page) {
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' })
  // If we're already authenticated the app redirects away from /login.
  if (!page.url().includes('/login')) return
  const username = page.locator('#username')
  const password = page.locator('#password')
  await username.waitFor({ state: 'visible', timeout: 10_000 })
  await username.fill(USER)
  await password.fill(PASS)
  await page.getByRole('button', { name: /sign in/i }).click()
  await page.waitForURL((url) => !url.toString().includes('/login'), { timeout: 15_000 })
}

async function shot(page, file, opts = {}) {
  const path = resolve(OUT_DIR, file)
  await page.screenshot({ path, ...opts })
  log('saved', file)
}

async function gotoAndSettle(page, path) {
  await page.goto(`${BASE_URL}${path}`, { waitUntil: 'networkidle' })
  // small settle for async data + animations
  await page.waitForTimeout(600)
}

async function captureSidebarOverview(page) {
  await gotoAndSettle(page, '/sessions')
  // Sidebar is the leftmost column. Width ~ 220-240px in production layout.
  // Grab the actual width from the DOM so we don't hard-code.
  const sidebar = page.locator('aside, nav').first()
  const box = await sidebar.boundingBox()
  const width = box?.width ?? 240
  await shot(page, 'sidebar-overview.png', {
    clip: { x: 0, y: 0, width: Math.ceil(width) + 8, height: VIEWPORT.height },
  })
}

async function captureSessionsLayout(page) {
  await gotoAndSettle(page, '/sessions')
  // Try to open the first live session so terminal + inspector are visible.
  const liveSession = page.locator('[data-state="live"], [data-session-status="live"]').first()
  if (await liveSession.count()) {
    await liveSession.click()
    await page.waitForTimeout(800)
  } else {
    // Fall back to the first row in the sessions list.
    const firstRow = page.locator('[role="button"]').filter({ hasText: /code|codex|claude|pda/i }).first()
    if (await firstRow.count()) {
      await firstRow.click()
      await page.waitForTimeout(800)
    }
  }
  await shot(page, 'sessions-layout.png', { fullPage: false })
}

async function captureSpawnDialog(page) {
  await gotoAndSettle(page, '/sessions')
  // Open the spawn dialog. The button text varies between "New session",
  // "Spawn session", or a + icon — try each.
  const triggers = [
    page.getByRole('button', { name: /spawn session/i }),
    page.getByRole('button', { name: /new session/i }),
    page.locator('button[aria-label*="new" i]'),
    page.locator('button[aria-label*="spawn" i]'),
  ]
  let opened = false
  for (const t of triggers) {
    if (await t.count()) {
      await t.first().click()
      opened = true
      break
    }
  }
  if (!opened) throw new Error('spawn-dialog: could not find trigger button')
  await page.waitForSelector('[role="dialog"]', { timeout: 5000 })
  await page.waitForTimeout(400)
  await shot(page, 'spawn-dialog.png')
  // Close dialog so subsequent captures aren't polluted.
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
}

async function openNewChannelDialog(page) {
  // Toolbar button — Plus icon. Try labelled buttons first.
  const triggers = [
    page.getByRole('button', { name: /new channel/i }),
    page.getByRole('button', { name: /register channel/i }),
    page.getByRole('button', { name: /add channel/i }),
    page.locator('header button:has(svg.lucide-plus)'),
    page.locator('button:has(svg.lucide-plus)').first(),
  ]
  for (const t of triggers) {
    if (await t.count()) {
      await t.first().click()
      await page.waitForSelector('[role="dialog"]', { timeout: 5000 })
      return
    }
  }
  throw new Error('channels: could not find New channel trigger')
}

async function captureChannelsKindPicker(page) {
  await gotoAndSettle(page, '/channels')
  await openNewChannelDialog(page)
  // Open the Kind <Select>.
  const kindTrigger = page.locator('#kind, [id="kind"]').first()
  await kindTrigger.click()
  // Radix-UI Select renders content into a portal — wait for it.
  await page.waitForSelector('[role="listbox"], [role="menu"]', { timeout: 5000 })
  await page.waitForTimeout(300)
  await shot(page, 'channels-kind-picker.png')
  await page.keyboard.press('Escape') // close listbox
  await page.waitForTimeout(200)
  await page.keyboard.press('Escape') // close dialog
  await page.waitForTimeout(300)
}

async function captureChannelsNotificationsPanel(page) {
  await gotoAndSettle(page, '/channels')
  // Try editing an existing channel so the notifications block is fully populated.
  const editButtons = page.locator('button:has(svg.lucide-pencil), button[aria-label*="edit" i]')
  if (await editButtons.count()) {
    await editButtons.first().click()
  } else {
    // No channels yet — open Register dialog instead, the panel still renders.
    await openNewChannelDialog(page)
  }
  await page.waitForSelector('[role="dialog"]', { timeout: 5000 })
  // Scroll the dialog body to bring the "Session notifications" header into view.
  const heading = page.locator('text=Session notifications').first()
  await heading.scrollIntoViewIfNeeded({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(400)
  // Crop to the dialog's bounding box for a clean panel-focused image.
  const dialog = page.locator('[role="dialog"]').first()
  const box = await dialog.boundingBox()
  if (box) {
    await shot(page, 'channels-notifications-panel.png', {
      clip: {
        x: Math.max(0, Math.floor(box.x) - 12),
        y: Math.max(0, Math.floor(box.y) - 12),
        width: Math.ceil(box.width) + 24,
        height: Math.ceil(box.height) + 24,
      },
    })
  } else {
    await shot(page, 'channels-notifications-panel.png')
  }
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
}

async function captureFullPage(page, path, file) {
  await gotoAndSettle(page, path)
  await shot(page, file, { fullPage: false })
}

async function main() {
  log('out dir =', OUT_DIR)
  log('base url =', BASE_URL)
  if (!existsSync(OUT_DIR)) mkdirSync(OUT_DIR, { recursive: true })

  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: DEVICE_SCALE_FACTOR,
    colorScheme: 'dark',
  })
  const page = await context.newPage()

  try {
    log('logging in as', USER)
    await login(page)
    log('logged in, starting captures')

    await captureSidebarOverview(page)
    await captureSessionsLayout(page)
    await captureSpawnDialog(page)
    await captureChannelsKindPicker(page)
    await captureChannelsNotificationsPanel(page)
    await captureFullPage(page, '/providers', 'providers-layout.png')
    await captureFullPage(page, '/integrations', 'integrations-layout.png')
    await captureFullPage(page, '/activity', 'activity-layout.png')

    log('done — all 8 screenshots written to', OUT_DIR)
  } finally {
    await context.close()
    await browser.close()
  }
}

main().catch((err) => {
  console.error('[tutorial-shots] FAILED:', err)
  process.exit(1)
})
