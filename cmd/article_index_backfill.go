package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"benetnasch/app/bootstrap"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/search"
	"benetnasch/app/infra/task"

	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/cobra"
)

var articleIndexBackfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "manage the persistent article index backfill",
	Args:  cobra.NoArgs,
}

var articleIndexCmd = &cobra.Command{
	Use:   "article-index",
	Short: "manage article search indexes",
	Args:  cobra.NoArgs,
}

var articleIndexBackfillStartCmd = &cobra.Command{
	Use:   "start",
	Short: "create or reuse a backfill and run it until completion or pause",
	Args:  cobra.NoArgs,
	RunE:  runArticleIndexBackfillStart,
}

var articleIndexBackfillRunCmd = &cobra.Command{
	Use:   "run",
	Short: "run an existing backfill from its persisted cursor",
	Args:  cobra.NoArgs,
	RunE:  runArticleIndexBackfill,
}

var articleIndexBackfillPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "request a cooperative pause after the current article",
	Args:  cobra.NoArgs,
	RunE:  requestArticleIndexBackfillPause,
}

var articleIndexBackfillResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "clear a pause or failed state so the next run can continue",
	Args:  cobra.NoArgs,
	RunE:  resumeArticleIndexBackfill,
}

var articleIndexBackfillStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "show the persisted backfill state",
	Args:  cobra.NoArgs,
	RunE:  showArticleIndexBackfillStatus,
}

var articleIndexBackfillPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "read public articles and estimate the local chunking workload",
	Args:  cobra.NoArgs,
	RunE:  runArticleIndexBackfillPlan,
}

var articleIndexProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "create and configure the versioned article chunk index",
	Args:  cobra.NoArgs,
	RunE:  runArticleIndexProvision,
}

var articleIndexSwapCmd = &cobra.Command{
	Use:   "swap",
	Short: "atomically exchange the current and candidate article chunk indexes",
	Args:  cobra.NoArgs,
	RunE:  runArticleIndexSwap,
}

func init() {
	articleIndexBackfillCmd.PersistentFlags().String("run-id", "", "backfill run ID (default: article-index-backfill-<index-version>)")
	articleIndexBackfillCmd.PersistentFlags().Int("page-size", task.DefaultArticleIndexBackfillPageSize, "articles read per source page (low-memory default: 1)")
	articleIndexBackfillCmd.PersistentFlags().Int("max-articles-per-run", 1, "maximum articles to process in this run (0 explicitly means until completion or pause)")
	articleIndexBackfillCmd.PersistentFlags().Duration("run-timeout", 10*time.Minute, "maximum time for one write-enabled backfill run")
	articleIndexBackfillCmd.PersistentFlags().Bool("allow-writes", false, "authorize database state and Meilisearch writes for backfill control")
	articleIndexCmd.AddCommand(articleIndexBackfillCmd)
	articleIndexBackfillCmd.AddCommand(
		articleIndexBackfillStartCmd,
		articleIndexBackfillRunCmd,
		articleIndexBackfillPauseCmd,
		articleIndexBackfillResumeCmd,
		articleIndexBackfillStatusCmd,
		articleIndexBackfillPlanCmd,
	)
	articleIndexBackfillPlanCmd.Flags().Int("after-article-id", 0, "exclusive article ID cursor for a read-only plan")
	articleIndexBackfillPlanCmd.Flags().Int("max-articles", 0, "maximum articles to inspect (0 means all; a positive value is a bounded sample)")
	articleIndexBackfillPlanCmd.Flags().Duration("timeout", 5*time.Minute, "maximum time for the read-only source scan")
	articleIndexProvisionCmd.Flags().Bool("allow-writes", false, "authorize writes to the configured Meilisearch index")
	articleIndexProvisionCmd.Flags().Duration("task-timeout", 10*time.Minute, "maximum time to wait for provisioning tasks")
	articleIndexProvisionCmd.Flags().Duration("task-poll-interval", 250*time.Millisecond, "interval between Meilisearch task status checks")
	articleIndexSwapCmd.Flags().Bool("allow-writes", false, "authorize the atomic Meilisearch index swap")
	articleIndexSwapCmd.Flags().Duration("task-timeout", 10*time.Minute, "maximum time to wait for the swap task")
	articleIndexSwapCmd.Flags().Duration("task-poll-interval", 250*time.Millisecond, "interval between swap task status checks")
	articleIndexSwapCmd.Flags().String("current-index", "", "current versioned article chunk index UID")
	articleIndexSwapCmd.Flags().String("current-index-version", "", "current index version, used to validate the current UID")
	articleIndexSwapCmd.Flags().String("current-provider", "", "provider contract for the current index")
	articleIndexSwapCmd.Flags().String("current-model", "", "model contract for the current index")
	articleIndexSwapCmd.Flags().String("current-model-version", "", "model version contract for the current index")
	articleIndexSwapCmd.Flags().Int("current-dimension", 0, "vector dimension contract for the current index")
	articleIndexSwapCmd.Flags().Int("current-batch-size", 0, "embedding batch size contract for the current index")
	articleIndexSwapCmd.Flags().String("candidate-index", "", "candidate UID; defaults to the configured article_chunks index")
	articleIndexCmd.AddCommand(articleIndexProvisionCmd)
	articleIndexCmd.AddCommand(articleIndexSwapCmd)
	rootCmd.AddCommand(articleIndexCmd)
}

