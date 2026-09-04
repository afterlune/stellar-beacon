<template>
  <el-card class="studio-card">
    <div slot="header" class="header-row">
      <span>AI Studio</span>
      <el-tag v-if="currentReview" size="small" type="info">Run {{ currentReview.runId }}</el-tag>
    </div>

    <el-alert
      title="AI 只生成预览和审核记录，不会自动写入文章。接受、部分接受、拒绝和重新生成都会留下审计记录。"
      type="info"
      :closable="false"
      show-icon />

    <el-form :model="form" label-width="96px" class="studio-form">
      <el-form-item label="操作">
        <el-select v-model="form.operation" placeholder="选择写作操作">
          <el-option label="续写" value="continue" />
          <el-option label="润色" value="polish" />
          <el-option label="摘要" value="summary" />
          <el-option label="标题" value="title" />
          <el-option label="纠错" value="correct" />
        </el-select>
      </el-form-item>
      <el-form-item label="文章 ID">
        <el-input v-model="form.articleId" type="number" placeholder="编辑已有文章时填写；新草稿可留空" />
      </el-form-item>
      <el-form-item label="标题">
        <el-input v-model="form.title" maxlength="256" show-word-limit />
      </el-form-item>
      <el-form-item label="正文">
        <el-input
          v-model="form.content"
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 24 }"
          placeholder="输入要处理的文章正文"
          maxlength="30000"
          show-word-limit />
      </el-form-item>
      <el-form-item label="补充要求">
        <el-input v-model="form.instruction" type="textarea" :rows="3" maxlength="2000" show-word-limit />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="generatePreview">生成预览</el-button>
        <el-button v-if="currentReview" @click="acceptReview">接受</el-button>
        <el-button v-if="currentReview" @click="partialDialog = true">部分接受</el-button>
        <el-button v-if="currentReview" type="warning" @click="rejectDialog = true">拒绝</el-button>
        <el-button v-if="currentReview" @click="regenerateReview">重新生成</el-button>
      </el-form-item>
    </el-form>

    <div v-if="currentReview" class="preview-grid">
      <section>
        <h3>生成预览</h3>
        <pre class="preview-text">{{ currentReview.preview }}</pre>
      </section>
      <section>
        <h3>Diff</h3>
        <pre class="diff-text">{{ currentReview.diff }}</pre>
      </section>
    </div>

    <el-divider>审核记录</el-divider>
    <el-table v-loading="listLoading" :data="reviews" stripe>
      <el-table-column prop="operation" label="操作" width="100" />
      <el-table-column prop="targetId" label="目标" width="100" />
      <el-table-column prop="status" label="状态" width="150" />
      <el-table-column prop="runId" label="Run ID" min-width="180" show-overflow-tooltip />
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="100">
        <template slot-scope="scope">
          <el-button type="text" @click="selectReview(scope.row)">查看</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pagination">
      <el-pagination
        background
        layout="prev, pager, next"
        :current-page="page.current"
        :page-size="page.size"
        :total="page.total"
        @current-change="changePage" />
    </div>

    <el-dialog title="部分接受" :visible.sync="partialDialog" width="640px">
      <el-input v-model="partialContent" type="textarea" :rows="10" maxlength="30000" show-word-limit />
      <span slot="footer">
        <el-button @click="partialDialog = false">取消</el-button>
        <el-button type="primary" @click="partialAccept">记录部分接受</el-button>
      </span>
    </el-dialog>

    <el-dialog title="拒绝生成物" :visible.sync="rejectDialog" width="520px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" maxlength="2000" show-word-limit placeholder="填写拒绝原因" />
      <span slot="footer">
        <el-button @click="rejectDialog = false">取消</el-button>
        <el-button type="danger" @click="rejectReview">记录拒绝</el-button>
      </span>
    </el-dialog>
  </el-card>
</template>

