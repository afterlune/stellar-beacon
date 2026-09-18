/**
 * The public HTTP contract shared by the blog and administration apps.
 *
 * The server emits the canonical `{ code, message, data }` envelope. The
 * parser also accepts the pre-migration `{ flag, code, message, data }`
 * envelope so local Playwright fixtures can be upgraded independently.
 */
export type ApiCode =
  | 'OK'
  | 'UNAUTHENTICATED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'RATE_LIMITED'
  | 'INVALID_ARGUMENT'
  | 'USERNAME_EXISTS'
  | 'USERNAME_NOT_FOUND'
  | 'ARTICLE_PASSWORD_INVALID'
  | 'OPERATION_FAILED'
  | 'INTERNAL'
  | 'MALFORMED_RESPONSE'
  | (string & {})

export interface ApiResponse<T = unknown> {
  code: ApiCode
  message: string
  data: T
}

/** @deprecated Use ApiResponse. Kept as a source-compatible type alias. */
export type ResultVO<T = unknown> = ApiResponse<T>

export interface Page<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
  hasMore?: boolean
  nextSince?: string

  /** Transitional aliases for views that have not yet adopted the new names. */
  records: T[]
  count: number
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

export interface AdminArticleView {
	id: number
	articleTitle?: string
	articleContent?: string
	articleContentHtml?: string
	articleCover?: string
	categoryName?: string
	tagNames?: string[]
	status?: number
	type?: number
	isTop?: number
	isFeatured?: number
	password?: string
	originalUrl?: string
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
  nextValidTime?: string
  canRunOnce?: boolean
  runOnceReason?: string
  [key: string]: unknown
}

export interface AdminJobTarget {
  target: string
  name: string
  description: string
  cronExample: string
}

export interface JobRunOutcome {
  jobId: number
  target: string
  processed: boolean
  message?: string
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
  email?: string
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

export interface NewsletterSubscriber {
  id: number
  email: string
  status: 'pending' | 'active' | 'unsubscribed' | string
  confirmedAt?: string
  createdAt?: string
  updatedAt?: string
  [key: string]: unknown
}

export interface NewsletterDelivery {
  id: number
  subscriberId: number
  articleId: number
  status: 'queued' | 'sending' | 'sent' | 'failed' | string
  attempts: number
  lastError?: string
  sentAt?: string
  createdAt?: string
  subscriberEmail?: string
  articleTitle?: string
  [key: string]: unknown
}

export interface GrowthSummaryItem {
  eventName: string
  day: string
  count: number
  createdAt?: string
}

export type DashboardRange = '7d' | '30d' | '12m'

export interface DashboardOverview {
  totalViews: number
  todayViews: number
  monthViews: number
  userCount: number
  articleCount: number
  messageCount: number
}

export interface DashboardTrend {
  period: string
  views: number
}

export interface DashboardRegion {
  name: string
  label: string
  code: string
  value: number
}

export interface DashboardDistribution {
  name: string
  value: number
}

export interface DashboardArticleRank {
  id: number
  title: string
  views: number
}

export interface GrowthSubscriberStats {
  total: number
  active: number
  pending: number
  unsubscribed: number
  confirmationRate: number
}

export interface GrowthDeliveryStats {
  queued: number
  sending: number
  sent: number
  failed: number
  successRate: number
}

export interface DashboardGrowthTrend {
  period: string
  shareClicks: number
  subscribeStarts: number
  subscribeConfirms: number
  unsubscribes: number
  deliverySent: number
  deliveryFailed: number
}

export interface DashboardGrowth {
  subscribers: GrowthSubscriberStats
  deliveries: GrowthDeliveryStats
  trend: DashboardGrowthTrend[]
}

export interface SMTPHealth {
  configured: boolean
  reachable: boolean
  host: string
  port: number
  tls: boolean
  auth: boolean
  checkedAt?: string
  message: string
}

export interface NewsletterHealth {
  subscribers: GrowthSubscriberStats
  deliveries: GrowthDeliveryStats
  smtp: SMTPHealth
  generatedAt?: string
}

export interface AdminDashboardAnalytics {
  range: DashboardRange
  unit: 'day' | 'month'
  overview: DashboardOverview
  trend: DashboardTrend[]
  regions: DashboardRegion[]
  categories: DashboardDistribution[]
  tags: DashboardDistribution[]
  articleRank: DashboardArticleRank[]
  growth: DashboardGrowth
  generatedAt: string
}

export type ContentAnalyticsRange = '7d' | '30d' | '90d' | '12m'

export interface ContentAnalyticsOverview {
  views: number
  uniqueReaders: number
  effectiveSessions: number
  avgActiveMs: number
  completionRate: number
}

export interface ContentAnalyticsTrend extends ContentAnalyticsOverview {
  period: string
}

export interface AdminContentAnalytics {
  range: ContentAnalyticsRange
  unit: 'day' | 'month'
  overview: ContentAnalyticsOverview
  trend: ContentAnalyticsTrend[]
  generatedAt: string
}

export interface ContentArticlePerformance {
  articleId: number
  articleTitle: string
  articleCover: string
  categoryName: string
  createTime: string
  views: number
  uniqueReaders: number
  effectiveSessions: number
  avgActiveMs: number
  completionRate: number
}

export interface ContentArticleAnalyticsDetail {
  articleId: number
  articleTitle: string
  articleCover: string
  categoryName: string
  createTime: string
  overview: ContentAnalyticsOverview
  trend: ContentAnalyticsTrend[]
}

export interface AdminMediaAsset {
  key: string
  url: string
  name: string
  size?: number
  contentType?: string
  lastModified?: string
  deletable?: boolean
}

export interface UserMenu {
  name: string
  path: string
  component: string
  icon?: string
  hidden?: boolean
  children?: UserMenu[]
}

interface LegacyResultVO<T = unknown> {
  flag: boolean
  code: number
  message: string
  data: T
}

export class ApiContractError extends Error {
  readonly code: ApiCode

