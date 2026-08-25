import { defineStore } from 'pinia'

export const useMetaStore = defineStore('metaStore', {
  state: () => {
    return {
      title: '红白的个人博客'
    }
  }
})