<script>
export default {
  name: 'AIStudio',
  data() {
    return {
      loading: false,
      listLoading: false,
      partialDialog: false,
      rejectDialog: false,
      partialContent: '',
      rejectReason: '',
      currentReview: null,
      reviews: [],
      page: { current: 1, size: 20, total: 0 },
      form: {
        operation: 'polish',
        articleId: '',
        title: '',
        content: '',
        instruction: ''
      }
    }
  },
  created() {
    this.loadReviews()
  },
  methods: {
    generatePreview() {
      if (!this.form.content.trim()) {
        this.$message.error('正文不能为空')
        return Promise.resolve(null)
      }
      this.loading = true
      const payload = {
        operation: this.form.operation,
        title: this.form.title,
        content: this.form.content,
        instruction: this.form.instruction
      }
      if (String(this.form.articleId).trim()) {
        payload.articleId = Number(this.form.articleId)
      }
      return this.axios
        .post('/api/admin/ai/writing/preview', payload)
        .then(({ data }) => {
          if (!data.flag) {
            this.$message.error(data.message)
            return
          }
          const preview = data.data || {}
          this.currentReview = preview
          this.partialContent = preview.preview || ''
          this.loadReviews()
        })
        .catch(() => this.$message.error('生成预览失败'))
        .finally(() => {
          this.loading = false
        })
    },
    loadReviews() {
      this.listLoading = true
      this.axios
        .get('/api/admin/ai/reviews', { params: { current: this.page.current, size: this.page.size } })
        .then(({ data }) => {
          if (!data.flag) {
            this.$message.error(data.message)
            return
          }
          const value = data.data || {}
          this.reviews = Array.isArray(value.records) ? value.records : []
          this.page.total = Number(value.count || 0)
        })
        .catch(() => this.$message.error('加载审核记录失败'))
        .finally(() => {
          this.listLoading = false
        })
    },
    changePage(current) {
      this.page.current = current
      this.loadReviews()
    },
    selectReview(review) {
      this.currentReview = {
        reviewId: review.id,
        runId: review.runId,
        operation: review.operation,
        preview: review.content,
        diff: review.diff
      }
      this.partialContent = review.content || ''
    },
    postAction(action, payload) {
      if (!this.currentReview || !this.currentReview.reviewId) return Promise.resolve()
      return this.axios.post('/api/admin/ai/reviews/' + this.currentReview.reviewId + '/' + action, payload || {})
    },
    acceptReview() {
      this.postAction('approve')
        .then(({ data }) => {
          if (data.flag) {
            this.$message.success('已记录接受')
            this.loadReviews()
          } else this.$message.error(data.message)
        })
        .catch(() => this.$message.error('记录接受失败'))
    },
    partialAccept() {
      if (!this.partialContent.trim()) {
        this.$message.error('部分接受内容不能为空')
        return
      }
      this.postAction('partial', { content: this.partialContent })
        .then(({ data }) => {
          if (data.flag) {
            this.partialDialog = false
            this.$message.success('已记录部分接受')
            this.loadReviews()
          } else this.$message.error(data.message)
        })
        .catch(() => this.$message.error('记录部分接受失败'))
    },
    rejectReview() {
      if (!this.rejectReason.trim()) {
        this.$message.error('拒绝原因不能为空')
        return
      }
      this.postAction('reject', { rejectReason: this.rejectReason })
        .then(({ data }) => {
          if (data.flag) {
            this.rejectDialog = false
            this.$message.success('已记录拒绝')
            this.loadReviews()
          } else this.$message.error(data.message)
        })
        .catch(() => this.$message.error('记录拒绝失败'))
    },
    regenerateReview() {
      const previousReview = this.currentReview
      if (!previousReview || !previousReview.reviewId) return
      this.generatePreview()
        .then(() => {
          if (!this.currentReview || this.currentReview.reviewId === previousReview.reviewId) return null
          return this.axios.post('/api/admin/ai/reviews/' + previousReview.reviewId + '/regenerate', {
            runId: this.currentReview.runId,
            content: this.currentReview.preview
          })
        })
        .then((response) => {
          if (response && response.data && !response.data.flag) {
            this.$message.error(response.data.message)
            return
          }
          if (response) this.$message.success('已记录重新生成')
          this.loadReviews()
        })
        .catch(() => this.$message.error('记录重新生成失败'))
    }
  }
}
</script>

<style scoped>
.studio-card {
  min-height: calc(100vh - 120px);
}
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.studio-form {
  margin-top: 1.5rem;
}
.preview-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}
.preview-grid section {
  min-width: 0;
}
.preview-text,
.diff-text {
  min-height: 180px;
  max-height: 480px;
  overflow: auto;
  padding: 1rem;
  white-space: pre-wrap;
  word-break: break-word;
  background: #f5f7fa;
  border-radius: 4px;
}
.diff-text {
  background: #111827;
  color: #e5e7eb;
}
.pagination {
  margin-top: 1rem;
  text-align: right;
}
@media (max-width: 900px) {
  .preview-grid {
    grid-template-columns: 1fr;
  }
}
</style>