func runArticleIndexProvision(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexWriteAuthorization(allowWrites); err != nil {
		return err
	}
	taskTimeout, err := cmd.Flags().GetDuration("task-timeout")
	if err != nil {
		return err
	}
	if taskTimeout <= 0 {
		return fmt.Errorf("task-timeout must be positive")
	}
	pollInterval, err := cmd.Flags().GetDuration("task-poll-interval")
	if err != nil {
		return err
	}
	if pollInterval < 0 {
		return fmt.Errorf("task-poll-interval cannot be negative")
	}

	runtime, err := bootstrap.InitializeArticleChunksIndexProvision()
	if err != nil {
		return fmt.Errorf("initialize article index provisioning: %w", err)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), taskTimeout)
	defer cancel()
	tasks, err := search.ProvisionArticleChunksIndex(ctx, runtime.Client, runtime.Spec)
	if err != nil {
		return fmt.Errorf("provision article chunk index: %w", err)
	}
	completed, err := search.WaitForArticleChunksTasks(ctx, runtime.Client.TaskReader(), runtime.Spec.UID, tasks, pollInterval)
	if err != nil {
		return fmt.Errorf("wait for article chunk index tasks: %w", err)
	}
	return writeArticleIndexProvisionOutput(cmd.OutOrStdout(), runtime.Spec, tasks, completed)
}

func requireArticleIndexWriteAuthorization(allowWrites bool) error {
	if !allowWrites {
		return fmt.Errorf("article-index write operations require explicit --allow-writes")
	}
	return nil
}

