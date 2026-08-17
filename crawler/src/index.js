import express from 'express'
import { chromium } from 'playwright'
import { lookup } from 'node:dns/promises'

const app = express(); app.use(express.json({ limit: '16kb' }))
app.get('/health', (_req, res) => res.json({ status: 'ok' }))
const isPrivateIp = (address) => address === '::1' || address.startsWith('127.') || address.startsWith('10.') || address.startsWith('192.168.') || address.startsWith('169.254.') || /^172\.(1[6-9]|2\d|3[0-1])\./.test(address) || address.startsWith('fc') || address.startsWith('fd') || address.startsWith('fe80:')
async function validateExternalUrl(rawUrl) {
  let target
  try { target = new URL(rawUrl) } catch { throw new Error('invalid') }
  if (!['http:', 'https:'].includes(target.protocol) || !target.hostname) throw new Error('invalid')
  const host = target.hostname.toLowerCase()
  if (host === 'localhost' || host.endsWith('.localhost') || host.endsWith('.local') || host.endsWith('.internal')) throw new Error('blocked')
  const resolved = await lookup(host, { all: true })
  if (!resolved.length || resolved.some(({ address }) => isPrivateIp(address))) throw new Error('blocked')
  return target.toString()
}
app.post('/fetch', async (req, res) => {
  if (!req.body?.url) return res.status(400).json({ error: 'url is required' })
  let browser
  try { const url = await validateExternalUrl(req.body.url); browser = await chromium.launch({ headless: true, args: typeof process.getuid === 'function' && process.getuid() === 0 ? ['--no-sandbox'] : [] }); const page = await browser.newPage(); await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 20000 }); const title = await page.title(); const text = (await page.locator('body').innerText()).slice(0, 20000); res.json({ title, text }) }
  catch (error) { const status = error.message === 'invalid' ? 400 : error.message === 'blocked' ? 403 : 502; res.status(status).json({ error: status === 403 ? 'url is blocked' : 'page fetch failed' }) }
  finally { await browser?.close() }
})
app.listen(process.env.PORT || 3000, () => console.log('crawler listening'))