  constructor(message: string, code: ApiCode = 'INTERNAL') {
    super(message)
    this.name = 'ApiContractError'
    this.code = code
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function legacyCode(code: number, flag: boolean): ApiCode {
  if (flag || code === 20000) return 'OK'
  switch (code) {
    case 40001:
    case 41000:
      return 'UNAUTHENTICATED'
    case 40300:
      return 'FORBIDDEN'
    case 40400:
      return 'NOT_FOUND'
    case 40900:
      return 'CONFLICT'
    case 42900:
      return 'RATE_LIMITED'
    case 52001:
      return 'USERNAME_EXISTS'
    case 52002:
      return 'USERNAME_NOT_FOUND'
    case 52003:
      return 'ARTICLE_PASSWORD_INVALID'
    case 52000:
      return 'INVALID_ARGUMENT'
    case 51000:
      return 'OPERATION_FAILED'
    default:
      return code >= 50000 ? 'INTERNAL' : 'OPERATION_FAILED'
  }
}

function numericCode(code: ApiCode): number {
  switch (code) {
    case 'OK': return 20000
    case 'UNAUTHENTICATED': return 40001
    case 'FORBIDDEN': return 40300
    case 'NOT_FOUND': return 40400
    case 'CONFLICT': return 40900
    case 'RATE_LIMITED': return 42900
    case 'INVALID_ARGUMENT': return 52000
    case 'USERNAME_EXISTS': return 52001
    case 'USERNAME_NOT_FOUND': return 52002
    case 'ARTICLE_PASSWORD_INVALID': return 52003
    case 'OPERATION_FAILED': return 51000
    case 'MALFORMED_RESPONSE': return 50000
    default: return 50000
  }
}

export function isSuccessCode(code: ApiCode): boolean {
  return code === 'OK' || code === 'SUCCESS'
}

export function isAuthenticationCode(code: ApiCode): boolean {
  return code === 'UNAUTHENTICATED' || code === 'AUTH_REQUIRED' || code === 'TOKEN_EXPIRED'
}

export function isResultVO<T = unknown>(value: unknown): value is ApiResponse<T> {
  return (
    isRecord(value) &&
    typeof value.code === 'string' &&
    typeof value.message === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  )
}

function isLegacyResultVO<T = unknown>(value: unknown): value is LegacyResultVO<T> {
  return (
    isRecord(value) &&
    typeof value.flag === 'boolean' &&
    typeof value.code === 'number' &&
    Number.isInteger(value.code) &&
    typeof value.message === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  )
}

export function parseResult<T = unknown>(value: unknown): ApiResponse<T> {
  if (isResultVO<T>(value)) return value
  if (isLegacyResultVO<T>(value)) {
    return {
      code: legacyCode(value.code, value.flag),
      message: value.message,
      data: value.data
    }
  }
  return {
    code: 'MALFORMED_RESPONSE',
    message: '接口响应格式错误',
    data: null as T
  }
}

/** Convert a canonical response to the legacy shape used by the blog UI. */
export function toLegacyResult<T>(value: ApiResponse<T>): LegacyResultVO<T> {
  return {
    flag: isSuccessCode(value.code),
    code: numericCode(value.code),
    message: value.message,
    data: toLegacyData(value.data) as T
  }
}

function toLegacyData(value: unknown): unknown {
  if (!isRecord(value) || (!Array.isArray(value.items) && !Array.isArray(value.records))) return value
  const records = (Array.isArray(value.records) ? value.records : value.items) as unknown[]
  const count = typeof value.count === 'number'
    ? value.count
    : typeof value.total === 'number'
      ? value.total
      : records.length
  return { ...value, records, count }
}

export function unwrapResult<T = unknown>(value: unknown): T {
  const result = parseResult<T>(value)
  if (!isSuccessCode(result.code)) throw new ApiContractError(result.message, result.code)
  return result.data
}

export function normalizePage<T>(value: unknown): Page<T> {
  if (!isRecord(value)) {
    return { items: [], total: 0, page: 1, pageSize: 0, records: [], count: 0 }
  }
  const items = Array.isArray(value.items)
    ? value.items as T[]
    : Array.isArray(value.records)
      ? value.records as T[]
      : []
  const totalValue = typeof value.total === 'number' ? value.total : value.count
  const total = typeof totalValue === 'number' && Number.isFinite(totalValue) ? totalValue : items.length
  const page = typeof value.page === 'number' && Number.isFinite(value.page) && value.page > 0 ? value.page : 1
  const pageSizeValue = typeof value.pageSize === 'number' ? value.pageSize : value.size
  const pageSize = typeof pageSizeValue === 'number' && Number.isFinite(pageSizeValue) && pageSizeValue >= 0
    ? pageSizeValue
    : items.length
  return {
    items,
    total,
    page,
    pageSize,
    records: items,
    count: total,
    hasMore: typeof value.hasMore === 'boolean' ? value.hasMore : undefined,
    nextSince: typeof value.nextSince === 'string' ? value.nextSince : undefined
  }
}

/** Reader reactions: `like` and `favorite` are the only accepted kinds. */
export type ReactionKind = 'like' | 'favorite'

export interface ReactionToggleResult {
  active: boolean
  likeCount: number
  favoriteCount: number
}

/** Article collections: an ordered group of published articles. */
export interface SeriesSummary {
  id: number
  seriesName: string
  seriesDesc: string
  cover: string
  articleCount: number
}
