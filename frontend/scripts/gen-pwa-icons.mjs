// Generates the PWA icons in frontend/public without any dependencies: the
// 4-point sparkle from favicon.svg (#4f6ef7) on a #eef2f9 tile, at the sizes
// the web manifest and iOS need. Node's zlib is used to encode the PNGs.
//
// Run: node scripts/gen-pwa-icons.mjs
import { deflateSync } from 'node:zlib'
import { writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

// --- Minimal PNG encoder (8-bit RGBA) --------------------------------------
const CRC_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n++) {
    let c = n
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    table[n] = c >>> 0
  }
  return table
})()

function crc32(buffer) {
  let crc = 0xffffffff
  for (let i = 0; i < buffer.length; i++) crc = CRC_TABLE[(crc ^ buffer[i]) & 0xff] ^ (crc >>> 8)
  return (crc ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
  const out = Buffer.alloc(12 + data.length)
  out.writeUInt32BE(data.length, 0)
  out.write(type, 4, 'ascii')
  data.copy(out, 8)
  out.writeUInt32BE(crc32(out.subarray(4, 8 + data.length)), 8 + data.length)
  return out
}

function encodePNG(width, height, rgba) {
  const signature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(width, 0)
  ihdr.writeUInt32BE(height, 4)
  ihdr[8] = 8 // bit depth
  ihdr[9] = 6 // color type: RGBA
  const stride = width * 4 + 1
  const raw = Buffer.alloc(stride * height)
  for (let y = 0; y < height; y++) {
    raw[y * stride] = 0 // filter: none
    rgba.copy(raw, y * stride + 1, y * width * 4, (y + 1) * width * 4)
  }
  return Buffer.concat([signature, chunk('IHDR', ihdr), chunk('IDAT', deflateSync(raw)), chunk('IEND', Buffer.alloc(0))])
}

// --- Shapes (favicon.svg geometry) -----------------------------------------
// The 4-point sparkle: outer vertices at N/E/S/W (radius r) and inner
// (concave) vertices at the diagonals (r * 0.62). Ray casting point-in-polygon.
function sparkleContains(x, y, cx, cy, r) {
  const s = r * 0.62
  const poly = [
    [cx, cy - r], [cx + s, cy - s], [cx + r, cy], [cx + s, cy + s],
    [cx, cy + r], [cx - s, cy + s], [cx - r, cy], [cx - s, cy - s],
  ]
  let inside = false
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [xi, yi] = poly[i]
    const [xj, yj] = poly[j]
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) inside = !inside
  }
  return inside
}

function roundedRectContains(x, y, size, radius) {
  const max = size - 1 - radius
  if (x < radius && y < radius) return (x - radius) ** 2 + (y - radius) ** 2 <= radius ** 2
  if (x > max && y < radius) return (x - max) ** 2 + (y - radius) ** 2 <= radius ** 2
  if (x < radius && y > max) return (x - radius) ** 2 + (y - max) ** 2 <= radius ** 2
  if (x > max && y > max) return (x - max) ** 2 + (y - max) ** 2 <= radius ** 2
  return x >= 0 && x < size && y >= 0 && y < size
}

// --- Rendering --------------------------------------------------------------
const TILE = [0xee, 0xf2, 0xf9] // #eef2f9
const SPARKLE = [0x4f, 0x6e, 0xf7] // #4f6ef7

// 4x supersampling for smooth edges. maskable fills the whole canvas (the OS
// crops the mask); otherwise the tile is a rounded square with transparency.
function render(size, { maskable = false, radius = null } = {}) {
  const ss = 4
  const cornerRadius = radius ?? Math.round(size * 0.25)
  const sparkleR = size * 0.34 // star spans ~59% of the tile, inside the mask safe zone
  const cx = size / 2
  const cy = size / 2
  const rgba = Buffer.alloc(size * size * 4)
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      let r = 0, g = 0, b = 0, a = 0
      for (let sy = 0; sy < ss; sy++) {
        for (let sx = 0; sx < ss; sx++) {
          const u = x + (sx + 0.5) / ss
          const v = y + (sy + 0.5) / ss
          if (!maskable && !roundedRectContains(u, v, size, cornerRadius)) continue
          a += 255
          if (sparkleContains(u, v, cx, cy, sparkleR)) {
            r += SPARKLE[0]; g += SPARKLE[1]; b += SPARKLE[2]
          } else {
            r += TILE[0]; g += TILE[1]; b += TILE[2]
          }
        }
      }
      const i = (y * size + x) * 4
      const n = ss * ss
      rgba[i] = Math.round(r / n)
      rgba[i + 1] = Math.round(g / n)
      rgba[i + 2] = Math.round(b / n)
      rgba[i + 3] = Math.round(a / n)
    }
  }
  return encodePNG(size, size, rgba)
}

const files = {
  'pwa-192x192.png': render(192),
  'pwa-512x512.png': render(512),
  'pwa-maskable-512x512.png': render(512, { maskable: true }),
  'apple-touch-icon.png': render(180, { maskable: true }), // iOS applies its own rounding
}

for (const [name, png] of Object.entries(files)) {
  const out = join(root, 'public', name)
  writeFileSync(out, png)
  console.log(`wrote public/${name} (${png.length} bytes)`)
}