func runArticleIndexSwap(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexWriteAuthorization(allowWrites); err != nil {
		return err
	}
	taskTimeout, err := cmd.Flags().GetDuration("task-timeout")
	if err != nil {
		return err
	}
	if taskTimeout <= 0 {
		return fmt.Errorf("task-timeout must be positive")
	}
	pollInterval, err := cmd.Flags().GetDuration("task-poll-interval")
	if err != nil {
		return err
	}
	if pollInterval < 0 {
		return fmt.Errorf("task-poll-interval cannot be negative")
	}

	currentIndex, err := requiredArticleIndexSwapStringFlag(cmd, "current-index")
	if err != nil {
		return err
	}
	currentIndexVersion, err := requiredArticleIndexSwapStringFlag(cmd, "current-index-version")
	if err != nil {
		return err
	}
	currentProvider, err := requiredArticleIndexSwapStringFlag(cmd, "current-provider")
	if err != nil {
		return err
	}
	currentModel, err := requiredArticleIndexSwapStringFlag(cmd, "current-model")
	if err != nil {
		return err
	}
	currentModelVersion, err := requiredArticleIndexSwapStringFlag(cmd, "current-model-version")
	if err != nil {
		return err
	}
	currentDimension, err := cmd.Flags().GetInt("current-dimension")
	if err != nil {
		return err
	}
	currentBatchSize, err := cmd.Flags().GetInt("current-batch-size")
	if err != nil {
		return err
	}
	candidateIndex, err := cmd.Flags().GetString("candidate-index")
	if err != nil {
		return err
	}
	candidateIndex = strings.TrimSpace(candidateIndex)

	runtime, err := bootstrap.InitializeArticleChunksIndexProvision()
	if err != nil {
		return fmt.Errorf("initialize article index swap: %w", err)
	}
	if candidateIndex == "" {
		candidateIndex = runtime.Spec.UID
	}
	if candidateIndex != runtime.Spec.UID {
		return fmt.Errorf("candidate index %q does not match the configured article chunk index %q", candidateIndex, runtime.Spec.UID)
	}

	currentSpec, err := newArticleIndexSwapCurrentSpec(currentIndex, currentIndexVersion, currentProvider, currentModel, currentModelVersion, currentDimension, currentBatchSize)
	if err != nil {
		return fmt.Errorf("validate current article index contract: %w", err)
	}
	plan, err := search.NewArticleChunksIndexSwapPlan(currentSpec, runtime.Spec)
	if err != nil {
		return fmt.Errorf("build article index swap plan: %w", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), taskTimeout)
	defer cancel()
	task, err := search.SwapArticleChunksIndex(ctx, runtime.Client, plan)
	if err != nil {
		return fmt.Errorf("swap article chunk indexes: %w", err)
	}
	completed, err := search.WaitForArticleChunksTasks(ctx, runtime.Client.TaskReader(), plan.Candidate.UID, []*meilisearch.TaskInfo{task}, pollInterval)
	if err != nil {
		return fmt.Errorf("wait for article index swap task: %w", err)
	}
	return writeArticleIndexSwapOutput(cmd.OutOrStdout(), plan, task, completed)
}

func requiredArticleIndexSwapStringFlag(cmd *cobra.Command, name string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("--%s is required", name)
	}
	return value, nil
}

func newArticleIndexSwapCurrentSpec(index, indexVersion, provider, model, modelVersion string, dimension, batchSize int) (search.ArticleChunksIndexSpec, error) {
	spec, err := search.NewArticleChunksIndexSpec(
		config.AIModelRoute{Provider: provider, Model: model},
		config.AIEmbeddingConfig{
			IndexVersion: indexVersion,
			ModelVersion: modelVersion,
			Dimension:    dimension,
			BatchSize:    batchSize,
		},
	)
	if err != nil {
		return search.ArticleChunksIndexSpec{}, err
	}
	if spec.UID != index {
		return search.ArticleChunksIndexSpec{}, fmt.Errorf("current index %q does not match index version %q", index, indexVersion)
	}
	return spec, nil
}

type articleIndexProvisionTaskOutput struct {
	TaskUID  int64  `json:"taskUid"`
	IndexUID string `json:"indexUid,omitempty"`
}

type articleIndexProvisionOutput struct {
	IndexUID           string                            `json:"indexUid"`
	IndexVersion       string                            `json:"indexVersion"`
	Provider           string                            `json:"provider"`
	Model              string                            `json:"model"`
	ModelVersion       string                            `json:"modelVersion"`
	Dimension          int                               `json:"dimension"`
	EmbeddingBatchSize int                               `json:"embeddingBatchSize"`
	Tasks              []articleIndexProvisionTaskOutput `json:"tasks"`
	CompletedTasks     []articleIndexProvisionTaskOutput `json:"completedTasks"`
}

type articleIndexSwapOutput struct {
	CurrentIndex     string `json:"currentIndex"`
	CandidateIndex   string `json:"candidateIndex"`
	SwapTaskUID      int64  `json:"swapTaskUid"`
	CompletedTaskUID int64  `json:"completedTaskUid"`
}

