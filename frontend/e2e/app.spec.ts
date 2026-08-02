import { expect, test } from '@playwright/test'

test('renders today dashboard shell', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('今天有')).toBeVisible()
  await expect(page.getByText('此刻,你感觉——')).toBeVisible()
})
