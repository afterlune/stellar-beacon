/**
 * Frontend-facing API contracts shared by blog and the future Vue 3 admin.
 * The backend keeps this envelope stable so an endpoint can be migrated
 * without changing callers or silently turning an error into empty data.
 */
export interface ResultVO<T = unknown> {
  flag: boolean
  code: number
  message: string
  data: T
}

export interface Page<T> {
  records: T[]
  count: number
  hasMore?: boolean
  nextSince?: string
}

export interface Article {
  id: number
  articleTitle: string
  articleContent: string
  createTime?: string
  updateTime?: string
  viewCount?: number
  categoryId?: number
  categoryName?: string
  top?: boolean
  featured?: boolean
  [key: string]: unknown
}

export interface ArticleDetail extends Article {
  articleCover: string
  preArticleCard: Article
  nextArticleCard: Article
}

export interface AdminArticle {
  id: number
  articleCover?: string
  articleTitle: string
  isTop?: number
  isFeatured?: number
  isDelete?: number
  status?: number
  type?: number
  createTime?: string
  categoryName?: string
  viewsCount?: number
  tagDTOs?: Tag[]
  [key: string]: unknown
}

export interface Category {
  id: number
  categoryName: string
  articleCount?: number
  [key: string]: unknown
}

export interface AdminCategory {
  id: number
  categoryName: string
  articleCount?: number | string
  createTime?: string
  [key: string]: unknown
}

export interface Tag {
  id: number
  tagName: string
  count?: number
  [key: string]: unknown
}

export interface AdminTag {
  id: number
  tagName: string
  articleCount?: number
  createTime?: string
  [key: string]: unknown
}

export interface AdminAlbum {
  id: number
  albumName: string
  albumDesc?: string
  albumCover?: string
  photoCount?: number
  status?: number
  createTime?: string
  [key: string]: unknown
}

export interface AdminFriendLink {
  id: number
  linkName: string
  linkAvatar: string
  linkAddress: string
  linkIntro: string
  createTime?: string
  [key: string]: unknown
}

export interface AdminTalk {
  id: number
  nickname?: string
  avatar?: string
  content?: string
  images?: string
  imgs?: string[]
  isTop?: number
  status?: number
  commentCount?: number
  createTime?: string
  [key: string]: unknown
}

export interface AdminPhoto {
  id: number
  photoName?: string
  photoDesc?: string
  photoSrc?: string
  [key: string]: unknown
}

export interface Comment {
  id: number
  commentContent: string
  createTime?: string
  nickname?: string
  [key: string]: unknown
}

export interface AdminComment {
  id: number
  avatar?: string
  nickname?: string
  replyNickname?: string
  articleTitle?: string
  commentContent?: string
  type?: number
  isReview?: number
  createTime?: string
  [key: string]: unknown
}

export interface AdminRole {
  id: number
  roleName: string
  isDisable?: number
  createTime?: string
  [key: string]: unknown
}

export interface AdminJob {
  id: number
  jobName: string
  jobGroup: string
  invokeTarget: string
  cronExpression: string
  misfirePolicy?: string | number
  concurrent?: number
  status?: number
  remark?: string
  createTime?: string
  updateTime?: string
  canRunOnce?: boolean
  runOnceReason?: string
  [key: string]: unknown
}

export interface JobRunOutcome {
  jobId: number
  target: string
  processed: boolean
}

export interface UserRole {
  id: number
  roleName: string
  isDisable?: number
  [key: string]: unknown
}

export interface AdminUser {
  id?: number
  userInfoId?: number
  avatar?: string
  nickname?: string
  loginType?: number
  ipAddress?: string
  ipSource?: string
  username?: string
  intro?: string
  website?: string
  roles?: UserRole[] | string[]
  createTime?: string
  lastLoginTime?: string
  isDisable?: number
  status?: number
  [key: string]: unknown
}

export interface Archive {
  time: string
  articles: Article[]
}

export interface Album {
  id: number
  albumName: string
  [key: string]: unknown
}

export interface Photo {
  id: number
  url?: string
  [key: string]: unknown
}

export interface WebsiteConfig {
  siteName?: string
  siteURL?: string
  siteAvatar?: string
  siteIntro?: string
  author?: string
  authorAvatar?: string
  authorIntro?: string
  notice?: string
  websiteCreateTime?: string
  [key: string]: unknown
}

export interface WebsiteSummary {
  viewCount: number
  articleCount: number
  talkCount: number
  categoryCount: number
  tagCount: number
  websiteConfigDTO: WebsiteConfig
}

export interface AgentFeatureFlags {
  publicChat: boolean
  vitals: boolean
  galaxy: boolean
  dreams: boolean
  capsules: boolean
  radio: boolean
  videos: boolean
  ttsEnabled: boolean
}

export interface AgentProfile {
  id: string
  name: string
  promptVersion: string
  systemPrompt: string
  opening?: string
  rhythmPrompts: Record<string, string>
  enabled: boolean
  updatedAt?: string
}

export interface AgentVitals {
  status: string
  lifeStage: string
  emotion: string
  emotionScores?: Record<string, number>
  phase: string
  timezone: string
  localTime: string
  nextTransitionAt: string
  articleCount: number
  categoryCount: number
  tagCount: number
  talkCount: number
  contentCount: number
  recentContentCount: number
  viewCount: number
  uniqueVisitorCount: number
  updatedAt: string
}

