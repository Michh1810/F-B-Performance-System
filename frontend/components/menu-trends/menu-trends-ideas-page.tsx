"use client"

import { useEffect, useMemo, useRef, useState } from "react"

import {
  ApiError,
  type ActiveMenuItem,
  type HashtagSuggestion,
  type MenuIdea,
  type RestaurantProfile,
  generateHashtagSuggestion,
  getLatestApprovedSuggestion,
  getRestaurantProfile,
  listActiveMenuItems,
  listHashtagSuggestions,
  listIdeas,
  promoteIdea,
  runMenuIdeaAgent,
  saveManualHashtagSuggestion,
  setIdeaStatus,
  updateHashtagSuggestionStatus,
  updateRestaurantProfile,
} from "@/lib/api"
import { computeLatestBatchMetrics } from "@/lib/idea-metrics"
import { useTrendIngestStatus } from "@/lib/use-trend-ingest-status"

import { IngestStatusBanner } from "@/components/ingest-status-banner"
import { PageHeader } from "./page-header"
import { RestaurantDescriptionCard } from "./restaurant-description-card"
import { HashtagDiscoveryCard } from "./hashtag-discovery-card"
import { HashtagSuggestionHistory } from "./hashtag-suggestion-history"
import { ActiveMenuSummary } from "./active-menu-summary"
import { ActiveMenuDrawer } from "./active-menu-drawer"
import { RunAgentButton, type RunPhase } from "./run-agent-button"
import { LatestRunSummary } from "./latest-run-summary"
import { IdeaFilters, type IdeaFilter, type IdeaSort } from "./idea-filters"
import { IdeaList } from "./idea-list"
import { IdeaReviewModal } from "./idea-review-modal"

function sameHashtags(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((tag, i) => tag === b[i])
}

const RUN_LOADING_PHASES = [
  "Fetching TikTok trends…",
  "Scanning active menu items…",
  "Generating menu recommendations…",
]

