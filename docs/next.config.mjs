import path from 'node:path'
import { fileURLToPath } from 'node:url'
import nextra from 'nextra'

const docsDir = path.dirname(fileURLToPath(import.meta.url))
const withNextra = nextra({})

export default withNextra({
  images: { unoptimized: true },
  outputFileTracingRoot: path.join(docsDir, '..'),
})