func writeArticleIndexProvisionOutput(output io.Writer, spec search.ArticleChunksIndexSpec, tasks []*meilisearch.TaskInfo, completed []*meilisearch.Task) error {
	if output == nil {
		return fmt.Errorf("article index provision output is nil")
	}
	value := articleIndexProvisionOutput{
		IndexUID:           spec.UID,
		IndexVersion:       spec.IndexVersion,
		Provider:           spec.Provider,
		Model:              spec.Model,
		ModelVersion:       spec.ModelVersion,
		Dimension:          spec.Dimension,
		EmbeddingBatchSize: spec.EmbeddingBatchSize,
		Tasks:              make([]articleIndexProvisionTaskOutput, 0, len(tasks)),
		CompletedTasks:     make([]articleIndexProvisionTaskOutput, 0, len(completed)),
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		value.Tasks = append(value.Tasks, articleIndexProvisionTaskOutput{TaskUID: task.TaskUID, IndexUID: task.IndexUID})
	}
	for _, task := range completed {
		if task == nil {
			continue
		}
		value.CompletedTasks = append(value.CompletedTasks, articleIndexProvisionTaskOutput{TaskUID: search.ArticleChunksTaskUID(task), IndexUID: task.IndexUID})
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeArticleIndexSwapOutput(output io.Writer, plan search.ArticleChunksIndexSwapPlan, task *meilisearch.TaskInfo, completed []*meilisearch.Task) error {
	if output == nil {
		return fmt.Errorf("article index swap output is nil")
	}
	if task == nil || task.TaskUID <= 0 {
		return fmt.Errorf("article index swap task is invalid")
	}
	if len(completed) != 1 || search.ArticleChunksTaskUID(completed[0]) != task.TaskUID {
		return fmt.Errorf("article index swap completion does not match the submitted task")
	}
	value := articleIndexSwapOutput{
		CurrentIndex:     plan.Current.UID,
		CandidateIndex:   plan.Candidate.UID,
		SwapTaskUID:      task.TaskUID,
		CompletedTaskUID: search.ArticleChunksTaskUID(completed[0]),
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func runArticleIndexBackfillStart(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexBackfillWriteAuthorization(allowWrites); err != nil {
		return err
	}
	maxArticles, runTimeout, err := articleIndexBackfillRunOptions(cmd)
	if err != nil {
		return err
	}
	runtime, err := initializeArticleIndexBackfill(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), runTimeout)
	defer cancel()
	if _, err := runtime.Backfill.Create(ctx); err != nil {
		return fmt.Errorf("create article index backfill: %w", err)
	}
	return executeArticleIndexBackfillWithLimit(ctx, cmd.OutOrStdout(), runtime.Backfill, maxArticles)
}

func runArticleIndexBackfill(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexBackfillWriteAuthorization(allowWrites); err != nil {
		return err
	}
	maxArticles, runTimeout, err := articleIndexBackfillRunOptions(cmd)
	if err != nil {
		return err
	}
	runtime, err := initializeArticleIndexBackfill(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), runTimeout)
	defer cancel()
	return executeArticleIndexBackfillWithLimit(ctx, cmd.OutOrStdout(), runtime.Backfill, maxArticles)
}

func requestArticleIndexBackfillPause(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexBackfillWriteAuthorization(allowWrites); err != nil {
		return err
	}
	runtime, err := initializeArticleIndexBackfillControl(cmd)
	if err != nil {
		return err
	}
	if err := runtime.States.RequestPause(cmd.Context(), runtime.RunID); err != nil {
		return fmt.Errorf("request article index backfill pause: %w", err)
	}
	state, err := runtime.States.Get(cmd.Context(), runtime.RunID)
	if err != nil {
		return err
	}
	return writeArticleIndexBackfillState(cmd.OutOrStdout(), state)
}

func resumeArticleIndexBackfill(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if err := requireArticleIndexBackfillWriteAuthorization(allowWrites); err != nil {
		return err
	}
	runtime, err := initializeArticleIndexBackfillControl(cmd)
	if err != nil {
		return err
	}
	if err := runtime.States.Resume(cmd.Context(), runtime.RunID); err != nil {
		return fmt.Errorf("resume article index backfill: %w", err)
	}
	state, err := runtime.States.Get(cmd.Context(), runtime.RunID)
	if err != nil {
		return err
	}
	return writeArticleIndexBackfillState(cmd.OutOrStdout(), state)
}

func showArticleIndexBackfillStatus(cmd *cobra.Command, _ []string) error {
	runtime, err := initializeArticleIndexBackfillControl(cmd)
	if err != nil {
		return err
	}
	state, err := runtime.States.Get(cmd.Context(), runtime.RunID)
	if err != nil {
		return err
	}
	return writeArticleIndexBackfillState(cmd.OutOrStdout(), state)
}

func runArticleIndexBackfillPlan(cmd *cobra.Command, _ []string) error {
	allowWrites, err := cmd.Flags().GetBool("allow-writes")
	if err != nil {
		return err
	}
	if allowWrites {
		return fmt.Errorf("article-index backfill plan is read-only; omit --allow-writes")
	}
	pageSize := task.DefaultArticleIndexPlanPageSize
	if pageSizeFlag := cmd.Flags().Lookup("page-size"); pageSizeFlag != nil && pageSizeFlag.Changed {
		pageSize, err = cmd.Flags().GetInt("page-size")
		if err != nil {
			return err
		}
	}
	afterArticleID, err := cmd.Flags().GetInt("after-article-id")
	if err != nil {
		return err
	}
	maxArticles, err := cmd.Flags().GetInt("max-articles")
	if err != nil {
		return err
	}
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	runtime, err := bootstrap.InitializeArticleIndexPlan(pageSize)
	if err != nil {
		return fmt.Errorf("initialize article index plan: %w", err)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	result, runErr := runtime.Plan.Run(ctx, afterArticleID, maxArticles)
	if writeErr := writeArticleIndexPlan(cmd.OutOrStdout(), result); writeErr != nil {
		if runErr != nil {
			return fmt.Errorf("run article index plan: %w; write plan: %v", runErr, writeErr)
		}
		return fmt.Errorf("write article index plan: %w", writeErr)
	}
	if runErr != nil {
		return fmt.Errorf("run article index plan: %w", runErr)
	}
	return nil
}

func requireArticleIndexBackfillWriteAuthorization(allowWrites bool) error {
	if !allowWrites {
		return fmt.Errorf("article-index backfill write operations require explicit --allow-writes")
	}
	return nil
}

func articleIndexBackfillRunOptions(cmd *cobra.Command) (int, time.Duration, error) {
	maxArticles, err := cmd.Flags().GetInt("max-articles-per-run")
	if err != nil {
		return 0, 0, err
	}
	if maxArticles < 0 {
		return 0, 0, fmt.Errorf("max-articles-per-run cannot be negative")
	}
	runTimeout, err := cmd.Flags().GetDuration("run-timeout")
	if err != nil {
		return 0, 0, err
	}
	if runTimeout <= 0 {
		return 0, 0, fmt.Errorf("run-timeout must be positive")
	}
	return maxArticles, runTimeout, nil
}

func initializeArticleIndexBackfill(cmd *cobra.Command) (bootstrap.ArticleIndexBackfillRuntime, error) {
	runID, err := cmd.Flags().GetString("run-id")
	if err != nil {
		return bootstrap.ArticleIndexBackfillRuntime{}, err
	}
	pageSize, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return bootstrap.ArticleIndexBackfillRuntime{}, err
	}
	return bootstrap.InitializeArticleIndexBackfill(runID, pageSize)
}

func initializeArticleIndexBackfillControl(cmd *cobra.Command) (bootstrap.ArticleIndexBackfillControlRuntime, error) {
	runID, err := cmd.Flags().GetString("run-id")
	if err != nil {
		return bootstrap.ArticleIndexBackfillControlRuntime{}, err
	}
	return bootstrap.InitializeArticleIndexBackfillControl(runID)
}

func executeArticleIndexBackfill(ctx context.Context, output io.Writer, backfill interface {
	Run(context.Context) (port.ArticleIndexBackfillState, error)
}) error {
	return executeArticleIndexBackfillWithLimit(ctx, output, backfill, 0)
}

func executeArticleIndexBackfillWithLimit(ctx context.Context, output io.Writer, backfill interface {
	Run(context.Context) (port.ArticleIndexBackfillState, error)
}, maxArticles int) error {
	if maxArticles < 0 {
		return fmt.Errorf("max-articles-per-run cannot be negative")
	}
	var (
		state port.ArticleIndexBackfillState
		err   error
	)
	if maxArticles > 0 {
		limited, ok := backfill.(interface {
			RunWithLimit(context.Context, int) (port.ArticleIndexBackfillState, error)
		})
		if !ok {
			return fmt.Errorf("article index backfill does not support a per-run article limit")
		}
		state, err = limited.RunWithLimit(ctx, maxArticles)
	} else {
		state, err = backfill.Run(ctx)
	}
	if writeErr := writeArticleIndexBackfillState(output, state); writeErr != nil {
		if err != nil {
			return fmt.Errorf("run article index backfill: %w; write status: %v", err, writeErr)
		}
		return fmt.Errorf("write article index backfill status: %w", writeErr)
	}
	if err != nil {
		return fmt.Errorf("run article index backfill: %w", err)
	}
	return nil
}

type articleIndexBackfillStateOutput struct {
	ID                 string                          `json:"id"`
	IndexUID           string                          `json:"indexUid"`
	IndexVersion       string                          `json:"indexVersion"`
	Provider           string                          `json:"provider"`
	Model              string                          `json:"model"`
	ModelVersion       string                          `json:"modelVersion"`
	Dimension          int                             `json:"dimension"`
	EmbeddingBatchSize int                             `json:"embeddingBatchSize"`
	PageSize           int                             `json:"pageSize"`
	Status             port.ArticleIndexBackfillStatus `json:"status"`
	Cursor             int                             `json:"cursor"`
	ProcessedArticles  int64                           `json:"processedArticles"`
	IndexedChunks      int64                           `json:"indexedChunks"`
	PauseRequested     bool                            `json:"pauseRequested"`
	LeaseOwner         string                          `json:"leaseOwner,omitempty"`
	LeaseUntil         string                          `json:"leaseUntil,omitempty"`
	LastError          string                          `json:"lastError,omitempty"`
	StartedAt          string                          `json:"startedAt,omitempty"`
	CompletedAt        string                          `json:"completedAt,omitempty"`
	CreatedAt          string                          `json:"createdAt,omitempty"`
	UpdatedAt          string                          `json:"updatedAt,omitempty"`
}

func writeArticleIndexBackfillState(output io.Writer, state port.ArticleIndexBackfillState) error {
	if output == nil {
		return fmt.Errorf("backfill output is nil")
	}
	value := articleIndexBackfillStateOutput{
		ID:                 state.ID,
		IndexUID:           state.IndexUID,
		IndexVersion:       state.IndexVersion,
		Provider:           state.Provider,
		Model:              state.Model,
		ModelVersion:       state.ModelVersion,
		Dimension:          state.Dimension,
		EmbeddingBatchSize: state.EmbeddingBatchSize,
		PageSize:           state.PageSize,
		Status:             state.Status,
		Cursor:             state.Cursor,
		ProcessedArticles:  state.ProcessedArticles,
		IndexedChunks:      state.IndexedChunks,
		PauseRequested:     state.PauseRequested,
		LeaseOwner:         state.LeaseOwner,
		LastError:          state.LastError,
	}
	if !state.LeaseUntil.IsZero() {
		value.LeaseUntil = state.LeaseUntil.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !state.StartedAt.IsZero() {
		value.StartedAt = state.StartedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !state.CompletedAt.IsZero() {
		value.CompletedAt = state.CompletedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !state.CreatedAt.IsZero() {
		value.CreatedAt = state.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !state.UpdatedAt.IsZero() {
		value.UpdatedAt = state.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeArticleIndexPlan(output io.Writer, result task.ArticleIndexPlanResult) error {
	if output == nil {
		return fmt.Errorf("article index plan output is nil")
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