export interface GalaxyPoint {
  articleId: number
  lifeStage: string
  x: number
  y: number
  updatedAt: string
}

export interface RadioEpisode {
  id: string
  title: string
  script: string
  phase: string
  tts: boolean
  sourceArticleId?: number
  publishedAt: string
}

export interface RadioPage {
  records: RadioEpisode[]
  count: number
}

export interface VideoRecord {
  id: string
  title: string
  description?: string
  source: 'local' | 'external'
  url: string
  embedUrl?: string
  mimeType?: string
  sizeBytes?: number
  createdAt?: string
  updatedAt?: string
}

export interface VideoPage extends Page<VideoRecord> {
  frameOrigins?: string[]
}

export interface AIWritingPreview {
	reviewId: string
	runId: string
	operation: string
	preview: string
	diff?: string
}

export interface AIVisionPreview {
	reviewId: string
	runId: string
	operation: string
	preview: string
}

export interface AIReview {
  id: string
  targetType?: string
  targetId?: string
  operation: string
  status: string
  runId: string
  content?: string
  diff?: string
  rejectReason?: string
  createdAt?: string
  updatedAt?: string
  expiresAt?: string
  [key: string]: unknown
}

export interface AgentReviewPolicy {
  id: string
  version: number
  reviewRequired: boolean
  reviewTtlSeconds: number
  maxCandidateRunes: number
  similarityThreshold: number
  dailyLimit: number
  perArticleLimit: number
  perActionLimit: number
  allowedActions: string[]
  sensitivePatterns: string[]
  updatedAt?: string
}

export type AgentMemoryAssertionStatus = 'active' | 'stale' | 'conflicted' | 'retracted'

export interface AgentMemoryAssertion {
  id: string
  subjectKey: string
  predicate: string
  object: string
  sourceType: string
  sourceId: string
  version: number
  confidence: number
  status: AgentMemoryAssertionStatus
  validFrom: string
  validUntil?: string
  createdAt: string
  updatedAt: string
}

export interface AgentMemoryAssertionRevision {
  assertionId: string
  revisionNo: number
  subjectKey: string
  predicate: string
  object: string
  sourceType: string
  sourceId: string
  version: number
  confidence: number
  status: AgentMemoryAssertionStatus
  validFrom: string
  validUntil?: string
  reason: string
  changedAt: string
}

export type AgentMemoryConflictStatus = 'open' | 'resolved' | 'rejected'

export interface AgentMemoryConflictMember {
  assertionId: string
  role: 'candidate' | 'winner' | 'rejected'
  addedAt: string
}

export interface AgentMemoryConflict {
  id: string
  subjectKey: string
  predicate: string
  status: AgentMemoryConflictStatus
  winnerAssertionId?: string
  resolution?: string
  createdAt: string
  resolvedAt?: string
  updatedAt: string
  members: AgentMemoryConflictMember[]
}

export interface AgentMemoryAssertionPage {
  items: AgentMemoryAssertion[]
  current: number
  size: number
  hasMore: boolean
}

export interface AgentMemoryHistory {
  assertionId: string
  items: AgentMemoryAssertionRevision[]
}

export interface AgentMemoryConflictPage {
  items: AgentMemoryConflict[]
  current: number
  size: number
  hasMore: boolean
}

export interface Talk {
  id: number
  content?: string
  [key: string]: unknown
}

export interface FriendLink {
  id: number
  name?: string
  url?: string
  [key: string]: unknown
}

export interface TopAndFeaturedArticles {
  topArticle: Article
  featuredArticles: Article[]
}

export interface UserMenu {
  name: string
  path: string
  component: string
  icon?: string
  hidden?: boolean
  children?: UserMenu[]
}

export interface LoginResponse extends AdminUser {
  token: string
}

export interface Capsule {
  id: string
  title: string
  content?: string
  deliverAt: string
  status: string
  createdAt?: string
  deliveredAt?: string
}

export class ApiContractError extends Error {
  readonly code: number

  constructor(message: string, code = 50000) {
    super(message)
    this.name = 'ApiContractError'
    this.code = code
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

export function isResultVO<T = unknown>(value: unknown): value is ResultVO<T> {
  return (
    isRecord(value) &&
    typeof value.flag === 'boolean' &&
    typeof value.code === 'number' &&
    Number.isInteger(value.code) &&
    typeof value.message === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  )
}

/** Normalize untrusted JSON at the HTTP boundary without exposing details. */
export function parseResult<T = unknown>(value: unknown): ResultVO<T> {
  if (!isResultVO<T>(value)) {
    return {
      flag: false,
      code: 50000,
      message: '接口响应格式错误',
      data: null as T
    }
  }
  return value
}

export function unwrapResult<T = unknown>(value: unknown): T {
  const result = parseResult<T>(value)
  if (!result.flag) throw new ApiContractError(result.message, result.code)
  return result.data
}

export function normalizePage<T>(value: unknown): Page<T> {
  if (!isRecord(value)) return { records: [], count: 0 }
  const records = Array.isArray(value.records) ? value.records as T[] : []
  const count = typeof value.count === 'number' && Number.isFinite(value.count) ? value.count : records.length
  return { records, count, hasMore: typeof value.hasMore === 'boolean' ? value.hasMore : undefined, nextSince: typeof value.nextSince === 'string' ? value.nextSince : undefined }
}
