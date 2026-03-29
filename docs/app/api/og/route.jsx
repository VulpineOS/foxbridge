import { ImageResponse } from 'next/og'

export const runtime = 'edge'

export async function GET(req) {
  const { searchParams } = new URL(req.url)
  const title = searchParams.get('title') || 'Foxbridge'
  const description = searchParams.get('description') || 'CDP-to-Firefox Protocol Proxy'

  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '60px 80px',
          background: 'linear-gradient(135deg, #1a0f00 0%, #3d1f00 50%, #1a0f00 100%)',
          fontFamily: 'system-ui, sans-serif',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
            marginBottom: '40px',
          }}
        >
          <div
            style={{
              width: '48px',
              height: '48px',
              borderRadius: '50%',
              background: '#F97316',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '24px',
            }}
          >
            🦊
          </div>
          <span
            style={{
              fontSize: '24px',
              color: '#fdba74',
              fontWeight: 600,
              letterSpacing: '-0.02em',
            }}
          >
            Foxbridge
          </span>
        </div>

        <div
          style={{
            fontSize: '56px',
            fontWeight: 800,
            color: '#ffffff',
            lineHeight: 1.15,
            letterSpacing: '-0.03em',
            marginBottom: '24px',
            maxWidth: '900px',
          }}
        >
          {title}
        </div>

        <div
          style={{
            fontSize: '24px',
            color: '#fed7aa',
            lineHeight: 1.4,
            maxWidth: '800px',
          }}
        >
          {description}
        </div>

        <div
          style={{
            position: 'absolute',
            bottom: '60px',
            left: '80px',
            fontSize: '20px',
            color: '#8a6b3d',
          }}
        >
          foxbridge.vulpineos.com
        </div>

        <div
          style={{
            position: 'absolute',
            top: '0',
            right: '0',
            width: '400px',
            height: '400px',
            background: 'radial-gradient(circle, rgba(249,115,22,0.15) 0%, transparent 70%)',
          }}
        />
      </div>
    ),
    {
      width: 1200,
      height: 630,
    },
  )
}
