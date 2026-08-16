// isMockVideoUrl flags evidence URLs produced by cmd/seed-idea-trends
// (dev-only mock signals authored to exercise the recommendation pipeline
// without an Apify token). Those URLs point at "@mockuser" and don't
// resolve to a real TikTok video, so they shouldn't be rendered as
// clickable links. Real signals come from cmd/trend-ingest and use actual
// creator handles. Shared by components/menu-trends and components/trend-videos
// — both render evidence/scraped-video links from the same trend_signals corpus.
export function isMockVideoUrl(url: string): boolean {
  return /tiktok\.com\/@mockuser\//.test(url)
}
