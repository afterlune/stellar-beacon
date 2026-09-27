import { defineStore } from 'pinia'

export const useCommentStore = defineStore('commentStore', {
  state: () => {
    return {
      recentComment: '' as any,
      type: '' as any,
      topicId: '' as any
    }
  },
  actions: {}
})