export function MenuTrendsIdeasPage() {
  // Restaurant description
  const [profile, setProfile] = useState<RestaurantProfile | null>(null)
  const [description, setDescription] = useState("")
  const [profileLoading, setProfileLoading] = useState(true)
  const [savingProfile, setSavingProfile] = useState(false)

  // Hashtags
  const [hashtags, setHashtags] = useState<string[]>([])
  const [savedHashtags, setSavedHashtags] = useState<string[]>([])
  const [generatingHashtags, setGeneratingHashtags] = useState(false)
  const [savingDraftHashtags, setSavingDraftHashtags] = useState(false)
  const [hashtagError, setHashtagError] = useState<string | null>(null)
  const [suggestions, setSuggestions] = useState<HashtagSuggestion[]>([])
  const [reviewingSuggestionId, setReviewingSuggestionId] = useState<string | null>(null)
  const ingestStatus = useTrendIngestStatus()

  // Active menu
  const [activeMenuItems, setActiveMenuItems] = useState<ActiveMenuItem[]>([])
  const [activeMenuLoading, setActiveMenuLoading] = useState(true)
  const [activeMenuError, setActiveMenuError] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Run agent
  const [runPhase, setRunPhase] = useState<RunPhase>("idle")
  const [runError, setRunError] = useState<string | null>(null)
  const [loadingPhaseIndex, setLoadingPhaseIndex] = useState(0)

  // Ideas
  const [ideas, setIdeas] = useState<MenuIdea[]>([])
  const [ideasLoading, setIdeasLoading] = useState(true)
  const [ideasError, setIdeasError] = useState<string | null>(null)
  const [filter, setFilter] = useState<IdeaFilter>("all")
  const [sort, setSort] = useState<IdeaSort>("newest")
  const [dismissingId, setDismissingId] = useState<string | null>(null)

  // Review modal
  const [reviewIdea, setReviewIdea] = useState<MenuIdea | null>(null)
  const [reviewOpen, setReviewOpen] = useState(false)
  const [reviewSubmitting, setReviewSubmitting] = useState(false)
  const [reviewError, setReviewError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    getRestaurantProfile(controller.signal)
      .then((p) => {
        setProfile(p)
        setDescription(p.description)
      })
      .catch(() => {
        // no profile saved yet — leave the description field empty
      })
      .finally(() => setProfileLoading(false))

    refreshSuggestions(controller.signal)
    refreshActiveMenu(controller.signal)
    refreshIdeas(controller.signal)

    return () => controller.abort()
  }, [])

  function refreshActiveMenu(signal?: AbortSignal) {
    setActiveMenuLoading(true)
    setActiveMenuError(null)
    listActiveMenuItems(signal)
      .then(setActiveMenuItems)
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        setActiveMenuError(err instanceof Error ? err.message : "Failed to load active menu items.")
      })
      .finally(() => setActiveMenuLoading(false))
  }

  function refreshIdeas(signal?: AbortSignal) {
    setIdeasLoading(true)
    setIdeasError(null)
    listIdeas(undefined, signal)
      .then(setIdeas)
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        setIdeasError(err instanceof Error ? err.message : "Failed to load ideas.")
      })
      .finally(() => setIdeasLoading(false))
  }

  // Seeds the editable hashtag draft from what's actually driving
  // trend-ingest: the most recently *approved* suggestion if one exists,
  // otherwise the most recently generated one (any status) so there's still
  // something to review/approve. syncBaseline also resets `savedHashtags`,
  // the dirty-check baseline for the Save button.
  function seedDraftFromSuggestions(list: HashtagSuggestion[]) {
    if (list.length === 0) return
    const seed = getLatestApprovedSuggestion(list) ?? list.reduce((a, b) => (a.generated_at > b.generated_at ? a : b))
    setHashtags(seed.hashtags)
    setSavedHashtags(seed.hashtags)
  }

  function refreshSuggestions(signal?: AbortSignal) {
    listHashtagSuggestions(undefined, signal)
      .then((list) => {
        setSuggestions(list)
        seedDraftFromSuggestions(list)
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        // no prior suggestions — start with an empty hashtag list
      })
  }

  async function handleSaveDescription() {
    setSavingProfile(true)
    try {
      const saved = await updateRestaurantProfile(description)
      setProfile(saved)
      setDescription(saved.description)
    } catch {
      // the Save button reappears since `dirty` is still true — the user can retry
    } finally {
      setSavingProfile(false)
    }
  }

  async function handleGenerateHashtags() {
    setHashtagError(null)
    setGeneratingHashtags(true)
    try {
      if (description.trim() && description !== profile?.description) {
        const saved = await updateRestaurantProfile(description)
        setProfile(saved)
        setDescription(saved.description)
      }
      const suggestion = await generateHashtagSuggestion()
      setHashtags(suggestion.hashtags)
      setSavedHashtags(suggestion.hashtags)
      setSuggestions((prev) => [suggestion, ...prev])
    } catch (err) {
      setHashtagError(err instanceof Error ? err.message : "Failed to generate hashtags.")
    } finally {
      setGeneratingHashtags(false)
    }
  }

  const hashtagsDirty = !sameHashtags(hashtags, savedHashtags)

  // Persists the current (edited) hashtag draft as a new "pending"
  // suggestion — see saveManualHashtagSuggestion. Without this, hashtags
  // added/removed via the card's Add/Remove buttons only ever lived in
  // local state and vanished on refresh.
  async function handleSaveDraftHashtags() {
    setHashtagError(null)
    setSavingDraftHashtags(true)
    try {
      const suggestion = await saveManualHashtagSuggestion(hashtags)
      setSavedHashtags(suggestion.hashtags)
      setSuggestions((prev) => [suggestion, ...prev])
    } catch (err) {
      setHashtagError(err instanceof Error ? err.message : "Failed to save hashtags.")
    } finally {
      setSavingDraftHashtags(false)
    }
  }

  async function handleApproveSuggestion(suggestion: HashtagSuggestion) {
    setReviewingSuggestionId(suggestion.id)
    try {
      await updateHashtagSuggestionStatus(suggestion.id, "approved")
      // Approving is the action that makes a suggestion the one
      // cmd/trend-ingest actually sweeps — sync the draft to match so the
      // card reflects what's now active instead of looking stale/dirty.
      setHashtags(suggestion.hashtags)
      setSavedHashtags(suggestion.hashtags)
      refreshSuggestions()
    } catch (err) {
      setHashtagError(err instanceof Error ? err.message : "Failed to approve suggestion.")
    } finally {
      setReviewingSuggestionId(null)
    }
  }

  async function handleRejectSuggestion(suggestion: HashtagSuggestion) {
    setReviewingSuggestionId(suggestion.id)
    try {
      await updateHashtagSuggestionStatus(suggestion.id, "rejected")
      refreshSuggestions()
    } catch (err) {
      setHashtagError(err instanceof Error ? err.message : "Failed to reject suggestion.")
    } finally {
      setReviewingSuggestionId(null)
    }
  }

  const activeSuggestionId = useMemo(
    () => getLatestApprovedSuggestion(suggestions)?.id ?? null,
    [suggestions]
  )

  const loadingPhaseTimer = useRef<ReturnType<typeof setInterval> | null>(null)

  async function handleRunAgent() {
    setRunPhase("running")
    setRunError(null)
    setLoadingPhaseIndex(0)
    loadingPhaseTimer.current = setInterval(() => {
      setLoadingPhaseIndex((i) => (i + 1) % RUN_LOADING_PHASES.length)
    }, 2200)

    try {
      await runMenuIdeaAgent()
      refreshIdeas()
      refreshActiveMenu()
      setRunPhase("idle")
    } catch (err) {
      setRunPhase("error")
      if (err instanceof ApiError && err.notImplemented) {
        setRunError("The run-agent endpoint isn't reachable (POST /api/ai/ideas/run returned 404/405).")
      } else {
        setRunError(err instanceof Error ? err.message : "Failed to run the Menu Idea Agent.")
      }
    } finally {
      if (loadingPhaseTimer.current) clearInterval(loadingPhaseTimer.current)
    }
  }

  function openReview(idea: MenuIdea) {
    setReviewIdea(idea)
    setReviewError(null)
    setReviewOpen(true)
  }

  async function handleDismiss(idea: MenuIdea) {
    setDismissingId(idea.id)
    try {
      await setIdeaStatus(idea.id, "dismissed")
      setIdeas((prev) => prev.map((i) => (i.id === idea.id ? { ...i, status: "dismissed" } : i)))
      if (reviewIdea?.id === idea.id) setReviewOpen(false)
    } catch (err) {
      setIdeasError(err instanceof Error ? err.message : "Failed to dismiss idea.")
    } finally {
      setDismissingId(null)
    }
  }

  async function handlePromote(idea: MenuIdea, pricing?: { priceCents: number; cogsCents: number }) {
    setReviewSubmitting(true)
    setReviewError(null)
    try {
      const updated = await promoteIdea(idea.id, pricing)
      setIdeas((prev) => prev.map((i) => (i.id === idea.id ? updated : i)))
      setReviewOpen(false)
    } catch (err) {
      setReviewError(err instanceof Error ? err.message : "Failed to approve idea.")
    } finally {
      setReviewSubmitting(false)
    }
  }

  const counts = useMemo(
    () => ({
      all: ideas.length,
      promotion: ideas.filter((i) => i.kind === "promotion").length,
      tweak: ideas.filter((i) => i.kind === "tweak").length,
    }),
    [ideas]
  )

  const visibleIdeas = useMemo(() => {
    const filtered = filter === "all" ? ideas : ideas.filter((i) => i.kind === filter)
    const sorted = [...filtered]
    if (sort === "newest") sorted.sort((a, b) => (a.generated_at < b.generated_at ? 1 : -1))
    else if (sort === "oldest") sorted.sort((a, b) => (a.generated_at > b.generated_at ? 1 : -1))
    else sorted.sort((a, b) => b.source_total_views - a.source_total_views)
    return sorted
  }, [ideas, filter, sort])

  const metrics = useMemo(() => computeLatestBatchMetrics(ideas), [ideas])

  return (
    <div className="flex flex-col gap-6 pb-16">
      <PageHeader />

      <div className="flex flex-col gap-6 px-8">
        <div className="flex flex-col gap-6 lg:flex-row">
          <RestaurantDescriptionCard
            value={description}
            onChange={setDescription}
            onSave={handleSaveDescription}
            dirty={description !== (profile?.description ?? "")}
            saving={savingProfile}
            loading={profileLoading}
            disabled={runPhase === "running"}
          />
          <HashtagDiscoveryCard
            hashtags={hashtags}
            onAdd={(tag) => setHashtags((prev) => [...prev, tag])}
            onRemove={(tag) => setHashtags((prev) => prev.filter((t) => t !== tag))}
            onGenerate={handleGenerateHashtags}
            generating={generatingHashtags}
            canGenerate={description.trim().length > 0}
            error={hashtagError}
            disabled={runPhase === "running"}
            dirty={hashtagsDirty}
            onSaveDraft={handleSaveDraftHashtags}
            savingDraft={savingDraftHashtags}
          />
        </div>

        <HashtagSuggestionHistory
          suggestions={suggestions}
          activeSuggestionId={activeSuggestionId}
          onApprove={handleApproveSuggestion}
          onReject={handleRejectSuggestion}
          busyId={reviewingSuggestionId}
        />

        <IngestStatusBanner status={ingestStatus} />

        <ActiveMenuSummary
          items={activeMenuItems}
          loading={activeMenuLoading}
          error={activeMenuError}
          onViewMenu={() => setDrawerOpen(true)}
        />
        <ActiveMenuDrawer open={drawerOpen} onOpenChange={setDrawerOpen} items={activeMenuItems} />

        <RunAgentButton
          phase={runPhase}
          loadingText={RUN_LOADING_PHASES[loadingPhaseIndex]}
          errorMessage={runError}
          hasHashtags={hashtags.length > 0}
          onRun={handleRunAgent}
        />

        <LatestRunSummary metrics={metrics} activeMenuItemsCount={activeMenuItems.length} />

        <IdeaFilters counts={counts} filter={filter} onFilterChange={setFilter} sort={sort} onSortChange={setSort} />

        <IdeaList
          ideas={visibleIdeas}
          loading={ideasLoading}
          error={ideasError}
          onReview={openReview}
          onDismiss={handleDismiss}
          dismissingId={dismissingId}
        />
      </div>

      <IdeaReviewModal
        idea={reviewIdea}
        open={reviewOpen}
        onOpenChange={setReviewOpen}
        onDismiss={handleDismiss}
        onPromote={handlePromote}
        submitting={reviewSubmitting}
        error={reviewError}
      />
    </div>
  )
}
